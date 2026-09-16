package handlers

import (
	"github.com/gokuai/yunku-cli/internal/pipeline"
)

// RegisterHandler runs during the Register phase — the first stage
// in the pipeline, executed while the Cobra command tree is being
// built. It validates that the registration context carries a
// non-empty command identifier.
//
// The handler is intentionally lightweight and side-effect free.
// This provides the structural hook for future extensions (e.g.
// dynamic command injection, feature gating, or Raw API fallback
// command registration) without adding any runtime overhead to
// the default path. Logging is handled at the call site in
// canonical.go, consistent with how PreParse logging is done
// in cobra.go.
type RegisterHandler struct{}

func (RegisterHandler) Name() string          { return "register" }
func (RegisterHandler) Phase() pipeline.Phase { return pipeline.Register }

func (RegisterHandler) Handle(ctx *pipeline.Context) error {
	return nil
}
