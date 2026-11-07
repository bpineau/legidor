# Reproducible agent evaluations

Measure the incremental benefit of Legidor within each agent, separating fiscal
correctness, source evidence, and resource use. Compare Claude Opus 5.5, Sonnet 5.5,
Codex Sol 6.1 and Astra 6
with no skill, Paperasse fiscaliste, Legidor, and both. Cross-agent differences
also include their native web tools and system prompts.

The runner is an independent Go executable (`cmd/legidor-eval`), using only the
standard library. It invokes the installed native agent CLIs. No JavaScript or
Python runtime, Paperasse runner, or calculator script is used.

## Protocol

- Import public Paperasse scenarios and criteria, retaining the upstream commit,
  MIT notice, and explicit amendments. Start with eight cases. Keep other cases
  available as a holdout. Never import private dossiers.
- Specify income and event years, missing assumptions, and source expectations
  equally in all arms. Keep expected answers outside the agent workspace.
- Provide identical arithmetic and native web access. A small stdio MCP broker
  exposes only selected skill documents and, in Legidor arms, allowlisted CLI
  research commands. Disable native execution, ambient skills, MCP integrations,
  memory and user/project instructions. The broker cannot execute shell code or
  read arbitrary paths. Verify the effective tool surface in smoke tests.
- Load Paperasse data/references as read-only resources, excluding its evals and
  scripts. Hash all resources, prompts, rubric, runner and Legidor binaries.
- Pin model identifiers and effort. Capture native JSONL events, final answer,
  exit status, wall time, CLI versions, input/output/cache tokens, available cost
  estimates and tool calls. Unavailable costs remain null, never zero.
- Each sample gets a distinct directory and fresh conversation. Resume keeps
  completed samples with their original metrics; it does not manufacture another
  measurement. Repetitions invoke the models again. Refuse incompatible resumes.
- Report Legidor cache mode explicitly. Use a separate cache from everyday usage;
  distinguish response caching from provider prompt caching.
- Grade blinded answers against the same criteria. Keep grading usage separate.
  Include failures and missing grades in denominators. Source citations alone do
  not establish that a source supports the claim. Review dubious grades manually.
- Start with one repetition of eight cases across sixteen configurations. Use the
  pilot to diagnose defects; repeat selected cases after fixes. Do not claim
  statistical significance from one sample or selectively publish winners.

## Implementation plan

1. `internal/eval/corpus.go`: corpus, upstream import, provenance, validation and
   resource hashes. Tests cover duplicate IDs, missing rubrics and hash changes.
2. `internal/eval/broker.go`: scoped document reading, arithmetic and Legidor
   command adapter over stdio MCP. Tests cover traversal, symlinks, denied
   commands, malformed arithmetic and protocol errors.
3. `internal/eval/agent.go`, `runner.go`: isolated adapters, bounded execution,
   JSONL parsing, failure retention, repetitions and resume. Test both native
   event formats, incomplete/error output, cache accounting and manifest changes.
4. `internal/eval/report.go`, `cmd/legidor-eval/main.go`: CLI, blind grading,
   JSON/Markdown aggregation and reproducible commands. Test missing grades and
   denominators. Add README instructions and attribution.
5. Run live capability probes, pilot, grade and inspect failures. Correct easy
   Legidor defects with regression tests, rerun affected cases, publish an
   aggregate report with limitations. Keep raw local traces out of Git.
6. Run `go test -race ./...`, `go vet ./...`, obtain a fresh code review, address
   material findings, commit and push. Update deployed skills if changed.
