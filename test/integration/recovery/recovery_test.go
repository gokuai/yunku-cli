package recovery_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokuai/yunku-cli/internal/app"
	"github.com/gokuai/yunku-cli/internal/recovery"
)

// TestRecoveryClosedLoop drives the full recovery lifecycle through the real
// command surface: capture a failure snapshot via the recovery store (the same
// path used when a command fails), then run `ykc recovery plan / execute /
// finalize` and verify the event log gains exactly one phase entry per step —
// in particular, execute/finalize must not recursively capture their own
// internal failures as new events.
func TestRecoveryClosedLoop(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YKC_CONFIG_DIR", configDir)

	// ── Capture: simulate a failed `contact member info` invocation ──────────
	store := recovery.NewStore(configDir)
	rawErr := errors.New("error_code: 40406, error_msg: 用户不存在")
	input := recovery.CaptureInput{
		CommandPath: []string{"contact", "member", "info"},
		ServerID:    "contact",
		ToolName:    "member_info",
		Args:        map[string]any{"ent_id": "12345", "member_id": "USER_MISSING"},
		Argv:        []string{"contact", "member", "info", "--ent-id", "12345", "--member-id", "USER_MISSING"},
		RawErr:      rawErr,
		WrappedErr:  rawErr,
	}
	captured, err := store.Capture(recovery.BuildContext(input), recovery.BuildReplay(input))
	if err != nil {
		t.Fatalf("store.Capture() error = %v", err)
	}
	if captured.EventID == "" {
		t.Fatal("expected captured event id")
	}

	last, err := store.LoadLastError()
	if err != nil {
		t.Fatalf("LoadLastError() error = %v", err)
	}
	if last.EventID != captured.EventID {
		t.Fatalf("LoadLastError().EventID = %q, want %q", last.EventID, captured.EventID)
	}
	if got := strings.Join(last.Context.CommandPath, " "); got != "contact member info" {
		t.Fatalf("captured command path = %q, want \"contact member info\"", got)
	}

	eventsPath := filepath.Join(configDir, "recovery", "recovery_events.jsonl")
	beforePlan, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("ReadFile(recovery_events.jsonl) error = %v", err)
	}
	if got := countPhase(beforePlan, "captured"); got != 1 {
		t.Fatalf("captured phase count after capture = %d, want 1", got)
	}

	// ── Plan ──────────────────────────────────────────────────────────────────
	root := app.NewRootCommand()
	var planOut bytes.Buffer
	root.SetOut(&planOut)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"recovery", "plan", "--event-id", captured.EventID, "-f", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(recovery plan) error = %v", err)
	}
	if !strings.Contains(planOut.String(), captured.EventID) {
		t.Fatalf("plan output missing event id:\n%s", planOut.String())
	}

	// ── Execute ───────────────────────────────────────────────────────────────
	root = app.NewRootCommand()
	var executeOut bytes.Buffer
	root.SetOut(&executeOut)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"recovery", "execute", "--event-id", captured.EventID, "-f", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(recovery execute) error = %v", err)
	}

	var bundle recovery.RecoveryBundle
	if err := json.Unmarshal(executeOut.Bytes(), &bundle); err != nil {
		t.Fatalf("json.Unmarshal(bundle) error = %v\noutput:\n%s", err, executeOut.String())
	}
	if bundle.EventID != captured.EventID {
		t.Fatalf("bundle.EventID = %q, want %q", bundle.EventID, captured.EventID)
	}

	afterExecute, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("ReadFile(recovery_events.jsonl after execute) error = %v", err)
	}
	if got := countPhase(afterExecute, "captured"); got != 1 {
		t.Fatalf("captured phase count after execute = %d, want 1 (execute must not recursively capture)", got)
	}
	if got := countPhase(afterExecute, "analyzed"); got != 1 {
		t.Fatalf("analyzed phase count after execute = %d, want 1", got)
	}

	// ── Finalize ──────────────────────────────────────────────────────────────
	executionFile := filepath.Join(configDir, "execution.json")
	executionPayload := `{"actions":["inspect_bundle","handoff"],"attempts":[{"command_summary":"ykc contact member info --ent-id 12345 --member-id USER_MISSING --format json","result":"failed","error_summary":"member still missing","source":"agent_analysis"}],"result":"handoff","error_summary":"contact member probe still failing"}`
	if err := os.WriteFile(executionFile, []byte(executionPayload), 0o600); err != nil {
		t.Fatalf("WriteFile(execution.json) error = %v", err)
	}

	root = app.NewRootCommand()
	var finalizeOut bytes.Buffer
	root.SetOut(&finalizeOut)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"recovery", "finalize",
		"--event-id", captured.EventID,
		"--outcome", "handoff",
		"--execution-file", executionFile,
		"-f", "json",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(recovery finalize) error = %v", err)
	}
	if !strings.Contains(finalizeOut.String(), `"execution_recorded": true`) {
		t.Fatalf("finalize output missing execution_recorded:\n%s", finalizeOut.String())
	}

	afterFinalize, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("ReadFile(recovery_events.jsonl after finalize) error = %v", err)
	}
	if got := countPhase(afterFinalize, "captured"); got != 1 {
		t.Fatalf("captured phase count after finalize = %d, want 1", got)
	}
	if got := countPhase(afterFinalize, "finalized"); got != 1 {
		t.Fatalf("finalized phase count after finalize = %d, want 1", got)
	}
}

func countPhase(data []byte, phase string) int {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, `"phase":"`+phase+`"`) {
			count++
		}
	}
	return count
}
