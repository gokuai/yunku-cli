// Package cli exposes public entry points for the ykc CLI so that external
// overlay modules can embed and launch the CLI after customising edition hooks.
package cli

import "github.com/gokuai/yunku-cli/internal/app"

// SetVersion overrides the version, build time and git commit strings
// that are displayed by `ykc version` and `ykc --version`.
// Typically called by overlay main.go with values injected via ldflags.
func SetVersion(v, buildTime, gitCommit string) {
	app.SetVersion(v, buildTime, gitCommit)
}

// Execute runs the root CLI command and returns the process exit code.
func Execute() int {
	return app.Execute()
}
