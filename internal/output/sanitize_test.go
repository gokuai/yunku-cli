package output

import "testing"

func TestSanitizeForTerminalStripsControlCharacters(t *testing.T) {
	t.Parallel()

	// ANSI escape sequences and control characters are fully stripped.
	got := SanitizeForTerminal("hello\x1b[31mworld\x07")
	want := "helloworld"
	if got != want {
		t.Fatalf("SanitizeForTerminal() = %q, want %q", got, want)
	}
}

func TestSanitizeForTerminalStripsDangerousUnicode(t *testing.T) {
	t.Parallel()

	got := SanitizeForTerminal("a\u202Eb\u200Bc")
	want := "abc"
	if got != want {
		t.Fatalf("SanitizeForTerminal() = %q, want %q", got, want)
	}
}

func TestSanitizeForTerminalPreservesReadableWhitespace(t *testing.T) {
	t.Parallel()

	got := SanitizeForTerminal("line1\nline2\tvalue")
	want := "line1\nline2\tvalue"
	if got != want {
		t.Fatalf("SanitizeForTerminal() = %q, want %q", got, want)
	}
}
