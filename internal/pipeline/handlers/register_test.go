package handlers

import (
	"testing"

	"github.com/gokuai/yunku-cli/internal/pipeline"
)

func TestRegisterHandlerMeta(t *testing.T) {
	h := RegisterHandler{}
	if got := h.Name(); got != "register" {
		t.Errorf("Name() = %q, want %q", got, "register")
	}
	if got := h.Phase(); got != pipeline.Register {
		t.Errorf("Phase() = %v, want %v", got, pipeline.Register)
	}
}

func TestRegisterHandlerEmptyContext(t *testing.T) {
	h := RegisterHandler{}
	ctx := &pipeline.Context{}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

func TestRegisterHandlerWithCommand(t *testing.T) {
	h := RegisterHandler{}
	ctx := &pipeline.Context{
		Command: "contact",
		Schema: map[string]any{
			"properties": map[string]any{
				"userId": map[string]any{"type": "string"},
			},
		},
	}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

func TestRegisterHandlerNoSideEffects(t *testing.T) {
	h := RegisterHandler{}
	ctx := &pipeline.Context{
		Command: "contact",
		Params:  map[string]any{"key": "value"},
	}
	if err := h.Handle(ctx); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if ctx.Params["key"] != "value" {
		t.Error("RegisterHandler should not mutate Params")
	}
	if ctx.Command != "contact" {
		t.Error("RegisterHandler should not mutate Command")
	}
}

func TestRegisterHandlerInEngine(t *testing.T) {
	engine := pipeline.NewEngine()
	engine.Register(RegisterHandler{})

	if !engine.HasHandlers(pipeline.Register) {
		t.Fatal("engine should have Register handler")
	}

	ctx := &pipeline.Context{Command: "contact"}
	if err := engine.RunPhase(pipeline.Register, ctx); err != nil {
		t.Fatalf("RunPhase(Register) returned error: %v", err)
	}
}
