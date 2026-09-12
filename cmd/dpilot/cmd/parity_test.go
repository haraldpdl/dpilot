package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/haraldpdl/dpilot/pkg/config"
	"github.com/haraldpdl/dpilot/pkg/ddev"
	"github.com/haraldpdl/dpilot/pkg/tui"
)

// envelope mirrors ddev's -j output shape.
type envelope struct {
	Level string          `json:"level"`
	Msg   string          `json:"msg"`
	Raw   json.RawMessage `json:"raw"`
	Time  string          `json:"time"`
}

func decodeEnvelope(t *testing.T, out string) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal([]byte(out), &e); err != nil {
		t.Fatalf("not a ddev-style json envelope: %v (%q)", err, out)
	}
	if e.Level != "info" || e.Time == "" || len(e.Raw) == 0 {
		t.Fatalf("envelope incomplete: %+v", e)
	}
	return e
}

func TestListAliasLs(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	newClient = func() ddev.Client { return listClient{} }
	if _, err := run(t, "ls"); err != nil {
		t.Fatalf("alias ls failed: %v", err)
	}
}

func TestDescribeAliasesStAndDesc(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "g", Members: []string{"db"}})
	newClient = func() ddev.Client { return listClient{} }
	for _, alias := range []string{"st", "desc"} {
		if _, err := run(t, alias, "g"); err != nil {
			t.Fatalf("alias %s failed: %v", alias, err)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		out, err := run(t, flag)
		if err != nil || !strings.HasPrefix(out, "dpilot ") {
			t.Fatalf("%s: err=%v out=%q", flag, err, out)
		}
	}
}

func TestJSONOutputIsRootFlagWithDdevEnvelope(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "mystack", Members: []string{"db"}})
	newClient = func() ddev.Client { return listClient{list: []ddev.Project{{Name: "db", Status: ddev.StatusRunning}}} }
	out, err := run(t, "-j", "list") // ddev form: flag before the verb
	if err != nil {
		t.Fatal(err)
	}
	e := decodeEnvelope(t, out)
	var rows []struct {
		Name    string `json:"name"`
		Running int    `json:"running"`
	}
	if err := json.Unmarshal(e.Raw, &rows); err != nil || len(rows) != 1 || rows[0].Name != "mystack" || rows[0].Running != 1 {
		t.Fatalf("raw should hold the list rows: %s (%v)", e.Raw, err)
	}
	if !strings.Contains(e.Msg, "mystack") || strings.Contains(e.Msg, "\x1b[") {
		t.Fatalf("msg should be the plain rendered table: %q", e.Msg)
	}
}

func TestJSONErrorLine(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	newClient = func() ddev.Client { return listClient{} }
	_, err := run(t, "-j", "describe", "nope")
	if err == nil {
		t.Fatal("expected an error for a missing group")
	}
	line := ErrorLine(err)
	var e map[string]any
	if json.Unmarshal([]byte(line), &e) != nil || e["level"] != "fatal" || !strings.Contains(e["msg"].(string), "nope") {
		t.Fatalf("with -j errors must be a fatal json line, got %q", line)
	}
	_, err = run(t, "describe", "nope")
	if line := ErrorLine(err); !strings.HasPrefix(line, "dpilot: ") {
		t.Fatalf("without -j errors keep the plain prefix, got %q", line)
	}
}

func TestStartAllStartsEveryGroupInNameOrder(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "b", Members: []string{"y"}})
	_ = config.Save(&config.Group{Name: "a", Members: []string{"x"}})
	rec := &recClient{}
	newClient = func() ddev.Client { return rec }
	if _, err := run(t, "start", "--all"); err != nil {
		t.Fatalf("start --all: %v", err)
	}
	if strings.Join(rec.started, ",") != "x,y" {
		t.Fatalf("expected x,y got %v", rec.started)
	}
}

func TestStopAllStopsEveryGroupInReverse(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	_ = config.Save(&config.Group{Name: "a", Members: []string{"x1", "x2"}})
	_ = config.Save(&config.Group{Name: "b", Members: []string{"y"}})
	rec := &recClient{}
	newClient = func() ddev.Client { return rec }
	if _, err := run(t, "stop", "-a"); err != nil {
		t.Fatalf("stop -a: %v", err)
	}
	if strings.Join(rec.stopped, ",") != "y,x2,x1" {
		t.Fatalf("expected y,x2,x1 got %v", rec.stopped)
	}
}

func TestLifecycleRequiresGroupOrAll(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	newClient = func() ddev.Client { return &recClient{} }
	for _, verb := range []string{"start", "stop", "restart"} {
		if _, err := run(t, verb); err == nil || !strings.Contains(err.Error(), "--all") {
			t.Fatalf("%s without args should ask for a group or --all, got %v", verb, err)
		}
	}
}

func TestDeletePromptsOnTerminal(t *testing.T) {
	t.Setenv("DPILOT_HOME", t.TempDir())
	withProjects("db")
	if err := config.Save(&config.Group{Name: "mystack", Members: []string{"db"}}); err != nil {
		t.Fatal(err)
	}
	isInteractive = func() bool { return true }
	defer func() { isInteractive = tui.IsInteractive; rootCmd.SetIn(nil) }()

	rootCmd.SetIn(strings.NewReader("n\n"))
	out, err := run(t, "delete", "mystack")
	if err != nil || !strings.Contains(out, "OK to delete group \"mystack\"? [Y/n]") || !strings.Contains(out, "cancelled") {
		t.Fatalf("declining should cancel cleanly: err=%v out=%q", err, out)
	}
	if ok, _ := config.Exists("mystack"); !ok {
		t.Fatal("group should survive a declined prompt")
	}

	rootCmd.SetIn(strings.NewReader("\n")) // blank accepts the default (yes), as in ddev
	if _, err := run(t, "delete", "mystack"); err != nil {
		t.Fatalf("accepting: %v", err)
	}
	if ok, _ := config.Exists("mystack"); ok {
		t.Fatal("group should be deleted after the default-yes prompt")
	}
}
