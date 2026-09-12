package ddev

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Client is the seam over the ddev CLI. Project names are always passed after
// a "--" terminator so ddev can never parse a name as a flag.
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

func (c *CLI) ensure() error {
	if _, err := exec.LookPath(c.Bin); err != nil {
		return fmt.Errorf("ddev not found on PATH: install ddev (https://ddev.com) to use dpilot")
	}
	return nil
}

// cancelGrace is how long an interrupted ddev gets to clean up (stop
// containers, release locks) before it is killed.
const cancelGrace = 10 * time.Second

// command builds the ddev invocation. When ctx is cancelled (Ctrl-C, a
// timeout) ddev receives SIGINT, as it would from the terminal, so it can
// shut down cleanly; only if it is still running after cancelGrace is it
// killed, which also unblocks us if a docker grandchild holds our pipes.
func (c *CLI) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = cancelGrace
	return cmd
}

func (c *CLI) capture(ctx context.Context, args ...string) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	var out, errBuf bytes.Buffer
	cmd := c.command(ctx, args...)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("ddev %v: %w", args, ctxErr)
		}
		return nil, fmt.Errorf("ddev %v: %w: %s", args, err, errBuf.String())
	}
	return out.Bytes(), nil
}

func (c *CLI) stream(ctx context.Context, args ...string) error {
	if err := c.ensure(); err != nil {
		return err
	}
	cmd := c.command(ctx, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("ddev %v: %w", args, ctxErr)
		}
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
	out, err := c.capture(ctx, "describe", "-j", "--", name)
	if err != nil {
		return nil, err
	}
	return ParseDescribe(out)
}

func (c *CLI) Start(ctx context.Context, name string) error {
	return c.stream(ctx, "start", "--", name)
}

func (c *CLI) Stop(ctx context.Context, name string) error {
	return c.stream(ctx, "stop", "--", name)
}
