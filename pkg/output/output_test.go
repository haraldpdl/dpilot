package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

func TestGroupsJSON(t *testing.T) {
	var buf bytes.Buffer
	rows := []GroupRow{{Name: "mystack", Members: 3, Running: 2}}
	if err := Groups(&buf, rows, true); err != nil {
		t.Fatal(err)
	}
	var e struct {
		Level, Msg, Time string
		Raw              []GroupRow
	}
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil {
		t.Fatalf("not valid json: %v", err)
	}
	if e.Level != "info" || e.Time == "" || !strings.Contains(e.Msg, "mystack") {
		t.Fatalf("expected a ddev-style envelope, got %s", buf.String())
	}
	got := e.Raw
	if got[0].Name != "mystack" || got[0].Running != 2 {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGroupsTableMentionsName(t *testing.T) {
	var buf bytes.Buffer
	if err := Groups(&buf, []GroupRow{{Name: "mystack", Members: 3, Running: 2}}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "mystack") {
		t.Fatalf("table missing name: %q", buf.String())
	}
}

func TestDescribeTableShowsMembersInOrder(t *testing.T) {
	var buf bytes.Buffer
	rows := []MemberRow{{Name: "db", Status: "running"}, {Name: "api", Status: "missing"}}
	if err := Describe(&buf, "mystack", rows, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Index(out, "db") > strings.Index(out, "api") {
		t.Fatalf("expected db before api: %q", out)
	}
}

func TestDescribeJSON(t *testing.T) {
	var buf bytes.Buffer
	rows := []MemberRow{{Name: "db", Status: "running"}, {Name: "api", Status: "missing"}}
	if err := Describe(&buf, "mystack", rows, true); err != nil {
		t.Fatal(err)
	}
	var e struct {
		Raw struct {
			Name    string      `json:"name"`
			Members []MemberRow `json:"members"`
		} `json:"raw"`
	}
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil {
		t.Fatalf("describe json invalid: %v", err)
	}
	got := e.Raw
	if got.Name != "mystack" || len(got.Members) != 2 || got.Members[1].Name != "api" || got.Members[1].Status != "missing" {
		t.Fatalf("unexpected describe json: %+v", got)
	}
}

func TestDescribeNoAnsiForNonTerminal(t *testing.T) {
	var buf bytes.Buffer
	rows := []MemberRow{{Name: "db", Status: "running"}}
	if err := Describe(&buf, "g", rows, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("a bytes.Buffer is not a terminal; expected no ANSI: %q", buf.String())
	}
}

func TestGroupsEmptyPrintsHintNotTable(t *testing.T) {
	var buf bytes.Buffer
	if err := Groups(&buf, nil, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "No dpilot groups found") || strings.Contains(out, "GROUP") {
		t.Fatalf("empty list should print a hint, not a header-only table: %q", out)
	}
}

func TestGroupsEmptyJSONIsArray(t *testing.T) {
	var buf bytes.Buffer
	if err := Groups(&buf, nil, true); err != nil {
		t.Fatal(err)
	}
	var e struct{ Raw json.RawMessage }
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil || strings.TrimSpace(string(e.Raw)) != "[]" {
		t.Fatalf("empty -j raw must be [], got %q", buf.String())
	}
}

func TestGroupsInvalidRowShowsError(t *testing.T) {
	var buf bytes.Buffer
	rows := []GroupRow{{Name: "broken", Error: `parse group "broken": yaml: field bogus not found`}, {Name: "ok", Members: 1}}
	if err := Groups(&buf, rows, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "broken") || !strings.Contains(out, "invalid") || !strings.Contains(out, "field bogus not found") {
		t.Fatalf("invalid group should be listed with its error: %q", out)
	}
	if !strings.Contains(out, "│       1 │") {
		t.Fatalf("numeric columns must stay right-aligned next to an invalid row: %q", out)
	}
	if strings.Count(out, "broken") != 2 { // once in the table, once in the error line
		t.Fatalf("the group name should not be repeated in the error line: %q", out)
	}
	buf.Reset()
	if err := Groups(&buf, rows, true); err != nil {
		t.Fatal(err)
	}
	var e struct{ Raw []GroupRow }
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil {
		t.Fatalf("json: %v", err)
	}
	got := e.Raw
	if len(got) != 2 || got[0].Error == "" || got[1].Error != "" {
		t.Fatalf("json should carry error only on the broken row: %s", buf.String())
	}
}

func TestTablesUseDdevLightBoxStyle(t *testing.T) {
	var buf bytes.Buffer
	if err := Groups(&buf, []GroupRow{{Name: "a", Members: 1}, {Name: "b", Members: 2}}, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "+-") || !strings.Contains(out, "│") || strings.Count(out, "├") != 2 {
		t.Fatalf("expected ddev's light box style with a separator after the header and between the two rows, got:\n%s", out)
	}
	buf.Reset()
	if err := Describe(&buf, "g", []MemberRow{{Name: "db", Status: "running"}}, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "+-") || !strings.Contains(buf.String(), "│") {
		t.Fatalf("describe should use the same style, got:\n%s", buf.String())
	}
}

func TestStatusCellsFollowDdevColours(t *testing.T) {
	text.EnableColors() // go-pretty reads NO_COLOR/TERM at init; make the test hermetic
	cases := map[string]string{
		"running": "\x1b[32mOK\x1b[0m",
		"stopped": "\x1b[31mstopped\x1b[0m",
		"missing": "\x1b[31mmissing\x1b[0m",
		"paused":  "\x1b[33mpaused\x1b[0m",
	}
	for status, want := range cases {
		if got := colorStatus(status, true); got != want {
			t.Errorf("%s: got %q want %q", status, got, want)
		}
	}
	if got := colorStatus("running", false); got != "OK" {
		t.Errorf("uncoloured running should read OK, got %q", got)
	}
}
