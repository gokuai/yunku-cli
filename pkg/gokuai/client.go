package gokuai

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strings"
	"time"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
)

// Client is the gokuai API client
type Client struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	BaseURL      string
}

// NewClient creates a new gokuai client
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{},
		BaseURL:      GetAPIHost(),
	}
}

// APIResponse is the common API response structure
type APIResponse struct {
	ErrorCode int    `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	Result    int    `json:"result,omitempty"`
	Msg       string `json:"msg,omitempty"`
}

// IsSuccess checks if the API call was successful
func (r *APIResponse) IsSuccess() bool {
	return r.ErrorCode == 0 || r.Result == 1
}

// Error returns the error message if any
func (r *APIResponse) Error() string {
	if r.ErrorCode != 0 {
		return fmt.Sprintf("error_code: %d, error_msg: %s", r.ErrorCode, r.ErrorMsg)
	}
	if r.Result == 0 {
		return r.Msg
	}
	return ""
}

// errorCodeCategory classifies a gokuai error_code per the official error
// code reference (https://developer.goukuai.cn/overview/errorcode.html).
// Every documented error_code shares a leading 3-digit HTTP-status-class
// prefix (e.g. 40003 / 4000304 -> "400", 40101 -> "401"):
//   - 400: request data invalid (bad/missing parameters)      -> Validation
//   - 401/403: not authenticated / no permission               -> Auth
//   - 404/405/5xx and anything unrecognized: upstream API fault -> API
func errorCodeCategory(code int) apperrors.Category {
	prefix := fmt.Sprintf("%d", code)
	if len(prefix) > 3 {
		prefix = prefix[:3]
	}
	switch prefix {
	case "400":
		return apperrors.CategoryValidation
	case "401", "403":
		return apperrors.CategoryAuth
	default:
		return apperrors.CategoryAPI
	}
}

// newAPIError builds a categorized error for a gokuai error_code/error_msg
// pair, optionally prefixed with a human-readable context (e.g. "删除部门失败").
func newAPIError(code int, msg, context string) error {
	if context != "" {
		msg = context + ": " + msg
	}
	opt := apperrors.WithReason(fmt.Sprintf("error_code=%d", code))
	switch errorCodeCategory(code) {
	case apperrors.CategoryValidation:
		return apperrors.NewValidation(msg, opt)
	case apperrors.CategoryAuth:
		return apperrors.NewAuth(msg, opt)
	default:
		return apperrors.NewAPI(msg, opt)
	}
}

// apiResponseError builds a categorized error from an APIResponse whose
// IsSuccess() check failed, preserving the existing "<context>: <message>"
// text while classifying the error by error_code.
func apiResponseError(resp *APIResponse, context string) error {
	return newAPIError(resp.ErrorCode, resp.Error(), context)
}

// doRequest performs an HTTP POST request
func (c *Client) doRequest(endpoint string, params map[string]string) ([]byte, error) {
	// Build URL
	reqURL := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	// Build form data
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	// Log request parameters (redact sensitive fields)
	logParams := redactParams(params)
	slog.Debug("gokuai API request",
		"endpoint", endpoint,
		"client_id", c.ClientID,
		"params", logParams,
	)

	// Create request
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	// Log response body
	slog.Debug("gokuai API response",
		"endpoint", endpoint,
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	return body, nil
}

// redactParams returns a copy of params with sensitive fields redacted
func redactParams(params map[string]string) map[string]string {
	if params == nil {
		return nil
	}
	sensitiveKeys := map[string]bool{
		"client_secret": true,
		"password":      true,
		"access_token":  true,
		"refresh_token": true,
		"sign":          true,
	}
	redacted := make(map[string]string)
	for k, v := range params {
		if sensitiveKeys[k] {
			redacted[k] = "***REDACTED***"
		} else {
			redacted[k] = v
		}
	}
	return redacted
}

// doSignedRequest performs a signed POST request
func (c *Client) doSignedRequest(endpoint string, params map[string]string) ([]byte, error) {
	signedParams := BuildSignedParams(params, c.ClientID, c.ClientSecret)
	return c.doRequest(endpoint, signedParams)
}

// CallAPI calls a gokuai API endpoint and returns the response
func (c *Client) CallAPI(endpoint string, params map[string]string) (*APIResponse, error) {
	body, err := c.doSignedRequest(endpoint, params)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CallAPIWithResult calls an API and unmarshals the result into the given type
func (c *Client) CallAPIWithResult(endpoint string, params map[string]string, result interface{}) error {
	body, err := c.doSignedRequest(endpoint, params)
	if err != nil {
		return err
	}

	// First check for error response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err == nil {
		if !apiResp.IsSuccess() {
			return apiResponseError(&apiResp, "API error")
		}
	}

	// Try to unmarshal into result
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return nil
}

// === User API Client ===

// UserClient is the gokuai user API client (uses access_token for auth)
// TokenInvalidErrorCode 是 access token 失效时服务端返回的 error_code。
// 收到该 code 时可通过 refresh token 换取新 token 后重试。
const TokenInvalidErrorCode = 40102

type UserClient struct {
	AccessToken string
	BearerToken string
	HTTPClient  *http.Client
	BaseURL     string

	// TokenRefresher 在 access token 失效(40102)时被调用，返回新的 access token。
	// 为 nil 时不进行自动刷新重试。设置后 doUserRequest 会在收到 40102 时
	// 刷新一次并重放请求；刷新失败则返回原始错误。
	TokenRefresher func() (string, error)
}

// NewUserClient creates a new user API client
func NewUserClient(accessToken string) *UserClient {
	return &UserClient{
		AccessToken: accessToken,
		HTTPClient:  &http.Client{},
		BaseURL:     GetAPIHost(),
	}
}

// NewBearerTokenUserClient creates a user API client that uses bearer token authentication.
// When BearerToken is set, requests skip signing and token params, and instead send
// Authorization and X-Auth-Type headers.
func NewBearerTokenUserClient(bearerToken string) *UserClient {
	return &UserClient{
		BearerToken: bearerToken,
		HTTPClient:  &http.Client{},
		BaseURL:     GetAPIHost(),
	}
}

// userSign generates the signature for user API requests
// Algorithm: hmac-sha1(joined_values, secret) with base64 encoding
// Values are sorted by key and joined with \n
func (c *UserClient) userSign(params map[string]string, secret string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build value string with \n separator (no URL encoding)
	var vals []string
	for _, k := range keys {
		if params[k] == "" {
			continue
		}
		vals = append(vals, params[k])
	}
	valueString := strings.Join(vals, "\n")

	// HMAC-SHA1 with base64 encoding
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(valueString))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// doUserRequest performs a signed GET request for user API
func (c *UserClient) doUserRequest(endpoint string, params map[string]string, secret string) ([]byte, error) {
	// Work on a copy so a retry with a refreshed token can re-sign cleanly
	// without stale token/sign values from the first attempt.
	signed := cloneParams(params)

	if c.BearerToken != "" {
		// Bearer token mode: skip signing and token, use headers instead
	} else {
		// Add access_token
		signed["token"] = c.AccessToken

		// Calculate sign using the provided secret
		signed["sign"] = c.userSign(signed, secret)
	}

	body, err := c.doSignedUserGet(endpoint, signed)
	if err != nil {
		return nil, err
	}

	// On token-invalid responses, refresh once and replay the request.
	if c.TokenRefresher != nil && isTokenInvalidResponse(body) {
		newToken, refreshErr := c.TokenRefresher()
		if refreshErr != nil || newToken == "" {
			slog.Debug("gokuai token refresh failed", "error", refreshErr)
			return body, nil
		}
		c.AccessToken = newToken

		signed = cloneParams(params)
		signed["token"] = newToken
		signed["sign"] = c.userSign(signed, secret)

		body, err = c.doSignedUserGet(endpoint, signed)
		if err != nil {
			return nil, err
		}
	}

	return body, nil
}

// doSignedUserGet executes a single GET request with the already-signed params.
func (c *UserClient) doSignedUserGet(endpoint string, params map[string]string) ([]byte, error) {
	// Build URL with query string
	reqURL := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	// Log request parameters (redact sensitive fields)
	logParams := redactParams(params)
	slog.Debug("gokuai user API request",
		"endpoint", endpoint,
		"params", logParams,
	)

	// Create GET request
	req, err := http.NewRequest("GET", reqURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add bearer token headers if in bearer token mode
	if c.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.BearerToken)
		req.Header.Set("X-Auth-Type", "mcp")
	}

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	// Log response body
	slog.Debug("gokuai user API response",
		"endpoint", endpoint,
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	return body, nil
}

// isTokenInvalidResponse reports whether the response body carries the
// access-token-invalid error code (40102).
func isTokenInvalidResponse(body []byte) bool {
	var errResp struct {
		ErrorCode int `json:"error_code"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return false
	}
	return errResp.ErrorCode == TokenInvalidErrorCode
}

