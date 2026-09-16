package scripts_test

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var expectedPackagedSkillTargets = []string{
	".agents/skills/ykc",
	".claude/skills/ykc",
	".cursor/skills/ykc",
	".gemini/skills/ykc",
	".codex/skills/ykc",
	".github/skills/ykc",
	".windsurf/skills/ykc",
	".augment/skills/ykc",
	".cline/skills/ykc",
	".amp/skills/ykc",
	".kiro/skills/ykc",
	".trae/skills/ykc",
	".openclaw/skills/ykc",
}

// seedDistArtifacts creates fake goreleaser output archives (empty tar.gz/zip
// files) and a checksums.txt stub so that post-goreleaser.sh can run without
// an actual goreleaser build.
func seedDistArtifacts(t *testing.T, distDir string, targets []string) {
	t.Helper()
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", distDir, err)
	}

	for _, target := range targets {
		p := filepath.Join(distDir, target)
		if err := os.WriteFile(p, []byte("fake-archive"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", p, err)
		}
	}

	// Create empty checksums.txt (goreleaser creates this)
	checksums := filepath.Join(distDir, "checksums.txt")
	var lines []string
	for _, target := range targets {
		lines = append(lines, "deadbeef00000000000000000000000000000000000000000000000000000000  "+target)
	}
	if err := os.WriteFile(checksums, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", checksums, err)
	}
}

func TestPostGoreleaserBuildsExpectedArtifacts(t *testing.T) {
	t.Parallel()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release", "post-goreleaser.sh"))
	if err != nil {
		t.Fatalf("Abs(post-goreleaser.sh) error = %v", err)
	}

	root := t.TempDir()
	distDir := filepath.Join(root, "dist")

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	archiveName := "ykc-" + hostOS + "-" + hostArch + ".tar.gz"
	if hostOS == "windows" {
		archiveName = "ykc-" + hostOS + "-" + hostArch + ".zip"
	}

	// Seed dist/ with fake goreleaser archives (simulate goreleaser output)
	seedDistArtifacts(t, distDir, []string{archiveName})

	cmd := exec.Command("sh", scriptPath)
	cmd.Env = append(os.Environ(),
		"YKC_PACKAGE_DIST_DIR="+distDir,
		"YKC_RELEASE_BASE_URL=https://downloads.example.com/releases/v1.2.3",
		"YKC_PACKAGE_VERSION=1.2.3",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-goreleaser.sh error = %v\noutput:\n%s", err, string(output))
	}

	for _, rel := range []string{
		"ykc-skills.zip",
		"checksums.txt",
		filepath.Join("npm", "yunku-cli", "package.json"),
		filepath.Join("homebrew", "yunku-cli.rb"),
		filepath.Join("homebrew", "yunku-cli-local.rb"),
	} {
		full := filepath.Join(distDir, rel)
		if _, err := os.Stat(full); err != nil {
			t.Fatalf("Stat(%s) error = %v\noutput:\n%s", full, err, string(output))
		}
	}

	// The skills zip must bundle the CLI reference doc and rewrite the
	// error-codes.md link so it resolves after installation.
	assertSkillsZipBundlesReference(t, filepath.Join(distDir, "ykc-skills.zip"))

	formulaPath := filepath.Join(distDir, "homebrew", "yunku-cli-local.rb")
	formulaData, err := os.ReadFile(formulaPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", formulaPath, err)
	}
	formulaText := string(formulaData)
	for _, want := range []string{
		"class YunkuCliLocal < Formula",
		"resource \"skills\" do",
		"Yunku CLI",
	} {
		if !strings.Contains(formulaText, want) {
			t.Fatalf("formula missing %q:\n%s", want, formulaText)
		}
	}

	releaseFormulaPath := filepath.Join(distDir, "homebrew", "yunku-cli.rb")
	releaseFormulaData, err := os.ReadFile(releaseFormulaPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", releaseFormulaPath, err)
	}
	releaseFormulaText := string(releaseFormulaData)
	for _, want := range []string{
		"class YunkuCli < Formula",
		"https://downloads.example.com/releases/v1.2.3/" + archiveName,
		"https://downloads.example.com/releases/v1.2.3/ykc-skills.zip",
	} {
		if !strings.Contains(releaseFormulaText, want) {
			t.Fatalf("release formula missing %q:\n%s", want, releaseFormulaText)
		}
	}

	packageJSONPath := filepath.Join(distDir, "npm", "yunku-cli", "package.json")
	packageJSON, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", packageJSONPath, err)
	}
	for _, want := range []string{
		"\"name\": \"yunku-cli\"",
		"Yunku CLI",
		"\"postinstall\": \"node install.js\"",
	} {
		if !strings.Contains(string(packageJSON), want) {
			t.Fatalf("package.json missing %q:\n%s", want, string(packageJSON))
		}
	}

	npmInstallPath := filepath.Join(distDir, "npm", "yunku-cli", "install.js")
	npmInstallData, err := os.ReadFile(npmInstallPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", npmInstallPath, err)
	}
	npmInstallText := string(npmInstallData)
	for _, target := range expectedPackagedSkillTargets {
		agentDir := strings.TrimSuffix(target, "/ykc")
		if !strings.Contains(npmInstallText, agentDir) {
			t.Fatalf("npm install.js missing %q:\n%s", agentDir, npmInstallText)
		}
	}

	for _, target := range expectedPackagedSkillTargets {
		if !strings.Contains(releaseFormulaText, target) {
			t.Fatalf("release formula missing %q:\n%s", target, releaseFormulaText)
		}
	}

	// Verify checksums.txt was updated to include skills zip
	checksumsData, err := os.ReadFile(filepath.Join(distDir, "checksums.txt"))
	if err != nil {
		t.Fatalf("ReadFile(checksums.txt) error = %v", err)
	}
	if !strings.Contains(string(checksumsData), "ykc-skills.zip") {
		t.Fatalf("checksums.txt missing ykc-skills.zip entry:\n%s", string(checksumsData))
	}
}

