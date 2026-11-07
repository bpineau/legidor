package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculate(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"(780000-500000)*0.03", "8400"}, {"15000-4000-18000-6000", "-13000"}, {"8000*0.172", "1376"}} {
		got, err := Calculate(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("%s: %s %v", tc.input, got, err)
		}
	}
	for _, s := range []string{"1/0", "exec(1)", "1<<1000000000", "x", "1e999999"} {
		if _, err := Calculate(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}
func TestResourceBoundary(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("secret"), 0600)
	os.WriteFile(filepath.Join(root, "data.json"), []byte("{}"), 0600)
	os.Symlink(outside, filepath.Join(root, "escape"))
	b := Broker{Root: root}
	if s, err := b.Read("data.json"); err != nil || s != "{}" {
		t.Fatal(s, err)
	}
	for _, p := range []string{"../secret", outside, "escape"} {
		if _, err := b.Read(p); err == nil {
			t.Errorf("read outside root: %s", p)
		}
	}
}
func TestLegidorCommandBoundary(t *testing.T) {
	for _, args := range [][]string{{"auth", "token"}, {"cache", "clear"}, {"refs", "/tmp/private"}, {"check", "/tmp/private"}, {"get", "CGI 1", "--config", "/tmp/private"}} {
		if err := ValidateLegidor(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	if err := ValidateLegidor([]string{"get", "CGI 197", "--date", "2026-01-01", "--part", "I"}); err != nil {
		t.Fatal(err)
	}
}
func TestParseClaudeUsage(t *testing.T) {
	s := `{"type":"system","subtype":"init","model":"claude-opus-5-5"}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"WebSearch"}]}}
{"type":"result","subtype":"success","is_error":false,"result":"answer","total_cost_usd":0.2,"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":30}}`
	r, err := ParseTrace("claude", strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	if r.Answer != "answer" || r.InputTokens != 140 || r.CacheReadTokens != 100 || r.CacheWriteTokens != 30 || r.OutputTokens != 20 || r.ToolCalls != 1 || r.CostUSD == nil || *r.CostUSD != 0.2 {
		t.Fatalf("%+v", r)
	}
}
func TestParseCodexUsageAndFailures(t *testing.T) {
	s := `{"type":"item.completed","item":{"type":"agent_message","text":"answer"}}
{"type":"item.completed","item":{"type":"web_search"}}
{"type":"turn.completed","usage":{"input_tokens":200,"cached_input_tokens":150,"output_tokens":30}}`
	r, err := ParseTrace("codex", strings.NewReader(s))
	if err != nil || r.InputTokens != 200 || r.CacheReadTokens != 150 || r.CostUSD != nil || r.ToolCalls != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	for _, s := range []string{`{"type":"turn.failed","error":{"message":"quota"}}`, `{"type":"item.completed","item":{"type":"agent_message","text":"partial"}}`, `garbage`} {
		if _, err := ParseTrace("codex", strings.NewReader(s)); err == nil {
			t.Fatalf("accepted failed trace %s", s)
		}
	}
	if _, err := ParseTrace("claude", strings.NewReader(`{"type":"result","is_error":true,"result":"error"}`)); err == nil {
		t.Fatal("accepted Claude failure")
	}
}
func TestResourceHashChanges(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "data.json")
	os.WriteFile(p, []byte("one"), 0600)
	a, err := HashTree(root)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, []byte("two"), 0600)
	b, _ := HashTree(root)
	if a == b {
		t.Fatal("resource mutation not hashed")
	}
}

func TestPartialClaudeUsageDeduplicatesStreamMessages(t *testing.T) {
	trace := `{"type":"assistant","message":{"id":"m1","usage":{"input_tokens":5,"output_tokens":1,"cache_read_input_tokens":50,"cache_creation_input_tokens":10},"content":[]}}
{"type":"assistant","message":{"id":"m1","usage":{"input_tokens":5,"output_tokens":9,"cache_read_input_tokens":50,"cache_creation_input_tokens":10},"content":[]}}
{"type":"assistant","message":{"id":"m2","usage":{"input_tokens":3,"output_tokens":4,"cache_read_input_tokens":65},"content":[]}}`
	m, err := ParseTrace("claude", strings.NewReader(trace))
	if err == nil || m.UsageComplete || m.InputTokens != 133 || m.OutputTokens != 13 {
		t.Fatalf("%+v %v", m, err)
	}
}
