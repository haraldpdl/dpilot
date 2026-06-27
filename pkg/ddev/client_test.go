package ddev

import (
	"context"
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
