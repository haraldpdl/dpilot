package config

import (
	"os"
	"strings"
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
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustPath(t, "g"), []byte("name: g\nbogus: 1\nmembers: [a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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

func TestPathRejectsUnsafeNames(t *testing.T) {
	tempHome(t)
	for _, name := range []string{"../escaped", "sub/dir", "..", "."} {
		if err := Save(&Group{Name: name, Members: []string{"x"}}); err == nil {
			t.Errorf("Save(%q): expected rejection, got nil", name)
		}
		if _, err := Load(name); err == nil {
			t.Errorf("Load(%q): expected rejection, got nil", name)
		}
	}
}

func TestValidateRejectsFlagLikeMembers(t *testing.T) {
	for _, m := range []string{"-RO", "--all", "-", "a b", "bad_name", "trailing-"} {
		g := &Group{Name: "g", Members: []string{m}}
		if err := g.Validate(); err == nil {
			t.Errorf("Validate member %q: expected rejection, got nil", m)
		}
	}
	for _, m := range []string{"db", "my-api", "site.local", "a1", "x"} {
		g := &Group{Name: "g", Members: []string{m}}
		if err := g.Validate(); err != nil {
			t.Errorf("Validate member %q: unexpected error %v", m, err)
		}
	}
}

func TestLoadUsesFilenameAsName(t *testing.T) {
	tempHome(t)
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustPath(t, "g"), []byte("name: other\nmembers: [a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load("g")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Name != "g" {
		t.Fatalf("expected filename to win, got name %q", got.Name)
	}
	got.Members = append(got.Members, "b")
	if err := Save(got); err != nil {
		t.Fatalf("save: %v", err)
	}
	if ok, _ := Exists("other"); ok {
		t.Fatal("Save wrote to the YAML name instead of the loaded file")
	}
	data, err := os.ReadFile(mustPath(t, "g"))
	if err != nil || !strings.Contains(string(data), "- b") || !strings.Contains(string(data), "name: g") {
		t.Fatalf("Save should rewrite the loaded file under its own name, got %q (%v)", data, err)
	}
}

func TestLegacyGroupNamesStayUsable(t *testing.T) {
	tempHome(t)
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"my group", "_shared", "-legacy"} {
		if err := os.WriteFile(mustPath(t, name), []byte("members: [a]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		g, err := Load(name)
		if err != nil {
			t.Fatalf("Load(%q): a group file made by an earlier release must still load: %v", name, err)
		}
		if err := Save(g); err != nil {
			t.Fatalf("Save(%q): %v", name, err)
		}
		if err := Delete(name); err != nil {
			t.Fatalf("Delete(%q): %v", name, err)
		}
	}
}

func TestLoadRejectsNonPositiveTimeout(t *testing.T) {
	tempHome(t)
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"0s", "-5s"} {
		if err := os.WriteFile(mustPath(t, "g"), []byte("members: [a]\nwait_timeout: "+v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load("g"); err == nil {
			t.Errorf("wait_timeout %q: expected rejection, got nil", v)
		}
	}
}

func TestValidateNameRules(t *testing.T) {
	for _, n := range []string{"", " ", " x", "-foo", ".hidden", "a b", "a/b", "..", "."} {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q): expected rejection, got nil", n)
		}
	}
	for _, n := range []string{"mystack", "my-stack", "my_stack", "v1.2", "A9"} {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q): unexpected error %v", n, err)
		}
	}
}

func TestLoadErrorsNameTheGroup(t *testing.T) {
	tempHome(t)
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustPath(t, "g"), []byte("members: [a, a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load("g")
	if err == nil || !strings.Contains(err.Error(), `group "g"`) {
		t.Fatalf("validation errors must name the group, got %v", err)
	}
}

func TestListSkipsNamelessFilesAndLoadNamesEmptyFiles(t *testing.T) {
	tempHome(t)
	if err := os.MkdirAll(mustDir(t), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustPath(t, ""), []byte("members: [a]\n"), 0o644); err != nil { // ".yaml"
		t.Fatal(err)
	}
	if err := os.WriteFile(mustPath(t, "empty"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	names, err := List()
	if err != nil || len(names) != 1 || names[0] != "empty" {
		t.Fatalf("a nameless .yaml must not be listed, got %v (%v)", names, err)
	}
	if _, err := Load("empty"); err == nil || !strings.Contains(err.Error(), "file is empty") {
		t.Fatalf("an empty file should say so, got %v", err)
	}
}
