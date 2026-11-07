package app

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/model"
)

func TestCLIParseAndUsage(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{{[]string{"resolve", "CSS L136-8 IV", "--parse-only", "--json"}, 0}, {[]string{"get", "CSS L136-8 IV", "--part", "III", "--json"}, 2}, {[]string{"search", "x", "--limit", "0", "--json"}, 2}, {[]string{"get", "--help"}, 0}, {[]string{"get", "CGI 32", "--fresh", "--offline", "--json"}, 2}} {
		var out, err bytes.Buffer
		code := Run(context.Background(), tc.args, strings.NewReader(""), &out, &err)
		if code != tc.code {
			t.Fatalf("%v code %d out %s err %s", tc.args, code, &out, &err)
		}
		if tc.args[len(tc.args)-1] == "--json" && !json.Valid(out.Bytes()) {
			t.Fatalf("invalid JSON %s", &out)
		}
	}
}
func TestMalformedManifestKeepsOrder(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, err bytes.Buffer
	code := Run(context.Background(), []string{"check", "-", "--json", "--offline"}, strings.NewReader("bad json\n{\"ref\":\"CSS L136-8\"}\n"), &out, &err)
	if code != 3 {
		t.Fatal(code, out.String(), err.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatal(out.String())
	}
	for _, l := range lines {
		if !json.Valid([]byte(l)) {
			t.Fatal(l)
		}
	}
}
func TestRejectIgnoredFlags(t *testing.T) {
	for _, args := range [][]string{{"check", "-", "--date", "2020-01-01", "--json"}, {"get", "CGI 32", "--in", "jorf", "--json"}, {"resolve", "CGI 32", "--full", "--json"}, {"get", "CGI 32, 2", "--parts", "--json"}} {
		var out, err bytes.Buffer
		if code := Run(context.Background(), args, strings.NewReader(""), &out, &err); code != 2 {
			t.Fatalf("%v code %d %s", args, code, &out)
		}
	}
}
func TestDiffKeepsPreamble(t *testing.T) {
	a := documentForDiff("Threshold 10.\nI.- Same.")
	b := documentForDiff("Threshold 11.\nI.- Same.")
	changes := diffChanges(a, b)
	if len(changes) != 1 || !strings.Contains(changes[0]["diff"], "[-10.-]{+11.+}") {
		t.Fatalf("%+v", changes)
	}
}
func TestTOCPagination(t *testing.T) {
	d := model.Document{ID: "JORFTEXT000000000001", ContentKind: "table_of_contents", Complete: true}
	for i := 0; i < 400; i++ {
		d.Items = append(d.Items, model.Link{ID: "JORFARTI000000000001", Title: strings.Repeat("A", 100)})
	}
	r := referenceForPagination()
	r.Kind = "text"
	first, e := paginate(d, r, "text", options{env: "production"})
	if e != nil || first.Complete || first.Cursor == "" || len(first.Items) >= len(d.Items) {
		t.Fatal("TOC not bounded")
	}
	count := len(first.Items)
	cursor := first.Cursor
	for cursor != "" {
		next, e := paginate(d, r, "text", options{env: "production", cursor: cursor})
		if e != nil {
			t.Fatal(e)
		}
		count += len(next.Items)
		cursor = next.Cursor
	}
	if count != 400 {
		t.Fatalf("lost or repeated items: %d", count)
	}
}
