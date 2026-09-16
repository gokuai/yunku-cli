// upload.go implements the GoKuai chunked file upload protocol.
//
// Protocol flow:
//  1. POST /m-open/1/file/create_file  (signed, excludes filehash/filesize)
//     → state=1: instant done (server already has the file)
//     → state=0: need to upload; response contains server URL and pathhash
//  2. POST {server}/upload_init?org_client_id=...  (no sign)
//     headers: x-gk-upload-{filename,pathhash,filehash,filesize}
//     → HTTP 202: instant done
//     → HTTP 200: response body contains {"session": "..."}
//  3. PUT  {server}/upload_part  (no sign)
//     headers: x-gk-upload-session, x-gk-upload-range ("start-end")
//     body: raw file bytes for that range
//     → HTTP 2xx: chunk accepted
//  4. POST {server}/upload_finish  (no sign)
//     header: x-gk-upload-session
//     → HTTP 2xx: upload complete
//
// On any error after a session is established, POST {server}/upload_abort is called.
package gokuai

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
)

// DefaultChunkSize is the default size of each upload chunk (4 MiB).
const DefaultChunkSize int64 = 4 * 1024 * 1024

// uploadInitResponse is the parsed body of a successful /upload_init (HTTP 200) response.
type uploadInitResponse struct {
	Session string `json:"session"`
}