// GetMountList 获取文件库列表 (用户 API)
// secret: 用户OAuth client_secret
func (c *UserClient) GetMountList(secret string) (*MountListResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/mount", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp MountListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// FileListResponseV2 is the response for user file list API
type FileListResponseV2 struct {
	Count FlexInt      `json:"count,omitempty"`
	List  []FileInfoV2 `json:"list"`
}

// FileInfoV2 represents a file for user API
type FileInfoV2 struct {
	Hash             string    `json:"hash"`
	MountID          FlexInt   `json:"mount_id"`
	Dir              FlexInt   `json:"dir"`
	Fullpath         string    `json:"fullpath"`
	Filename         string    `json:"filename"`
	Filehash         string    `json:"filehash"`
	Filesize         FlexInt64 `json:"filesize"`
	CreateMemberID   FlexInt   `json:"create_member_id"`
	CreateMemberName string    `json:"create_member_name"`
	CreateDateline   FlexInt   `json:"create_dateline"`
	LastMemberID     FlexInt   `json:"last_member_id"`
	LastMemberName   string    `json:"last_member_name"`
	LastDateline     FlexInt   `json:"last_dateline"`
	Thumbnail        string    `json:"thumbnail"`
	Lock             FlexInt   `json:"lock"`
}

// FileInfoResponseV2 is the response for user file info API
type FileInfoResponseV2 struct {
	Hash             string          `json:"hash"`
	MountID          FlexInt         `json:"mount_id"`
	Dir              FlexInt         `json:"dir"`
	Fullpath         string          `json:"fullpath"`
	Filename         string          `json:"filename"`
	Filehash         string          `json:"filehash"`
	Filesize         FlexInt64       `json:"filesize"`
	CreateMemberID   FlexInt         `json:"create_member_id"`
	CreateMemberName string          `json:"create_member_name"`
	CreateDateline   FlexInt         `json:"create_dateline"`
	LastMemberID     FlexInt         `json:"last_member_id"`
	LastMemberName   string          `json:"last_member_name"`
	LastDateline     FlexInt         `json:"last_dateline"`
	Lock             FlexInt         `json:"lock"`
	Favorite         json.RawMessage `json:"favorite,omitempty"`
	URI              string          `json:"uri,omitempty"`
	Preview          string          `json:"preview"`
	Thumbnail        string          `json:"thumbnail"`
	Property         EmbeddedJSON    `json:"property"`
}

// FileAttributeResponse is the response for file attribute API
type FileAttributeResponse struct {
	FileCount   FlexInt `json:"file_count"`
	FolderCount FlexInt `json:"folder_count"`
	FileSize    FlexInt `json:"file_size"`
}

// CreateFolderResponseV2 is the response for user create folder API
type CreateFolderResponseV2 struct {
	Fullpath string `json:"fullpath"`
	Hash     string `json:"hash"`
}

// DownloadURLResponseV2 is the response for user file open/download API
type DownloadURLResponseV2 struct {
	Hash     string    `json:"hash"`
	Filehash string    `json:"filehash"`
	Filesize FlexInt64 `json:"filesize"`
	Lock     FlexInt   `json:"lock"`
	Uris     []string  `json:"uris"`
}

// PreviewURLResponseV2 is the response for user preview URL API
type PreviewURLResponseV2 struct {
	URL string `json:"url"`
}

// CopyMoveResponse is the response for copy/move API
type CopyMoveResponse struct {
	Hash     string `json:"hash,omitempty"`
	Fullpath string `json:"fullpath,omitempty"`
}

// RenameResponse is the response for rename API
type RenameResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
}

// FileUpdateResponseV2 is a group of file updates returned by /2/updates/file_updates API
type FileUpdateResponseV2 struct {
	ToDatelineMS   FlexInt64               `json:"to_datelinems"`
	FromDatelineMS FlexInt64               `json:"from_datelinems"`
	ActMemberID    FlexInt                 `json:"act_member_id"`
	ActCount       FlexInt                 `json:"act_count"`
	ListCount      FlexInt                 `json:"list_count"`
	Info           []FileUpdateInfoGroupV2 `json:"info"`
}

// FileUpdateInfoGroupV2 groups file updates by action type
type FileUpdateInfoGroupV2 struct {
	List           []FileUpdateItemV2 `json:"list"`
	Fullpath       []string           `json:"fullpath"`
	FullpathCount  FlexInt            `json:"fullpath_count"`
	Count          FlexInt            `json:"count"`
	ToDatelineMS   FlexInt64          `json:"to_datelinems"`
	FromDatelineMS FlexInt64          `json:"from_datelinems"`
	Name           string             `json:"name"`
	Act            FlexInt            `json:"act"`
}

// FileUpdateItemV2 represents a single file update event
type FileUpdateItemV2 struct {
	ID            string                 `json:"_id"`
	EntID         string                 `json:"ent_id"`
	MountID       FlexInt                `json:"mount_id"`
	StoragePoint  string                 `json:"storage_point"`
	Type          FlexInt                `json:"type"`
	Act           FlexInt                `json:"act"`
	Hash          string                 `json:"hash"`
	Dir           FlexInt                `json:"dir"`
	Fullpath      string                 `json:"fullpath"`
	Filename      string                 `json:"filename"`
	Filehash      string                 `json:"filehash"`
	Filesize      FlexInt64              `json:"filesize"`
	Version       FlexInt                `json:"version"`
	MemberID      FlexInt                `json:"member_id"`
	Dateline      FlexInt64              `json:"dateline"`
	DatelineMS    FlexInt64              `json:"datelinems"`
	Del           bool                   `json:"del"`
	Message       string                 `json:"message"`
	Property      map[string]interface{} `json:"property"`
	ActMemberID   FlexInt                `json:"act_member_id"`
	Sort          FlexInt                `json:"sort"`
	Opts          []interface{}          `json:"opts"`
	OrgID         string                 `json:"org_id"`
	OrgName       string                 `json:"org_name"`
	OrgLogo       string                 `json:"org_logo"`
	OrgAvatar     string                 `json:"org_avatar"`
	MemberName    string                 `json:"member_name"`
	MemberPhoto   string                 `json:"member_photo"`
	RenderText    string                 `json:"render_text"`
	ActName       string                 `json:"act_name"`
	DateTxt       string                 `json:"date_txt"`
	TimeAgo       string                 `json:"timeago"`
	RenderContent string                 `json:"render_content"`
	Unread        FlexInt                `json:"unread"`
}

