package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	authpkg "github.com/gokuai/yunku-cli/internal/auth"
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/internal/executor"
	"github.com/gokuai/yunku-cli/internal/httpclient"
	"github.com/gokuai/yunku-cli/internal/logging"
	"github.com/gokuai/yunku-cli/internal/safety"
	"github.com/gokuai/yunku-cli/pkg/edition"
)

const (
	runtimeContentScanEnv             = "YKC_RUNTIME_CONTENT_SCAN"
	runtimeContentScanEnforceEnv      = "YKC_RUNTIME_CONTENT_SCAN_ENFORCE"
	runtimeContentScanReportOutputEnv = "YKC_RUNTIME_CONTENT_SCAN_REPORT"

	// Environment variables for request headers (passed from caller)
	envYkcAgent     = "YKC_AGENT"
	envYkcTraceID   = "YKC_TRACE_ID"
	envYkcSessionID = "YKC_SESSION_ID"
	envYkcMessageID = "YKC_MESSAGE_ID"
)

func newCommandRunnerWithFlags(flags *GlobalFlags) *runtimeRunner {
	var httpClient *http.Client
	if flags != nil && flags.Timeout > 0 {
		httpClient = &http.Client{Timeout: time.Duration(flags.Timeout) * time.Second}
	}
	client := httpclient.NewClient(httpClient)
	client.ExtraHeaders = resolveIdentityHeaders()
	return &runtimeRunner{
		httpClient:         client,
		globalFlags:        flags,
		fallback:           echoRunner{},
		scanner:            newRuntimeContentScanner(),
		enforceContentScan: runtimeFlagEnabled(os.Getenv(runtimeContentScanEnforceEnv), false),
		includeScanReport:  runtimeFlagEnabled(os.Getenv(runtimeContentScanReportOutputEnv), false),
	}
}

type runtimeRunner struct {
	httpClient         *httpclient.Client
	globalFlags        *GlobalFlags
	fallback           executor.Runner
	scanner            safety.Scanner
	enforceContentScan bool
	includeScanReport  bool
}

type echoRunner struct{}

func (echoRunner) Run(_ context.Context, invocation executor.Invocation) (executor.Result, error) {
	if invocation.DryRun {
		return executor.Result{
			Invocation: invocation,
			Response: map[string]any{
				"dry_run": true,
				"request": ToolCallRequest(invocation.Tool, invocation.Params),
				"note":    "execution skipped by --dry-run",
			},
		}, nil
	}
	return executor.Result{Invocation: invocation}, nil
}

func ToolCallRequest(tool string, params map[string]any) map[string]any {
	if params == nil {
		params = map[string]any{}
	}
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": params,
		},
	}
}

func (r *runtimeRunner) Run(ctx context.Context, invocation executor.Invocation) (executor.Result, error) {
	if r.httpClient == nil {
		return r.fallback.Run(ctx, invocation)
	}

	// Mock mode: skip catalog validation, use a placeholder endpoint.
	if r.globalFlags != nil && r.globalFlags.Mock {
		endpoint := fmt.Sprintf("https://mock-%s.gokuai.com", invocation.CanonicalProduct)
		if override, ok := productEndpointOverride(invocation.CanonicalProduct); ok {
			endpoint = override
		}
		return r.executeInvocation(ctx, endpoint, invocation)
	}

	// Prefetch the Keychain token in the background. Keychain access costs
	// ~70ms on macOS; starting it here lets the load overlap with endpoint
	// resolution and catalog loading below.
	go getCachedRuntimeToken(ctx)

	if shouldUseDirectRuntime(invocation) {
		if endpoint, ok := directRuntimeEndpoint(invocation.CanonicalProduct, invocation.Tool); ok {
			return r.executeInvocation(ctx, endpoint, invocation)
		}
	}

	return r.fallback.Run(ctx, invocation)
}

