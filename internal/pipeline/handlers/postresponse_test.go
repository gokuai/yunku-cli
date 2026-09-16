package handlers

import (
	"testing"

	"github.com/gokuai/yunku-cli/internal/pipeline"
)

func TestPostResponseHandlerMeta(t *testing.T) {
	h := PostResponseHandler{}
	if got := h.Name(); got != "postresponse" {
		t.Errorf("Name() = %q, want %q", got, "postresponse")
	}
	if got := h.Phase(); got != pipeline.PostResponse {
		t.Errorf("Phase() = %v, want %v", got, pipeline.PostResponse)
	}
}

func TestPostResponseHandlerEmptyContext(t *testing.T) {
	h := PostResponseHandler{}
	ctx := &pipeline.Context{}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

func TestPostResponseHandlerNoSideEffects(t *testing.T) {
	h := PostResponseHandler{}
	ctx := &pipeline.Context{
		Command: "contact.user.search",
		Response: map[string]any{
			"users": []any{
				map[string]any{"userId": "u001"},
			},
			"total": 1,
		},
	}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if ctx.Response["total"] != 1 {
		t.Error("PostResponseHandler should not mutate Response")
	}
}

func TestPostResponseHandlerNilResponse(t *testing.T) {
	h := PostResponseHandler{}
	ctx := &pipeline.Context{
		Command:  "ding.send",
		Response: nil,
	}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

func TestPostResponseHandlerInEngine(t *testing.T) {
	engine := pipeline.NewEngine()
	engine.Register(PostResponseHandler{})

	if !engine.HasHandlers(pipeline.PostResponse) {
		t.Fatal("engine should have PostResponse handler")
	}

	ctx := &pipeline.Context{
		Command:  "contact.user.get",
		Response: map[string]any{"users": []any{}},
	}
	if err := engine.RunPhase(pipeline.PostResponse, ctx); err != nil {
		t.Fatalf("RunPhase(PostResponse) returned error: %v", err)
	}
}
