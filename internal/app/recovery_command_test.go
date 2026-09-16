package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gokuai/yunku-cli/internal/recovery"
)

func TestRecoveryPlanReadsLastSnapshotAndPrintsJSON(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YKC_CONFIG_DIR", configDir)
	writeRecoverySnapshot(t, configDir, recovery.LastError{
		EventID:    "evt_test",
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Context: recovery.RecoveryContext{
			CommandPath:   []string{"approval", "instance", "get"},
			ServerID:      "approval",
			ToolName:      "get_approval_instance",
			OperationKind: recovery.OperationRead,
			CLIErrorCode:  "RESOURCE_NOT_FOUND",
			RawError:      "resource_not_found",
			Fingerprint:   "fp-1",
		},
		Replay: recovery.Replay{
			ServerID:        "approval",
			ToolName:        "get_approval_instance",
			OperationKind:   recovery.OperationRead,
			ToolArgs:        map[string]any{"instanceId": "ins_1"},
			RedactedCommand: "ykc approval instance get --instance-id ins_1 --format json",
		},
	})

	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"recovery", "plan", "--last", "-f", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(recovery plan) error = %v", err)
	}
	if !strings.Contains(out.String(), `"event_id": "evt_test"`) {
		t.Fatalf("output missing event id:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `"category": "resource"`) {
		t.Fatalf("output missing resource category:\n%s", out.String())
	}
}

func TestRecoveryExecuteReadsLastSnapshotAndPrintsJSON(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YKC_CONFIG_DIR", configDir)
	writeRecoverySnapshot(t, configDir, recovery.LastError{
		EventID:    "evt_exec",
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Context: recovery.RecoveryContext{
			CommandPath:   []string{"approval", "instance", "get"},
			ServerID:      "approval",
			ToolName:      "get_approval_instance",
			OperationKind: recovery.OperationRead,
			CLIErrorCode:  "RESOURCE_NOT_FOUND",
			RawError:      "resource_not_found",
			Fingerprint:   "fp-2",
		},
		Replay: recovery.Replay{
			ServerID:        "approval",
			ToolName:        "get_approval_instance",
			OperationKind:   recovery.OperationRead,
			ToolArgs:        map[string]any{"instanceId": "ins_1"},
			RedactedCommand: "ykc approval instance get --instance-id ins_1 --format json",
		},
	})

	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"recovery", "execute", "--last", "-f", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(recovery execute) error = %v", err)
	}
	if !strings.Contains(out.String(), `"event_id": "evt_exec"`) {
		t.Fatalf("output missing event id:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `"status": "needs_agent_action"`) {
		t.Fatalf("output missing bundle status:\n%s", out.String())
	}
}

func TestRecoveryFinalizeRequiresEventIDAndOutcome(t *testing.T) {
	root := NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"recovery", "finalize"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute(recovery finalize) error = nil, want validation")
	}
	if !strings.Contains(err.Error(), "--event-id") {
		t.Fatalf("error = %v, want event-id requirement", err)
	}
}

func TestRecoveryPlanRejectsLastAndEventIDTogether(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YKC_CONFIG_DIR", configDir)
	writeRecoverySnapshot(t, configDir, recovery.LastError{
		EventID:    "evt_conflict",
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Context: recovery.RecoveryContext{
			CommandPath:   []string{"approval", "instance", "get"},
			ServerID:      "approval",
			ToolName:      "get_approval_instance",
			OperationKind: recovery.OperationRead,
			CLIErrorCode:  "RESOURCE_NOT_FOUND",
			RawError:      "resource_not_found",
			Fingerprint:   "fp-conflict",
		},
	})

	root := NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"recovery", "plan", "--last", "--event-id", "evt_conflict"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute(recovery plan) error = nil, want conflict validation")
	}
	if !strings.Contains(err.Error(), "--last") || !strings.Contains(err.Error(), "--event-id") {
		t.Fatalf("error = %v, want mutually exclusive flags", err)
	}
}

func TestRecoveryFinalizeAcceptsLegacyExecutionFile(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("YKC_CONFIG_DIR", configDir)
	writeRecoverySnapshot(t, configDir, recovery.LastError{
		EventID:    "evt_legacy_finalize",
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Context: recovery.RecoveryContext{
			CommandPath:   []string{"approval", "instance", "get"},
			ServerID:      "approval",
			ToolName:      "get_approval_instance",
			OperationKind: recovery.OperationUnknown,
			RawError:      "unexpected upstream failure",
			Fingerprint:   "fp-legacy-finalize",
		},
	})

	executionPath := filepath.Join(configDir, "legacy_execution.json")
	if err := os.WriteFile(executionPath, []byte(`{"action":"verify_resource_exists","attempts":2,"result":"failed","error":"resource still missing"}`), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy execution) error = %v", err)
	}

	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"recovery", "finalize",
		"--event-id", "evt_legacy_finalize",
		"--outcome", "failed",
		"--execution-file", executionPath,
		"-f", "json",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(recovery finalize) error = %v", err)
	}
	if !strings.Contains(out.String(), `"execution_recorded": true`) {
		t.Fatalf("output missing execution_recorded flag:\n%s", out.String())
	}

	data, err := os.ReadFile(filepath.Join(configDir, "recovery", "recovery_events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile(recovery_events.jsonl) error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, `"phase":"finalized"`) {
		t.Fatalf("expected finalized event, got %s", lastLine)
	}
	if !strings.Contains(lastLine, `"legacy_execution_file"`) {
		t.Fatalf("expected legacy execution attempts to be normalized, got %s", lastLine)
	}
}

func writeRecoverySnapshot(t *testing.T, configDir string, last recovery.LastError) {
	t.Helper()

	recoveryDir := filepath.Join(configDir, "recovery")
	if err := os.MkdirAll(recoveryDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(recovery) error = %v", err)
	}
	data, err := json.MarshalIndent(last, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(recoveryDir, "last_error.json"), append(data, '\n'), 0o600); err != nil {
		t.Fatalf("WriteFile(last_error.json) error = %v", err)
	}
}
