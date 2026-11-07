package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func benchmarkFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "corpus.json")
	if err := writeJSON(p, Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "arithmetic", Prompt: "2+2", Assertions: []string{"four", "explains"}}}}); err != nil {
		t.Fatal(err)
	}
	h, _ := HashFile(p)
	writeJSON(filepath.Join(root, "manifest.json"), map[string]any{"corpus_sha256": h, "repetitions": 1, "cli_versions": map[string]string{"codex": "test-cli 1"}})
	grades := GradeSet{}
	for _, arm := range []string{"none", "legidor", "paperasse", "both"} {
		os.MkdirAll(filepath.Join(root, arm), 0700)
		writeJSON(filepath.Join(root, arm, "result.json"), Sample{CaseID: 1, Agent: "codex", RequestedModel: "test-sol", Arm: arm, Repetition: 1, Status: "ok", DurationMS: 2000, Measurement: Measurement{Model: "test-sol", InputTokens: 100, OutputTokens: 10, CacheReadTokens: 60, UsageComplete: true, ToolCalls: 2}})
		pass, fail := true, false
		grades.Grades = append(grades.Grades, Grade{Sample: arm, Criteria: []Criterion{{Index: 1, Pass: &fail, Reason: "missing explanation"}, {Index: 0, Pass: &pass, Reason: "correct"}}, Evidence: 1, Temporal: 2})
	}
	os.MkdirAll(filepath.Join(root, "grading", "case-001"), 0700)
	writeJSON(filepath.Join(root, "grading", "case-001", "grades.json"), grades)
	return root
}

func TestAnthropicBenchmarkExport(t *testing.T) {
	root := benchmarkFixture(t)
	out := filepath.Join(t.TempDir(), "export")
	if err := ExportAnthropic(root, out, "codex", "test-sol", "both", "paperasse"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Runs []struct {
			Configuration string
			Result        struct {
				PassRate float64 `json:"pass_rate"`
				Tokens   int64
			}
			Expectations []struct {
				Text     string
				Passed   bool
				Evidence string
			}
		}
		Metadata struct {
			Model string `json:"executor_model"`
		}
	}
	// Read the viewer's exact field names, independently of the exporter types.
	var rawDoc map[string]json.RawMessage
	json.Unmarshal(b, &rawDoc)
	var summary struct {
		With map[string]struct {
			Mean   float64
			Stddev float64
		} `json:"with_skill"`
		Without map[string]struct {
			Mean   float64
			Stddev float64
		} `json:"without_skill"`
		Delta map[string]string `json:"delta"`
	}
	if err := json.Unmarshal(rawDoc["run_summary"], &summary); err != nil {
		t.Fatal(err)
	}
	if summary.With["pass_rate"].Mean != 0.5 || summary.Without["tokens"].Mean != 110 || summary.Delta["tokens"] != "+0.000000" {
		t.Fatalf("incompatible summary: %s", rawDoc["run_summary"])
	}
	delete(rawDoc, "run_summary")
	withoutSummary, _ := json.Marshal(rawDoc)
	if err := json.Unmarshal(withoutSummary, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Runs) != 2 || doc.Metadata.Model != "test-sol" || doc.Runs[0].Configuration != "with_skill" || doc.Runs[1].Configuration != "without_skill" {
		t.Fatalf("mixed or mislabeled configurations: %s", b)
	}
	for _, run := range doc.Runs {
		if run.Result.PassRate != 0.5 || run.Result.Tokens != 110 || run.Expectations[0].Text != "four" || !run.Expectations[0].Passed || run.Expectations[1].Passed {
			t.Fatalf("wrong mapping or tokens: %s", b)
		}
	}
	grading, err := os.ReadFile(filepath.Join(out, "case-001", "with_skill", "run-1", "grading.json"))
	if err != nil || !strings.Contains(string(grading), `"evidence": "correct"`) {
		t.Fatalf("missing compatible grading: %s %v", grading, err)
	}
	if err := ExportAnthropic(root, out, "codex", "test-sol", "both", "paperasse"); err == nil {
		t.Fatal("overwrote prior export")
	}
}

func TestExportStatisticsUseSampleDeviation(t *testing.T) {
	s := summarize([]float64{2, 4, 6})
	if s.Mean != 4 || s.Stddev != 2 || s.Min != 2 || s.Max != 6 {
		t.Fatalf("wrong statistics: %+v", s)
	}
}

func TestAnthropicExportRejectsIncompleteOrMixedPairs(t *testing.T) {
	for _, kind := range []string{"failed", "ungraded", "partial-usage", "unstarted", "different-model", "changed-corpus"} {
		t.Run(kind, func(t *testing.T) {
			root := benchmarkFixture(t)
			p := filepath.Join(root, "legidor", "result.json")
			b, _ := os.ReadFile(p)
			var s Sample
			json.Unmarshal(b, &s)
			switch kind {
			case "failed":
				s.Status = "failed"
			case "partial-usage":
				s.UsageComplete = false
			case "different-model":
				s.Model = "different"
			case "ungraded":
				os.Remove(filepath.Join(root, "grading", "case-001", "grades.json"))
			case "changed-corpus":
				os.WriteFile(filepath.Join(root, "corpus.json"), append([]byte(" "), mustRead(t, filepath.Join(root, "corpus.json"))...), 0600)
			}
			writeJSON(p, s)
			if kind == "unstarted" {
				os.Remove(p)
			}
			out := filepath.Join(t.TempDir(), "export")
			if err := ExportAnthropic(root, out, "codex", "test-sol", "legidor", "none"); err == nil {
				t.Fatal("export hid an incomplete or changed comparison")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("partial export left behind")
			}
		})
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
