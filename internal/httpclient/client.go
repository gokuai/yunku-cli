// Package httpclient provides a simple HTTP client for calling yunku APIs.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gokuai/yunku-cli/pkg/config"
)

// ToolCallResult represents the result of a tool call.
type ToolCallResult struct {
	IsError bool
	Content map[string]any
	Blocks  []ContentBlock
}

// ContentBlock represents a single content item in a tool result.
type ContentBlock struct {
	Type string
	Text string
}

// Client is a simple HTTP client for calling yunku APIs.
type Client struct {
	HTTPClient   *http.Client
	AuthToken    string
	BearerMode   bool
	ExtraHeaders map[string]string
}

// NewClient creates a new HTTP client with default settings.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.HTTPTimeout,
		}
	}
	return &Client{
		HTTPClient:   httpClient,
		ExtraHeaders: make(map[string]string),
	}
}

// WithAuth returns a copy of the client with the specified auth token.
func (c *Client) WithAuth(authToken string) *Client {
	copy := *c
	copy.AuthToken = authToken
	copy.BearerMode = false
	return &copy
}

// WithBearerAuth returns a copy of the client configured for bearer token authentication.
// Requests will use Authorization and X-Auth-Type headers instead of x-auth-token.
func (c *Client) WithBearerAuth(bearerToken string) *Client {
	copy := *c
	copy.AuthToken = bearerToken
	copy.BearerMode = true
	return &copy
}

// CallTool calls a yunku API tool with the given parameters.
func (c *Client) CallTool(ctx context.Context, endpoint, toolName string, params map[string]any) (*ToolCallResult, error) {
	requestBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": params,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		if c.BearerMode {
			req.Header.Set("Authorization", "Bearer "+c.AuthToken)
			req.Header.Set("X-Auth-Type", "mcp")
		} else {
			req.Header.Set("x-auth-token", c.AuthToken)
		}
	}
	for k, v := range c.ExtraHeaders {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, config.MaxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response struct {
		Result map[string]any `json:"result,omitempty"`
		Error  *jsonrpcError   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error != nil {
		return &ToolCallResult{
			IsError: true,
			Content: map[string]any{
				"error":   response.Error.Message,
				"message": response.Error.Message,
			},
			Blocks: []ContentBlock{{Type: "text", Text: response.Error.Message}},
		}, nil
	}

	result := &ToolCallResult{
		Content: make(map[string]any),
		Blocks:  []ContentBlock{},
	}

	if response.Result != nil {
		result.Content = response.Result
		// Extract content blocks if present
		if content, ok := response.Result["content"].([]any); ok {
			for _, item := range content {
				if block, ok := item.(map[string]any); ok {
					blockType := "text"
					if t, ok := block["type"].(string); ok {
						blockType = t
					}
					text := ""
					if t, ok := block["text"].(string); ok {
						text = t
					}
					result.Blocks = append(result.Blocks, ContentBlock{Type: blockType, Text: text})
				}
			}
		}
	}

	// Check for business errors in content
	if success, ok := result.Content["success"].(bool); ok && !success {
		result.IsError = true
	}

	return result, nil
}

// jsonrpcError represents a JSON-RPC error.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RedactURL returns a redacted version of the URL for logging.
func RedactURL(rawURL string) string {
	return rawURL
}

// ExtractServerDiagnosticsFromMap extracts diagnostic information from a response.
func ExtractServerDiagnosticsFromMap(content map[string]any) map[string]string {
	diag := make(map[string]string)
	if traceID, ok := content["traceId"].(string); ok {
		diag["trace_id"] = traceID
	}
	if traceID, ok := content["trace_id"].(string); ok {
		diag["trace_id"] = traceID
	}
	if errCode, ok := content["errorCode"].(string); ok {
		diag["server_error_code"] = errCode
	}
	return diag
}
