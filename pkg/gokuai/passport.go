package gokuai

import (
	"net/url"
	"os"
	"strings"
)

// defaultPassportDomain is the default host of the gokuai passport web
// OAuth2 login page (gokuai-passport 服务的 /oauth/login 页面)。
const defaultPassportDomain = "https://passport.gokuai.com"

// GetPassportDomain returns the passport web-authorization domain from the
// GOKUAI_PASSPORT_DOMAIN env var, falling back to defaultPassportDomain.
// Bare hosts without scheme are assumed to be https://.
// Like GetAPIHost this is a function so the env var (loaded from .env at
// startup) is read at call time rather than process start.
func GetPassportDomain() string {
	if d := os.Getenv("GOKUAI_PASSPORT_DOMAIN"); d != "" {
		if !strings.HasPrefix(d, "http://") && !strings.HasPrefix(d, "https://") {
			d = "https://" + d
		}
		return strings.TrimRight(d, "/")
	}
	return defaultPassportDomain
}

// BuildPassportLoginURL constructs the passport /oauth/login authorization
// page URL:
//
//	{domain}/oauth/login?client_id=...&response_type=code&redirect_uri=...&state=...
//
// Passport exchanges the authorization code server-side once the user logs in
// (see gokuai-passport OAuthController::postLogin): the browser is redirected
// straight to redirectURI with access_token, refresh_token and expires_in as
// query parameters. A loopback redirectURI therefore lets a CLI receive the
// token set directly, without a second code-exchange request.
//
// state is forwarded for CSRF correlation but passport does not echo it back
// on the final token redirect; the random loopback port and one-shot receiver
// are the CLI's protections against stray callbacks.
func BuildPassportLoginURL(clientID, redirectURI, state, scope string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	if state != "" {
		q.Set("state", state)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	return GetPassportDomain() + "/oauth/login?" + q.Encode()
}
