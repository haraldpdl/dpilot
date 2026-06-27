package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/haraldpdl/dpilot/pkg/ddev"
)

// fakeClock advances deterministically: every Sleep moves Now forward.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time        { return c.t }
func (c *fakeClock) Sleep(d time.Duration) { c.t = c.t.Add(d) }

// fakeClient scripts ddev behavior for tests.
type fakeClient struct {
	startErr map[string]error
	stopErr  map[string]error
	started  []string
	stopped  []string
	// describeSeq returns successive Describe results per name; the last is repeated.
	describeSeq map[string][]*ddev.Describe
	describeIdx map[string]int
	list        []ddev.Project
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		startErr:    map[string]error{},
		stopErr:     map[string]error{},
		describeSeq: map[string][]*ddev.Describe{},
		describeIdx: map[string]int{},
	}
}

func running(name string) *ddev.Describe {
	return &ddev.Describe{Name: name, Status: ddev.StatusRunning}
}

func stopped(name string) *ddev.Describe {
	return &ddev.Describe{Name: name, Status: ddev.StatusStopped}
}

func (f *fakeClient) List(context.Context) ([]ddev.Project, error) { return f.list, nil }

func (f *fakeClient) Describe(_ context.Context, name string) (*ddev.Describe, error) {
	seq := f.describeSeq[name]
	if len(seq) == 0 {
		return stopped(name), nil
	}
	i := f.describeIdx[name]
	if i >= len(seq) {
		i = len(seq) - 1
	}
	f.describeIdx[name] = f.describeIdx[name] + 1
	return seq[i], nil
}

func (f *fakeClient) Start(_ context.Context, name string) error {
	if err := f.startErr[name]; err != nil {
		return err
	}
	f.started = append(f.started, name)
	return nil
}

func (f *fakeClient) Stop(_ context.Context, name string) error {
	if err := f.stopErr[name]; err != nil {
		return err
	}
	f.stopped = append(f.stopped, name)
	return nil
}

var _ ddev.Client = (*fakeClient)(nil)

func errFor(name string) error { return fmt.Errorf("boom: %s", name) }