// ComputeFileSHA1 computes the lower-hex SHA1 digest and byte length of a file.
func ComputeFileSHA1(filePath string) (hash string, size int64, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	h := sha1.New()
	size, err = io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("hashing %s: %w", filePath, err)
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

// UploadFile performs the complete upload flow for a local file:
//  1. Computes SHA1 / file size
//  2. Calls create_file to register the file
//  3. If state=1 (hash match), returns immediately
//  4. Otherwise, runs the upload_init → upload_part(s) → upload_finish sequence
//
// chunkSize controls how many bytes are sent per upload_part call.
// Pass ≤0 to use DefaultChunkSize.
// extraParams are merged into the create_file request (e.g. op_id, op_name, overwrite).
func (c *OrgClient) UploadFile(
	localPath string,
	fullpath string,
	chunkSize int64,
	extraParams map[string]string,
) (*EntFileCreateFileResponse, error) {

	// ── Step 1: compute SHA1 and file size ────────────────────────────────
	filehash, filesize, err := ComputeFileSHA1(localPath)
	if err != nil {
		return nil, err
	}
	filename := filepath.Base(localPath)
	slog.Info("file hash computed",
		"path", localPath,
		"sha1", filehash,
		"size", filesize,
	)

	// ── Step 2: call create_file ──────────────────────────────────────────
	params := map[string]string{
		"fullpath": fullpath,
		"filehash": filehash,
		"filesize": strconv.FormatInt(filesize, 10),
	}
	for k, v := range extraParams {
		params[k] = v
	}

	createResp, err := c.CreateFile(params)
	if err != nil {
		return nil, fmt.Errorf("create_file: %w", err)
	}

	// ── Step 3: instant done (hash match) ─────────────────────────────────
	if int(createResp.State) == 1 {
		slog.Info("upload instant done (hash match)", "fullpath", fullpath)
		return createResp, nil
	}

	if createResp.Server == "" {
		return nil, fmt.Errorf("create_file returned state=0 but no upload server URL")
	}

	serverURL := normalizeUploadServerURL(createResp.Server)

	// ── Step 4: upload_init ───────────────────────────────────────────────
	session, err := c.doUploadInit(serverURL, filename, createResp.Hash, filehash, filesize)
	if err != nil {
		return nil, fmt.Errorf("upload_init: %w", err)
	}
	if session == "" {
		// HTTP 202 → server already has the content
		slog.Info("upload_init instant done", "fullpath", fullpath)
		return createResp, nil
	}

	// ── Step 5: upload parts ──────────────────────────────────────────────
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	if err := c.doUploadParts(serverURL, localPath, filesize, chunkSize, session); err != nil {
		abortErr := c.doUploadAbort(serverURL, session)
		if abortErr != nil {
			slog.Warn("upload_abort failed", "error", abortErr)
		}
		return nil, fmt.Errorf("uploading parts: %w", err)
	}

	// ── Step 6: upload_finish ─────────────────────────────────────────────
	if err := c.doUploadFinish(serverURL, session); err != nil {
		abortErr := c.doUploadAbort(serverURL, session)
		if abortErr != nil {
			slog.Warn("upload_abort failed", "error", abortErr)
		}
		return nil, fmt.Errorf("upload_finish: %w", err)
	}

	slog.Info("upload complete", "fullpath", fullpath, "size", filesize, "chunks", chunksRequired(filesize, chunkSize))
	return createResp, nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// doUploadInit sends POST /upload_init to the upload server.
// Returns the session string (empty string means HTTP 202, instant done).
func (c *OrgClient) doUploadInit(
	serverURL, filename, pathhash, filehash string,
	filesize int64,
) (string, error) {
	reqURL := fmt.Sprintf("%s/upload_init?org_client_id=%s",
		serverURL, url.QueryEscape(c.OrgClientID))

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(nil))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-gk-upload-filename", url.QueryEscape(filename))
	req.Header.Set("x-gk-upload-pathhash", pathhash)
	req.Header.Set("x-gk-upload-filehash", filehash)
	req.Header.Set("x-gk-upload-filesize", strconv.FormatInt(filesize, 10))

	slog.Debug("upload_init request",
		"url", reqURL,
		"pathhash", pathhash,
		"filehash", filehash,
		"filesize", filesize,
	)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", apperrors.NewAPI(fmt.Sprintf("HTTP request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	slog.Debug("upload_init response", "status", resp.StatusCode, "body", string(body))

	switch resp.StatusCode {
	case http.StatusAccepted: // 202 → instant done
		return "", nil
	case http.StatusOK: // 200 → need chunks
		var initResp uploadInitResponse
		if err := json.Unmarshal(body, &initResp); err != nil {
			return "", fmt.Errorf("parsing upload_init response: %w (body: %s)", err, body)
		}
		if initResp.Session == "" {
			return "", apperrors.NewAPI(fmt.Sprintf("upload_init returned empty session (body: %s)", body))
		}
		return initResp.Session, nil
	default:
		return "", apperrors.NewAPI(fmt.Sprintf("upload_init HTTP %d: %s", resp.StatusCode, body))
	}
}

// doUploadParts reads localPath and sends it in chunkSize-byte segments via PUT /upload_part.
func (c *OrgClient) doUploadParts(
	serverURL, localPath string,
	filesize, chunkSize int64,
	session string,
) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	reqURL := fmt.Sprintf("%s/upload_part", serverURL)

	var offset int64
	chunkNum := 0
	total := chunksRequired(filesize, chunkSize)

	for offset < filesize {
		end := offset + chunkSize - 1
		if end >= filesize {
			end = filesize - 1
		}
		size := end - offset + 1

		chunk := make([]byte, size)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return fmt.Errorf("reading chunk %d: %w", chunkNum, err)
		}

		rangeHdr := fmt.Sprintf("%d-%d", offset, end)
		slog.Debug("uploading chunk",
			"chunk", chunkNum+1,
			"total", total,
			"range", rangeHdr,
			"bytes", size,
		)

		req, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewReader(chunk))
		if err != nil {
			return fmt.Errorf("creating upload_part request (chunk %d): %w", chunkNum, err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("x-gk-upload-session", session)
		req.Header.Set("x-gk-upload-range", rangeHdr)
		req.ContentLength = size

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("upload_part %d: %w", chunkNum, err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		slog.Debug("upload_part response",
			"chunk", chunkNum+1,
			"status", resp.StatusCode,
			"body", string(respBody),
		)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return apperrors.NewAPI(fmt.Sprintf("upload_part %d (range %s) returned HTTP %d: %s",
				chunkNum, rangeHdr, resp.StatusCode, respBody))
		}

		offset = end + 1
		chunkNum++
	}

	return nil
}

// doUploadFinish sends POST /upload_finish to complete the upload.
func (c *OrgClient) doUploadFinish(serverURL, session string) error {
	reqURL := fmt.Sprintf("%s/upload_finish", serverURL)

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("x-gk-upload-session", session)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return apperrors.NewAPI(fmt.Sprintf("HTTP request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	slog.Debug("upload_finish response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperrors.NewAPI(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, body))
	}
	return nil
}

