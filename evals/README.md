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

The reviewed full suite is `full.json`: all 50 upstream IDs are retained, with
270 criteria after the documented amendments. It complements the eight-case pilot.
Dates, ambiguous facts and tool-specific requirements have been reviewed before
measurement. In particular, successful historical research must not fail merely
because Paperasse lacks that year's local data, and naming another Paperasse skill
is not required. Every case records its review date, amendments and category.
The two orientation questions remain in the suite, but are identified separately
from tax calculations; historical research has its own category too.

To compare the three individual configurations with Sonnet and then grade them:

```sh
make build build-eval
bin/legidor-eval campaign -corpus evals/full.json \
  -agents claude -claude-model claude-sonnet-5-5 \
  -arms none,paperasse,legidor \
  -max-samples 150 -workers 1 -repetitions 1 \
  -paperasse /path/to/paperasse/fiscaliste -legidor "$PWD/bin/legidor" \
  -judge-agent claude -judge-model claude-sonnet-5-5 \
  -out /tmp/legidor-full-comparison
```

This deliberately starts 150 evaluated conversations plus one blinded grading
conversation per case. It is not a smoke test. `campaign` requires an explicit
judge model and otherwise uses the same flags and isolation as `run`. It records
its phase in private `campaign.json`, stops before grading if execution fails,
and never retries failures automatically. A process lock rejects duplicate
`campaign` launches into the same directory. Do not run separate `run` or `grade`
commands against an active campaign. Successful samples and grades are reused
on an identical resume. A failed sample remains failed: investigate it and use a
separate run for retries. Keep the executable and all input snapshots fixed while
a campaign is running. A Sonnet judge is economical but can introduce family bias;
inspect close or surprising verdicts before drawing conclusions.

For development, start with two Sonnet samples and an explicit size limit:

```sh
make build build-eval
bin/legidor-eval run -agents claude -claude-model claude-sonnet-5-5 \
  -cases 3,13 -arms legidor -max-samples 2 -workers 1 \
  -legidor "$PWD/bin/legidor" -out /tmp/legidor-sonnet-smoke
bin/legidor-eval grade -out /tmp/legidor-sonnet-smoke \
  -agent claude -model claude-sonnet-5-5
```

The default stops scheduling new samples after the first failure, including quota
errors. Already running samples finish and retain their measurements. Opt into
`-continue-on-error` only deliberately. `-max-samples N` rejects a larger matrix
before launching any agents; zero (the default) leaves the size unrestricted.
It counts conversations, not provider tokens or money. Grading makes separate
model calls, one per case with successful answers.

For the pilot comparison (24 samples with Sonnet and the three default arms):

```sh
make build
make build-eval
bin/legidor-eval run \
  -paperasse /path/to/paperasse/fiscaliste \
  -legidor "$PWD/bin/legidor" \
  -max-samples 24 \
  -out /tmp/legidor-pilot
bin/legidor-eval grade -out /tmp/legidor-pilot
bin/legidor-eval report -out /tmp/legidor-pilot
```

Authenticate the selected native agent CLI beforehand. Legidor uses its usual private
PISTE configuration. No credentials are copied into the repository. The runner
supports macOS and Linux. Model runs consume the configured account's quota.
There are no paid API calls in `go test ./...`.

Defaults: Claude only with `claude-sonnet-5-5`, medium effort, eight cases,
three arms (`none,paperasse,legidor`),
one repetition, a shuffled sequential schedule (seed 42), and eight minutes per
sample. Model identifiers and CLI flags depend on the installed agent versions;
use `-claude-model` and `-codex-model` to test another agent version explicitly.
Codex is opt-in with `-agents codex` (default Codex model: `gpt-6.1-sol`).
Opus and Astra are disabled even when named explicitly: `run`, `campaign` and
`grade` reject them before model calls unless `-allow-expensive-models` is supplied.
The guard includes plural model flags and the campaign judge. The standalone
`grade` command also defaults to Sonnet; `campaign` still requires an explicit
judge model. Old frozen runner binaries retain their original defaults.

```sh
# One case across the three default configurations.
bin/legidor-eval run -cases 11 -paperasse /path/to/paperasse/fiscaliste \
  -legidor "$PWD/bin/legidor" -out /tmp/legidor-smoke

# Repeated measurements, each a new conversation, in a new run directory.
bin/legidor-eval run -cases 3,13 -repetitions 3 -cache cold \
  -paperasse /path/to/paperasse/fiscaliste \
  -legidor "$PWD/bin/legidor" -out /tmp/legidor-repeated
```

`-agents` and `-arms` select the matrix. The combined `both` arm remains available
only when requested, for example `-arms none,legidor,both`.
`-workers 2` runs at most two agent conversations concurrently; the default is
one. Cold/fresh cache modes keep a separate cache for every sample. The manifest
records the worker count because concurrency can affect latency and quotas.

Model IDs appear in sample directories, individual results and aggregate tables;
tables also include CLI versions. To explicitly include Sol alongside Sonnet:

```sh
bin/legidor-eval run -agents claude,codex \
  -claude-model claude-sonnet-5-5 -codex-model gpt-6.1-sol \
  -paperasse /path/to/paperasse/fiscaliste -legidor "$PWD/bin/legidor" \
  -max-samples 48 -workers 1 -out /tmp/legidor-sonnet-sol
```

