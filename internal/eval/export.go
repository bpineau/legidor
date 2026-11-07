package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type exportedExpectation struct {
	Text     string `json:"text"`
	Passed   bool   `json:"passed"`
	Evidence string `json:"evidence"`
}

type exportedResult struct {
	PassRate    float64 `json:"pass_rate"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	Total       int     `json:"total"`
	TimeSeconds float64 `json:"time_seconds"`
	Tokens      int64   `json:"tokens"`
	ToolCalls   int     `json:"tool_calls"`
}

type exportedRun struct {
	EvalID        int                   `json:"eval_id"`
	EvalName      string                `json:"eval_name"`
	Configuration string                `json:"configuration"`
	RunNumber     int                   `json:"run_number"`
	Result        exportedResult        `json:"result"`
	Expectations  []exportedExpectation `json:"expectations"`
	Notes         []string              `json:"notes"`
	Legidor       map[string]any        `json:"x_legidor"`
}

type statistics struct {
	Mean   float64 `json:"mean"`
	Stddev float64 `json:"stddev"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

// ExportAnthropic projects one complete, graded model/arm pair onto the upstream
// benchmark and grading formats. It does not invoke a model. Incomplete campaigns
// remain in Legidor's native report: dropping failed or unstarted cells here would
// turn a selective export into misleading evidence of improvement.
func ExportAnthropic(root, dest, agent, model, with, without string) (err error) {
	if agent == "" || model == "" || with == "" || without == "" || with == without {
		return fmt.Errorf("explicit agent, model and two distinct arms are required")
	}
	if _, e := os.Lstat(dest); !os.IsNotExist(e) {
		return fmt.Errorf("export destination must not exist: %s", dest)
	}
	corpusPath := filepath.Join(root, "corpus.json")
	if err := verifyGradingCorpus(root, corpusPath); err != nil {
		return err
	}
	corpus, err := LoadCorpus(corpusPath)
	if err != nil {
		return err
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Repetitions int               `json:"repetitions"`
		IDs         []string          `json:"ids"`
		Versions    map[string]string `json:"cli_versions"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return err
	}
	if manifest.Repetitions < 1 {
		return fmt.Errorf("manifest must record positive repetitions")
	}
	samples, err := Samples(root)
	if err != nil {
		return err
	}
	type cell struct {
		caseID, repetition int
		arm                string
	}
	type entry struct {
		id     string
		sample Sample
	}
	cells := map[cell]entry{}
	for id, s := range samples {
		if s.Agent != agent || s.Model != model || (s.Arm != with && s.Arm != without) {
			continue
		}
		key := cell{s.CaseID, s.Repetition, s.Arm}
		if _, ok := cells[key]; ok {
			return fmt.Errorf("duplicate sample for case %d, arm %s, repetition %d", s.CaseID, s.Arm, s.Repetition)
		}
		cells[key] = entry{id, s}
	}
	var runs []exportedRun
	var ids []int
	for _, c := range corpus.Cases {
		if len(manifest.IDs) > 0 && !contains(manifest.IDs, strconv.Itoa(c.ID)) {
			continue
		}
		ids = append(ids, c.ID)
		gradesBytes, err := os.ReadFile(filepath.Join(root, "grading", fmt.Sprintf("case-%03d", c.ID), "grades.json"))
		if err != nil {
			return fmt.Errorf("case %d: complete grades required: %w", c.ID, err)
		}
		var grades GradeSet
		if err := json.Unmarshal(gradesBytes, &grades); err != nil {
			return err
		}
		bySample := map[string]Grade{}
		for _, g := range grades.Grades {
			if _, exists := bySample[g.Sample]; exists {
				return fmt.Errorf("duplicate grade for %s", g.Sample)
			}
			bySample[g.Sample] = g
		}
		for rep := 1; rep <= manifest.Repetitions; rep++ {
			for _, arm := range []string{with, without} {
				e, ok := cells[cell{c.ID, rep, arm}]
				if !ok {
					return fmt.Errorf("missing case %d, arm %s, repetition %d for %s/%s; native report retains incomplete campaigns", c.ID, arm, rep, agent, model)
				}
				s := e.sample
				g, ok := bySample[e.id]
				if !ok || s.Status != "ok" || !s.UsageComplete {
					return fmt.Errorf("sample %s requires success, complete usage and grading; use native report for incomplete campaigns", e.id)
				}
				if _, err := ParseGrades(string(raw(GradeSet{Grades: []Grade{g}})), map[string]int{e.id: len(c.Assertions)}); err != nil {
					return err
				}
				configuration := "with_skill"
				if arm == without {
					configuration = "without_skill"
				}
				r := exportedRun{EvalID: c.ID, EvalName: c.Name, Configuration: configuration, RunNumber: rep, Expectations: make([]exportedExpectation, len(c.Assertions)), Notes: []string{}, Result: exportedResult{Total: len(c.Assertions), Tokens: s.InputTokens + s.OutputTokens, TimeSeconds: float64(s.DurationMS) / 1000, ToolCalls: s.ToolCalls}, Legidor: map[string]any{"source_sample": e.id, "arm": arm, "agent": agent, "model": s.Model, "requested_model": s.RequestedModel, "cli_version": manifest.Versions[agent], "status": s.Status, "usage_complete": s.UsageComplete, "input_tokens": s.InputTokens, "output_tokens": s.OutputTokens, "cache_read_tokens": s.CacheReadTokens, "cache_write_tokens": s.CacheWriteTokens, "estimated_cost_usd": s.CostUSD, "evidence_score": g.Evidence, "temporal_score": g.Temporal}}
				for _, criterion := range g.Criteria {
					r.Expectations[criterion.Index] = exportedExpectation{c.Assertions[criterion.Index], *criterion.Pass, criterion.Reason}
					if *criterion.Pass {
						r.Result.Passed++
					}
				}
				r.Result.Failed = r.Result.Total - r.Result.Passed
				r.Result.PassRate = float64(r.Result.Passed) / float64(r.Result.Total)
				if g.Notes != "" {
					r.Notes = append(r.Notes, g.Notes)
				}
				runs = append(runs, r)
			}
		}
	}
	if len(runs) == 0 || len(runs) != len(cells) {
		return fmt.Errorf("selected samples do not match the manifest's planned cases and repetitions")
	}
	if len(manifest.IDs) > 0 && len(ids) != len(manifest.IDs) {
		return fmt.Errorf("manifest contains unknown or duplicate case IDs")
	}
	summary := map[string]any{}
	aggregates := map[string]map[string]statistics{}
	for _, config := range []string{"with_skill", "without_skill"} {
		values := map[string][]float64{"pass_rate": {}, "time_seconds": {}, "tokens": {}}
		for _, r := range runs {
			if r.Configuration == config {
				values["pass_rate"] = append(values["pass_rate"], r.Result.PassRate)
				values["time_seconds"] = append(values["time_seconds"], r.Result.TimeSeconds)
				values["tokens"] = append(values["tokens"], float64(r.Result.Tokens))
			}
		}
		aggregates[config] = map[string]statistics{}
		for key, v := range values {
			aggregates[config][key] = summarize(v)
		}
		summary[config] = aggregates[config]
	}
	delta := map[string]string{}
	for _, key := range []string{"pass_rate", "time_seconds", "tokens"} {
		delta[key] = fmt.Sprintf("%+.6f", aggregates["with_skill"][key].Mean-aggregates["without_skill"][key].Mean)
	}
	summary["delta"] = delta
	// Grading costs belong to the whole blinded case comparison, not a sample.
	gradingMetadata := map[string]json.RawMessage{}
	if b, e := os.ReadFile(filepath.Join(root, "grading", "manifest.json")); e == nil {
		if err := json.Unmarshal(b, &gradingMetadata); err != nil {
			return err
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	judge := "unknown"
	if b, ok := gradingMetadata["model"]; ok {
		if err := json.Unmarshal(b, &judge); err != nil {
			return err
		}
	}
	name := corpus.SkillName
	if name == "" {
		name = "legidor"
	}
	benchmark := map[string]any{
		"metadata": map[string]any{"skill_name": name, "executor_model": model, "analyzer_model": judge, "timestamp": time.Now().UTC().Format(time.RFC3339), "evals_run": ids, "runs_per_configuration": manifest.Repetitions},
		"runs":     runs, "run_summary": summary,
		"notes":     []string{"Explicit arm mapping: with_skill=" + with + ", without_skill=" + without + ".", "Only complete paired campaigns can be exported. Tokens include cache reads; grader usage is separate. Unmeasured error counts, per-tool counts and claims verification are omitted, not zero.", "Statistics use sample standard deviation across selected cases and repetitions; they are descriptive, not confidence intervals."},
		"x_legidor": map[string]any{"agent": agent, "cli_version": manifest.Versions[agent], "with_arm": with, "without_arm": without, "manifest": json.RawMessage(manifestBytes), "grading_manifest": gradingMetadata},
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	if err := os.Mkdir(dest, 0700); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dest)
		}
	}()
	if err = writeJSON(filepath.Join(dest, "benchmark.json"), benchmark); err != nil {
		return err
	}
	if err = writeJSON(filepath.Join(dest, "evals.json"), corpus); err != nil {
		return err
	}
	for _, r := range runs {
		dir := filepath.Join(dest, fmt.Sprintf("case-%03d", r.EvalID), r.Configuration, fmt.Sprintf("run-%d", r.RunNumber))
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		grading := map[string]any{"expectations": r.Expectations, "summary": map[string]any{"passed": r.Result.Passed, "failed": r.Result.Failed, "total": r.Result.Total, "pass_rate": r.Result.PassRate}, "execution_metrics": map[string]any{"total_tool_calls": r.Result.ToolCalls}, "timing": map[string]any{"executor_duration_seconds": r.Result.TimeSeconds}, "x_legidor": r.Legidor}
		if err = writeJSON(filepath.Join(dir, "grading.json"), grading); err != nil {
			return err
		}
		timing := map[string]any{"total_tokens": r.Result.Tokens, "duration_ms": int64(math.Round(r.Result.TimeSeconds * 1000)), "total_duration_seconds": r.Result.TimeSeconds}
		if err = writeJSON(filepath.Join(dir, "timing.json"), timing); err != nil {
			return err
		}
	}
	return nil
}

func summarize(values []float64) statistics {
	s := statistics{Min: values[0], Max: values[0]}
	for _, v := range values {
		s.Mean += v
		s.Min = math.Min(s.Min, v)
		s.Max = math.Max(s.Max, v)
	}
	s.Mean /= float64(len(values))
	if len(values) > 1 {
		for _, v := range values {
			s.Stddev += (v - s.Mean) * (v - s.Mean)
		}
		s.Stddev = math.Sqrt(s.Stddev / float64(len(values)-1))
	}
	return s
}
