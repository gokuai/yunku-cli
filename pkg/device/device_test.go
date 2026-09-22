package device

import (
	"encoding/json"
	"testing"
)

func TestParams(t *testing.T) {
	params := Params("test-client-id")

	info, ok := params["info"]
	if !ok {
		t.Fatal("Params missing info")
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(info), &parsed); err != nil {
		t.Fatalf("info is not valid JSON: %v", err)
	}
	if parsed["CLIENT_ID"] != "test-client-id" {
		t.Errorf("CLIENT_ID = %q, want %q", parsed["CLIENT_ID"], "test-client-id")
	}
	for _, key := range []string{"VERSION", "BRAND", "DISPLAY"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("info missing key %s", key)
		}
	}
}

func TestCollect(t *testing.T) {
	// 值取决于运行平台，只需保证不 panic
	_ = Collect()
}
