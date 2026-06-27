package ddev

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Client is the seam over the ddev CLI.
type Client interface {
	List(ctx context.Context) ([]Project, error)
	Describe(ctx context.Context, name string) (*Describe, error)
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
}

// CLI is the real exec-based Client.
type CLI struct {
	Bin    string
	Stdout io.Writer
	Stderr io.Writer
}

// New returns a CLI that streams lifecycle output to the process stdio.
func New() *CLI {
	return &CLI{Bin: "ddev", Stdout: os.Stdout, Stderr: os.Stderr}
}

func (c *CLI) capture(ctx context.Context, args ...string) ([]byte, error) {
	var out, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ddev %v: %w: %s", args, err, errBuf.String())
	}
	return out.Bytes(), nil
}

func (c *CLI) stream(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ddev %v: %w", args, err)
	}
	return nil
}

func (c *CLI) List(ctx context.Context) ([]Project, error) {
	out, err := c.capture(ctx, "list", "-j")
	if err != nil {
		return nil, err
	}
	return ParseList(out)
}

func (c *CLI) Describe(ctx context.Context, name string) (*Describe, error) {
	out, err := c.capture(ctx, "describe", name, "-j")
	if err != nil {
		return nil, err
	}
	return ParseDescribe(out)
}

func (c *CLI) Start(ctx context.Context, name string) error {
	return c.stream(ctx, "start", name)
}

func (c *CLI) Stop(ctx context.Context, name string) error {
	return c.stream(ctx, "stop", name)
}
