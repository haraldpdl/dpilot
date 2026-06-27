package orchestrator

import (
	"context"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
)

// GroupSummary is a group's name with member and running-member counts.
type GroupSummary struct {
	Name    string
	Members int
	Running int
}

// GroupSummaries returns a summary for every configured group, computing the
// running-member count from a single ddev project listing.
func GroupSummaries(ctx context.Context, c ddev.Client) ([]GroupSummary, error) {
	names, err := config.List()
	if err != nil {
		return nil, err
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
			return nil, err
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
