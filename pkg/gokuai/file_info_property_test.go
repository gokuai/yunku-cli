package gokuai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFileInfoPropertyRendersAsJSON(t *testing.T) {
	// The API returns property as a JSON-encoded string.
	input := `{
		"hash": "abc",
		"mount_id": "1",
		"property": "{\"k\":\"v\",\"n\":42}"
	}`

	var info FileInfoResponseV2
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// property must be re-emitted as a nested JSON object, not a quoted string.
	if !strings.Contains(string(out), `"property":{"k":"v","n":42}`) {
		t.Fatalf("property not rendered as JSON object: %s", out)
	}
	if strings.Contains(string(out), `"{\"k\"`) {
		t.Fatalf("property still rendered as escaped string: %s", out)
	}
}

func TestFileInfoPropertyAlreadyObject(t *testing.T) {
	input := `{"hash":"abc","property":{"k":"v"}}`

	var info FileInfoResponseV2
	if err := json.Unmarshal([]byte(input), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"property":{"k":"v"}`) {
		t.Fatalf("property not preserved as object: %s", out)
	}
}

func TestFileInfoPropertyNullAndEmpty(t *testing.T) {
	for _, input := range []string{
		`{"hash":"a","property":null}`,
		`{"hash":"a","property":""}`,
	} {
		var info FileInfoResponseV2
		if err := json.Unmarshal([]byte(input), &info); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}
		out, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(out), `"property":""`) {
			t.Fatalf("expected empty property for input %s, got %s", input, out)
		}
	}
}
