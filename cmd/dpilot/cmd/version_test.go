package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("dpilot")) {
		t.Fatalf("expected version output to mention dpilot, got %q", buf.String())
	}
}
