package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

// Clock is the time seam (mockable in tests).
type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

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

// Start starts members in order, waiting for each to be ready. Fail-fast.
func (o *Orchestrator) Start(ctx context.Context, g *config.Group) error {
	n := len(g.Members)
	for i, m := range g.Members {
		fmt.Fprintf(o.Out, "Starting %s (%d/%d)...\n", m, i+1, n)
		if err := o.Client.Start(ctx, m); err != nil {
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
		d, err := o.Client.Describe(ctx, name)
		if err != nil {
			return fmt.Errorf("describe %s: %w", name, err)
		}
		if d.Ready() {
			return nil
		}
		if !o.Clock.Now().Before(deadline) {
			return fmt.Errorf("timeout waiting for %s to become ready after %s", name, timeout)
		}
		o.Clock.Sleep(o.Poll)
	}
}

// Stop stops members in reverse order, best-effort, joining any errors.
func (o *Orchestrator) Stop(ctx context.Context, g *config.Group) error {
	var errs []error
	for i := len(g.Members) - 1; i >= 0; i-- {
		m := g.Members[i]
		fmt.Fprintf(o.Out, "Stopping %s...\n", m)
		if err := o.Client.Stop(ctx, m); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", m, err))
		}
	}
	return errors.Join(errs...)
}

// Restart stops then starts.
func (o *Orchestrator) Restart(ctx context.Context, g *config.Group) error {
	if err := o.Stop(ctx, g); err != nil {
		return err
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