// GetFileList 获取文件列表 (用户 API)
func (c *UserClient) GetFileList(mountID int, params map[string]string, secret string) (*FileListResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/ls", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileListResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetFileInfo 获取文件(夹)信息 (用户 API)
func (c *UserClient) GetFileInfo(mountID int, params map[string]string, secret string) (*FileInfoResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/info", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileInfoResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SearchFiles 搜索文件 (用户 API)
func (c *UserClient) SearchFiles(mountID int, params map[string]string, secret string) (*FileListResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/search", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileListResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetDownloadURL 获取文件下载地址 (用户 API)
func (c *UserClient) GetDownloadURL(mountID int, params map[string]string, secret string) (*DownloadURLResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/open", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp DownloadURLResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetPreviewURL 获取文件预览地址 (用户 API)
func (c *UserClient) GetPreviewURL(mountID int, params map[string]string, secret string) (*PreviewURLResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/preview_url", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp PreviewURLResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFolder 创建文件夹 (用户 API)
func (c *UserClient) CreateFolder(mountID int, params map[string]string, secret string) (*CreateFolderResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/create_folder", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp CreateFolderResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CopyFile 复制文件 (用户 API)
func (c *UserClient) CopyFile(mountID int, params map[string]string, secret string) (json.RawMessage, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/copy", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	return body, nil
}

// MoveFile 移动文件 (用户 API)
func (c *UserClient) MoveFile(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/move", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// RenameFile 重命名文件 (用户 API)
func (c *UserClient) RenameFile(mountID int, params map[string]string, secret string) (*RenameResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/rename", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp RenameResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// DeleteFile 删除文件 (用户 API)
func (c *UserClient) DeleteFile(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/del", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// GetFileUpdates 获取文件最近更新列表 (用户 API)
func (c *UserClient) GetFileUpdates(mountID int, params map[string]string, secret string) ([]FileUpdateResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/updates/file_updates", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	// 接口在「没有更新」时返回成功对象 {"error_code":0}（而非空数组），
	// 直接反序列化到 []FileUpdateResponseV2 会失败。把这种对象响应当作
	// 「无更新」处理，返回空切片。
	if trimmed := bytes.TrimSpace(body); len(trimmed) > 0 && trimmed[0] == '{' {
		return []FileUpdateResponseV2{}, nil
	}

	var resp []FileUpdateResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetFileStat 获取统计信息 (用户 API)
func (c *UserClient) GetFileStat(secret string) (map[string]interface{}, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/file/stat", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetFileLinks 获取文件外链列表 (用户 API, GET /m-api/1/file/file_link_list)
func (c *UserClient) GetFileLinks(mountID int, params map[string]string, secret string) (*FileLinkListResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/file_link_list", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileLinkListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFileLinkResponse 创建文件外链响应
type CreateFileLinkResponse struct {
	Code  string `json:"code"`
	Link  string `json:"link"`
	QRURL string `json:"qr_url"`
}

// CreateFileLink 创建文件外链 (用户 API, POST /m-api/1/file/create_file_link)
func (c *UserClient) CreateFileLink(mountID int, params map[string]string, secret string) (*CreateFileLinkResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/file/create_file_link", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp CreateFileLinkResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CloseFileLink 关闭文件外链 (用户 API, POST /m-api/1/file/close_file_link)
func (c *UserClient) CloseFileLink(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/file/close_file_link", p, secret)
	if err != nil {
		return err
	}

	return checkAPIError(body)
}

// FileLockRequest 文件锁请求参数
type FileLockRequest struct {
	MountID  int
	Fullpath string
	Lock     string // "lock" or "unlock"
}

// LockFile 上锁文件 (用户 API)
func (c *UserClient) LockFile(req FileLockRequest, secret string) error {
	params := map[string]string{
		"mount_id": fmt.Sprintf("%d", req.MountID),
		"fullpath": req.Fullpath,
		"lock":     req.Lock,
	}

	body, err := c.doUserRequest("/m-api/1/file/lock", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// FileTagRequest 文件标签请求参数
type FileTagRequest struct {
	MountID  int
	Fullpath string
	Tag      string
}

// AddFileTag 添加文件标签 (用户 API)
func (c *UserClient) AddFileTag(req FileTagRequest, secret string) error {
	params := map[string]string{
		"mount_id": fmt.Sprintf("%d", req.MountID),
		"fullpath": req.Fullpath,
		"tag":      req.Tag,
	}

	body, err := c.doUserRequest("/m-api/1/file/add_tag", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// DeleteFileTag 删除文件标签 (用户 API)
func (c *UserClient) DeleteFileTag(req FileTagRequest, secret string) error {
	params := map[string]string{
		"mount_id": fmt.Sprintf("%d", req.MountID),
		"fullpath": req.Fullpath,
		"tag":      req.Tag,
	}

	body, err := c.doUserRequest("/m-api/1/file/del_tag", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// FavoritesRequest 收藏夹请求参数
type FavoritesRequest struct {
	MountID  int
	Fullpath string
	FavID    int // -1 for default favorites
}

// AddToFavorites 添加文件到收藏夹 (用户 API)
func (c *UserClient) AddToFavorites(req FavoritesRequest, secret string) error {
	params := map[string]string{
		"mount_id": fmt.Sprintf("%d", req.MountID),
		"fullpath": req.Fullpath,
		"fav_id":   fmt.Sprintf("%d", req.FavID),
	}

	body, err := c.doUserRequest("/m-api/1/favorites/add_file", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// RemoveFromFavorites 从收藏夹移除 (用户 API)
func (c *UserClient) RemoveFromFavorites(req FavoritesRequest, secret string) error {
	params := map[string]string{
		"mount_id": fmt.Sprintf("%d", req.MountID),
		"fullpath": req.Fullpath,
		"fav_id":   fmt.Sprintf("%d", req.FavID),
	}

	body, err := c.doUserRequest("/m-api/1/favorites/del_file", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// GetFavorites 获取收藏夹文件列表 (用户 API)
func (c *UserClient) GetFavorites(favID int, start, size int, secret string) (*FileListResponseV2, error) {
	params := map[string]string{
		"fav_id": fmt.Sprintf("%d", favID),
		"start":  fmt.Sprintf("%d", start),
		"size":   fmt.Sprintf("%d", size),
	}

	body, err := c.doUserRequest("/m-api/1/favorites/get_files", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileListResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateLibraryResponse is the response for create library API
type CreateLibraryResponse struct {
	OrgID        FlexInt `json:"org_id"`
	MountID      FlexInt `json:"mount_id"`
	StoragePoint string  `json:"storage_point"`
}

// CreateLibrary 创建文件库 (用户 API)
func (c *UserClient) CreateLibrary(params map[string]string, secret string) (*CreateLibraryResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/create", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp CreateLibraryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetRecycleList 获取回收站列表 (用户 API)
func (c *UserClient) GetRecycleList(mountID int, params map[string]string, secret string) (*RecycleResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/recycle", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp RecycleResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// RecoverFiles 恢复已删除文件 (用户 API)
func (c *UserClient) RecoverFiles(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/recover", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// DeleteCompletely 彻底删除文件 (用户 API)
func (c *UserClient) DeleteCompletely(mountID int, params map[string]string, secret string) (*QueueIDResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/del_completely", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp QueueIDResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// ClearRecycle 清空回收站 (用户 API)
func (c *UserClient) ClearRecycle(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/clear", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetFileHistoryV2 获取文件历史 (用户 API v2)
func (c *UserClient) GetFileHistoryV2(mountID int, params map[string]string, secret string) (*FileHistoryResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/history", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileHistoryResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// RevertFileHistory 还原历史版本 (用户 API)
func (c *UserClient) RevertFileHistory(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/revert", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetOrgStat 获取库统计信息 (用户 API)
func (c *UserClient) GetOrgStat(mountID int, secret string) (*StatResponse, error) {
	params := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/file/stat", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp StatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// LibraryInfoResponse is the response for library info API
type LibraryInfoResponse struct {
	OrgID               FlexInt   `json:"org_id"`
	MountID             FlexInt   `json:"mount_id"`
	EntID               FlexInt   `json:"ent_id"`
	MemberID            FlexInt   `json:"member_id"`
	StoragePoint        string    `json:"storage_point"`
	StorageCache        FlexInt   `json:"storage_cahce"`
	OrgName             string    `json:"org_name"`
	OrgDescription      string    `json:"org_description"`
	OrgLogo             string    `json:"org_logo"`
	OrgLogoURL          string    `json:"org_logo_url"`
	SizeUse             FlexInt64 `json:"size_use"`
	SizeTotal           FlexInt64 `json:"size_total"`
	MemberLimit         FlexInt   `json:"member_limit"`
	MemberCount         FlexInt   `json:"member_count"`
	ExternalMemberCount FlexInt   `json:"external_member_count"`
	GroupCount          FlexInt   `json:"group_count"`
	AddDateline         FlexInt   `json:"add_dataline"`
	CanManage           FlexInt   `json:"can_manage"`
	CanQuit             FlexInt   `json:"can_quit"`
	CanDelete           FlexInt   `json:"can_delete"`
	FileCount           FlexInt   `json:"file_count"`
	FolderCount         FlexInt   `json:"folder_count"`
}

// LibraryMembersResponse is the response for library members API
type LibraryMembersResponse struct {
	List  []LibraryMemberInfo `json:"list"`
	Count FlexInt             `json:"count"`
}

// LibraryMemberInfo represents a library member
type LibraryMemberInfo struct {
	OrgID         FlexInt   `json:"org_id"`
	MountID       FlexInt   `json:"mount_id"`
	EntID         FlexInt   `json:"ent_id"`
	MemberID      FlexInt   `json:"member_id"`
	MemberName    string    `json:"member_name"`
	MemberLetter  string    `json:"member_letter"`
	MemberEmail   string    `json:"member_email"`
	MemberType    FlexInt   `json:"member_type"`
	RoleID        FlexInt   `json:"role_id"`
	RoleIDs       []string  `json:"role_ids"`
	IsStand       FlexInt   `json:"is_stand"`
	State         FlexInt   `json:"state"`
	AddTime       FlexInt64 `json:"addtime"`
	GroupFullpath string    `json:"group_fullpath"`
}

func (c *UserClient) GetLibraryInfo(orgID int, secret string) (*LibraryInfoResponse, error) {
	params := map[string]string{
		"org_id":   fmt.Sprintf("%d", orgID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/library/info", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp LibraryInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetLibraryMembers 获取库成员列表 (用户 API)
func (c *UserClient) GetLibraryMembers(orgID int, params map[string]string, secret string) (*LibraryMembersResponse, error) {
	p := map[string]string{
		"org_id":   fmt.Sprintf("%d", orgID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/members", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp LibraryMembersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SearchLibraryMembersByKeyword 按关键字搜索库成员 (用户 API)
// 使用 mount_id 标识库（区别于 GetLibraryMembers 的 org_id），
// 用于 @成员 提及解析（keyword=成员名, size=1, start=0）。
func (c *UserClient) SearchLibraryMembersByKeyword(mountID int, keyword, secret string) (*LibraryMembersResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"keyword":  keyword,
		"size":     "1",
		"start":    "0",
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/library/members", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp LibraryMembersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetClientOAuth 获取用户授权的 client_id 和 client_secret
// 这是企业授权接口，需要先在企业管理后台创建设备应用获得 client_id 和 client_secret
func (c *Client) GetClientOAuth() (*ClientOAuthResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_client_oauth", params)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ClientOAuthResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if resp.Result != 1 {
		return nil, fmt.Errorf("获取授权失败: %s", resp.Msg)
	}

	return &resp, nil
}

// === Account APIs (User) ===

// AccountInfoResponse is the response for account info API
type AccountInfoResponse struct {
	UUID          string            `json:"uuid"`
	MemberID      FlexInt           `json:"member_id"`
	MemberEmail   string            `json:"member_email"`
	MemberName    string            `json:"member_name"`
	MemberPhone   string            `json:"member_phone"`
	MemberAccount string            `json:"member_account"`
	Avatar        string            `json:"avatar"`
	Language      string            `json:"language"`
	Validate      FlexInt           `json:"validate"`
	Settings      FlexStringMap     `json:"settings"`
	FavoriteNames map[string]string `json:"favorite_names"`
	Favorites     []FavoriteItem    `json:"favorites"`
	YunkuCount    FlexInt           `json:"yunku_count"`
	Oauths        []string          `json:"oauths"`
	SiteURL       string            `json:"site_url"`
	Property      *AccountProperty  `json:"property"`
	Source        string            `json:"source"`
}

// AccountProperty 账户属性
type AccountProperty struct {
	EditPassword FlexInt `json:"edit_password"`
}

// FavoriteItem represents a favorite collection
type FavoriteItem struct {
	ID    int    `json:"id"`
	Type  int    `json:"type"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GetAccountInfo 获取用户相关信息 (用户 API)
func (c *UserClient) GetAccountInfo(secret string) (*AccountInfoResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/info", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AccountInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// FindPassword 找回密码 (用户 API)
func (c *UserClient) FindPassword(email string, secret string) error {
	params := map[string]string{
		"email":    email,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/findpassword", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// SetAccountInfoResponse 设置用户相关信息响应
type SetAccountInfoResponse struct {
	MemberName string `json:"member_name"`
	AvatarURL  string `json:"avatar_url"`
}

// SetAccountInfo 设置用户相关信息 (用户 API, POST /m-api/1/account/set_info)
// 当 fileBytes 非空时使用 multipart/form-data 上传头像文件
func (c *UserClient) SetAccountInfo(params map[string]string, fileBytes []byte, fileName string, secret string) (*SetAccountInfoResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	// 无文件上传: 使用普通请求
	if len(fileBytes) == 0 {
		body, err := c.doUserRequest("/m-api/1/account/set_info", p, secret)
		if err != nil {
			return nil, err
		}
		if err := checkAPIError(body); err != nil {
			return nil, err
		}
		var resp SetAccountInfoResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
		}
		return &resp, nil
	}

	// 有文件上传: 使用 multipart/form-data
	reqURL := fmt.Sprintf("%s/m-api/1/account/set_info", c.BaseURL)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 写入文本字段 (dateline, member_name, mobile, language)
	for k, v := range p {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("failed to write %s field: %w", k, err)
		}
	}

	// 写入 token
	if err := writer.WriteField("token", c.AccessToken); err != nil {
		return nil, fmt.Errorf("failed to write token field: %w", err)
	}

	// 计算签名: 包含所有文本字段 + token (file 不参与签名)
	signParams := map[string]string{}
	for k, v := range p {
		signParams[k] = v
	}
	signParams["token"] = c.AccessToken
	sign := c.userSign(signParams, secret)
	if err := writer.WriteField("sign", sign); err != nil {
		return nil, fmt.Errorf("failed to write sign field: %w", err)
	}

	// 写入文件字段，显式设置 Content-Type 以便服务端正确识别图片格式
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	h.Set("Content-Type", http.DetectContentType(fileBytes))
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	slog.Debug("gokuai user API set_info request",
		"endpoint", "/m-api/1/account/set_info",
		"file_name", fileName,
	)

	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	slog.Debug("gokuai user API set_info response",
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var result SetAccountInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// AccountEntInfoResponse is the response for account ent info API
type AccountEntInfoResponse struct {
	EntID         FlexInt     `json:"ent_id"`
	Name          string      `json:"name"`
	Domain        string      `json:"domain"`
	ProductID     FlexInt     `json:"product_id"`
	EndDateline   FlexInt     `json:"end_dateline"`
	MemberCount   FlexInt     `json:"member_count"`
	MemberLimit   FlexInt     `json:"member_limit"`
	AddTime       FlexInt     `json:"addtime"`
	Logo          string      `json:"logo"`
	LogoURL       string      `json:"logo_url"`
	OrgCount      FlexInt     `json:"org_count"`
	OrgLimit      FlexInt     `json:"org_limit"`
	Backend       FlexInt     `json:"backend"`
	MemberCreates interface{} `json:"member_creates"`
	TotalSpace    FlexInt     `json:"total_space"`
	UsedSpace     FlexInt     `json:"used_space"`
}

// GetAccountEntInfo 获取企业信息 (用户 API)
func (c *UserClient) GetAccountEntInfo(entID string, secret string) (*AccountEntInfoResponse, error) {
	params := map[string]string{
		"ent_id":   entID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/ent_info", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AccountEntInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// DeviceInfo represents device information
type DeviceInfo struct {
	DeviceID        FlexInt `json:"device_id"`
	DeviceName      string  `json:"device_name"`
	OSName          string  `json:"os_name"`
	OSVersion       string  `json:"os_version"`
	LastActivity    string  `json:"last_activity"`
	AllowEdit       FlexInt `json:"allow_edit"`
	AllowDelete     FlexInt `json:"allow_delete"`
	IsCurrentDevice FlexInt `json:"is_current_device"`
	State           FlexInt `json:"state"`
}

// DeviceListResponse is the response for device list API
type DeviceListResponse struct {
	Devices []DeviceInfo `json:"devices"`
}

// GetDeviceList 获取设备列表 (用户 API)
func (c *UserClient) GetDeviceList(secret string) (*DeviceListResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/device_list", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp DeviceListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// ToggleDevice 启用/禁用设备 (用户 API)
func (c *UserClient) ToggleDevice(deviceID string, state string, secret string) error {
	params := map[string]string{
		"device_id": deviceID,
		"state":     state,
		"dateline":  fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/toggle_device", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// DeleteDevice 删除设备 (用户 API)
func (c *UserClient) DeleteDevice(deviceID string, secret string) error {
	params := map[string]string{
		"device_id": deviceID,
		"dateline":  fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/del_device", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// DisableNewDevice 启用/禁用"禁用新设备" (用户 API)
func (c *UserClient) DisableNewDevice(state string, secret string) error {
	params := map[string]string{
		"state":    state,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/disable_new_device", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// ChangePassword 修改密码 (用户 API)
func (c *UserClient) ChangePassword(oldPassword, newPassword, refreshToken, secret string) error {
	// 旧密码需要 MD5 加密
	oldPasswordMD5 := fmt.Sprintf("%x", md5.Sum([]byte(oldPassword)))

	params := map[string]string{
		"old_password":  oldPasswordMD5,
		"new_password":  newPassword,
		"refresh_token": refreshToken,
		"dateline":      fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/changepassword", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// ServerInfo represents server information
type ServerInfo struct {
	Host       string  `json:"host"`
	Hostname   string  `json:"hostname"`
	HostnameIn string  `json:"hostname-in"`
	Port       FlexInt `json:"port"`
	HTTPS      FlexInt `json:"https"`
	Path       string  `json:"path"`
	Sign       string  `json:"sign"`
	Dateline   int     `json:"dateline"`
}

// GetServers 获取服务器列表 (用户 API)
func (c *UserClient) GetServers(serverType, storagePoint, secret string) ([]ServerInfo, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	if serverType != "" {
		params["type"] = serverType
	}
	if storagePoint != "" {
		params["storage_point"] = storagePoint
	}

	body, err := c.doUserRequest("/m-api/1/account/servers", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp []ServerInfo
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// SettingItem represents a setting item
type SettingItem struct {
	Setting  string `json:"setting"`
	Value    int    `json:"value"`
	Property string `json:"property"`
}

// SettingResponse is the response for setting API
type SettingResponse struct {
	List []SettingItem `json:"list"`
}

// GetSettings 获取客户端设置 (用户 API)
func (c *UserClient) GetSettings(secret string) (*SettingResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/setting", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp SettingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// ResendMail 发送邮箱验证链接 (用户 API)
func (c *UserClient) ResendMail(email, secret string) error {
	params := map[string]string{
		"email":    email,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/resend_mail", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// BindMail 绑定邮箱地址 (用户 API)
func (c *UserClient) BindMail(email, secret string) error {
	params := map[string]string{
		"email":    email,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/bind_mail", params, secret)
	if err != nil {
		return err
	}

	return checkAPIError(body)
}

// UpdateEnt 更新企业信息 (用户 API)
func (c *UserClient) UpdateEnt(entID, name, logo, secret string) error {
	params := map[string]string{
		"ent_id":   entID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	if name != "" {
		params["name"] = name
	}
	if logo != "" {
		params["logo"] = logo
	}

	body, err := c.doUserRequest("/m-api/1/account/update_ent", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// UploadAvatarResponse is the response for upload avatar API
type UploadAvatarResponse struct {
	URI       string `json:"uri"`
	AvatarURL string `json:"avatar_url"`
}

// UploadAvatar 上传头像 (用户 API, POST multipart/form-data)
func (c *UserClient) UploadAvatar(fileBytes []byte, fileName, avatarType, secret string) (*UploadAvatarResponse, error) {
	reqURL := fmt.Sprintf("%s/m-api/1/account/upload_avatar", c.BaseURL)

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add type field
	if err := writer.WriteField("type", avatarType); err != nil {
		return nil, fmt.Errorf("failed to write type field: %w", err)
	}

	// Add file field with correct Content-Type for image
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	h.Set("Content-Type", http.DetectContentType(fileBytes))
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	// Add token and sign fields
	if err := writer.WriteField("token", c.AccessToken); err != nil {
		return nil, fmt.Errorf("failed to write token field: %w", err)
	}

	// Sign: sign the params (type + token) before closing the multipart writer
	params := map[string]string{
		"type":  avatarType,
		"token": c.AccessToken,
	}
	sign := c.userSign(params, secret)
	if err := writer.WriteField("sign", sign); err != nil {
		return nil, fmt.Errorf("failed to write sign field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	slog.Debug("gokuai user API upload_avatar request",
		"endpoint", "/m-api/1/account/upload_avatar",
		"file", fileName,
		"type", avatarType,
	)

	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	slog.Debug("gokuai user API upload_avatar response",
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var result UploadAvatarResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &result, nil
}

// SoftwareInfo represents software information
type SoftwareInfo struct {
	URL         string                 `json:"url"`
	QRURL       string                 `json:"qr_url"`
	AppStoreURL string                 `json:"app_store_url"`
	DeviceName  string                 `json:"device_name"`
	Settings    map[string]interface{} `json:"settings"`
}

// GetSoftware 获取客户端列表 (用户 API)
func (c *UserClient) GetSoftware(secret string) ([]SoftwareInfo, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/software", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp []SoftwareInfo
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// === Library APIs (User) ===

// LibraryGroupInfo represents a library group/department
type LibraryGroupInfo struct {
	OrgID   FlexInt `json:"org_id"`
	Name    string  `json:"name"`
	Count   FlexInt `json:"count"`
	MountID FlexInt `json:"mount_id"`
	EntID   FlexInt `json:"ent_id"`
	GroupID FlexInt `json:"group_id"`
	RoleID  FlexInt `json:"role_id"`
	AddTime FlexInt `json:"addtime"`
	Path    string  `json:"path"`
}

// LibraryGroupsResponse is the response for library groups API
type LibraryGroupsResponse struct {
	List []LibraryGroupInfo `json:"list"`
}

// GetLibraryGroups 获取库部门列表 (用户 API)
func (c *UserClient) GetLibraryGroups(orgID string, withInfo bool, secret string) (*LibraryGroupsResponse, error) {
	params := map[string]string{
		"org_id":   orgID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	if withInfo {
		params["with_info"] = "1"
	}

	body, err := c.doUserRequest("/m-api/1/library/groups", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp LibraryGroupsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddLibraryGroupResponse is the response for add library group API
type AddLibraryGroupResponse struct {
	OrgID   FlexInt `json:"org_id"`
	MountID FlexInt `json:"mount_id"`
	EntID   FlexInt `json:"ent_id"`
	GroupID FlexInt `json:"group_id"`
	RoleID  FlexInt `json:"role_id"`
	AddTime FlexInt `json:"addtime"`
	Path    string  `json:"path"`
}

// AddLibraryGroup 添加库部门 (用户 API)
func (c *UserClient) AddLibraryGroup(params map[string]string, secret string) (*AddLibraryGroupResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/add_group", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AddLibraryGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddLibraryGroups 添加多个部门到库 (用户 API)
func (c *UserClient) AddLibraryGroups(params map[string]string, secret string) (*AddLibraryGroupResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/add_groups", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AddLibraryGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// UpdateLibraryGroup 修改库部门 (用户 API)
func (c *UserClient) UpdateLibraryGroup(params map[string]string, secret string) (*AddLibraryGroupResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/update_group", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AddLibraryGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddLibraryMemberResponse is the response for add library member API
type AddLibraryMemberResponse struct {
	List      []LibraryMemberResult `json:"list"`
	ErrorList []LibraryMemberError  `json:"error_list"`
}

// LibraryMemberResult represents a successfully added member
type LibraryMemberResult struct {
	OrgID      FlexInt `json:"org_id"`
	MountID    FlexInt `json:"mount_id"`
	EntID      FlexInt `json:"ent_id"`
	MemberID   FlexInt `json:"member_id"`
	MemberType FlexInt `json:"member_type"`
	RoleID     FlexInt `json:"role_id"`
	State      FlexInt `json:"state"`
	AddTime    FlexInt `json:"addtime"`
}

// LibraryMemberError represents an error when adding a member
type LibraryMemberError struct {
	MemberID   FlexInt `json:"member_id"`
	MemberName string  `json:"member_name"`
	ErrorCode  FlexInt `json:"error_code"`
	ErrorMsg   string  `json:"error_msg"`
}

// AddLibraryMember 添加库成员 (用户 API)
func (c *UserClient) AddLibraryMember(params map[string]string, secret string) (*AddLibraryMemberResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/add_member", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AddLibraryMemberResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// UpdateLibraryMember 修改库成员 (用户 API)
func (c *UserClient) UpdateLibraryMember(params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/update_member", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// RemoveLibraryMember 移除库成员 (用户 API)
func (c *UserClient) RemoveLibraryMember(orgID, memberIDs, secret string) error {
	params := map[string]string{
		"org_id":      orgID,
		"_member_ids": memberIDs,
		"dateline":    fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/library/remove_member", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// DeleteLibrary 删除库 (用户 API)
func (c *UserClient) DeleteLibrary(orgID, secret string) error {
	params := map[string]string{
		"org_id":   orgID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/library/delete", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// RemoveLibraryGroup 移除库部门 (用户 API)
func (c *UserClient) RemoveLibraryGroup(entID, orgID, groupID, secret string) error {
	params := map[string]string{
		"ent_id":   entID,
		"org_id":   orgID,
		"group_id": groupID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/library/remove_group", params, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// UpdateLibrary 更新库信息 (用户 API)
func (c *UserClient) UpdateLibrary(params map[string]string, secret string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/library/update", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// SearchGroupResponse is the response for search group API
type SearchGroupResponse struct {
	List []SearchGroupItem `json:"list"`
}

// SearchGroupItem represents a search group result
type SearchGroupItem struct {
	ID   FlexInt `json:"id"`
	Name string  `json:"name"`
	Path string  `json:"path"`
}

// SearchLibraryGroup 搜索库部门 (用户 API)
func (c *UserClient) SearchLibraryGroup(orgID, keyword, secret string) (*SearchGroupResponse, error) {
	params := map[string]string{
		"org_id":   orgID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	if keyword != "" {
		params["keyword"] = keyword
	}

	body, err := c.doUserRequest("/m-api/1/library/search_group", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp SearchGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// === /2/file APIs ===

// CreateFileV2Response is the response for create file API v2.
// When State==0 the caller must run the upload_init→upload_part(s)→upload_finish flow.
// When State==1 the server already has the file content (hash match, instant done).
type CreateFileV2Response struct {
	Hash     string  `json:"hash"`
	Fullpath string  `json:"fullpath"`
	State    FlexInt `json:"state"`  // 0=need upload, 1=instant done (hash match)
	Server   string  `json:"server"` // upload server base URL (when state=0)
}

// CreateFileV2 创建文件 (用户 API v2)
func (c *UserClient) CreateFileV2(mountID int, params map[string]string, secret string) (*CreateFileV2Response, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/create_file", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp CreateFileV2Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFileForUpload calls /m-api/2/file/create_file with filehash and filesize present in
// the request but excluded from the signature, as required by the GoKuai upload protocol.
// Use this (rather than CreateFileV2) when you need to upload actual file content.
func (c *UserClient) CreateFileForUpload(
	mountID int,
	fullpath, filehash string,
	filesize int64,
	secret string,
	extraParams map[string]string,
) (*CreateFileV2Response, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"fullpath": fullpath,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
		"token":    c.AccessToken,
	}
	for k, v := range extraParams {
		if v != "" {
			p[k] = v
		}
	}

	// Sign without filehash / filesize (protocol requirement)
	p["sign"] = Sign(p, secret, "filehash", "filesize")

	// Add filehash and filesize after signing (sent in the request, not signed)
	p["filehash"] = filehash
	p["filesize"] = fmt.Sprintf("%d", filesize)

	// Reuse doUserRequest's transport by building a raw GET (consistent with other user API calls).
	// We bypass the auto-sign in doUserRequest by using our pre-signed params directly.
	reqURL := fmt.Sprintf("%s/m-api/2/file/create_file", c.BaseURL)
	q := url.Values{}
	for k, v := range p {
		q.Set(k, v)
	}

	slog.Debug("user create_file request",
		"endpoint", "/m-api/2/file/create_file",
		"fullpath", fullpath,
		"filehash", filehash,
		"filesize", filesize,
		"params", p,
	)

	req, err := http.NewRequest("GET", reqURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	slog.Debug("user create_file response", "status", resp.StatusCode, "body", string(body))

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var result CreateFileV2Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, body)
	}

	return &result, nil
}

// MoveFileV2 移动文件 (用户 API v2)
func (c *UserClient) MoveFileV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/move", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// RenameFileV2 重命名文件 (用户 API v2)
func (c *UserClient) RenameFileV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/rename", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// GetFileInfoV2 获取文件信息 (用户 API v2)
func (c *UserClient) GetFileInfoV2(mountID int, params map[string]string, secret string) (*FileInfoResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/info", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileInfoResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetFileAttributeV2 获取文件属性 (用户 API v2)
func (c *UserClient) GetFileAttributeV2(mountID int, params map[string]string, secret string) (*FileAttributeResponse, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/attribute", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileAttributeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetFileOpenURLV2 获取文件下载地址 (用户 API v2)
func (c *UserClient) GetFileOpenURLV2(mountID int, params map[string]string, secret string) (*DownloadURLResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/open", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp DownloadURLResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetPreviewURLV2 获取预览URL (用户 API v2)
func (c *UserClient) GetPreviewURLV2(mountID int, params map[string]string, secret string) (*PreviewURLResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/preview_url", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp PreviewURLResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFolderV2 创建文件夹 (用户 API v2)
func (c *UserClient) CreateFolderV2(mountID int, params map[string]string, secret string) (*CreateFolderResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/create_folder", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp CreateFolderResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateStructureV2 创建目录结构 (用户 API v2)
func (c *UserClient) CreateStructureV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/create_structure", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// SaveFileV2 保存文件 (用户 API v2)
func (c *UserClient) SaveFileV2(mountID int, params map[string]string, secret string) (*FileInfoResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/save", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileInfoResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SearchFileV2 搜索文件 (用户 API v2)
func (c *UserClient) SearchFileV2(mountID int, params map[string]string, secret string) (*FileListResponseV2, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/search", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileListResponseV2
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddFileRemarkV2 添加文件评论 (用户 API v2, POST /m-api/2/file/add_remark)
func (c *UserClient) AddFileRemarkV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/2/file/add_remark", p, secret)
	if err != nil {
		return err
	}

	return checkAPIError(body)
}

// GetFileRemarkV2 获取文件评论 (用户 API v2)
func (c *UserClient) GetFileRemarkV2(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/remark", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetCEditURLV2 获取协作编辑URL (用户 API v2)
func (c *UserClient) GetCEditURLV2(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/cedit_url", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// AddLifecycleV2Response 设置文件生命周期响应
type AddLifecycleV2Response struct {
	RunAt string `json:"run_at"`
}

// AddLifecycleV2 设置文件生命周期 (用户 API v2, POST /m-api/2/file/add_lifecycle)
func (c *UserClient) AddLifecycleV2(mountID int, params map[string]string, secret string) (*AddLifecycleV2Response, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/2/file/add_lifecycle", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AddLifecycleV2Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// DelLifecycleV2 取消文件生命周期 (用户 API v2, POST /m-api/2/file/del_lifecycle)
func (c *UserClient) DelLifecycleV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/2/file/del_lifecycle", p, secret)
	if err != nil {
		return err
	}

	return checkAPIError(body)
}

// GetLifecycleV2 获取文件生命周期 (用户 API v2)
func (c *UserClient) GetLifecycleV2(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/get_lifecycle", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// SetFileRemindV2 设置文件提醒 (用户 API v2, POST /m-api/2/file/set_fileremind)
func (c *UserClient) SetFileRemindV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/2/file/set_fileremind", p, secret)
	if err != nil {
		return err
	}

	return checkAPIError(body)
}

// DelFileRemindV2 删除文件提醒 (用户 API v2)
func (c *UserClient) DelFileRemindV2(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/del_fileremind", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// UpdateFileLink 更新文件链接设置 (用户 API)
func (c *UserClient) UpdateFileLink(params map[string]string, secret string) (map[string]interface{}, error) {
	body, err := c.doUserRequest("/m-api/1/file/update_file_link", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetFileLinkDetail 获取文件外链详情 (用户 API)
func (c *UserClient) GetFileLinkDetail(params map[string]string, secret string) (map[string]interface{}, error) {
	body, err := c.doUserRequest("/m-api/1/file/file_link_detail", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetQueueStatus 获取文件执行队列状态 (用户 API)
func (c *UserClient) GetQueueStatus(mountID int, params map[string]string, secret string) ([]map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/queue", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	stateMap := map[float64]string{
		0: "等待中",
		1: "执行中",
		2: "已完成",
		3: "执行出错",
	}

	for _, item := range resp {
		item["state"] = stateMap[item["state"].(float64)]
	}

	return resp, nil
}

// CheckFileExist 检查文件是否存在 (用户 API v2)
func (c *UserClient) CheckFileExist(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/exist", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetFileURL 获取文件地址 (用户 API v2)
func (c *UserClient) GetFileURL(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/open", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// ImageSearch 图片搜索 (用户 API v2)
func (c *UserClient) ImageSearch(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/image_search", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// ImageSearchUpload 图片搜索（上传本地图片，用户 API v2, POST multipart/form-data）
// 与 ImageSearch 的区别在于以 multipart 方式上传 file 字段，文本字段的签名
// 规则与用户 API 一致（file 不参与签名）。
func (c *UserClient) ImageSearchUpload(mountID int, params map[string]string, fileBytes []byte, fileName, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	reqURL := fmt.Sprintf("%s/m-api/2/file/image_search", c.BaseURL)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 写入文本字段
	for k, v := range p {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("failed to write %s field: %w", k, err)
		}
	}

	// 写入 token
	if err := writer.WriteField("token", c.AccessToken); err != nil {
		return nil, fmt.Errorf("failed to write token field: %w", err)
	}

	// 计算签名: 所有文本字段 + token (file 不参与签名)
	signParams := map[string]string{}
	for k, v := range p {
		signParams[k] = v
	}
	signParams["token"] = c.AccessToken
	sign := c.userSign(signParams, secret)
	if err := writer.WriteField("sign", sign); err != nil {
		return nil, fmt.Errorf("failed to write sign field: %w", err)
	}

	// 写入文件字段
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	h.Set("Content-Type", http.DetectContentType(fileBytes))
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	slog.Debug("gokuai user API image_search request",
		"endpoint", "/m-api/2/file/image_search",
		"file", fileName,
	)

	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	slog.Debug("gokuai user API image_search response",
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return result, nil
}

// GetFileRemind 获取文件到期提醒 (用户 API v2)
func (c *UserClient) GetFileRemind(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/get_fileremind", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetFileReminds 获取文件到期提醒列表 (用户 API v2)
func (c *UserClient) GetFileReminds(params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/2/file/get_filereminds", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// SetFileKeyword 设置文件标签 (用户 API)
func (c *UserClient) SetFileKeyword(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/keyword", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// GetMemberPermissions 获取文件夹成员权限 (用户 API)
func (c *UserClient) GetMemberPermissions(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/get_member_permissions", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetGroupPermissions 获取文件夹部门权限 (用户 API)
func (c *UserClient) GetGroupPermissions(mountID int, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/get_group_permissions", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// SetPermission 设置文件夹权限 (用户 API)
func (c *UserClient) SetPermission(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/set_permission", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// SetPermissionInherit 设置文件夹权限是否继承 (用户 API)
func (c *UserClient) SetPermissionInherit(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/set_permission_inherit", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// ResetPermission 重置文件夹权限 (用户 API)
func (c *UserClient) ResetPermission(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/reset_permission", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// ResetAllPermission 重置文件夹所有权限 (用户 API)
func (c *UserClient) ResetAllPermission(mountID int, params map[string]string, secret string) error {
	p := map[string]string{
		"mount_id": fmt.Sprintf("%d", mountID),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/file/reset_all_permission", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// === Enterprise (ent) API Methods ===

// checkAPIError checks if the API response body contains an error_code.
// Many gokuai enterprise APIs return {"error_code": N, "error_msg": "..."} on failure
// instead of the expected data structure. This must be called before unmarshaling
// into a typed response to avoid confusing JSON unmarshal errors.
func checkAPIError(body []byte) error {
	var errResp struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return nil // not JSON or different structure, let caller handle
	}
	if errResp.ErrorCode != 0 {
		msg := fmt.Sprintf("error_code: %d, error_msg: %s", errResp.ErrorCode, errResp.ErrorMsg)
		return newAPIError(errResp.ErrorCode, msg, "")
	}
	return nil
}

// DelClientOAuth 删除个人授权
func (c *Client) DelClientOAuth() error {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_client_oauth", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除授权失败")
	}

	return nil
}

// AddSyncMember 添加或修改同步成员
func (c *Client) AddSyncMember(params map[string]string) (*SyncMemberResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_sync_member", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp SyncMemberResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// DelSyncMember 删除同步成员
func (c *Client) DelSyncMember(members string) error {
	params := map[string]string{
		"members":  members,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_sync_member", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除同步成员失败")
	}

	return nil
}

// AddSyncGroup 添加或修改同步部门
func (c *Client) AddSyncGroup(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_sync_group", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "添加/修改同步部门失败")
	}

	return nil
}

// DelSyncGroup 删除同步部门
func (c *Client) DelSyncGroup(groups string) error {
	params := map[string]string{
		"groups":   groups,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_sync_group", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除同步部门失败")
	}

	return nil
}

// AddSyncGroupMember 添加同步部门的成员
func (c *Client) AddSyncGroupMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_sync_group_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "添加同步部门成员失败")
	}

	return nil
}

// DelSyncGroupMember 删除同步部门的成员
func (c *Client) DelSyncGroupMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_sync_group_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除同步部门成员失败")
	}

	return nil
}

// DelSyncMemberGroup 删除同步成员的所属部门
func (c *Client) DelSyncMemberGroup(members string) error {
	params := map[string]string{
		"members":  members,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_sync_member_group", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除同步成员部门失败")
	}

	return nil
}

// GetMemberByOutID 通过外部帐号获取成员信息
func (c *Client) GetMemberByOutID(params map[string]string) (map[string]MemberInfo, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_member_by_out_id", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]MemberInfo
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetGroupByOutID 通过外部部门ID获取部门信息
func (c *Client) GetGroupByOutID(params map[string]string) (map[string]GroupInfo, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_group_by_out_id", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]GroupInfo
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// AddSyncAdmin 添加管理员
func (c *Client) AddSyncAdmin(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_sync_admin", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "添加管理员失败")
	}

	return nil
}

// AddMember 添加成员
func (c *Client) AddMember(params map[string]string) (*SyncMemberResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_member", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp SyncMemberResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SetMember 修改成员
func (c *Client) SetMember(params map[string]string) (*SyncMemberResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/set_member", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp SyncMemberResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// DelMember 删除成员
func (c *Client) DelMember(members string) error {
	params := map[string]string{
		"members":  members,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_member", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除成员失败")
	}

	return nil
}

// AddGroup 添加部门
func (c *Client) AddGroup(params map[string]string) (*AddGroupResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_group", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AddGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SetGroup 修改部门
func (c *Client) SetGroup(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/set_group", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "修改部门失败")
	}

	return nil
}

// DelGroup 删除部门
func (c *Client) DelGroup(groups string) error {
	params := map[string]string{
		"groups":   groups,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_group", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除部门失败")
	}

	return nil
}

// AddGroupMember 添加部门成员
func (c *Client) AddGroupMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/add_group_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "添加部门成员失败")
	}

	return nil
}

// DelGroupMember 删除部门成员
func (c *Client) DelGroupMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_group_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除部门成员失败")
	}

	return nil
}

// DelMemberGroup 删除成员的所属部门
func (c *Client) DelMemberGroup(members string) error {
	params := map[string]string{
		"members":  members,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/del_member_group", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除成员部门失败")
	}

	return nil
}

// GetMembers 成员列表
func (c *Client) GetMembers(params map[string]string) (*MemberListResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_members", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp MemberListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetMember 成员信息
func (c *Client) GetMember(params map[string]string) (*MemberDetailResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_member", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp MemberDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetGroups 部门列表
func (c *Client) GetGroups() (GroupListResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_groups", params)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp GroupListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetGroupMembers 部门中成员列表
func (c *Client) GetGroupMembers(params map[string]string) (*GroupMemberListResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_group_members", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp GroupMemberListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetRoles 角色列表
func (c *Client) GetRoles() (RolesResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/ent/get_roles", params)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp RolesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// doUserPostRequest performs a signed POST request for user API
func (c *UserClient) doUserPostRequest(endpoint string, params map[string]string, secret string) ([]byte, error) {
	signed := cloneParams(params)
	signed["token"] = c.AccessToken
	signed["sign"] = c.userSign(signed, secret)

	body, err := c.doSignedUserPost(endpoint, signed)
	if err != nil {
		return nil, err
	}

	if c.TokenRefresher != nil && isTokenInvalidResponse(body) {
		newToken, refreshErr := c.TokenRefresher()
		if refreshErr != nil || newToken == "" {
			slog.Debug("gokuai token refresh failed", "error", refreshErr)
			return body, nil
		}
		c.AccessToken = newToken

		signed = cloneParams(params)
		signed["token"] = newToken
		signed["sign"] = c.userSign(signed, secret)

		body, err = c.doSignedUserPost(endpoint, signed)
		if err != nil {
			return nil, err
		}
	}

	return body, nil
}

func cloneParams(params map[string]string) map[string]string {
	clone := make(map[string]string, len(params))
	for k, v := range params {
		clone[k] = v
	}
	return clone
}

func (c *UserClient) doSignedUserPost(endpoint string, params map[string]string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	// Build form data
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	logParams := redactParams(params)
	slog.Debug("gokuai user API POST request",
		"endpoint", endpoint,
		"params", logParams,
	)

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	slog.Debug("gokuai user API POST response",
		"endpoint", endpoint,
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	return body, nil
}

// === Contact APIs (User) ===

// ContactGroupInfo represents a contact group/department
type ContactGroupInfo struct {
	GroupID    FlexInt     `json:"id"`
	EntID      FlexInt     `json:"ent_id"`
	Name       string      `json:"name"`
	ParentID   FlexInt     `json:"parent_id"`
	State      FlexInt     `json:"state"`
	Hidden     FlexInt     `json:"hidden"`
	Order      FlexInt     `json:"order"`
	OutID      string      `json:"out_id"`
	MemberIDs  interface{} `json:"member_ids"`
	ChildCount FlexInt     `json:"child_count"`
}

// ContactGroupListResponse is the response for contact group list API
type ContactGroupListResponse struct {
	List []ContactGroupInfo `json:"list"`
}

// ContactMemberInfo represents a contact member
type ContactMemberInfo struct {
	MemberID        FlexInt `json:"member_id"`
	MemberName      string  `json:"member_name"`
	MemberEmail     string  `json:"member_email"`
	MemberPhone     string  `json:"member_phone"`
	MemberTitle     string  `json:"member_title"`
	MemberType      FlexInt `json:"member_type"`
	MemberLetter    string  `json:"member_letter"`
	Avatar          string  `json:"avatar"`
	State           FlexInt `json:"state"`
	Account         string  `json:"account"`
	OutID           string  `json:"out_id"`
	EntID           FlexInt `json:"ent_id"`
	EnableCreateOrg FlexInt `json:"enable_create_org"`
}

// ContactMemberListResponse is the response for contact member list API
type ContactMemberListResponse struct {
	List  []ContactMemberInfo `json:"list"`
	Count FlexInt             `json:"count"`
}

// ContactRootGroupResponse is the response for contact root group API
type ContactRootGroupResponse struct {
	Name    string  `json:"name"`
	GroupID FlexInt `json:"group_id"`
	State   FlexInt `json:"state"`
	Hidden  FlexInt `json:"hidden"`
}

// GetContactGroupList 获取部门列表 (用户 API)
func (c *UserClient) GetContactGroupList(entID, groupID string, params map[string]string, secret string) (*ContactGroupListResponse, error) {
	p := map[string]string{
		"ent_id":   entID,
		"group_id": groupID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/contact/group_list", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ContactGroupListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SearchContactGroup 查询部门列表 (用户 API)
func (c *UserClient) SearchContactGroup(entID, keyword string, secret string) (*ContactGroupListResponse, error) {
	params := map[string]string{
		"ent_id":   entID,
		"keyword":  keyword,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/contact/search_group", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ContactGroupListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetContactGroupMemberList 获取部门成员列表 (用户 API)
func (c *UserClient) GetContactGroupMemberList(params map[string]string, secret string) (*ContactMemberListResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/contact/group_member_list", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ContactMemberListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetContactMemberInfo 获取成员信息 (用户 API)
func (c *UserClient) GetContactMemberInfo(entID, memberID string, params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"ent_id":     entID,
		"_member_id": memberID,
		"dateline":   fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/contact/member_info", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// GetContactMemberGroups 获取成员所属部门列表 (用户 API)
func (c *UserClient) GetContactMemberGroups(entID, memberID string, secret string) (*ContactGroupListResponse, error) {
	params := map[string]string{
		"ent_id":     entID,
		"_member_id": memberID,
		"dateline":   fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/contact/member_groups", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ContactGroupListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddContactGroup 添加部门 (用户 API)
func (c *UserClient) AddContactGroup(params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/contact/add_group", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// UpdateContactGroup 更新部门 (用户 API)
func (c *UserClient) UpdateContactGroup(params map[string]string, secret string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/contact/update_group", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// DelContactGroup 删除部门 (用户 API)
func (c *UserClient) DelContactGroup(params map[string]string, secret string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/contact/del_group", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// AddContactMember 添加成员 (用户 API)
func (c *UserClient) AddContactMember(params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/contact/add_member", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// UpdateContactMember 修改成员 (用户 API)
func (c *UserClient) UpdateContactMember(params map[string]string, secret string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/contact/update_member", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// RemoveContactMember 移除成员 (用户 API)
func (c *UserClient) RemoveContactMember(params map[string]string, secret string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserPostRequest("/m-api/1/contact/remove_member", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// AddContactGroupMember 添加部门成员 (用户 API)
func (c *UserClient) AddContactGroupMember(params map[string]string, secret string) (map[string]interface{}, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/contact/add_group_member", p, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// RemoveContactGroupMember 移除部门成员 (用户 API)
func (c *UserClient) RemoveContactGroupMember(params map[string]string, secret string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doUserRequest("/m-api/1/contact/remove_group_member", p, secret)
	if err != nil {
		return err
	}
	return checkAPIError(body)
}

// GetContactRootGroup 获取根部门属性 (用户 API)
func (c *UserClient) GetContactRootGroup(entID string, secret string) (*ContactRootGroupResponse, error) {
	params := map[string]string{
		"ent_id":   entID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/contact/root_group", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ContactRootGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetEntLog 管理日志
func (c *Client) GetEntLog(params map[string]string) (*EntLogResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/ent/log", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntLogResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// === Enterprise Library (Org) API Methods ===

// OrgClient is the gokuai org API client (uses org_client_id + org_client_secret for auth)
type OrgClient struct {
	OrgClientID     string
	OrgClientSecret string
	HTTPClient      *http.Client
	BaseURL         string
}

// NewOrgClient creates a new org API client
func NewOrgClient(orgClientID, orgClientSecret string) *OrgClient {
	return &OrgClient{
		OrgClientID:     orgClientID,
		OrgClientSecret: orgClientSecret,
		HTTPClient:      &http.Client{},
		BaseURL:         GetAPIHost(),
	}
}

// doOrgSignedRequest performs a signed POST request for org client
func (c *OrgClient) doOrgSignedRequest(endpoint string, params map[string]string) ([]byte, error) {
	signedParams := BuildOrgSignedParams(params, c.OrgClientID, c.OrgClientSecret)

	// Build URL
	reqURL := fmt.Sprintf("%s%s", c.BaseURL, endpoint)

	// Build form data
	formData := url.Values{}
	for k, v := range signedParams {
		formData.Set(k, v)
	}

	// Log request parameters (redact sensitive fields)
	logParams := redactParams(signedParams)
	slog.Debug("gokuai org API request",
		"endpoint", endpoint,
		"org_client_id", c.OrgClientID,
		"params", logParams,
	)

	// Create request
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("request failed: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("failed to read response: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}

	// Log response body
	slog.Debug("gokuai org API response",
		"endpoint", endpoint,
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	return body, nil
}

// === OrgClient File API Methods ===

// GetFileList 获取文件列表
func (c *OrgClient) GetFileList(params map[string]string) (*EntFileListResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/ls", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetFileUpdates 获取文件最近更新列表
func (c *OrgClient) GetFileUpdates(params map[string]string) (*EntFileUpdateResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/updates", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileUpdateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetFileUpdatesCount 获取文件更新数量
func (c *OrgClient) GetFileUpdatesCount(params map[string]string) (*EntFileUpdatesCountResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/updates_count", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileUpdatesCountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetDownloadURL 获取文件下载链接
func (c *OrgClient) GetDownloadURL(params map[string]string) (*DownloadURLResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/download_url", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp DownloadURLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetPreviewURL 获取文件预览/批注链接
func (c *OrgClient) GetPreviewURL(params map[string]string) (*PreviewURLResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/preview_url", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp PreviewURLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetExportURL 获取文件导出链接
func (c *OrgClient) GetExportURL(params map[string]string) (*ExportURLResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/preview_download_url", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp ExportURLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetCeditURL 获取文件协同编辑链接
func (c *OrgClient) GetCeditURL(params map[string]string) (*CeditURLResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/cedit_url", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp CeditURLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetFileInfo 获取文件(夹)信息
func (c *OrgClient) GetFileInfo(params map[string]string) (*FileInfoResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/info", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp FileInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SearchFiles 文件搜索
func (c *OrgClient) SearchFiles(params map[string]string) (*EntFileListResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/search", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFolder 创建文件夹
func (c *OrgClient) CreateFolder(params map[string]string) (*EntFileCreateFolderResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/create_folder", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileCreateFolderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFile 上传文件(步骤1: 请求上传)
func (c *OrgClient) CreateFile(params map[string]string) (*EntFileCreateFileResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/create_file", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileCreateFileResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetUploadServers 获取上传服务器
func (c *OrgClient) GetUploadServers(params map[string]string) (*UploadServersResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/upload_servers", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp UploadServersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CopyFile 复制文件(夹)
func (c *OrgClient) CopyFile(params map[string]string) (*EntFileCopyResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/copy", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileCopyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// MultiCopyFile 高级复制文件(夹)
func (c *OrgClient) MultiCopyFile(params map[string]string) ([]EntFileCopyResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/mcopy", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp []EntFileCopyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// MoveFile 移动文件(夹)
func (c *OrgClient) MoveFile(params map[string]string) (*EntFileCopyResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/move", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileCopyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// DeleteFile 删除文件(夹)
// DeleteFile 删除文件(夹)
func (c *OrgClient) DeleteFile(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/del", params)
}

// DeleteCompletely 彻底删除文件(夹)
func (c *OrgClient) DeleteCompletely(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/del_completely", params)
}

// GetRecycleList 回收站
func (c *OrgClient) GetRecycleList(params map[string]string) (*RecycleResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/recycle", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp RecycleResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// doOrgAction performs a signed POST request to an org endpoint and returns
// the parsed JSON response. Non-empty error_code values are returned as errors.
// Used by write/mutation endpoints whose JSON response body should be surfaced
// to the caller rather than being swallowed.
func (c *OrgClient) doOrgAction(endpoint string, params map[string]string) (map[string]any, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}
	body, err := c.doOrgSignedRequest(endpoint, p)
	if err != nil {
		return nil, err
	}
	if err := checkAPIError(body); err != nil {
		return nil, err
	}
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}
	return resp, nil
}

// RecoverFiles 恢复已删除文件
func (c *OrgClient) RecoverFiles(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/recover", params)
}

// GetFileHistory 获取文件历史
func (c *OrgClient) GetFileHistory(params map[string]string) (*EntFileHistoryResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/history", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileHistoryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CreateFileLink 获取文件外链
func (c *OrgClient) CreateFileLink(params map[string]string) (*EntFileLinkResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/link", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileLinkResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// CloseFileLink 关闭文件外链
func (c *OrgClient) CloseFileLink(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/link_close", params)
}

// GetFileLinks 获取开启外链的文件列表
func (c *OrgClient) GetFileLinks(params map[string]string) ([]EntFileLinksResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/links", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp []EntFileLinksResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// LockFile 文件锁的操作
func (c *OrgClient) LockFile(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/lock", params)
}

// SetPermissionInherit 设置文件夹权限继承状态
func (c *OrgClient) SetPermissionInherit(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/set_permission_inherit", params)
}

// GetAllPermission 获取文件夹单独设置的权限
func (c *OrgClient) GetAllPermission(params map[string]string) (*EntFilePermissionResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/get_all_permission", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFilePermissionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SetFilePermission 修改文件夹权限
func (c *OrgClient) SetFilePermission(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/file_permission", params)
}

// ResetPermission 重置或移除文件夹权限
func (c *OrgClient) ResetPermission(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/reset_permission", params)
}

// GetFilePermission 获取文件权限
func (c *OrgClient) GetFilePermission(params map[string]string) (*EntFilePermissionResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/get_permission", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFilePermissionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// get_permission may return []string for specific user
		var permList []string
		if err2 := json.Unmarshal(body, &permList); err2 == nil {
			return &EntFilePermissionResponse{
				Members: map[string][]string{"permissions": permList},
			}, nil
		}
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddTag 添加标签
func (c *OrgClient) AddTag(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/add_tag", params)
}

// DelTag 删除标签
func (c *OrgClient) DelTag(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/del_tag", params)
}

// SetMetadata 添加或修改元数据
func (c *OrgClient) SetMetadata(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/set_metadata", params)
}

// DelMetadata 删除元数据
func (c *OrgClient) DelMetadata(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/1/file/del_metadata", params)
}

// GetStat 统计信息
func (c *OrgClient) GetStat(params map[string]string) (*EntFileStatResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/stat", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileStatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetQueueStatus 查询队列状态
func (c *OrgClient) GetQueueStatus(params map[string]string) (*EntFileQueueStatusResponse, error) {
	p := map[string]string{}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doOrgSignedRequest("/m-open/1/file/queue_status", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntFileQueueStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// === v2 File Permission Methods ===

// BatchSetPermission 批量设置权限
func (c *OrgClient) BatchSetPermission(params map[string]string) (map[string]any, error) {
	return c.doOrgAction("/m-open/2/file/batch_set_permission", params)
}

// CreateOrg 创建库
func (c *Client) CreateOrg(params map[string]string) (*EntOrgCreateResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/create", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgCreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SetOrg 修改库
func (c *Client) SetOrg(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/set", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "修改库失败")
	}

	return nil
}

// GetOrgInfo 库信息
func (c *Client) GetOrgInfo(params map[string]string) (*EntOrgInfoResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/info", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SearchOrg 搜索库
func (c *Client) SearchOrg(params map[string]string) (*EntOrgSearchResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/search", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetOrgInfoByMember 个人文件库信息
func (c *Client) GetOrgInfoByMember(params map[string]string) (*EntOrgInfoResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/info_by_member", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// SetOrgByMember 设置个人文件库
func (c *Client) SetOrgByMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/set_by_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "设置个人文件库失败")
	}

	return nil
}

// GetOrgList 获取库列表
func (c *Client) GetOrgList(params map[string]string) (*EntOrgListResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/ls", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// BindOrg 获取库授权
func (c *Client) BindOrg(params map[string]string) (*EntOrgBindResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/bind", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgBindResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// UnbindOrg 取消库授权
func (c *Client) UnbindOrg(orgClientID string) error {
	params := map[string]string{
		"org_client_id": orgClientID,
		"dateline":      fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/org/unbind", params)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "取消库授权失败")
	}

	return nil
}

// GetOrgMembers 获取库成员列表
func (c *Client) GetOrgMembers(params map[string]string) (*EntOrgMembersResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/get_members", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgMembersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// GetOrgMember 查询库成员信息
func (c *Client) GetOrgMember(params map[string]string) (map[string]EntOrgMemberInfo, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/get_member", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp map[string]EntOrgMemberInfo
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return resp, nil
}

// SetOrgOwner 设置库拥有者
func (c *Client) SetOrgOwner(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/set_owner", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "设置库拥有者失败")
	}

	return nil
}

// AddOrgMember 添加库成员
func (c *Client) AddOrgMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/add_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "添加库成员失败")
	}

	return nil
}

// SetOrgMemberRole 修改库成员角色
func (c *Client) SetOrgMemberRole(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/set_member_role", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "修改库成员角色失败")
	}

	return nil
}

// DelOrgMember 删除库成员
func (c *Client) DelOrgMember(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/del_member", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除库成员失败")
	}

	return nil
}

// GetOrgGroups 获取库部门列表
func (c *Client) GetOrgGroups(orgID string) (*EntOrgGroupsResponse, error) {
	params := map[string]string{
		"org_id":   orgID,
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doSignedRequest("/m-open/1/org/get_groups", params)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgGroupsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// AddOrgGroup 库上添加部门
func (c *Client) AddOrgGroup(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/add_group", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "库上添加部门失败")
	}

	return nil
}

// DelOrgGroup 删除库上的部门
func (c *Client) DelOrgGroup(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/del_group", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除库上的部门失败")
	}

	return nil
}

// SetOrgGroupRole 修改库上部门的角色
func (c *Client) SetOrgGroupRole(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/set_group_role", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "修改库上部门的角色失败")
	}

	return nil
}

// DestroyOrg 删除库
func (c *Client) DestroyOrg(params map[string]string) error {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/destroy", p)
	if err != nil {
		return err
	}

	if err := checkAPIError(body); err != nil {
		return err
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	if !resp.IsSuccess() {
		return apiResponseError(&resp, "删除库失败")
	}

	return nil
}

// GetOrgLog 库日志
func (c *Client) GetOrgLog(params map[string]string) (*EntOrgLogResponse, error) {
	p := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}
	for k, v := range params {
		if v != "" {
			p[k] = v
		}
	}

	body, err := c.doSignedRequest("/m-open/1/org/log", p)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp EntOrgLogResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}
