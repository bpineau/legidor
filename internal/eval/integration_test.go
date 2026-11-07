package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain provides an offline stand-in for the native CLI protocol. It never
// connects to a model or reads credentials. The production runner still launches
// real child processes, captures JSONL and applies its normal resume checks.
func TestMain(m *testing.M) {
	if os.Getenv("LEGIDOR_EVAL_FAKE_CLI") == "1" {
		for _, arg := range os.Args[1:] {
			if arg == "--version" {
				fmt.Println("fake-cli 1")
				os.Exit(0)
			}
		}
		if os.Getenv("LEGIDOR_EVAL_FAKE_WAIT") == "1" {
			time.Sleep(time.Minute)
			os.Exit(0)
		}
		if strings.Contains(strings.Join(os.Args, " "), "--ignore-user-config") {
			entries, err := os.ReadDir(os.Getenv("CODEX_HOME"))
			if err != nil || len(entries) != 1 || entries[0].Name() != "auth.json" {
				os.Exit(3)
			}
		}
		if p := os.Getenv("LEGIDOR_EVAL_FAKE_CALLS"); p != "" {
			f, _ := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			fmt.Fprintln(f, "call")
			f.Close()
		}
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"8400"}}`)
		fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":32,"cached_input_tokens":16,"output_tokens":4}}`)
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func fakeCLI(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(exe, filepath.Join(root, "codex")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LEGIDOR_EVAL_FAKE_CLI", "1")
	auth := t.TempDir()
	os.WriteFile(filepath.Join(auth, "auth.json"), []byte(`{}`), 0600)
	t.Setenv("CODEX_HOME", auth)
	calls := filepath.Join(root, "calls")
	t.Setenv("LEGIDOR_EVAL_FAKE_CALLS", calls)
	return calls
}
func TestOfflineRunnerResumeAndPromptBoundary(t *testing.T) {
	calls := fakeCLI(t)
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus.json")
	writeJSON(corpus, Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "synthetic", Prompt: "Calculate (780000-500000)*0.03", ExpectedOutput: "PRIVATE_EXPECTED_OUTPUT_CANARY", Assertions: []string{"PRIVATE_RUBRIC_CANARY"}}}})
	o := Options{Corpus: corpus, Output: filepath.Join(root, "results"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test-sol", Effort: "medium", CacheMode: "cold", Repetitions: 1, Timeout: 5 * time.Second}
	if err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(calls)
	if string(b) != "call\n" {
		t.Fatalf("resume ran model again: %q", b)
	}
	prompt, _ := os.ReadFile(filepath.Join(o.Output, "case-001-codex-test-sol-none-r1", "attempt-001", "prompt.txt"))
	if strings.Contains(string(prompt), "PRIVATE_RUBRIC_CANARY") || strings.Contains(string(prompt), "PRIVATE_EXPECTED_OUTPUT_CANARY") {
		t.Fatal("rubric leaked into evaluated conversation")
	}
	summary, _ := os.ReadFile(filepath.Join(o.Output, "summary.json"))
	var rows []Summary
	json.Unmarshal(summary, &rows)
	if len(rows) != 1 || rows[0].InputTokens != 32 || rows[0].ExpectedCriteria != 1 || rows[0].IncompleteUsage != 0 {
		t.Fatalf("%s", summary)
	}
	o.CodexModel = "different"
	if err := Run(context.Background(), o); err == nil {
		t.Fatal("mixed models in one run")
	}
}
func TestInvocationTimeoutRetainsTrace(t *testing.T) {
	fakeCLI(t)
	t.Setenv("LEGIDOR_EVAL_FAKE_WAIT", "1")
	root := t.TempDir()
	start := time.Now()
	m, err := Invoke(context.Background(), Invocation{Agent: "codex", Model: "test-sol", Effort: "medium", Work: filepath.Join(root, "work"), Output: filepath.Join(root, "out"), Prompt: "test", Timeout: 100 * time.Millisecond})
	if err == nil || m.UsageComplete || time.Since(start) > 5*time.Second {
		t.Fatalf("timeout not enforced: %+v %v", m, err)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "trace.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestOfflineParallelRunner(t *testing.T) {
	fakeCLI(t)
	root := t.TempDir()
	corpus := filepath.Join(root, "cases.json")
	writeJSON(corpus, Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "one", Prompt: "p", Assertions: []string{"a"}}, {ID: 2, Name: "two", Prompt: "q", Assertions: []string{"b"}}}})
	o := Options{Corpus: corpus, Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test-sol", Effort: "medium", CacheMode: "cold", Repetitions: 2, Workers: 2, Timeout: 5 * time.Second}
	if err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	samples, err := Samples(o.Output)
	if err != nil || len(samples) != 4 {
		t.Fatalf("%d %v", len(samples), err)
	}
	for _, s := range samples {
		if s.Status != "ok" || s.Model != "test-sol" {
			t.Fatalf("%+v", s)
		}
	}
}

func TestModelMatrixCreatesDistinctArtifacts(t *testing.T) {
	fakeCLI(t)
	root := t.TempDir()
	corpus := filepath.Join(root, "cases.json")
	writeJSON(corpus, Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "one", Prompt: "p", Assertions: []string{"a"}}}})
	o := Options{Corpus: corpus, Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModels: []string{"test-sol-6.1", "test-astra-6"}, Effort: "medium", CacheMode: "cold", Repetitions: 1, Workers: 2, Timeout: 5 * time.Second}
	if err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	samples, err := Samples(o.Output)
	if err != nil || len(samples) != 2 {
		t.Fatalf("%d %v", len(samples), err)
	}
	for _, model := range o.CodexModels {
		p := filepath.Join(o.Output, "case-001-codex-"+model+"-none-r1", "result.json")
		if _, err := os.Stat(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDuplicateMatrixCellsRejectedBeforeModelsRun(t *testing.T) {
	calls := fakeCLI(t)
	root := t.TempDir()
	corpus := filepath.Join(root, "cases.json")
	writeJSON(corpus, Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "one", Prompt: "p", Assertions: []string{"a"}}}})
	o := Options{Corpus: corpus, Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none", "none"}, CodexModel: "test-sol", Effort: "medium", CacheMode: "cold", Repetitions: 1, Workers: 2, Timeout: 5 * time.Second}
	if err := Run(context.Background(), o); err == nil {
		t.Fatal("duplicate arms accepted")
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatal("ran models before rejecting duplicates")
	}
	o.Arms = []string{"none"}
	o.CodexModels = []string{"vendor/model", "vendor_model"}
	if err := Run(context.Background(), o); err == nil {
		t.Fatal("colliding model directories accepted")
	}
}
