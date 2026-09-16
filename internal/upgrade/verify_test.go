package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile")
	content := []byte("hello world")
	os.WriteFile(path, content, 0644)

	hash, err := ComputeSHA256(path)
	if err != nil {
		t.Fatalf("ComputeSHA256() error = %v", err)
	}

	expected := sha256.Sum256(content)
	want := hex.EncodeToString(expected[:])
	if hash != want {
		t.Errorf("ComputeSHA256() = %q, want %q", hash, want)
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile")
	content := []byte("hello world")
	os.WriteFile(path, content, 0644)

	expected := sha256.Sum256(content)
	hash := hex.EncodeToString(expected[:])

	if err := VerifySHA256(path, hash); err != nil {
		t.Errorf("VerifySHA256() with correct hash: error = %v", err)
	}

	if err := VerifySHA256(path, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("VerifySHA256() with wrong hash: expected error")
	}
}

func TestParseChecksumFile(t *testing.T) {
	content := `abc123def456  ykc-darwin-arm64.tar.gz
fedcba987654  ykc-linux-amd64.tar.gz
`

	result := ParseChecksumFile(content)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result["ykc-darwin-arm64.tar.gz"] != "abc123def456" {
		t.Errorf("darwin hash = %q", result["ykc-darwin-arm64.tar.gz"])
	}
	if result["ykc-linux-amd64.tar.gz"] != "fedcba987654" {
		t.Errorf("linux hash = %q", result["ykc-linux-amd64.tar.gz"])
	}
}

func TestVerifyFileFromChecksums(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myfile.bin")
	content := []byte("test content")
	os.WriteFile(path, content, 0644)

	expected := sha256.Sum256(content)
	hash := hex.EncodeToString(expected[:])
	checksums := hash + "  myfile.bin\nother  otherfile.bin\n"

	if err := VerifyFileFromChecksums(path, "myfile.bin", checksums); err != nil {
		t.Errorf("VerifyFileFromChecksums() with correct hash: error = %v", err)
	}

	if err := VerifyFileFromChecksums(path, "missing.bin", checksums); err == nil {
		t.Error("VerifyFileFromChecksums() with missing file: expected error")
	}
}