// doUploadAbort sends POST /upload_abort to cancel an in-progress upload.
func (c *OrgClient) doUploadAbort(serverURL, session string) error {
	reqURL := fmt.Sprintf("%s/upload_abort", serverURL)

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("x-gk-upload-session", session)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// normalizeUploadServerURL returns a clean base URL from the server field
// returned by create_file. The value may be comma-separated or lack a scheme.
func normalizeUploadServerURL(raw string) string {
	if idx := strings.Index(raw, ","); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimSpace(raw)
	if raw != "" && !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "http://" + raw
	}
	return strings.TrimRight(raw, "/")
}

// ── UserClient upload ─────────────────────────────────────────────────────────

// UploadFile performs the complete upload flow for a local file using the user API.
//
// Protocol differences vs OrgClient.UploadFile:
//   - create_file called via /m-api/2/file/create_file (signed with user secret,
//     filehash/filesize excluded from signature)
//   - upload_init does NOT include org_client_id in URL; instead the access_token
//     is passed via the x-gk-token request header
//
// chunkSize controls chunk size (≤0 uses DefaultChunkSize).
// secret is the user OAuth client_secret used for API request signing.
// extraParams are merged into the create_file request (e.g. overwrite).
func (c *UserClient) UploadFile(
	mountID int,
	localPath string,
	fullpath string,
	chunkSize int64,
	secret string,
	extraParams map[string]string,
) (*CreateFileV2Response, error) {

	// ── Step 1: compute SHA1 and file size ────────────────────────────────
	filehash, filesize, err := ComputeFileSHA1(localPath)
	if err != nil {
		return nil, err
	}
	filename := filepath.Base(localPath)
	slog.Info("file hash computed",
		"path", localPath,
		"sha1", filehash,
		"size", filesize,
	)

	// ── Step 2: create_file (excludes filehash/filesize from sign) ────────
	createResp, err := c.CreateFileForUpload(mountID, fullpath, filehash, filesize, secret, extraParams)
	if err != nil {
		return nil, fmt.Errorf("create_file: %w", err)
	}

	// ── Step 3: instant done (hash match) ─────────────────────────────────
	if int(createResp.State) == 1 {
		slog.Info("upload instant done (hash match)", "fullpath", fullpath)
		return createResp, nil
	}

	if createResp.Server == "" {
		return nil, fmt.Errorf("create_file returned state=0 but no upload server URL")
	}

	serverURL := normalizeUploadServerURL(createResp.Server)

	// ── Step 4: upload_init (user API: x-gk-token instead of ?org_client_id) ──
	session, err := c.doUserUploadInit(serverURL, filename, createResp.Hash, filehash, filesize)
	if err != nil {
		return nil, fmt.Errorf("upload_init: %w", err)
	}
	if session == "" {
		// HTTP 202 → server already has the content
		slog.Info("upload_init instant done", "fullpath", fullpath)
		return createResp, nil
	}

	// ── Step 5: upload parts (shared with OrgClient) ──────────────────────
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	if err := c.doUserUploadParts(serverURL, localPath, filesize, chunkSize, session); err != nil {
		_ = c.doUserUploadAbort(serverURL, session)
		return nil, fmt.Errorf("uploading parts: %w", err)
	}

	// ── Step 6: upload_finish ─────────────────────────────────────────────
	if err := c.doUserUploadFinish(serverURL, session); err != nil {
		_ = c.doUserUploadAbort(serverURL, session)
		return nil, fmt.Errorf("upload_finish: %w", err)
	}

	slog.Info("upload complete",
		"fullpath", fullpath,
		"size", filesize,
		"chunks", chunksRequired(filesize, chunkSize),
	)
	return createResp, nil
}

