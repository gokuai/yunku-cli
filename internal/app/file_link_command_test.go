package app

import (
	"encoding/json"
	"testing"
)

func TestParseAnnotationExtension(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "plain object",
			input: `{"extension":"pdf|jpeg|jpg|png|gif|psd|bmp|tif|tiff"}`,
			want:  "pdf|jpeg|jpg|png|gif|psd|bmp|tif|tiff",
		},
		{
			name:  "json-encoded string",
			input: `"{\"extension\":\"pdf|jpg\"}"`,
			want:  "pdf|jpg",
		},
		{
			name:  "empty",
			input: ``,
			want:  "",
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseAnnotationExtension(json.RawMessage(c.input))
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", c.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
