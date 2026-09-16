package gokuai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlexStringMapEmptyArray(t *testing.T) {
	// PHP-style API returns an empty map as [].
	input := `{"member_name":"张三","settings":[]}`

	var info AccountInfoResponse
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.Settings == nil {
		t.Fatal("settings should be an empty non-nil map")
	}

	out, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"settings":{}`) {
		t.Fatalf("settings should render as empty object: %s", out)
	}
}

func TestFlexStringMapNull(t *testing.T) {
	input := `{"settings":null}`

	var info AccountInfoResponse
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"settings":{}`) {
		t.Fatalf("settings should render as empty object: %s", out)
	}
}

func TestFlexStringMapNormal(t *testing.T) {
	input := `{"settings":{"k":"v"}}`

	var info AccountInfoResponse
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.Settings["k"] != "v" {
		t.Fatalf("expected settings.k=v, got: %v", info.Settings)
	}

	out, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"settings":{"k":"v"}`) {
		t.Fatalf("settings should render as object: %s", out)
	}
}
