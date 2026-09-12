package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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
	if !strings.Contains(out, "|       1 |") {
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
		t.Fatalf("json should carry error only on the broken row: %s (%v)", buf.String(), err)
	}
}
