package gokuai

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	// defaultAPIHost is the default gokuai API host
	defaultAPIHost = "https://yk3.gokuai.com"
	// APIVersion is the API version
	APIVersion = "1"
)

// GetAPIHost returns the gokuai API host from GOKUAI_API_HOST env var
// with fallback to defaultAPIHost ("yk3.gokuai.com").
// This is a function (not a package-level var) so it reads the env var
// after loadDotEnv() has been called.
func GetAPIHost() string {
	if h := os.Getenv("GOKUAI_API_HOST"); h != "" {
		if !strings.HasPrefix(h, "http") {
			return fmt.Sprintf("https://%s", h)
		}
		return h
	}
	return defaultAPIHost
}

// signExcludeKeys are keys excluded from signature calculation
var signExcludeKeys = map[string]bool{
	"sign": true,
}

// Sign generates the signature for gokuai API requests
// Algorithm: hmac-sha1(values_joined_by_\n, secret) with base64 encoding
// Values are sorted by key and joined with \n, excluding "sign" key
func Sign(params map[string]string, secret string, excludeKeys ...string) string {
	excludes := make(map[string]bool)
	for k := range signExcludeKeys {
		excludes[k] = true
	}
	for _, k := range excludeKeys {
		excludes[k] = true
	}

	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build value string with \n separator, skipping excluded and empty values
	var vals []string
	for _, k := range keys {
		if excludes[k] {
			continue
		}
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

// BuildSignedParams adds dateline and sign to params
func BuildSignedParams(params map[string]string, clientID, clientSecret string) map[string]string {
	if params == nil {
		params = make(map[string]string)
	}
	// Add dateline
	params["dateline"] = fmt.Sprintf("%d", time.Now().Unix())

	// Add client_id if not present
	if _, ok := params["client_id"]; !ok && clientID != "" {
		params["client_id"] = clientID
	}

	// Calculate sign (excluding filehash and filesize for file creation endpoints)
	params["sign"] = Sign(params, clientSecret, "filehash", "filesize")

	return params
}

// BuildOrgSignedParams adds dateline and sign to params for org client requests
func BuildOrgSignedParams(params map[string]string, orgClientID, orgClientSecret string) map[string]string {
	if params == nil {
		params = make(map[string]string)
	}
	// Add dateline
	params["dateline"] = fmt.Sprintf("%d", time.Now().Unix())

	// Add org_client_id if not present
	if _, ok := params["org_client_id"]; !ok && orgClientID != "" {
		params["org_client_id"] = orgClientID
	}

	// Calculate sign (excluding filehash and filesize for file creation endpoints)
	params["sign"] = Sign(params, orgClientSecret, "filehash", "filesize")

	return params
}
