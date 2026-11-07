package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	Corpus, Output, Paperasse, Skill, Legidor  string
	Agents, Arms, IDs                          []string
	ClaudeModel, CodexModel, Effort, CacheMode string
	Repetitions                                int
	Seed                                       int64
	Timeout                                    time.Duration
}
type Sample struct {
	CaseID     int       `json:"case_id"`
	Agent      string    `json:"agent"`
	Arm        string    `json:"arm"`
	Repetition int       `json:"repetition"`
	Status     string    `json:"status"`
	Trace      string    `json:"trace,omitempty"`
	Error      string    `json:"error,omitempty"`
	Started    time.Time `json:"started"`
	DurationMS int64     `json:"duration_ms"`
	Measurement
}

// EnsureManifest prevents mixing resource/model/protocol versions when resuming.
func EnsureManifest(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err == nil {
		if !bytes.Equal(bytes.TrimSpace(old), bytes.TrimSpace(b)) {
			return errors.New("manifest differs: use a new output directory for changed inputs")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return writeJSON(path, v)
}
func contains(ss []string, s string) bool {
	for _, a := range ss {
		if a == s {
			return true
		}
	}
	return false
}
func (o Options) validate() error {
	if o.Output == "" || o.Corpus == "" || o.Repetitions < 1 || o.Timeout <= 0 || len(o.Agents) == 0 || len(o.Arms) == 0 {
		return errors.New("corpus, output, agents, arms, positive repetitions and timeout are required")
	}
	for _, a := range o.Agents {
		if a != "claude" && a != "codex" {
			return fmt.Errorf("unknown agent %s", a)
		}
	}
	for _, a := range o.Arms {
		if a != "none" && a != "paperasse" && a != "legidor" && a != "both" {
			return fmt.Errorf("unknown arm %s", a)
		}
	}
	if o.CacheMode != "cold" && o.CacheMode != "shared" && o.CacheMode != "fresh" {
		return errors.New("cache mode must be cold, shared or fresh")
	}
	return nil
}

func Run(ctx context.Context, o Options) error {
	if err := o.validate(); err != nil {
		return err
	}
	corpus, err := LoadCorpus(o.Corpus)
	if err != nil {
		return err
	}
	o.Output, err = filepath.Abs(o.Output)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(o.Output, 0700); err != nil {
		return err
	}
	// A staging snapshot is rebuilt and hashed before validating a resume.
	stage, err := os.MkdirTemp("", "legidor-eval-resources-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if contains(o.Arms, "paperasse") || contains(o.Arms, "both") {
		if err = Snapshot(o.Paperasse, filepath.Join(stage, "paperasse")); err != nil {
			return err
		}
	}
	if contains(o.Arms, "legidor") || contains(o.Arms, "both") {
		if err = Snapshot(o.Skill, filepath.Join(stage, "legidor")); err != nil {
			return err
		}
		o.Legidor, err = exec.LookPath(o.Legidor)
		if err != nil {
			return err
		}
		o.Legidor, _ = filepath.Abs(o.Legidor)
	}
	resourceHash, err := HashTree(stage)
	if err != nil {
		return err
	}
	corpusHash, err := HashFile(o.Corpus)
	if err != nil {
		return err
	}
	self, _ := os.Executable()
	runnerHash, err := HashFile(self)
	if err != nil {
		return err
	}
	versions := map[string]string{}
	for _, agent := range o.Agents {
		b, err := exec.CommandContext(ctx, agent, "--version").Output()
		if err != nil {
			return err
		}
		versions[agent] = strings.TrimSpace(string(b))
	}
	binHash := ""
	if o.Legidor != "" && (contains(o.Arms, "legidor") || contains(o.Arms, "both")) {
		binHash, err = HashFile(o.Legidor)
		if err != nil {
			return err
		}
	}
	manifest := map[string]any{"protocol": 1, "corpus_sha256": corpusHash, "resources_sha256": resourceHash, "runner_sha256": runnerHash, "legidor_sha256": binHash, "cli_versions": versions, "claude_model": o.ClaudeModel, "codex_model": o.CodexModel, "effort": o.Effort, "cache_mode": o.CacheMode, "repetitions": o.Repetitions, "seed": o.Seed, "timeout_seconds": o.Timeout.Seconds(), "agents": o.Agents, "arms": o.Arms, "ids": o.IDs, "as_of": corpus.AsOf}
	if err = EnsureManifest(filepath.Join(o.Output, "manifest.json"), manifest); err != nil {
		return err
	}
	corpusBytes, err := os.ReadFile(o.Corpus)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(o.Output, "corpus.json"), corpusBytes, 0600); err != nil {
		return err
	}
	type job struct {
		c          Case
		agent, arm string
		rep        int
	}
	var jobs []job
	for rep := 1; rep <= o.Repetitions; rep++ {
		for _, c := range corpus.Cases {
			if len(o.IDs) > 0 && !contains(o.IDs, strconv.Itoa(c.ID)) {
				continue
			}
			for _, agent := range o.Agents {
				for _, arm := range o.Arms {
					jobs = append(jobs, job{c, agent, arm, rep})
				}
			}
		}
	}
	if len(jobs) == 0 {
		return errors.New("no matching cases")
	}
	rng := rand.New(rand.NewSource(o.Seed))
	rng.Shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })
	failures := 0
	for i, j := range jobs {
		if err = ctx.Err(); err != nil {
			return err
		}
		id := fmt.Sprintf("case-%03d-%s-%s-r%d", j.c.ID, j.agent, j.arm, j.rep)
		dir := filepath.Join(o.Output, id)
		if _, err = os.Stat(filepath.Join(dir, "result.json")); err == nil {
			var previous Sample
			b, readErr := os.ReadFile(filepath.Join(dir, "result.json"))
			if readErr != nil {
				return readErr
			}
			if err := json.Unmarshal(b, &previous); err != nil {
				return err
			}
			if previous.Status != "ok" {
				failures++
			}
			fmt.Fprintf(os.Stderr, "[%d/%d] resume %s\n", i+1, len(jobs), id)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if _, err := os.Stat(dir); err == nil {
			previous, err := recoverInterrupted(dir, Sample{CaseID: j.c.ID, Agent: j.agent, Arm: j.arm, Repetition: j.rep})
			if err != nil {
				return err
			}
			if err = writeJSON(filepath.Join(dir, "result.json"), previous); err != nil {
				return err
			}
			failures++
			fmt.Fprintf(os.Stderr, "[%d/%d] retained interrupted sample %s\n", i+1, len(jobs), id)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		work, err := os.MkdirTemp("", "legidor-eval-work-")
		if err != nil {
			return err
		}
		err = func() error {
			defer os.RemoveAll(work)
			resources := filepath.Join(work, "resources")
			if err := os.MkdirAll(resources, 0700); err != nil {
				return err
			}
			for _, name := range []string{"paperasse", "legidor"} {
				if j.arm == name || j.arm == "both" {
					if err := Snapshot(filepath.Join(stage, name), filepath.Join(resources, name)); err != nil {
						return err
					}
				}
			}
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
			attempt, err := newAttempt(dir)
			if err != nil {
				return err
			}
			cache := filepath.Join(attempt, "cache")
			if o.CacheMode == "shared" {
				cache = filepath.Join(o.Output, "shared-cache")
			}
			broker := Broker{Root: resources, Cache: cache, Fresh: o.CacheMode == "fresh", Log: filepath.Join(attempt, "tools.jsonl")}
			if j.arm == "legidor" || j.arm == "both" {
				broker.Legidor = o.Legidor
			}
			config := filepath.Join(attempt, "broker.json")
			if err := writeJSON(config, broker); err != nil {
				return err
			}
			prompt := fmt.Sprintf("Research date: %s. All facts below are synthetic. Answer the question, show the calculations and cite precise official sources for material rules, including their applicability to the stated dates. If a fact is missing, explain a conditional answer instead of inventing it.\n\n%s\n", corpus.AsOf, j.c.Prompt)
			if j.arm != "none" {
				prompt += "\nAllocated skills (read these first using read_resource):\n"
				for _, name := range []string{"paperasse", "legidor"} {
					if j.arm == name || j.arm == "both" {
						prompt += name + "/SKILL.md\n"
					}
				}
				prompt += "\nResolve links in each SKILL.md relative to that skill's allocated directory. "
			}
			prompt += "\nUse calculate for arithmetic, native web tools for research"
			if broker.Legidor != "" {
				prompt += ", and the legidor tool to execute the CLI recipes (argv array, no shell)"
			}
			prompt += ". No shell or script runtime is available."
			model := o.ClaudeModel
			if j.agent == "codex" {
				model = o.CodexModel
			}
			sample := Sample{CaseID: j.c.ID, Agent: j.agent, Arm: j.arm, Repetition: j.rep, Started: time.Now().UTC(), Status: "ok", Trace: filepath.Join(filepath.Base(attempt), "trace.jsonl")}
			fmt.Fprintf(os.Stderr, "[%d/%d] run %s (%s)\n", i+1, len(jobs), id, model)
			m, err := Invoke(ctx, Invocation{Agent: j.agent, Model: model, Effort: o.Effort, Work: work, Output: attempt, Prompt: prompt, BrokerConfig: config, Timeout: o.Timeout})
			sample.Measurement = m
			sample.DurationMS = time.Since(sample.Started).Milliseconds()
			if err != nil {
				sample.Status = "failed"
				sample.Error = err.Error()
				failures++
			}
			if err := writeJSON(filepath.Join(dir, "result.json"), sample); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, "answer.md"), []byte(m.Answer), 0600)
		}()
		if err != nil {
			return err
		}
		if err = Report(o.Output); err != nil {
			return err
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d failed samples retained; inspect results", failures)
	}
	return nil
}

