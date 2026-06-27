package ddev

import (
	"os"
	"testing"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

func TestParseListFindsFixtureProject(t *testing.T) {
	projects, err := ParseList(read(t, "list.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var found bool
	for _, p := range projects {
		if p.Name == "dpilot-fixture" {
			found = true
			if p.Status != StatusRunning {
				t.Fatalf("expected running, got %q", p.Status)
			}
		}
	}
	if !found {
		t.Fatalf("dpilot-fixture not in list: %+v", projects)
	}
}

func TestParseDescribeRunningIsReady(t *testing.T) {
	d, err := ParseDescribe(read(t, "describe_running.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Status != StatusRunning {
		t.Fatalf("expected running, got %q", d.Status)
	}
	if !d.Ready() {
		t.Fatalf("running project with healthy services should be ready: %+v", d)
	}
}

func TestParseDescribeStoppedNotReady(t *testing.T) {
	d, err := ParseDescribe(read(t, "describe_stopped.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Ready() {
		t.Fatalf("stopped project must not be ready: %+v", d)
	}
}
