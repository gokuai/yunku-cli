package logging

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// LogCommandStart logs the beginning of a command execution.
func LogCommandStart(logger *slog.Logger, executionId, product, tool, endpoint, version string, authPresent bool, timeoutSec int) {
	if logger == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("execution_id", executionId),
		slog.String("product", product),
		slog.String("tool", tool),
		slog.String("endpoint", redactEndpoint(endpoint)),
		slog.String("cli_version", version),
		slog.String("os", runtime.GOOS),
		slog.String("arch", runtime.GOARCH),
		slog.Bool("auth_token_present", authPresent),
	}
	if timeoutSec > 0 {
		attrs = append(attrs, slog.Int("timeout_sec", timeoutSec))
	}
	logger.LogAttrs(context.TODO(), slog.LevelInfo, "command_start", attrs...)
}

// LogCommandEnd logs the end of a command execution.
func LogCommandEnd(logger *slog.Logger, executionId, product, tool string, success bool, duration time.Duration, errCategory, errReason string) {
	if logger == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("execution_id", executionId),
		slog.String("product", product),
		slog.String("tool", tool),
		slog.Bool("success", success),
		slog.String("duration", duration.Truncate(time.Millisecond).String()),
	}
	if !success {
		attrs = append(attrs, slog.String("error_category", errCategory))
		attrs = append(attrs, slog.String("error_reason", errReason))
	}
	logger.LogAttrs(context.TODO(), slog.LevelInfo, "command_end", attrs...)
}

// redactEndpoint removes query parameters from endpoint URLs in logs.
func redactEndpoint(endpoint string) string {
	for i := 0; i < len(endpoint); i++ {
		if endpoint[i] == '?' || endpoint[i] == '#' {
			return endpoint[:i]
		}
	}
	return endpoint
}
