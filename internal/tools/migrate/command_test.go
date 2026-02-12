package migrate

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewRootCommandStructure(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Use != "migrate" {
		t.Fatalf("unexpected root use: %s", cmd.Use)
	}
	if len(cmd.Commands()) != 3 {
		t.Fatalf("expected 3 subcommands, got %d", len(cmd.Commands()))
	}
	for _, name := range []string{"up", "status", "plan"} {
		if c, _, err := cmd.Find([]string{name}); err != nil || c == nil {
			t.Fatalf("expected subcommand %q: err=%v", name, err)
		}
	}
}

func TestRunCIPathSuccessAndError(t *testing.T) {
	opts := &options{ci: true, timeout: time.Second}
	details, err := run(opts, "title", "status", func(ctx context.Context) ([]string, error) {
		return []string{"ok"}, nil
	})
	if err != nil || len(details) != 1 || details[0] != "ok" {
		t.Fatalf("expected success details, got details=%v err=%v", details, err)
	}

	_, err = run(opts, "title", "status", func(ctx context.Context) ([]string, error) {
		return nil, context.DeadlineExceeded
	})
	if err == nil {
		t.Fatal("expected propagated error")
	}
}

func TestLoadConfigDBEnvParseError(t *testing.T) {
	envFile := t.TempDir() + "/bad.env"
	content := "JWT_ACCESS_TTL=not-a-duration\n"
	if err := osWriteFile(envFile, []byte(content)); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	if _, _, err := loadConfigDB(envFile); err == nil || !strings.Contains(err.Error(), "JWT_ACCESS_TTL") {
		t.Fatalf("expected config parse error, got %v", err)
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
