package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileMissingIsNoop(t *testing.T) {
	if err := LoadEnvFile(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("missing env file should be ignored: %v", err)
	}
}

func TestLoadEnvFileLoadsAndPreservesExisting(t *testing.T) {
	t.Setenv("EXISTING_KEY", "from-env")
	file := filepath.Join(t.TempDir(), "test.env")
	content := "# comment\nEXISTING_KEY=from-file\nNEW_KEY=hello\nQUOTED=\"x\"\nINVALID_LINE\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := LoadEnvFile(file); err != nil {
		t.Fatalf("load env file: %v", err)
	}
	if got := os.Getenv("EXISTING_KEY"); got != "from-env" {
		t.Fatalf("expected existing var to be preserved, got %q", got)
	}
	if got := os.Getenv("NEW_KEY"); got != "hello" {
		t.Fatalf("unexpected NEW_KEY=%q", got)
	}
	if got := os.Getenv("QUOTED"); got != "x" {
		t.Fatalf("unexpected QUOTED=%q", got)
	}
}

func TestLoadEnvFileOpenError(t *testing.T) {
	dir := t.TempDir()
	if err := LoadEnvFile(dir); err == nil {
		t.Fatal("expected error when path is a directory")
	}
}
