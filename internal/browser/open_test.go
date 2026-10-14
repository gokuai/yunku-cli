package browser

import (
	"reflect"
	"testing"
)

func TestOpenArgs(t *testing.T) {
	const url = "https://passport.gokuai.com/oauth/login?x=1"

	cases := []struct {
		goos    string
		wantBin string
		wantErr bool
	}{
		{"darwin", "open", false},
		{"windows", "rundll32", false},
		{"linux", "xdg-open", false},
		{"freebsd", "xdg-open", false},
		{"plan9", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.goos, func(t *testing.T) {
			args, err := openArgs(tc.goos, url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("openArgs(%q) expected error, got %v", tc.goos, args)
				}
				return
			}
			if err != nil {
				t.Fatalf("openArgs(%q) unexpected error: %v", tc.goos, err)
			}
			if args[0] != tc.wantBin {
				t.Errorf("binary = %q, want %q", args[0], tc.wantBin)
			}
			if args[len(args)-1] != url {
				t.Errorf("last arg = %q, want the URL", args[len(args)-1])
			}
		})
	}
}

func TestBrowserArgs_EnvOverride(t *testing.T) {
	const url = "https://passport.gokuai.com/oauth/login"

	t.Run("BROWSER set wins", func(t *testing.T) {
		t.Setenv("BROWSER", "firefox --private-window")
		want := []string{"firefox", "--private-window", url}
		got, err := browserArgs(url)
		if err != nil {
			t.Fatalf("browserArgs error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("browserArgs = %v, want %v", got, want)
		}
	})

	t.Run("BROWSER empty falls back to platform default", func(t *testing.T) {
		t.Setenv("BROWSER", "")
		got, err := browserArgs(url)
		if err != nil {
			t.Fatalf("browserArgs error: %v", err)
		}
		if got[len(got)-1] != url {
			t.Errorf("last arg = %q, want the URL", got[len(got)-1])
		}
	})
}
