package handlers

import (
	"testing"

	"github.com/gokuai/yunku-cli/internal/pipeline"
)

func TestPreRequestHandlerMeta(t *testing.T) {
	h := PreRequestHandler{}
	if got := h.Name(); got != "prerequest" {
		t.Errorf("Name() = %q, want %q", got, "prerequest")
	}
	if got := h.Phase(); got != pipeline.PreRequest {
		t.Errorf("Phase() = %v, want %v", got, pipeline.PreRequest)
	}
}

func TestPreRequestHandlerEmptyContext(t *testing.T) {
	h := PreRequestHandler{}
	ctx := &pipeline.Context{}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

func TestPreRequestHandlerNoSideEffects(t *testing.T) {
	h := PreRequestHandler{}
	ctx := &pipeline.Context{
		Command: "contact.user.get",
		Params: map[string]any{
			"userId": "u001",
			"name":   "test",
		},
		Payload: map[string]any{
			"userId": "u001",
			"name":   "test",
		},
	}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(ctx.Params) != 2 || ctx.Params["userId"] != "u001" || ctx.Params["name"] != "test" {
		t.Errorf("PreRequestHandler should not mutate Params, got %v", ctx.Params)
	}
	if len(ctx.Payload) != 2 || ctx.Payload["userId"] != "u001" || ctx.Payload["name"] != "test" {
		t.Errorf("PreRequestHandler should not mutate Payload, got %v", ctx.Payload)
	}
}

func TestPreRequestHandlerNilPayload(t *testing.T) {
	h := PreRequestHandler{}
	ctx := &pipeline.Context{
		Command: "chat.send_message",
		Params:  map[string]any{"userId": "u001"},
	}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

func TestPreRequestHandlerInEngine(t *testing.T) {
	engine := pipeline.NewEngine()
	engine.Register(PreRequestHandler{})

	if !engine.HasHandlers(pipeline.PreRequest) {
		t.Fatal("engine should have PreRequest handler")
	}

	ctx := &pipeline.Context{
		Command: "ding.send",
		Params:  map[string]any{"text": "test"},
		Payload: map[string]any{"text": "test"},
	}
	if err := engine.RunPhase(pipeline.PreRequest, ctx); err != nil {
		t.Fatalf("RunPhase(PreRequest) returned error: %v", err)
	}
}
