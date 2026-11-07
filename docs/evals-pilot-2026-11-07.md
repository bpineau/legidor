# Evaluation framework readiness: 2026-11-07

The Go runner is ready for a controlled comparison. This engineering pilot
validated the execution path and exposed useful Legidor defects. It does not
establish a ranking or prove a token saving: the four-model campaign was stopped
after Claude exhausted its quota and the operator limited further work to Sol.

## Coverage and interruption

The planned matrix contained eight synthetic cases, four resource configurations
and four models, with one conversation per cell: 128 samples. Configurations were
no skill, Paperasse fiscaliste, Legidor, and both. All had native web access and
the same Go arithmetic helper. Effort was medium, caches were cold per sample,
and at most four conversations ran concurrently.

| Requested model | Native CLI | Successful answers | Quota failures | Interrupted |
|---|---|---:|---:|---:|
| `claude-opus-5-5` | Claude Code 2.1.319 | 15 | 16 | 0 |
| `claude-sonnet-5-5` | Claude Code 2.1.319 | 9 | 19 | 0 |
| `gpt-6.1-sol` | Codex 0.167.0 | 25 | 0 | 2 |
| `gpt-6-astra` | Codex 0.167.0 | 21 | 0 | 2 |

There were 70 successful answers, 39 failed/interrupted attempts and 19 unstarted
cells. "Successful" means the native conversation completed, not that its tax
answer was correct. These answers were not all graded. In particular, quota
failures must not be interpreted as fiscal mistakes or as cheap successful runs.
Unequal coverage makes aggregate resource totals unsuitable for ranking arms.

Model IDs are recorded in directories, results and manifests. Claude supplies its
effective model in initialization events. Where Codex does not expose an effective
model, the result uses the requested identifier; this is not an independently
verified backend revision. CLI versions and binary/resource hashes are retained.
Earlier capability probes and abandoned calibration runs are separate from this
matrix and excluded from its counts.

## Problems found and corrected

- Older numbered texts such as `ordonnance 96-50 art. 19` were rejected because
  the parser required a four-digit year. Two-digit law years now work without
  guessing their century.
- Natural law citations containing their enactment date were rejected. They now
  parse, and the date is checked against official metadata or the official title.
  Enactment date remains distinct from the requested article-version date.
- Agents requested consecutive BOFiP paragraphs with `90..120`. This now returns
  the bounded source passage with context and intervening headings. Missing,
  reversed or ambiguous bounds still fail explicitly.
- Text identifiers followed by `art.`, `art` or `article` now accept those marker
  variants. Legal subdivision labels remain exact rather than guessed.
- `LEGIDOR_CACHE_DIR` now isolates response and OAuth caches without redirecting
  credential lookup. The broker overrides inherited cache settings per sample.

The numbered-text and BOFiP changes were checked against the live APIs and covered
by regression tests. Deployed CLI and skills were updated. Remaining limitations
include BOFiP news identifiers (`ACTU-...`) and exact legal subdivision labels:
agents should inspect `--parts` instead of inventing a selector or an option.

## Harness checks and quota controls

Capability probes caught ambient Codex instructions and missing MCP approvals;
the runner now uses an isolated configuration directory and explicit tool access.
Skill resources exclude scripts, evaluation answers, hidden files and external
symlinks. Expected answers never enter the evaluated conversation. Authentication
stays outside the repository. This is controlled tool exposure, not an adversarial
operating-system sandbox.

The runner retains partial usage and failed attempts, rejects incompatible
resumes, randomizes blinded grading order and keeps grading costs separate.
Missing usage is marked as incomplete. Reports include failed and ungraded
attempts in coverage denominators. All model and tool traces remain private;
provider token accounting and estimated prices are not subscription invoices.

After the quota interruption, the default was changed to stop scheduling after
the first failed sample. Already running samples finish. `-max-samples` can reject
an oversized matrix before any agent starts. Development instructions now begin
with two Sol samples, one worker, and a Sol judge. A full campaign remains an
explicit, larger operation. The judge can be selected independently of the tested
models, including when one provider has no quota available.

## Corpus readiness and next comparison

A final bounded validation used only `gpt-6.1-sol`: two Legidor-arm conversations
(PER and PEA), followed by two Sol grading calls. All four native calls completed.
The Go report recorded 8/12 passes against that run's frozen original rubric,
with evidence and temporal scores of 2/2 for both answers. This validates the
execution/grading/reporting pipeline, not comparative efficacy. Self-grading is
another limitation.

Manual inspection confirmed that the PEA answer omitted the rubric's advice
about waiting until five years and its detailed treatment of an exception; the
PER answer's only failed criterion requested unrelated employer-matching advice.
The latter criterion had already been removed from the next corpus revision.
The PEA run successfully used the repaired older-ordinance reference. Repeated
incorrect subdivision selectors also led to a short instruction in the deployed
skill to inspect `--parts` and copy exact labels.

Answering used 499,877 cumulative input tokens (422,528 reported cached reads),
5,467 output tokens and 170.7 seconds. Grading separately used 23,148 input tokens,
870 output tokens and 24.9 seconds. Codex did not provide a dollar cost. These
figures illustrate why even a two-case validation needs a deliberate quota budget;
they are not a token-saving claim.

The eight public Paperasse scenarios cover IR, PER, dividends, qualified RSUs,
CEHR, rental deficits, PEA and life insurance. Dates and missing assumptions were
made explicit; provenance and amendments are recorded in the corpus. The current
rubric has 48 criteria. The initial matrix used 51. Three criteria were removed
because they requested optional advice or discussion of facts expressly excluded
by the synthetic question. Earlier run directories retain their original rubric;
their scores must not be silently compared with the revised rubric.

The next four-model comparison should use a new output directory, the current
binary/skill/corpus and a fixed judge, with quota available for both answering and
grading. Start with a small balanced subset before expanding to 128 samples.
Review material grading disagreements manually, then repeat affected cases or use
held-out cases. A single repetition of eight scenarios is diagnostic, not a
statistical estimate of general model performance. Evidence grades measure what
the answer demonstrates, not independent verification of every citation.

See [the evaluation guide](../evals/README.md) for reproducible commands and
measurement definitions.
