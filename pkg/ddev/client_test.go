package ddev

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIDdevNotFound(t *testing.T) {
	c := &CLI{Bin: "dpilot-no-such-ddev-binary"}
	if _, err := c.List(context.Background()); err == nil || !strings.Contains(err.Error(), "ddev not found") {
		t.Fatalf("expected a ddev-not-found error, got %v", err)
	}
	if err := c.Start(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "ddev not found") {
		t.Fatalf("expected a ddev-not-found error from Start, got %v", err)
	}
}

func fakeCLI(t *testing.T) (*CLI, string) {
	t.Helper()
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("DPILOT_ARGS_FILE", argsFile)
	bin, err := filepath.Abs("testdata/fakeddev.sh")
	if err != nil {
		t.Fatal(err)
	}
	return &CLI{Bin: bin, Stdout: io.Discard, Stderr: io.Discard}, argsFile
}

func TestCLIPassesProjectNameAfterTerminator(t *testing.T) {
	c, argsFile := fakeCLI(t)
	ctx := context.Background()
	if err := c.Start(ctx, "-RO"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := c.Stop(ctx, "-RO"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := c.Describe(ctx, "-RO"); err != nil {
		t.Fatalf("describe: %v", err)
	}
	if _, err := c.List(ctx); err != nil {
		t.Fatalf("list: %v", err)
	}
	b, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(b))
	want := "start -- -RO\nstop -- -RO\ndescribe -j -- -RO\nlist -j"
	if got != want {
		t.Fatalf("argv mismatch\n got: %q\nwant: %q", got, want)
	}
}
