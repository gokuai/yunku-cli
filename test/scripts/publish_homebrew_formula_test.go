package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishHomebrewFormulaPushesUpdatedFormula(t *testing.T) {
	t.Parallel()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release", "publish-homebrew-formula.sh"))
	if err != nil {
		t.Fatalf("Abs(publish-homebrew-formula.sh) error = %v", err)
	}

	root := t.TempDir()
	remoteDir := filepath.Join(root, "tap.git")
	mustRun(t, root, "git", "init", "--bare", remoteDir)
	seedTapRepo(t, remoteDir, "main", "class OldFormula < Formula\nend\n")

	sourceFormula := filepath.Join(root, "yunku-cli.rb")
	mustWriteFile(t, sourceFormula, []byte("class YunkuCli < Formula\n  desc \"Yunku CLI\"\nend\n"), 0o644)

	cmd := exec.Command("sh", scriptPath)
	cmd.Env = append(os.Environ(),
		"YKC_TAP_REPO_URL="+remoteDir,
		"YKC_TAP_BRANCH=main",
		"YKC_FORMULA_SOURCE="+sourceFormula,
		"YKC_GIT_NAME=Yunku Bot",
		"YKC_GIT_EMAIL=ykc@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("publish-homebrew-formula.sh error = %v\noutput:\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "Published Homebrew formula") {
		t.Fatalf("publish output missing success message:\n%s", string(output))
	}

	cloneDir := filepath.Join(root, "check")
	mustRun(t, root, "git", "clone", "--branch", "main", remoteDir, cloneDir)
	got, err := os.ReadFile(filepath.Join(cloneDir, "Formula", "yunku-cli.rb"))
	if err != nil {
		t.Fatalf("ReadFile(published formula) error = %v", err)
	}
	if string(got) != "class YunkuCli < Formula\n  desc \"Yunku CLI\"\nend\n" {
		t.Fatalf("published formula = %q", string(got))
	}
}

func TestPublishHomebrewFormulaSkipsWhenFormulaUnchanged(t *testing.T) {
	t.Parallel()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release", "publish-homebrew-formula.sh"))
	if err != nil {
		t.Fatalf("Abs(publish-homebrew-formula.sh) error = %v", err)
	}

	root := t.TempDir()
	remoteDir := filepath.Join(root, "tap.git")
	mustRun(t, root, "git", "init", "--bare", remoteDir)
	initialFormula := "class YunkuCli < Formula\n  desc \"Yunku CLI\"\nend\n"
	seedTapRepo(t, remoteDir, "main", initialFormula)

	sourceFormula := filepath.Join(root, "yunku-cli.rb")
	mustWriteFile(t, sourceFormula, []byte(initialFormula), 0o644)

	beforeHead := strings.TrimSpace(mustOutput(t, root, "git", "ls-remote", remoteDir, "refs/heads/main"))

	cmd := exec.Command("sh", scriptPath)
	cmd.Env = append(os.Environ(),
		"YKC_TAP_REPO_URL="+remoteDir,
		"YKC_TAP_BRANCH=main",
		"YKC_FORMULA_SOURCE="+sourceFormula,
		"YKC_GIT_NAME=Yunku Bot",
		"YKC_GIT_EMAIL=ykc@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("publish-homebrew-formula.sh error = %v\noutput:\n%s", err, string(output))
	}
	if !strings.Contains(string(output), "No formula changes to publish.") {
		t.Fatalf("publish output missing no-op message:\n%s", string(output))
	}

	afterHead := strings.TrimSpace(mustOutput(t, root, "git", "ls-remote", remoteDir, "refs/heads/main"))
	if beforeHead != afterHead {
		t.Fatalf("remote head changed unexpectedly:\nbefore: %s\nafter:  %s", beforeHead, afterHead)
	}
}

func seedTapRepo(t *testing.T, remoteDir, branch, formulaContent string) {
	t.Helper()

	workDir := t.TempDir()
	mustRun(t, t.TempDir(), "git", "clone", remoteDir, workDir)
	mustRun(t, workDir, "git", "config", "user.name", "Seed User")
	mustRun(t, workDir, "git", "config", "user.email", "seed@example.com")
	mustWriteFile(t, filepath.Join(workDir, "Formula", "yunku-cli.rb"), []byte(formulaContent), 0o644)
	mustRun(t, workDir, "git", "add", "Formula/yunku-cli.rb")
	mustRun(t, workDir, "git", "commit", "-m", "seed")
	mustRun(t, workDir, "git", "branch", "-M", branch)
	mustRun(t, workDir, "git", "push", "origin", branch)
}

func mustRun(t *testing.T, workdir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v error = %v\noutput:\n%s", name, args, err, string(output))
	}
}

func mustOutput(t *testing.T, workdir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v error = %v\noutput:\n%s", name, args, err, string(output))
	}
	return string(output)
}
