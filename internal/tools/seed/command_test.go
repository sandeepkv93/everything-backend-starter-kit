package seed

import (
	"context"
	"testing"
)

func TestNewRootCommandStructure(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Use != "seed" {
		t.Fatalf("unexpected root use: %s", cmd.Use)
	}
	for _, name := range []string{"apply", "dry-run", "verify-local-email"} {
		if c, _, err := cmd.Find([]string{name}); err != nil || c == nil {
			t.Fatalf("expected subcommand %q: err=%v", name, err)
		}
	}
	verify, _, err := cmd.Find([]string{"verify-local-email"})
	if err != nil {
		t.Fatalf("find verify-local-email: %v", err)
	}
	if f := verify.Flags().Lookup("email"); f == nil {
		t.Fatal("expected --email flag on verify-local-email")
	}
}

func TestRunCIPath(t *testing.T) {
	opts := &options{ci: true}
	details, err := run(opts, "title", "apply", func(ctx context.Context) ([]string, error) {
		return []string{"done"}, nil
	})
	if err != nil || len(details) != 1 {
		t.Fatalf("expected success details, got details=%v err=%v", details, err)
	}
}
