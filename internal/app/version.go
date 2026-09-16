package app

var version = "dev"

// SetVersion overrides the version, build time and git commit strings.
// Called by pkg/cli.SetVersion for overlay modules that inject their own
// version info via ldflags.
func SetVersion(v, bt, gc string) {
	if v != "" {
		version = v
	}
	if bt != "" {
		buildTime = bt
	}
	if gc != "" {
		gitCommit = gc
	}
}

// Version returns the current CLI version string, including build metadata
// when injected via ldflags (buildTime, gitCommit).
func Version() string {
	if buildTime != "unknown" || gitCommit != "unknown" {
		return version + " (" + gitCommit + ", " + buildTime + ")"
	}
	return version
}

// RawVersion returns the bare version string without build metadata.
func RawVersion() string { return version }

// BuildTime returns the build timestamp injected via ldflags.
func BuildTime() string { return buildTime }

// GitCommit returns the git commit hash injected via ldflags.
func GitCommit() string { return gitCommit }
