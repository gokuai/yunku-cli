package handlers

import (
	"github.com/gokuai/yunku-cli/internal/pipeline"
)

// PostResponseHandler runs in the PostResponse phase — after the
// transport returns a result and before the output is written to
// stdout. It receives the raw response and can mutate it.
//
// Default behaviour: no-op pass-through. This establishes the
// extension point for:
//   - Output format transformation (e.g. table, CSV, YAML renderers)
//   - Response field filtering or redaction
//   - Pagination metadata injection
//   - Response caching or analytics collection
//
// Logging is handled at the integration point in canonical.go,
// consistent with how other phases log at their call sites.
type PostResponseHandler struct{}

func (PostResponseHandler) Name() string          { return "postresponse" }
func (PostResponseHandler) Phase() pipeline.Phase { return pipeline.PostResponse }

func (PostResponseHandler) Handle(ctx *pipeline.Context) error {
	return nil
}
