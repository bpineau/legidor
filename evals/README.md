# Agent evaluations

This suite measures whether Legidor helps an agent answer French tax questions.
It separates fiscal correctness, evidence quality, dates, tokens, elapsed time
and tool calls. It does not assume that adding a skill saves tokens.

The independent runner and its arithmetic helper use Go's standard library.
There is no Node.js, npm, Python, pip or uv dependency. Only the installed native
Claude Code and Codex executables, Go and Legidor are needed. Paperasse contributes
public scenario text, grading criteria and optional read-only documentation/data,
not its runner or executable scripts.

## Run

For development, start with two Sol samples and an explicit size limit:

```sh
make build build-eval
bin/legidor-eval run -agents codex -codex-model gpt-6.1-sol \
  -cases 3,13 -arms legidor -max-samples 2 -workers 1 \
  -legidor "$PWD/bin/legidor" -out /tmp/legidor-sol-smoke
bin/legidor-eval grade -out /tmp/legidor-sol-smoke \
  -agent codex -model gpt-6.1-sol
```

The default stops scheduling new samples after the first failure, including quota
errors. Already running samples finish and retain their measurements. Opt into
`-continue-on-error` only deliberately. `-max-samples N` rejects a larger matrix
before launching any agents; zero (the default) leaves the size unrestricted.
It counts conversations, not provider tokens or money. Grading makes separate
model calls, one per case with successful answers.

For a full comparison (64 samples with the default two models):

```sh
make build
make build-eval
bin/legidor-eval run \
  -paperasse /path/to/paperasse/fiscaliste \
  -legidor "$PWD/bin/legidor" \
  -max-samples 64 \
  -out /tmp/legidor-pilot
bin/legidor-eval grade -out /tmp/legidor-pilot
bin/legidor-eval report -out /tmp/legidor-pilot
```

Authenticate the two native agent CLIs beforehand. Legidor uses its usual private
PISTE configuration. No credentials are copied into the repository. The runner
supports macOS and Linux. Model runs consume the configured account's quota.
There are no paid API calls in `go test ./...`.

Defaults: `claude-opus-5-5`, `gpt-6.1-sol`, medium effort, eight cases, four arms,
one repetition, a shuffled sequential schedule (seed 42), and eight minutes per
sample. Model identifiers and CLI flags depend on the installed agent versions;
use `-claude-model` and `-codex-model` to test another agent version explicitly.

```sh
# One case across all eight configurations.
bin/legidor-eval run -cases 11 -paperasse /path/to/paperasse/fiscaliste \
  -legidor "$PWD/bin/legidor" -out /tmp/legidor-smoke

# Repeated measurements, each a new conversation, in a new run directory.
bin/legidor-eval run -cases 3,13 -repetitions 3 -cache cold \
  -paperasse /path/to/paperasse/fiscaliste \
  -legidor "$PWD/bin/legidor" -out /tmp/legidor-repeated
```

`-agents claude,codex` and `-arms none,paperasse,legidor,both` select the matrix.
`-workers 2` runs at most two agent conversations concurrently; the default is
one. Cold/fresh cache modes keep a separate cache for every sample. The manifest
records the worker count because concurrency can affect latency and quotas.

Model IDs appear in sample directories, individual results and aggregate tables;
tables also include CLI versions. Run additional model pairs in separate roots:

```sh
bin/legidor-eval run -claude-model claude-sonnet-5-5 -codex-model gpt-6-astra \
  -paperasse /path/to/paperasse/fiscaliste -legidor "$PWD/bin/legidor" \
  -workers 2 -out /tmp/legidor-sonnet-astra
```

Or run all four models in one matrix with a global concurrency limit:

```sh
bin/legidor-eval run \
  -claude-models claude-opus-5-5,claude-sonnet-5-5 \
  -codex-models gpt-6.1-sol,gpt-6-astra -workers 4 \
  -max-samples 128 \
  -paperasse /path/to/paperasse/fiscaliste -legidor "$PWD/bin/legidor" \
  -out /tmp/legidor-four-models
```

The plural model flags override their singular counterparts. Each requested model
gets every selected case, arm and repetition; it is also recorded separately from
the effective model reported by the agent (when available).

`-corpus` accepts another corpus with the same JSON schema as `pilot.json`.
Repeating an identical command resumes recorded samples with their original
metrics. It does not count reused answers as fresh zero-cost measurements. A
changed manifest requires a new output directory. Failed samples remain visible;
use a separate run to retest them rather than silently replacing them.

## What is isolated

Both agents receive the same case, arithmetic tool and their native web tools.
Neither can run arbitrary shell commands. A small stdio MCP adapter exposes only:

- `calculate`: bounded arithmetic, implemented in Go;
- `read_resource`: allocated skill documents/data, with path and symlink checks;
- `legidor`: read-only CLI commands, present only in Legidor arms. `refs` and
  `check` take content through standard input, never arbitrary local filenames.

