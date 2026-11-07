// legidor-eval runs reproducible research comparisons using native agent CLIs.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	bench "github.com/bpineau/legidor/internal/eval"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func list(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: legidor-eval run|grade|report|serve [options]")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	f := flag.NewFlagSet(os.Args[1], flag.ContinueOnError)
	switch os.Args[1] {
	case "import-paperasse":
		repo := f.String("repo", "", "public Paperasse checkout")
		dest := f.String("out", "evals", "destination directory")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		if *repo == "" {
			return fmt.Errorf("-repo is required")
		}
		return bench.ImportPaperasse(*repo, *dest)
	case "serve":
		p := f.String("config", "", "private broker configuration")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		b, e := os.ReadFile(*p)
		if e != nil {
			return e
		}
		var broker bench.Broker
		if e = json.Unmarshal(b, &broker); e != nil {
			return e
		}
		return broker.Serve(ctx, os.Stdin, os.Stdout)
	case "report":
		out := f.String("out", "", "run directory")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		if *out == "" {
			return fmt.Errorf("-out is required")
		}
		return bench.Report(*out)
	case "grade":
		out := f.String("out", "", "run directory")
		corpus := f.String("corpus", "evals/pilot.json", "corpus")
		agent := f.String("agent", "claude", "judge CLI: claude or codex")
		model := f.String("model", "claude-opus-5-5", "judge model")
		timeout := f.Duration("timeout", 10*time.Minute, "per-case timeout")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		if *out == "" {
			return fmt.Errorf("-out is required")
		}
		abs, e := filepath.Abs(*out)
		if e != nil {
			return e
		}
		return bench.GradeAnswers(ctx, abs, *corpus, *agent, *model, *timeout)
	case "run":
		var o bench.Options
		f.StringVar(&o.Corpus, "corpus", "evals/pilot.json", "corpus JSON")
		f.StringVar(&o.Output, "out", "", "output directory (private, outside Git)")
		f.StringVar(&o.Paperasse, "paperasse", "", "path to Paperasse fiscaliste directory")
		f.StringVar(&o.Skill, "skill", "skills/legidor", "Legidor skill directory")
		f.StringVar(&o.Legidor, "legidor", "legidor", "Legidor executable")
		f.StringVar(&o.ClaudeModel, "claude-model", "claude-opus-5-5", "exact Claude model")
		f.StringVar(&o.CodexModel, "codex-model", "gpt-6.1-sol", "exact Codex model")
		claudeModels := f.String("claude-models", "", "comma-separated Claude model IDs (overrides -claude-model)")
		codexModels := f.String("codex-models", "", "comma-separated Codex model IDs (overrides -codex-model)")
		f.StringVar(&o.Effort, "effort", "medium", "reasoning effort")
		f.StringVar(&o.CacheMode, "cache", "cold", "cold (per-sample), shared (run-local), or fresh")
		f.IntVar(&o.Repetitions, "repetitions", 1, "fresh conversations per cell")
		f.IntVar(&o.Workers, "workers", 1, "maximum concurrent agent conversations (1..16)")
		f.IntVar(&o.MaxSamples, "max-samples", 0, "reject a larger matrix before model calls (0 = unlimited)")
		f.BoolVar(&o.ContinueOnError, "continue-on-error", false, "continue scheduling after failed samples (default stops)")
		f.Int64Var(&o.Seed, "seed", 42, "schedule shuffle seed")
		f.DurationVar(&o.Timeout, "timeout", 8*time.Minute, "per-sample deadline")
		agents := f.String("agents", "claude,codex", "comma-separated agents")
		arms := f.String("arms", "none,paperasse,legidor,both", "comma-separated arms")
		ids := f.String("cases", "", "comma-separated case IDs (default all)")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		o.Agents = list(*agents)
		o.Arms = list(*arms)
		o.IDs = list(*ids)
		o.ClaudeModels = list(*claudeModels)
		o.CodexModels = list(*codexModels)
		return bench.Run(ctx, o)
	default:
		return fmt.Errorf("unknown command %q", os.Args[1])
	}
}
