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
	var got []GroupRow
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not valid json: %v", err)
	}
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
