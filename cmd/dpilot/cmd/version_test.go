package cmd

import "testing"

func TestResolveVersionPrefersLdflags(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	Version = "1.2.3"
	if got := resolveVersion(); got != "1.2.3" {
		t.Fatalf("resolveVersion() = %q, want 1.2.3", got)
	}
}

func TestResolveVersionNeverEmpty(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	Version = "dev"
	if got := resolveVersion(); got == "" {
		t.Fatal("resolveVersion() returned empty")
	}
}
