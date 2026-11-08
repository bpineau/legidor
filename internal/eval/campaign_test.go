package eval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCampaignStopsBeforeGradingWhenExecutionFails(t *testing.T) {
	calls := fakeCLI(t)
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus.json")
	writeJSON(corpus, Corpus{AsOf: "2026-11-08", Cases: []Case{{ID: 1, Name: "fixture", Prompt: "p", Assertions: []string{"a"}}}})
	o := Options{Corpus: corpus, Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test-sol", CacheMode: "cold", Repetitions: 1, Timeout: 100 * time.Millisecond}
	t.Setenv("LEGIDOR_EVAL_FAKE_WAIT", "1")
	if err := Campaign(context.Background(), o, "claude", "test-sonnet", time.Second); err == nil {
		t.Fatal("failed execution reported as complete")
	}
	var state struct {
		Phase string
		Error string
	}
	b, err := os.ReadFile(filepath.Join(o.Output, "campaign.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &state); err != nil {
		t.Fatal(err)
	}
	if state.Phase != "execution_failed" || state.Error == "" {
		t.Fatalf("wrong state: %s", b)
	}
	if _, err = os.Stat(filepath.Join(o.Output, "grading")); !os.IsNotExist(err) {
		t.Fatal("graded after failed execution")
	}
	if b, err = os.ReadFile(calls); err == nil && len(b) > 0 {
		t.Fatal("unexpected completed invocation")
	}
}

func TestCampaignValidatesJudgeBeforeAnyCalls(t *testing.T) {
	calls := fakeCLI(t)
	root := t.TempDir()
	o := Options{Corpus: filepath.Join(root, "absent"), Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test-sol", CacheMode: "cold", Repetitions: 1, Timeout: time.Second}
	err := Campaign(context.Background(), o, "bad-agent", "test", time.Second)
	if err == nil || !strings.Contains(err.Error(), "judge") {
		t.Fatalf("invalid judge accepted: %v", err)
	}
	if _, err = os.Stat(calls); !os.IsNotExist(err) {
		t.Fatal("called a model before judge validation")
	}
}

func TestCampaignCompletesAndResumesWithoutNewCalls(t *testing.T) {
	calls := fakeCLI(t)
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus.json")
	writeJSON(corpus, Corpus{AsOf: "2026-11-08", Cases: []Case{{ID: 1, Name: "fixture", Prompt: "p", Assertions: []string{"a"}}}})
	t.Setenv("LEGIDOR_EVAL_FAKE_GRADE", `{"grades":[{"sample":"answer-1","criteria":[{"index":0,"pass":true,"reason":"correct"}],"evidence":2,"temporal":2,"notes":""}]}`)
	o := Options{Corpus: corpus, Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test-sol", CacheMode: "cold", Repetitions: 1, Timeout: 5 * time.Second}
	for range 2 {
		if err := Campaign(context.Background(), o, "codex", "test-judge", 5*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	var state struct{ Phase string }
	json.Unmarshal(mustRead(t, filepath.Join(o.Output, "campaign.json")), &state)
	if state.Phase != "complete" {
		t.Fatalf("campaign did not finish: %+v", state)
	}
	if string(mustRead(t, calls)) != "call\ncall\n" {
		t.Fatal("resume re-ran execution or grading")
	}
	var rows []Summary
	json.Unmarshal(mustRead(t, filepath.Join(o.Output, "summary.json")), &rows)
	if len(rows) != 1 || rows[0].Graded != 1 || rows[0].Passed != 1 {
		t.Fatalf("missing graded report: %+v", rows)
	}
}

func TestCampaignRejectsConcurrentLaunchIntoSameDirectory(t *testing.T) {
	root := t.TempDir()
	f, err := os.OpenFile(filepath.Join(root, ".campaign.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	o := Options{Corpus: filepath.Join(root, "absent.json"), Output: root, Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test-sol", CacheMode: "cold", Repetitions: 1, Timeout: time.Second}
	err = Campaign(context.Background(), o, "codex", "test-judge", time.Second)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("concurrent launch not rejected: %v", err)
	}
	if _, err = os.Stat(filepath.Join(root, "campaign.json")); !os.IsNotExist(err) {
		t.Fatal("overwrote the active campaign state")
	}
}
