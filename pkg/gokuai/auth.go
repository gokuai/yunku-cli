package gokuai

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AuthClient 够快云库认证客户端
type AuthClient struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	BaseURL      string
}

// TokenResponse 登录响应
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	Error            string `json:"error,omitempty"`
	ErrorDesc        string `json:"error_description,omitempty"`
}

// NewAuthClient 创建认证客户端
func NewAuthClient(clientID, clientSecret string) *AuthClient {
	return &AuthClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		BaseURL:      GetAPIHost(),
	}
}

// LoginByPassword 用户名密码登录
func (c *AuthClient) LoginByPassword(username, password string) (*TokenResponse, error) {
	// 密码需要 MD5 加密
	passwordMD5 := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	return c.requestToken("password", map[string]string{
		"username": username,
		"password": passwordMD5,
	})
}

// RefreshToken 刷新 token
func (c *AuthClient) RefreshToken(refreshToken string) (*TokenResponse, error) {
	return c.requestToken("refresh_token", map[string]string{
		"refresh_token": refreshToken,
	})
}

// ExchangeToken 第三方授权换取 token
func (c *AuthClient) ExchangeToken(exchangeToken, domain string) (*TokenResponse, error) {
	return c.requestToken("exchange_token", map[string]string{
		"exchange_token": exchangeToken,
		"domain":         domain,
	})
}

// requestToken 通用请求 token
func (c *AuthClient) requestToken(grantType string, extraParams map[string]string) (*TokenResponse, error) {
	dateline := strconv.FormatInt(time.Now().Unix(), 10)

	params := map[string]string{
		"client_id":  c.ClientID,
		"grant_type": grantType,
		"dateline":   dateline,
	}
	for k, v := range extraParams {
		params[k] = v
	}

	sign := c.sign(params)
	params["sign"] = sign

	// 转换为 url.Values
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	// Log request parameters (redact sensitive fields)
	logParams := redactParams(params)
	slog.Debug("gokuai auth request",
		"endpoint", "/m-api/oauth2/token2",
		"client_id", c.ClientID,
		"grant_type", grantType,
		"params", logParams,
	)

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/m-api/oauth2/token2", formData)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// Log response body
	slog.Debug("gokuai auth response",
		"endpoint", "/m-api/oauth2/token2",
		"status_code", resp.StatusCode,
		"body", string(body),
	)

	var result TokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}

	return &result, nil
}

// sign 生成签名
// 签名算法: 按参数名排序，用 \n 连接值，然后用 client_secret 作为密钥进行 hmac-sha1 加密，最后 base64 编码
func (c *AuthClient) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		if params[k] == "" {
			continue
		}
		parts = append(parts, params[k])
	}

	queryString := strings.Join(parts, "\n")
	h := hmac.New(sha1.New, []byte(c.ClientSecret))
	h.Write([]byte(queryString))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}


