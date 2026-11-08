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
	"text/tabwriter"
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
		return fmt.Errorf("usage: legidor-eval run|campaign|grade|report|list|convert|import-paperasse|export-anthropic|serve [options]")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	f := flag.NewFlagSet(os.Args[1], flag.ContinueOnError)
	switch os.Args[1] {
	case "export-anthropic":
		root := f.String("run", "", "completed Legidor run directory")
		out := f.String("out", "", "new private export directory")
		agent := f.String("agent", "", "agent CLI to export")
		model := f.String("model", "", "exact recorded model to export")
		with := f.String("with", "", "arm mapped to with_skill")
		without := f.String("without", "", "arm mapped to without_skill")
		if err := f.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *root == "" || *out == "" {
			return fmt.Errorf("-run and -out are required")
		}
		return bench.ExportAnthropic(*root, *out, *agent, *model, *with, *without)
	case "convert":
		corpus := f.String("corpus", "", "source corpus (Anthropic or legacy Legidor JSON)")
		out := f.String("out", "", "destination JSON file")
		asOf := f.String("as-of", "", "research date YYYY-MM-DD (required when missing from source)")
		if err := f.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *corpus == "" || *out == "" {
			return fmt.Errorf("-corpus and -out are required")
		}
		return bench.ConvertCorpus(*corpus, *out, *asOf)
	case "list":
		corpus := f.String("corpus", "evals/pilot.json", "corpus JSON")
		if err := f.Parse(os.Args[2:]); err != nil {
			return err
		}
		c, err := bench.LoadCorpus(*corpus)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSCENARIO\tCRITERIA\tFILES")
		total := 0
		for _, s := range c.Cases {
			fmt.Fprintf(w, "%d\t%s\t%d\t%d\n", s.ID, s.Name, len(s.Assertions), len(s.Files))
			total += len(s.Assertions)
		}
		fmt.Fprintf(w, "\n%d cases, %d criteria; research date %s\n", len(c.Cases), total, c.AsOf)
		return w.Flush()
	case "import-paperasse":
		repo := f.String("repo", "", "public Paperasse checkout")
		dest := f.String("out", "evals", "destination directory")
		all := f.Bool("all", false, "import every upstream scenario unchanged into all.json")
		asOf := f.String("as-of", "2026-11-07", "research date YYYY-MM-DD")
		if e := f.Parse(os.Args[2:]); e != nil {
			return e
		}
		if *repo == "" {
			return fmt.Errorf("-repo is required")
		}
		return bench.ImportPaperasseCorpus(*repo, *dest, *all, *asOf)
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
	case "run", "campaign":
		var o bench.Options
		var judgeAgent, judgeModel string
		var gradingTimeout time.Duration
		if os.Args[1] == "campaign" {
			f.StringVar(&judgeAgent, "judge-agent", "claude", "judge CLI")
			f.StringVar(&judgeModel, "judge-model", "", "explicit judge model (required)")
			f.DurationVar(&gradingTimeout, "grading-timeout", 10*time.Minute, "per-case grading deadline")
		}
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
		if os.Args[1] == "campaign" {
			return bench.Campaign(ctx, o, judgeAgent, judgeModel, gradingTimeout)
		}
		return bench.Run(ctx, o)
	default:
		return fmt.Errorf("unknown command %q", os.Args[1])
	}
}
