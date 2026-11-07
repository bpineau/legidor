package eval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestManifestRejectsChangedResume(t *testing.T) {
	p := filepath.Join(t.TempDir(), "manifest.json")
	if err := EnsureManifest(p, map[string]string{"model": "one"}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureManifest(p, map[string]string{"model": "one"}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureManifest(p, map[string]string{"model": "two"}); err == nil {
		t.Fatal("accepted incompatible resume")
	}
}
func TestCorpusValidation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cases.json")
	for _, c := range []Corpus{{AsOf: "2026-11-07"}, {AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "one", Prompt: "p"}}}, {AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "one", Prompt: "p", Assertions: []string{"a"}}, {ID: 1, Name: "two", Prompt: "p", Assertions: []string{"a"}}}}} {
		writeJSON(p, c)
		if _, err := LoadCorpus(p); err == nil {
			t.Fatal("accepted invalid corpus")
		}
	}
}
func TestGradeValidation(t *testing.T) {
	valid := `{"grades":[{"sample":"a","criteria":[{"index":0,"pass":true,"reason":"correct"}],"evidence":1,"temporal":2,"notes":""}]}`
	expected := map[string]int{"a": 1}
	if _, err := ParseGrades(valid, expected); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{"grades":[]}`, strings.Replace(valid, `"index":0`, `"index":1`, 1), strings.Replace(valid, `"evidence":1`, `"evidence":3`, 1), strings.Replace(valid, `"pass":true,`, "", 1)} {
		if _, err := ParseGrades(bad, expected); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}
func TestReportCountsFailuresAndMissingGrades(t *testing.T) {
	root := t.TempDir()
	for i, s := range []Sample{{Agent: "codex", Arm: "none", Status: "ok", Measurement: Measurement{InputTokens: 100}}, {Agent: "codex", Arm: "none", Status: "failed"}} {
		p := filepath.Join(root, string(rune('a'+i)))
		os.MkdirAll(p, 0700)
		writeJSON(filepath.Join(p, "result.json"), s)
	}
	if err := Report(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []Summary
	json.Unmarshal(data, &rows)
	if len(rows) != 1 || rows[0].Attempts != 2 || rows[0].Succeeded != 1 || rows[0].Graded != 0 || rows[0].InputTokens != 100 {
		t.Fatalf("%s", data)
	}
}

func TestIsolatedCodexHomeOnlyLinksAuthentication(t *testing.T) {
	source := t.TempDir()
	os.WriteFile(filepath.Join(source, "auth.json"), []byte(`{"tokens":{}}`), 0600)
	os.WriteFile(filepath.Join(source, "AGENTS.md"), []byte("canary instruction"), 0600)
	work := t.TempDir()
	home, err := isolatedCodexHome(work, source)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 1 || entries[0].Name() != "auth.json" {
		t.Fatalf("unexpected files: %v %v", entries, err)
	}
	target, err := os.Readlink(filepath.Join(home, "auth.json"))
	if err != nil || target != filepath.Join(source, "auth.json") {
		t.Fatal(target, err)
	}
}

func TestGradeCorpusMustMatchRun(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "corpus.json")
	os.WriteFile(p, []byte(`{"cases":[]}`), 0600)
	h, _ := HashFile(p)
	writeJSON(filepath.Join(root, "manifest.json"), map[string]string{"corpus_sha256": h})
	if err := verifyGradingCorpus(root, p); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, []byte(`{"cases":[1]}`), 0600)
	if err := verifyGradingCorpus(root, p); err == nil {
		t.Fatal("accepted changed rubric")
	}
}
func TestAttemptsNeverOverwriteEarlierEvidence(t *testing.T) {
	root := t.TempDir()
	a, err := newAttempt(root)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(a, "trace.jsonl"), []byte("old evidence"), 0600)
	b, err := newAttempt(root)
	if err != nil || a == b {
		t.Fatal(a, b, err)
	}
	data, _ := os.ReadFile(filepath.Join(a, "trace.jsonl"))
	if string(data) != "old evidence" {
		t.Fatal("lost evidence")
	}
}
func TestBlindOrderChangesAcrossCases(t *testing.T) {
	ids := []string{"claude-both", "claude-legidor", "claude-none", "claude-paperasse", "codex-both", "codex-legidor", "codex-none", "codex-paperasse"}
	a := blindOrder(ids, 1)
	b := blindOrder(ids, 2)
	if strings.Join(a, ",") == strings.Join(b, ",") {
		t.Fatal("same confounded order")
	}
	if strings.Join(ids, ",") != "claude-both,claude-legidor,claude-none,claude-paperasse,codex-both,codex-legidor,codex-none,codex-paperasse" {
		t.Fatal("mutated original order")
	}
}
func TestFailureRemainsInFiscalDenominator(t *testing.T) {
	root := t.TempDir()
	writeJSON(filepath.Join(root, "corpus.json"), Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "case", Prompt: "p", Assertions: []string{"a", "b"}}}})
	for i, s := range []Sample{{CaseID: 1, Agent: "codex", Arm: "none", Status: "ok"}, {CaseID: 1, Agent: "codex", Arm: "none", Status: "failed"}} {
		p := filepath.Join(root, string(rune('a'+i)))
		os.MkdirAll(p, 0700)
		writeJSON(filepath.Join(p, "result.json"), s)
	}
	p := filepath.Join(root, "grading", "case-001")
	os.MkdirAll(p, 0700)
	yes := true
	writeJSON(filepath.Join(p, "grades.json"), GradeSet{Grades: []Grade{{Sample: "a", Criteria: []Criterion{{Index: 0, Pass: &yes, Reason: "ok"}, {Index: 1, Pass: &yes, Reason: "ok"}}, Evidence: 2, Temporal: 2}}})
	if err := Report(root); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "summary.json"))
	var rows []Summary
	json.Unmarshal(b, &rows)
	if len(rows) != 1 || rows[0].ExpectedCriteria != 4 || rows[0].Passed != 2 || rows[0].Failed != 1 {
		t.Fatalf("%s", b)
	}
}

func TestRecoverInterruptedSamplePreservesPartialEvidence(t *testing.T) {
	root := t.TempDir()
	attempt, err := newAttempt(root)
	if err != nil {
		t.Fatal(err)
	}
	trace := `{"type":"assistant","message":{"id":"m","usage":{"input_tokens":12,"output_tokens":5},"content":[]}}` + "\n"
	os.WriteFile(filepath.Join(attempt, "trace.jsonl"), []byte(trace), 0600)
	sample, err := recoverInterrupted(root, Sample{Agent: "claude", CaseID: 1, Arm: "none"})
	if err != nil {
		t.Fatal(err)
	}
	if sample.Status != "failed" || sample.UsageComplete || sample.InputTokens != 12 {
		t.Fatalf("%+v", sample)
	}
	b, _ := os.ReadFile(filepath.Join(attempt, "trace.jsonl"))
	if string(b) != trace {
		t.Fatal("changed trace")
	}
}

func TestGradingCostsIncludeFailedAttempts(t *testing.T) {
	root := t.TempDir()
	cost := 0.1
	for _, n := range []string{"attempt-001", "attempt-002"} {
		p := filepath.Join(root, "grading", "case-001", n)
		os.MkdirAll(p, 0700)
		writeJSON(filepath.Join(p, "usage.json"), GradeUsage{Measurement: Measurement{InputTokens: 50, CostUSD: &cost, UsageComplete: true}, DurationMS: 100})
	}
	if err := Report(root); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "grading-summary.json"))
	var s GradingSummary
	json.Unmarshal(b, &s)
	if s.Attempts != 2 || s.InputTokens != 100 || s.CostUSD == nil || *s.CostUSD != 0.2 {
		t.Fatalf("%s", b)
	}
}

func TestReportKeepsModelsSeparate(t *testing.T) {
	root := t.TempDir()
	for i, model := range []string{"claude-opus-5-5", "claude-sonnet-5-5"} {
		p := filepath.Join(root, string(rune('a'+i)))
		os.MkdirAll(p, 0700)
		writeJSON(filepath.Join(p, "result.json"), Sample{Agent: "claude", Arm: "none", Status: "ok", Measurement: Measurement{Model: model, UsageComplete: true}})
	}
	if err := Report(root); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "summary.json"))
	var rows []Summary
	json.Unmarshal(b, &rows)
	if len(rows) != 2 {
		t.Fatalf("models merged: %s", b)
	}
}

func TestParallelJobsBoundConcurrency(t *testing.T) {
	var active, maxActive, completed atomic.Int64
	err := parallel(context.Background(), 12, 2, func(_ int) error {
		n := active.Add(1)
		for {
			old := maxActive.Load()
			if n <= old || maxActive.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		active.Add(-1)
		completed.Add(1)
		return nil
	})
	if err != nil || completed.Load() != 12 || maxActive.Load() > 2 || maxActive.Load() < 2 {
		t.Fatalf("completed=%d max=%d err=%v", completed.Load(), maxActive.Load(), err)
	}
}

func TestInterruptedCodexKeepsRequestedModel(t *testing.T) {
	root := t.TempDir()
	attempt, err := newAttempt(root)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(attempt, "trace.jsonl"), []byte("{\"type\":\"thread.started\"}\n"), 0600)
	s, err := recoverInterrupted(root, Sample{Agent: "codex", RequestedModel: "test-astra"})
	if err != nil || s.Model != "test-astra" {
		t.Fatalf("%+v %v", s, err)
	}
	s.Model = ""
	writeJSON(filepath.Join(root, "result.json"), s)
	samples, err := Samples(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range samples {
		if got.Model != "test-astra" {
			t.Fatalf("lost persisted model: %+v", got)
		}
	}
}