func TestPostGoreleaserAllPlatformNpmAssets(t *testing.T) {
	t.Parallel()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release", "post-goreleaser.sh"))
	if err != nil {
		t.Fatalf("Abs(post-goreleaser.sh) error = %v", err)
	}

	root := t.TempDir()
	distDir := filepath.Join(root, "dist")

	allArchives := []string{
		"ykc-darwin-amd64.tar.gz",
		"ykc-darwin-arm64.tar.gz",
		"ykc-linux-amd64.tar.gz",
		"ykc-linux-arm64.tar.gz",
		"ykc-windows-amd64.zip",
		"ykc-windows-arm64.zip",
	}

	// Seed dist/ with all platform archives (simulate goreleaser --target all)
	seedDistArtifacts(t, distDir, allArchives)

	cmd := exec.Command("sh", scriptPath)
	cmd.Env = append(os.Environ(),
		"YKC_PACKAGE_DIST_DIR="+distDir,
		"YKC_RELEASE_BASE_URL=https://downloads.example.com/releases/v9.9.9",
		"YKC_PACKAGE_VERSION=9.9.9",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-goreleaser.sh error = %v\noutput:\n%s", err, string(output))
	}

	for _, rel := range append(allArchives, "ykc-skills.zip", "checksums.txt") {
		full := filepath.Join(distDir, rel)
		if _, err := os.Stat(full); err != nil {
			t.Fatalf("Stat(%s) error = %v\noutput:\n%s", full, err, string(output))
		}
	}

	packageAssetsDir := filepath.Join(distDir, "npm", "yunku-cli", "assets")
	for _, rel := range append(allArchives, "ykc-skills.zip") {
		if _, err := os.Stat(filepath.Join(packageAssetsDir, rel)); err != nil {
			t.Fatalf("npm asset missing %q: %v", rel, err)
		}
	}
}

func TestPostGoreleaserUsesFlattenedSkillsSourceRoot(t *testing.T) {
	t.Parallel()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release", "post-goreleaser.sh"))
	if err != nil {
		t.Fatalf("Abs(post-goreleaser.sh) error = %v", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", scriptPath, err)
	}

	text := string(data)
	if !strings.Contains(text, `cp -R "$ROOT/skills/."`) {
		t.Fatalf("post-goreleaser.sh missing flattened skills staging copy:\n%s", text)
	}
	if strings.Contains(text, `cd "$ROOT/skills/ykc"`) {
		t.Fatalf("post-goreleaser.sh still references legacy nested skills root:\n%s", text)
	}
	// The packaged zip must bundle the CLI reference doc so that
	// references/error-codes.md links resolve after installation.
	if !strings.Contains(text, `cli-reference.md`) {
		t.Fatalf("post-goreleaser.sh missing bundled cli-reference.md staging:\n%s", text)
	}
}

// assertSkillsZipBundlesReference verifies the packaged skills zip contains
// the bundled CLI reference doc and that error-codes.md links to it instead
// of the repo-relative ../../docs/reference.md path.
func assertSkillsZipBundlesReference(t *testing.T, zipPath string) {
	t.Helper()

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("zip.OpenReader(%s) error = %v", zipPath, err)
	}
	defer zr.Close()

	var haveReference bool
	var errorCodes *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case "references/cli-reference.md":
			haveReference = true
		case "references/error-codes.md":
			errorCodes = f
		}
	}
	if !haveReference {
		t.Fatal("skills zip missing references/cli-reference.md")
	}
	if errorCodes == nil {
		t.Fatal("skills zip missing references/error-codes.md")
	}

	rc, err := errorCodes.Open()
	if err != nil {
		t.Fatalf("open error-codes.md in zip: %v", err)
	}
	defer rc.Close()
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error-codes.md in zip: %v", err)
	}
	if strings.Contains(string(content), "../../docs/reference.md") {
		t.Fatal("packaged error-codes.md still links to repo-relative ../../docs/reference.md")
	}
	if !strings.Contains(string(content), "./cli-reference.md") {
		t.Fatal("packaged error-codes.md missing rewritten ./cli-reference.md link")
	}
}
