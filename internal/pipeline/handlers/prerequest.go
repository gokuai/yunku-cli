package handlers

import (
	"github.com/gokuai/yunku-cli/internal/pipeline"
)

// PreRequestHandler runs in the PreRequest phase — after parameter
// validation succeeds and just before the JSON-RPC call is dispatched.
// It receives the final payload and can inspect or mutate it.
//
// Default behaviour: no-op pass-through. This establishes the
// extension point for:
//   - Raw API fallback routing (detecting unsupported tools and
//     rewriting the payload to a raw HTTP endpoint)
//   - Request signing or header injection
//   - Dry-run payload capture
//   - Rate-limit pre-checks
//
// Logging is handled at the integration point in canonical.go,
// consistent with how other phases log at their call sites.
type PreRequestHandler struct{}

func (PreRequestHandler) Name() string          { return "prerequest" }
func (PreRequestHandler) Phase() pipeline.Phase { return pipeline.PreRequest }

func (PreRequestHandler) Handle(ctx *pipeline.Context) error {
	return nil
}