func (r *runtimeRunner) executeInvocation(ctx context.Context, endpoint string, invocation executor.Invocation) (result executor.Result, retErr error) {
	invokeStart := time.Now()
	execID := generateExecutionID()

	fl := FileLoggerInstance()

	defer func() {
		var errCat, errReason string
		if retErr != nil {
			var typed *apperrors.Error
			if errors.As(retErr, &typed) {
				errCat = string(typed.Category)
				errReason = typed.Reason
			} else {
				errCat = "unknown"
				errReason = retErr.Error()
			}
		}
		logging.LogCommandEnd(fl, execID,
			invocation.CanonicalProduct, invocation.Tool,
			retErr == nil, time.Since(invokeStart), errCat, errReason)
	}()

	authToken := r.resolveAuthToken(ctx)

	var timeoutSec int
	if r.globalFlags != nil {
		timeoutSec = r.globalFlags.Timeout
	}
	logging.LogCommandStart(fl, execID,
		invocation.CanonicalProduct, invocation.Tool, endpoint, version, authToken != "", timeoutSec)

	if invocation.DryRun {
		return executor.Result{
			Invocation: invocation,
			Response: map[string]any{
				"dry_run":  true,
				"endpoint": httpclient.RedactURL(endpoint),
				"request":  ToolCallRequest(invocation.Tool, invocation.Params),
				"note":     "execution skipped by --dry-run",
			},
		}, nil
	}

	// Mock mode: return predefined mock response without network call.
	if r.globalFlags != nil && r.globalFlags.Mock {
		invocation.Implemented = true
		return executor.Result{
			Invocation: invocation,
			Response: map[string]any{
				"endpoint": httpclient.RedactURL(endpoint),
				"content": map[string]any{
					"success": true,
					"result":  []any{},
					"_mock":   true,
					"_tool":   invocation.Tool,
				},
			},
		}, nil
	}

	// Fail-fast: reject unauthenticated requests before making network calls.
	if strings.TrimSpace(authToken) == "" {
		return executor.Result{}, apperrors.NewAuth(
			"未登录，请先执行 ykc auth login",
			apperrors.WithReason("not_authenticated"),
			apperrors.WithHint("运行 'ykc auth login' 完成登录后重试"),
			apperrors.WithActions("ykc auth login"),
		)
	}

	tc := r.httpClient.WithAuth(authToken)
	if r.globalFlags != nil && r.globalFlags.Token != "" {
		tc = r.httpClient.WithBearerAuth(authToken)
	}

	callCtx := ctx
	if r.globalFlags != nil && r.globalFlags.Timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, time.Duration(r.globalFlags.Timeout)*time.Second)
		defer cancel()
	}

	callResult, err := tc.CallTool(callCtx, endpoint, invocation.Tool, invocation.Params)
	if err != nil {
		if isAuthError(err) {
			if fn := edition.Get().OnAuthError; fn != nil {
				_ = fn(defaultConfigDir(), err)
			}
		}
		captureRuntimeFailure(invocation, err, err)
		return executor.Result{}, err
	}

	if fn := edition.Get().ClassifyToolResult; fn != nil {
		if editionErr := fn(callResult.Content); editionErr != nil {
			return executor.Result{}, editionErr
		}
	}

	if callResult.IsError {
		diagMap := httpclient.ExtractServerDiagnosticsFromMap(callResult.Content)
		logBusinessError(fl, "tool_error", invocation, callResult.Content, diagMap)
		diag := apperrors.ServerDiagnostics{
			TraceID:         diagMap["trace_id"],
			ServerErrorCode: diagMap["server_error_code"],
		}
		apiErr := apperrors.NewAPI(
			extractErrorMessage(callResult),
			apperrors.WithOperation("tools/call"),
			apperrors.WithReason("tool_error"),
			apperrors.WithServerKey(invocation.CanonicalProduct),
			apperrors.WithHint("API 返回了错误响应，请检查参数和文档。"),
			apperrors.WithServerDiag(diag),
		)
		captureRuntimeFailure(invocation, apiErr, apiErr)
		return executor.Result{}, apiErr
	}

	scanReport, err := r.scanContent(callResult.Content)
	if err != nil {
		return executor.Result{}, err
	}

	if bizErr := detectBusinessError(callResult.Content); bizErr != "" {
		diagMap := httpclient.ExtractServerDiagnosticsFromMap(callResult.Content)
		logBusinessError(fl, "business_error", invocation, callResult.Content, diagMap)
		diag := apperrors.ServerDiagnostics{
			TraceID:         diagMap["trace_id"],
			ServerErrorCode: diagMap["server_error_code"],
		}
		return executor.Result{}, apperrors.NewAPI(bizErr,
			apperrors.WithOperation("tools/call"),
			apperrors.WithReason("business_error"),
			apperrors.WithServerKey(invocation.CanonicalProduct),
			apperrors.WithHint("API 返回了业务错误，请检查必填参数和值。"),
			apperrors.WithServerDiag(diag),
		)
	}

	invocation.Implemented = true
	response := map[string]any{
		"endpoint": httpclient.RedactURL(endpoint),
		"content":  callResult.Content,
	}
	if r.includeScanReport && scanReport.Scanned {
		response["safety"] = scanReport
	}
	return executor.Result{Invocation: invocation, Response: response}, nil
}

func (r *runtimeRunner) resolveAuthToken(ctx context.Context) string {
	explicitToken := ""
	if r != nil && r.globalFlags != nil {
		explicitToken = r.globalFlags.Token
	}
	return resolveRuntimeAuthToken(ctx, explicitToken)
}

func resolveRuntimeAuthToken(ctx context.Context, explicitToken string) string {
	if token := strings.TrimSpace(explicitToken); token != "" {
		return token
	}
	return getCachedRuntimeToken(ctx)
}

// Cached token state for process lifetime
var (
	cachedRuntimeToken     string
	cachedRuntimeTokenOnce sync.Once
)

