package config

import (
	"os"
	"testing"
	"time"
)

func tempHome(t *testing.T) {
	t.Helper()
	t.Setenv("DPILOT_HOME", t.TempDir())
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tempHome(t)
	g := &Group{Name: "mystack", WaitTimeout: Duration(90 * time.Second), Members: []string{"db", "api"}}
	if err := Save(g); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load("mystack")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Name != "mystack" || got.WaitTimeout.Duration() != 90*time.Second {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if len(got.Members) != 2 || got.Members[0] != "db" || got.Members[1] != "api" {
		t.Fatalf("members mismatch: %+v", got.Members)
	}
}

func TestLoadAppliesDefaultTimeout(t *testing.T) {
	tempHome(t)
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustPath(t, "g"), []byte("name: g\nmembers:\n  - a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load("g")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.WaitTimeout.Duration() != DefaultWaitTimeout {
		t.Fatalf("expected default timeout, got %v", got.WaitTimeout.Duration())
	}
}

func TestValidateRejectsDuplicateMembers(t *testing.T) {
	g := &Group{Name: "g", Members: []string{"a", "a"}}
	if err := g.Validate(); err == nil {
		t.Fatal("expected duplicate member error")
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	tempHome(t)
	_ = os.MkdirAll(mustDir(t), 0o755)
	_ = os.WriteFile(mustPath(t, "g"), []byte("name: g\nbogus: 1\nmembers: [a]\n"), 0o644)
	if _, err := Load("g"); err == nil {
		t.Fatal("expected unknown-key error")
	}
}

func TestListSorted(t *testing.T) {
	tempHome(t)
	for _, n := range []string{"b", "a", "c"} {
		if err := Save(&Group{Name: n, Members: []string{"x"}}); err != nil {
			t.Fatal(err)
		}
	}
	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 || names[0] != "a" || names[2] != "c" {
		t.Fatalf("expected sorted names, got %v", names)
	}
}

// helpers
func mustDir(t *testing.T) string {
	d, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func mustPath(t *testing.T, n string) string { d, _ := Dir(); return d + "/" + n + ".yaml" }
