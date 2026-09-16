package errors

import "strings"

// ServerDiagnostics holds server-side diagnostic fields extracted from
// API response bodies or HTTP response headers. Fields are populated
// on a best-effort basis during error construction.
type ServerDiagnostics struct {
	TraceID         string `json:"trace_id,omitempty"`
	ServerErrorCode string `json:"server_error_code,omitempty"`
	TechnicalDetail string `json:"technical_detail,omitempty"`
	ServerRetryable *bool  `json:"server_retryable,omitempty"`
}

// IsEmpty returns true when no diagnostic field has been populated.
func (d ServerDiagnostics) IsEmpty() bool {
	return d.TraceID == "" && d.ServerErrorCode == "" &&
		d.TechnicalDetail == "" && d.ServerRetryable == nil
}

// WithServerDiag attaches server diagnostics to the error.
func WithServerDiag(diag ServerDiagnostics) Option {
	if diag.IsEmpty() {
		return func(*Error) {}
	}
	return func(e *Error) {
		e.ServerDiag = diag
		// Override retryable if server explicitly specified.
		if diag.ServerRetryable != nil {
			e.Retryable = *diag.ServerRetryable
		}
	}
}

// WithTraceID records the server-provided trace identifier.
// Used when only the trace ID is available (e.g. from HTTP headers)
// without a full ServerDiagnostics struct.
func WithTraceID(id string) Option {
	id = strings.TrimSpace(id)
	if id == "" {
		return func(*Error) {}
	}
	return func(e *Error) {
		e.ServerDiag.TraceID = id
	}
}
