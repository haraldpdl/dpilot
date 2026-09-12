package orchestrator

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

// GroupSummary is a group's name with member and running-member counts. A
// group whose file cannot be loaded is still listed, with Err set and zero
// counts, so one broken file never hides the others.
type GroupSummary struct {
	Name    string
	Members int
	Running int
	Err     string
}

// GroupSummaries returns a summary for every configured group, computing the
// running-member count from a single ddev project listing. Only a failure to
// list the groups directory or to reach ddev is returned as an error.
func GroupSummaries(ctx context.Context, c ddev.Client) ([]GroupSummary, error) {
	names, err := config.List()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}
	projects, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	running := map[string]bool{}
	for _, p := range projects {
		if p.Status == ddev.StatusRunning {
			running[p.Name] = true
		}
	}
	summaries := make([]GroupSummary, 0, len(names))
	for _, n := range names {
		g, err := config.Load(n)
		if err != nil {
			summaries = append(summaries, GroupSummary{Name: n, Err: err.Error()})
			continue
		}
		s := GroupSummary{Name: n, Members: len(g.Members)}
		for _, m := range g.Members {
			if running[m] {
				s.Running++
			}
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