The plural model flags override their singular counterparts. Each requested model
gets every selected case, arm and repetition; it is also recorded separately from
the effective model reported by the agent (when available).

`-corpus` accepts the interoperable scenario format described below, or legacy
Legidor `cases` files. The runner prints selected/available cases and planned
sample counts before starting. Unknown or duplicate `-cases` IDs are errors.
Repeating an identical command resumes recorded samples with their original
metrics. It does not count reused answers as fresh zero-cost measurements. A
changed manifest requires a new output directory. Failed samples remain visible;
use a separate run to retest them rather than silently replacing them.

## Scenario interchange

`pilot.json` uses the [Anthropic skill-creator scenario format](https://github.com/anthropics/skills/blob/2c5958ecac7cee2b583967a2c4981d0f4a75c579/skills/skill-creator/references/schemas.md):
`skill_name`, `evals`, and each scenario's `id`, `prompt`, `expected_output` and
`files`. It also retains human-readable names. Both `assertions` and
`expectations` are written identically because upstream SKILL.md and schemas.md
use different names. Either spelling is accepted; conflicting copies are rejected.

Legidor adds `x_legidor` for the research date, source commit/hash and amendments.
Unknown extension fields survive conversion. Expected output and criteria are
available to the judge only, never the evaluated agent. Dates must be explicit;
converting a file does not validate its tax assumptions or update its answers.

```sh
# Convert an upstream scenario file, without running agents.
bin/legidor-eval convert -corpus /path/to/skill/evals/evals.json \
  -as-of 2026-11-07 -out /tmp/converted-evals.json
bin/legidor-eval list -corpus /tmp/converted-evals.json

# Convert a legacy Legidor corpus; its embedded date is retained.
bin/legidor-eval convert -corpus /path/to/old-corpus.json \
  -out /tmp/converted-legacy.json
```

`files` references and draft scenarios without criteria are preserved during
conversion. This research runner currently requires nonempty criteria and rejects
selected scenarios with attachments before calling a model. It cannot yet serve
arbitrary PDF/image inputs. The imported Paperasse fiscal corpus has no attachments.
Missing scenario names get a stable `case-ID` fallback; IDs must be nonnegative
and unique. Run snapshots retain their original bytes and hashes: do not replace
an old run's `corpus.json` with a converted file.

## Result interchange

Export a completed comparison to Anthropic-compatible `benchmark.json`, per-run
`grading.json` and `timing.json`, plus the scenario file. This is a Go data export,
not a dependency on Anthropic's Python runner or a complete viewer workspace.

```sh
bin/legidor-eval export-anthropic -run /tmp/legidor-pilot \
  -agent codex -model gpt-6.1-sol -with legidor -without none \
  -out /tmp/legidor-vs-none-export

# Measures the incremental value of Legidor alongside Paperasse.
bin/legidor-eval export-anthropic -run /tmp/legidor-pilot \
  -agent codex -model gpt-6.1-sol -with both -without paperasse \
  -out /tmp/legidor-plus-paperasse-export
```

The exact recorded model and two distinct arms are mandatory. The export maps
them explicitly to `with_skill` and `without_skill`; it never pools models or
silently drops cells. Every planned case/repetition in that pair must have a
successful answer, complete usage and complete grading. The destination must be
new. Old derived manifests without a planned matrix cannot establish completeness.

Native manifests, model/CLI versions, cache counts, cost estimates, and evidence
and temporal scores remain in `x_legidor`. Unknown error counts, per-tool counts
and verified-claim results are omitted, not fabricated. Token totals include
cache reads; timing covers execution, with grading usage kept in the native
report. Summary statistics use sample standard deviation across selected cases
and repetitions; they are not confidence intervals.

Use the native report for failed, interrupted or partly graded campaigns. It
includes planned and unstarted counts when the original matrix is available;
otherwise planned coverage is explicitly unknown. All exports remain private
artifacts and must be reviewed before sharing.

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
Select another judge explicitly with `-agent codex -model gpt-6.1-sol`, for
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

The pinned upstream fiscal suite contains 50 scenarios and 275 assertions. The
eight-case pilot is a bounded diagnostic subset chosen to cover several personal
tax topics while controlling development costs, not a claim of exhaustive fiscal
coverage. Running only PER, RSU and PEA further narrows that coverage. Other
upstream topics include household taxation, LMNP, crypto, IFI, BSPCE, declaration
workflows, withholding, tax credits, foreign accounts and departure from France.
They can serve as additional tests or holdouts after reviewing their assumptions.

Import every upstream case unchanged, including its expected output and criteria:

```sh
bin/legidor-eval import-paperasse -repo /path/to/paperasse -all \
  -as-of 2026-11-07 -out /tmp/paperasse-fiscal-evals
bin/legidor-eval list -corpus /tmp/paperasse-fiscal-evals/all.json
```

This copies the source license and records the commit and source hash. It does
not apply pilot amendments and does not run models. Select a bounded subset with
`run -corpus /tmp/paperasse-fiscal-evals/all.json -cases ... -max-samples ...`.
Using all 50 cases with the three default arms means 150 conversations per
model/repetition, plus grading. Adding `both` raises this to 200; do not launch
larger matrices accidentally. Paperasse's executable calculators
remain unavailable in every configuration: this tests its documentation/data,
not the full executable Paperasse product.

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