// doUserUploadInit sends POST /upload_init to the upload server using user API auth.
// The access token is passed via the x-gk-token header (no org_client_id in URL).
// Returns the session string, or "" when HTTP 202 (instant done).
func (c *UserClient) doUserUploadInit(
	serverURL, filename, pathhash, filehash string,
	filesize int64,
) (string, error) {
	reqURL := fmt.Sprintf("%s/upload_init", serverURL) // no ?org_client_id for user API

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(nil))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-gk-upload-filename", url.QueryEscape(filename))
	req.Header.Set("x-gk-upload-pathhash", pathhash)
	req.Header.Set("x-gk-upload-filehash", filehash)
	req.Header.Set("x-gk-upload-filesize", strconv.FormatInt(filesize, 10))
	req.Header.Set("x-gk-token", c.AccessToken) // user API auth

	slog.Debug("user upload_init request",
		"url", reqURL,
		"pathhash", pathhash,
		"filehash", filehash,
		"filesize", filesize,
	)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", apperrors.NewAPI(fmt.Sprintf("HTTP request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	slog.Debug("user upload_init response", "status", resp.StatusCode, "body", string(body))

	switch resp.StatusCode {
	case http.StatusAccepted: // 202 → instant done
		return "", nil
	case http.StatusOK: // 200 → need chunks
		var initResp uploadInitResponse
		if err := json.Unmarshal(body, &initResp); err != nil {
			return "", fmt.Errorf("parsing upload_init response: %w (body: %s)", err, body)
		}
		if initResp.Session == "" {
			return "", apperrors.NewAPI(fmt.Sprintf("upload_init returned empty session (body: %s)", body))
		}
		return initResp.Session, nil
	default:
		return "", apperrors.NewAPI(fmt.Sprintf("upload_init HTTP %d: %s", resp.StatusCode, body))
	}
}

// doUserUploadParts uploads file chunks via PUT /upload_part (same protocol as OrgClient).
func (c *UserClient) doUserUploadParts(
	serverURL, localPath string,
	filesize, chunkSize int64,
	session string,
) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	reqURL := fmt.Sprintf("%s/upload_part", serverURL)
	var offset int64
	chunkNum := 0
	total := chunksRequired(filesize, chunkSize)

	for offset < filesize {
		end := offset + chunkSize - 1
		if end >= filesize {
			end = filesize - 1
		}
		size := end - offset + 1

		chunk := make([]byte, size)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return fmt.Errorf("reading chunk %d: %w", chunkNum, err)
		}

		rangeHdr := fmt.Sprintf("%d-%d", offset, end)
		slog.Debug("uploading chunk",
			"chunk", chunkNum+1,
			"total", total,
			"range", rangeHdr,
			"bytes", size,
		)

		req, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewReader(chunk))
		if err != nil {
			return fmt.Errorf("creating upload_part request (chunk %d): %w", chunkNum, err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("x-gk-upload-session", session)
		req.Header.Set("x-gk-upload-range", rangeHdr)
		req.ContentLength = size

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("upload_part %d: %w", chunkNum, err)
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		slog.Debug("upload_part response",
			"chunk", chunkNum+1,
			"status", resp.StatusCode,
			"body", string(respBody),
		)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return apperrors.NewAPI(fmt.Sprintf("upload_part %d (range %s) returned HTTP %d: %s",
				chunkNum, rangeHdr, resp.StatusCode, respBody))
		}

		offset = end + 1
		chunkNum++
	}
	return nil
}

// doUserUploadFinish sends POST /upload_finish to complete the upload.
func (c *UserClient) doUserUploadFinish(serverURL, session string) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/upload_finish", serverURL), bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("x-gk-upload-session", session)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return apperrors.NewAPI(fmt.Sprintf("HTTP request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	slog.Debug("user upload_finish response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperrors.NewAPI(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, body))
	}
	return nil
}

// doUserUploadAbort sends POST /upload_abort to cancel an in-progress upload.
func (c *UserClient) doUserUploadAbort(serverURL, session string) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/upload_abort", serverURL), bytes.NewReader(nil))
	if err != nil {
		return err
	}
	req.Header.Set("x-gk-upload-session", session)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// chunksRequired returns the number of chunks needed to upload a file.
func chunksRequired(filesize, chunkSize int64) int64 {
	if filesize == 0 {
		return 1
	}
	n := filesize / chunkSize
	if filesize%chunkSize != 0 {
		n++
	}
	return n
}
