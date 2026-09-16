package output

import "github.com/gokuai/yunku-cli/pkg/validate"

// SanitizeForTerminal strips ANSI escape sequences, control characters, and
// dangerous Unicode from text before it is printed to a terminal.
// Delegates to the validate package which provides the canonical implementation.
func SanitizeForTerminal(text string) string {
	return validate.SanitizeForTerminal(text)
}
