package gokuai

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/url"
	"time"
)

// BuildSSOURL wraps a target URL with the /account/sso authentication
// parameters required by the web console:
//
//	/account/sso?token={token}&url={url}&n={随机数}&t={当前时间戳}&s={签名}
//
// The signature is computed over token/url/n/dateline and follows the same
// convention as the rest of the gokuai API (HMAC-SHA1 over the non-empty
// parameter values joined by "\n" in alphabetical key order, base64 encoded),
// using the provided secret.
func BuildSSOURL(token, targetURL, secret string) (string, error) {
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"token":    token,
		"url":      targetURL,
		"n":        nonce,
		"t": fmt.Sprintf("%d", time.Now().Unix()),
	}
	params["sign"] = Sign(params, secret)

	ssoURL := fmt.Sprintf("%s/account/sso?token=%s&url=%s&n=%s&t=%s&s=%s",
		GetAPIHost(),
		url.QueryEscape(params["token"]),
		url.QueryEscape(params["url"]),
		url.QueryEscape(params["n"]),
		url.QueryEscape(params["t"]),
		url.QueryEscape(params["sign"]),
	)
	return ssoURL, nil
}

// randomNonce returns a purely numeric random nonce (up to 19 digits).
func randomNonce() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return fmt.Sprintf("%d", binary.BigEndian.Uint64(b[:])), nil
}