The adapter is benchmark infrastructure, not a public general-purpose MCP server.
It invokes the real Legidor executable. It does change the interaction compared
with unrestricted shell access, so results describe this controlled harness.
Paperasse calculator scripts cannot run; agents can read its data and use the
same arithmetic tool as the baseline.

Claude uses restricted mode, explicit tools, an explicit system prompt, disabled
slash commands, no ambient settings or MCP servers, and no persisted session.
Codex gets a temporary configuration directory linking only its existing private
authentication file, disabled ambient skills/memory/instructions and shell tools,
and the explicitly approved benchmark MCP tools. Account authentication remains
with the native CLIs. This is tool isolation, not an adversarial OS sandbox.

Resources exclude scripts, evals, hidden files and external symlinks. A hash covers
all allocated files, including nested references and tax data. The manifest also
records corpus/rubric, runner and Legidor hashes, exact models, CLI versions,
effort, schedule seed, repetitions, time limit and cache mode.

## Caches and measurements

- `-cache cold` starts each sample with its own empty Legidor cache. Repeated
  reads within a sample can hit it.
- `-cache shared` shares a run-local cache. It is empty at the start of a new run,
  then warmed by earlier samples in the recorded shuffled order.
- `-cache fresh` appends `--fresh` to Legidor research calls. OAuth refresh still
  follows normal expiry; this is not a way to disable provider prompt caching.

No mode uses the operator's everyday Legidor cache.
The adapter overrides `LEGIDOR_CACHE_DIR` and `XDG_CACHE_HOME` in each Legidor
child process, including any inherited override, to enforce this isolation.
Provider prompt caching is
reported separately and may occur in every arm. Normalized input tokens include
cache reads and writes. A provider's reported cost is an estimate, not a bill;
Codex's unavailable cost stays null. Native tool-call counting conventions differ,
so compare efficiency within an agent first. Grader tokens/costs are separate.

Raw traces, prompts, answers and tool results stay in the private output directory.
They may include machine paths and account metadata: never commit them. Publish
only reviewed aggregate reports. Keep the exact binary used for a long run; do not
rebuild that executable while its samples are running.

## Grading and interpretation

`grade` defaults to Opus with no tools, no evaluated skill and no configuration labels.
Select another judge explicitly with `-agent codex -model gpt-6-astra`, for
example when Claude quota is unavailable. The grading manifest pins this choice;
use the same judge for every comparison in a report and disclose family bias.
It grades fiscal assertions separately from evidence and temporal scores (0..2).
Expected answers never enter the evaluated conversation. Every verdict includes a
reason; raw grader output and usage remain available for audit.

The evidence score measures what the answer demonstrates in text. It does not
independently verify every URL, legal quotation or conclusion. Review material
failures and disputed criteria against primary sources before using results to
change Legidor. An LLM judge is fallible, and using a model to grade its own family can
introduce model-family bias. Report this and manually inspect close comparisons.

Eight cases with one repetition are a diagnostic pilot, not a statistical proof.
Compare coverage before scores, inspect case-level failures, and repeat affected
cases after changes. Keep cases outside the pilot as holdouts. Compare models only
with the caveat that their native prompts, web access and token accounting differ.

## Provenance and amendments

The pilot derives from [Paperasse fiscaliste evaluations](https://github.com/romainsimon/paperasse/tree/f5717490bc2f088f7587c239e33323478ce03918/fiscaliste/evals),
commit `f5717490bc2f088f7587c239e33323478ce03918`, under the
[MIT license](PAPERASSE-LICENSE). The source file hash and per-case amendments are
recorded in `pilot.json`. Cases 1, 3, 4, 6, 11, 12, 13 and 14 cover IR, PER,
dividends, RSUs, CEHR, rental deficits, PEA and life insurance. The other upstream
cases are not included in the pilot.

Dates and missing synthetic assumptions were clarified before measurement. In
particular, the PER ceiling is explicitly available from prior-year income, the
RSU plan is explicitly qualified, and the PEA gains are all earned in the stated
year. These are adaptations, not an unchanged reproduction of Paperasse's published
scores. Typography is normalized to ASCII quotes/dashes. Amounts are public
fictional fixtures, never a private dossier.

The current rubric has 48 criteria. Three optional or out-of-scenario criteria
were removed after the initial engineering pilot: employer matching for the
personal PER question, CEHR smoothing explicitly excluded by the facts, and
pre-2026 gains in a PEA scenario containing only 2026 gains. Their removals are
recorded as amendments. Earlier runs retain their original 51-criterion corpus;
do not regrade them with the revised rubric or silently compare the scores.

To regenerate from the pinned public checkout:

```sh
bin/legidor-eval import-paperasse -repo /path/to/paperasse -out evals
```

The importer records the checked-out commit and source digest. Review changes to
the corpus and criteria before accepting a new upstream revision. Do not reuse an
old result directory with a new corpus or modified skill.