// getCachedRuntimeToken returns a cached access token, loading it only once per process.
func getCachedRuntimeToken(ctx context.Context) string {
	cachedRuntimeTokenOnce.Do(func() {
		loadStart := time.Now()
		defer func() { RecordTiming(ctx, "auth_keychain", time.Since(loadStart)) }()

		token, tokenErr := ensureGokuaiAccessToken()
		if tokenErr != nil && errors.Is(tokenErr, authpkg.ErrTokenDecryption) {
			slog.Error(tokenErr.Error())
			return
		}
		if token != "" {
			cachedRuntimeToken = token
		}
	})
	return cachedRuntimeToken
}

// generateExecutionID returns a random 16-char hex string used to correlate
// all log entries belonging to a single command invocation.
func generateExecutionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ResetRuntimeTokenCache clears the cached token, forcing a reload on next access.
func ResetRuntimeTokenCache() {
	cachedRuntimeTokenOnce = sync.Once{}
	cachedRuntimeToken = ""
}

func newRuntimeContentScanner() safety.Scanner {
	if !runtimeFlagEnabled(os.Getenv(runtimeContentScanEnv), true) {
		return nil
	}
	return safety.NewContentScanner()
}

func (r *runtimeRunner) scanContent(content map[string]any) (safety.Report, error) {
	if r == nil || r.scanner == nil {
		return safety.Report{Scanned: false}, nil
	}
	report := r.scanner.ScanPayload(content)
	if r.enforceContentScan && len(report.Findings) > 0 {
		return report, apperrors.NewValidation("运行时响应被内容安全扫描拦截")
	}
	return report, nil
}

func runtimeFlagEnabled(raw string, defaultValue bool) bool {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return defaultValue
	}
	switch trimmed {
	case "0", "false", "no", "n", "off":
		return false
	default:
		return true
	}
}

func isAuthError(err error) bool {
	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		return appErr.Category == apperrors.CategoryAuth
	}
	return false
}

func productEndpointOverride(productID string) (string, bool) {
	key := "YKC_" + strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(productID), "-", "_")) + "_URL"
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", false
	}
	return value, true
}

// resolveIdentityHeaders loads or creates agent identity and returns HTTP
// headers to inject into requests.
func resolveIdentityHeaders() map[string]string {
	id := authpkg.EnsureExists(defaultConfigDir())
	headers := id.Headers()
	if headers == nil {
		headers = make(map[string]string)
	}

	// Inject environment variable based headers for gateway tracking
	envHeaders := map[string]string{
		"x-ykc-agent":      os.Getenv(envYkcAgent),
		"x-ykc-trace-id":   os.Getenv(envYkcTraceID),
		"x-ykc-session-id": os.Getenv(envYkcSessionID),
		"x-ykc-message-id": os.Getenv(envYkcMessageID),
	}
	for k, v := range envHeaders {
		if v != "" {
			headers[k] = v
		}
	}
	if fn := edition.Get().MergeHeaders; fn != nil {
		headers = fn(headers)
	}
	return headers
}

// detectBusinessError checks the response content for business errors.
func detectBusinessError(content map[string]any) string {
	success, ok := content["success"]
	if !ok {
		return ""
	}
	b, ok := success.(bool)
	if !ok || b {
		return ""
	}
	if msg, ok := content["errorMsg"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	if code, ok := content["errorCode"].(string); ok && strings.TrimSpace(code) != "" {
		return "business error: code " + strings.TrimSpace(code)
	}
	return "business error: success=false"
}

// extractErrorMessage builds an error message from a ToolCallResult.
func extractErrorMessage(result *httpclient.ToolCallResult) string {
	for _, block := range result.Blocks {
		text := strings.TrimSpace(block.Text)
		if text != "" {
			return text
		}
	}
	if msg, ok := result.Content["message"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	if msg, ok := result.Content["error"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	return "API returned an error response"
}

// logBusinessError logs tool errors and business errors to the file logger.
func logBusinessError(logger *slog.Logger, reason string, inv executor.Invocation, content map[string]any, diag map[string]string) {
	if logger == nil {
		return
	}
	attrs := []any{
		"product", inv.CanonicalProduct,
		"tool", inv.Tool,
		"reason", reason,
	}
	if traceID, ok := diag["trace_id"]; ok {
		attrs = append(attrs, "trace_id", traceID)
	}
	if errCode, ok := diag["server_error_code"]; ok {
		attrs = append(attrs, "server_error_code", errCode)
	}
	if msg, ok := content["error"].(string); ok {
		attrs = append(attrs, "error", msg)
	}
	if msg, ok := content["errorMsg"].(string); ok {
		attrs = append(attrs, "errorMsg", msg)
	}
	if msg, ok := content["message"].(string); ok {
		attrs = append(attrs, "message", msg)
	}
	logger.Warn("business_error", attrs...)
}
