package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

// Clock is the time seam (mockable in tests). Sleep returns early with the
// context's error when the context is cancelled.
type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Orchestrator sequences ddev project lifecycle for a group.
type Orchestrator struct {
	Client ddev.Client
	Clock  Clock
	Poll   time.Duration
	Out    io.Writer
}

// New builds an Orchestrator with the real clock and a 2s poll interval.
func New(c ddev.Client) *Orchestrator {
	return &Orchestrator{Client: c, Clock: realClock{}, Poll: 2 * time.Second, Out: os.Stdout}
}

// Start starts members in order, waiting for each to be ready. Fail-fast, and
// it stops as soon as ctx is cancelled (Ctrl-C) rather than driving ddev on.
func (o *Orchestrator) Start(ctx context.Context, g *config.Group) error {
	n := len(g.Members)
	for i, m := range g.Members {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("aborted before starting %s: %w", m, err)
		}
		fmt.Fprintf(o.Out, "Starting %s (%d/%d)...\n", m, i+1, n)
		if err := o.Client.Start(ctx, m); err != nil {
			if cerr := ctx.Err(); cerr != nil {
				return fmt.Errorf("aborted while starting %s (%v): %w", m, err, cerr)
			}
			return fmt.Errorf("start %s: %w", m, err)
		}
		if err := o.waitReady(ctx, m, g.WaitTimeout.Duration()); err != nil {
			return err
		}
		fmt.Fprintf(o.Out, "%s: ready\n", m)
	}
	return nil
}

func (o *Orchestrator) waitReady(ctx context.Context, name string, timeout time.Duration) error {
	deadline := o.Clock.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("aborted while waiting for %s: %w", name, err)
		}
		d, err := o.Client.Describe(ctx, name)
		if err != nil {
			if cerr := ctx.Err(); cerr != nil {
				return fmt.Errorf("aborted while waiting for %s (%v): %w", name, err, cerr)
			}
			return fmt.Errorf("describe %s: %w", name, err)
		}
		if d.Ready() {
			return nil
		}
		if !o.Clock.Now().Before(deadline) {
			return fmt.Errorf("timeout waiting for %s to become ready after %s", name, timeout)
		}
		if err := o.Clock.Sleep(ctx, o.Poll); err != nil {
			return fmt.Errorf("aborted while waiting for %s: %w", name, err)
		}
	}
}

// Stop stops members in reverse order, best-effort, joining any errors. Once
// ctx is cancelled it stops driving ddev and returns a single error naming
// the members not known to be stopped (the interrupted one included),
// instead of one failure per remaining member.
func (o *Orchestrator) Stop(ctx context.Context, g *config.Group) error {
	var errs []error
	for i := len(g.Members) - 1; i >= 0; i-- {
		m := g.Members[i]
		if err := ctx.Err(); err != nil {
			return abortedStop(g.Members[:i+1], err, errs)
		}
		fmt.Fprintf(o.Out, "Stopping %s...\n", m)
		if err := o.Client.Stop(ctx, m); err != nil {
			if cerr := ctx.Err(); cerr != nil {
				return abortedStop(g.Members[:i+1], cerr, errs)
			}
			errs = append(errs, fmt.Errorf("stop %s: %w", m, err))
		}
	}
	return errors.Join(errs...)
}

// abortedStop builds the single error returned when a stop is interrupted;
// left lists the members in group order whose state is now unknown.
func abortedStop(left []string, cause error, errs []error) error {
	errs = append(errs, fmt.Errorf("aborted: %d member(s) not stopped (%s): %w", len(left), strings.Join(left, ", "), cause))
	return errors.Join(errs...)
}

// Restart stops (best-effort) then starts (fail-fast). A stop error never
// blocks the start; it is reported and start proceeds. A cancelled stop does
// block it: nothing is started after Ctrl-C.
func (o *Orchestrator) Restart(ctx context.Context, g *config.Group) error {
	if err := o.Stop(ctx, g); err != nil {
		if ctx.Err() != nil {
			return err
		}
		fmt.Fprintf(o.Out, "restart: stop reported errors, continuing to start: %v\n", err)
	}
	return o.Start(ctx, g)
}

// MemberState pairs a member with its live ddev status.
type MemberState struct {
	Name   string
	Status ddev.ProjectStatus
}

// Statuses returns each member's current state in group order.
func (o *Orchestrator) Statuses(ctx context.Context, g *config.Group) ([]MemberState, error) {
	projects, err := o.Client.List(ctx)
	if err != nil {
		return nil, err
	}
	byName := map[string]ddev.ProjectStatus{}
	for _, p := range projects {
		byName[p.Name] = p.Status
	}
	states := make([]MemberState, 0, len(g.Members))
	for _, m := range g.Members {
		st, ok := byName[m]
		if !ok {
			st = ddev.StatusMissing
		}
		states = append(states, MemberState{Name: m, Status: st})
	}
	return states, nil
}