func recoverInterrupted(dir string, s Sample) (Sample, error) {
	s.Status = "failed"
	s.Error = "interrupted before result persistence; usage and duration may be incomplete; use a new run to retry"
	traces, err := filepath.Glob(filepath.Join(dir, "attempt-*", "trace.jsonl"))
	if err != nil {
		return s, err
	}
	trace := filepath.Join(dir, "trace.jsonl")
	if len(traces) > 0 {
		trace = traces[len(traces)-1]
	}
	if f, err := os.Open(trace); err == nil {
		s.Measurement, _ = ParseTrace(s.Agent, f)
		f.Close()
		s.Trace, _ = filepath.Rel(dir, trace)
	} else if !os.IsNotExist(err) {
		return s, err
	}
	return s, nil
}

func Samples(root string) (map[string]Sample, error) {
	samples := map[string]Sample{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "cache" || d.Name() == "shared-cache" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "result.json" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var s Sample
		if err = json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s.Agent == "" {
			return nil
		}
		// Re-parse older/partial measurements when native evidence is available.
		// This also upgrades early pilot artifacts without modifying their traces.
		if !s.UsageComplete {
			trace := s.Trace
			if trace == "" {
				trace = "trace.jsonl"
			}
			if f, err := os.Open(filepath.Join(filepath.Dir(p), trace)); err == nil {
				m, _ := ParseTrace(s.Agent, f)
				f.Close()
				if m.Model == "" {
					m.Model = s.Model
				}
				if m.Answer == "" {
					m.Answer = s.Answer
				}
				s.Measurement = m
			}
		}
		samples[filepath.Base(filepath.Dir(p))] = s
		return nil
	})
	return samples, err
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
