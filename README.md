# legidor

A Go CLI that gives AI agents reliable access to official French legal sources: Legifrance through PISTE, current BOFiP doctrine and Judilibre decisions. It resolves human references, checks identity and versions, selects subdivisions, and verifies batches of citations without an always-loaded MCP tool schema.

The binary uses only the Go standard library. It does not infer legal applicability or calculate taxes.

## Install

Go 1.23 or later:

```sh
go install github.com/bpineau/legidor/cmd/legidor@latest
```

From a checkout:

```sh
make test
make install
make install-skill
```

Put the binary directory on PATH (the checkout installer uses `~/.local/bin`). `make install-skill` installs the same portable skill under `~/.agents/skills/legidor` and `~/.claude/skills/legidor`. For another agent, install `skills/legidor` in that agent's skills directory. The skill calls the CLI through the shell and has no provider-specific tool dependency.

## Configure PISTE

Create a PISTE application subscribed to Legifrance and Judilibre, accepting each API's terms. Store credentials outside the checkout in `~/.config/legidor/piste.env` (or `$XDG_CONFIG_HOME/legidor/piste.env`):

```text
PISTE_CLIENT_ID=your-client-id
PISTE_CLIENT_SECRET=your-client-secret
```

Use directory mode 0700 and file mode 0600. The file is parsed as data, never sourced as shell code. Environment variables override file values. `PISTE_API_KEY` is recognized for status reporting; OAuth is the supported authentication method. No credential value is printed.

```sh
legidor auth status
```

Production is the default. `--env sandbox` uses separate endpoints, `piste-sandbox.env` and a separate cache. OAuth expiry comes from the server; tokens renew automatically, including one renewal after an HTTP 401. Network timeouts and retry counts are bounded, and long Retry-After delays are returned to the caller without early retries.

BOFiP needs no PISTE credentials.

## Use

```sh
legidor get 'CSS L136-8' --date 2025-12-31 --parts
legidor get 'CSS L136-8 IV' --date 2025-12-31 --json
legidor get 'CGI 150-0 D, 11' --date 2025-12-31
legidor get 'BOI-RFPI-DECLA-10 §60'
legidor get 'LFI 2024 art. 110'
legidor get 'CPCE R322-39..R322-42' --date 2025-12-31
legidor get 'Cass. 3e civ. 2023-10-19 n° 22-15.536' --json
legidor search 'abus de droit' --in cetat --limit 5
legidor versions 'CGI 32' --since 2024-01-01
legidor diff 'CGI 32' --from 2025-01-01 --to 2026-03-01
```

Quote references containing spaces. Options may appear before or after the reference. Every command has standalone `--help` with examples, defaults and common errors.

`--date` means the text version in force on that calendar date. It is not the law applicable to income earned during that year: publication, version validity and applicability are different facts. Read NOTA, modifying provisions and transitional rules separately.

Direct identifiers select their specific version. A date that contradicts that version is an error. Official article URLs preserve their date. Article and code numbers are never silently corrected. Recognized aliases include CGI and annexes I-IV, LPF, CSS, CMF/CoMoFi, civil, construction, insurance, labor, enforcement, consumer, commercial, local-government, public-health and urban-planning codes.

Numbered laws without a date resolve their JORF publication; with a date they resolve consolidated LODA. Rectifications can create several candidates. Use a returned text identifier with `art. NUMBER` to select an article. Named LFI/LFSS references use verified entries where available, otherwise a title search with identity checks. LFR requires a specific law number because several rectifying laws can exist in one year.

## Verify citations

```json
{"ref":"CGI 32, 2, c","date":"2025-12-31","where":"references/example.md:12"}
{"ref":"BOI-RFPI-DECLA-10 §60","where":"references/example.md:18"}
```

Add `quote` for actual wording to verify, then:

```sh
legidor check citations.jsonl --json
```

Output is one JSON line per input citation, in input order. Duplicate source reads are reused. Separate fields report `identity`, `version`, `part`, `quote` and `changed_since`. `applicability` is always `not_evaluated`. Missing evidence is never converted to a successful check. A table of contents cannot verify substantive quotations.

Quote normalization changes only whitespace and typographic quote punctuation. It preserves case, digits and negation, and rejects matches cutting through words or numbers. `[...]` separates fragments that must appear in order. Exact matching is relative to rendered source text; whitespace in upstream HTML is normalized during rendering.

```sh
legidor refs notes.md > citations.jsonl
git diff | legidor refs - --diff > changes.jsonl
```

Extraction happens locally and proposes detected references, not an exhaustive audit. Review and complete the manifest. Diff entries retain added/removed/context labels; `check` checks every supplied entry, so filter removed lines if they are outside the audit scope. Files, quotes and `where` values are not sent to source APIs. Free-text search queries are sent, so avoid including personal facts in them.

## Output and reliability

Text is compact by default; `--json` uses the stable `legidor/1` contract described in [docs/contract.md](docs/contract.md). Source URLs, retrieval time, environment and cache/offline status accompany documents.

| Exit | Meaning |
|---|---|
| 0 | Complete processing, not an overall legal endorsement |
| 1 | Established verification failure |
| 2 | Invalid input or conflicting options |
| 3 | Incomplete or indeterminate result |

Batch precedence is 3, then 2, then 1. Diagnostics go to stderr. JSON stdout contains results or structured errors only.

The supported structural subset includes Roman numerals, uppercase letters, numbered clauses, degree clauses, lowercase letters and common suffixes. Ancestor introductory text is retained. `--parts` shows recognized labels; ambiguous labels fail. `alinéa N` is a rendered paragraph position, not an authoritative legal numbering. Complex HTML, tables, images and embedded objects are reported as unsupported; no flattened approximation is represented as complete source text.

Whole laws return their table of contents. Text output normally stops at 20000 characters with `complete=false`, omitted size and a continuation cursor that checks content identity. `--full` lifts this output cap. Requested subdivisions are never truncated. Collections use official order, include intercalated articles and are bounded at 200 articles.

## Cache

Private token and response caches live under `~/.cache/legidor/ENVIRONMENT` or `$XDG_CACHE_HOME/legidor/ENVIRONMENT`, with atomic writes. Validated source JSON is cached; normalization and identity checks run again on reads. Errors and HTML error pages are not stored as legal content.

Current consultations, resolution, search and BOFiP use a 30-day cache. Dated historical consultations and searches, direct article versions and decision consultations use 90 days. These are reuse budgets, not guarantees that the source has remained unchanged. A cached article's end date, state and links can change upstream even if its wording does not.

`--max-age 24h` overrides the response cache age for a command (Go duration syntax: h, m, s). Older entries are fetched again. `--fresh` bypasses all response caches, even entries younger than 24 hours; it cannot be combined with `--max-age` or `--offline`. OAuth expiry always follows the server. `--offline` requires entries within the selected age budget and never falls back to expired entries.

Named code articles without an explicit date reuse their current-version resolution across calendar days. `date` remains the requested day; `provenance.resolution_date` and `resolution_retrieved_at` preserve the original search date and retrieval time. A known version end forces a new resolution. Explicit dates, general searches and tables of contents retain separate dated keys. Cached evidence must not be described as checked against the source today.

```sh
legidor cache stats --json
legidor get 'CIV 1240' --max-age 24h
legidor get 'CIV 1240' --fresh
legidor cache clear
```

To warm a bounded topic collection, keep a local JSONL manifest of references and optional dates, then run `legidor check topics.jsonl --json > results.jsonl`. Inspect per-row errors and completeness, then verify with `legidor check topics.jsonl --offline --json`. Repeating the first command fetches only missing or expired entries; add `--max-age 24h` or `--fresh` when needed. Keep personal manifests and reports outside public repositories. Warming caches source responses, not generated legal conclusions, and does not make unsupported content complete.

## Coverage and limits

- Legifrance: code articles and sections, ranges, LODA/JORF, targeted KALI, CETAT, JURI and CONSTIT. NOR and ELI URLs containing a NOR or numbered text are supported. Ambiguous publications return candidate identifiers.
- BOFiP: current doctrine only, refreshed weekly by the publisher. It is administrative doctrine, potentially opposable under conditions, not legislation. Historical queries fail explicitly.
- Judilibre: decision lookup and source zones; search defaults to Cour de cassation. Direct identifiers can consult other covered jurisdictions. Partial decisions retain `complete=false`.
- Out of scope: historical BOFiP, PDF-only material, unsupported complex tables/annexes, tax forms/notices, BOSS, URSSAF, Unedic, HCSF, bulk mirrors, evidence bundles and an MCP server.

No anti-bot bypass is attempted. A result absent from a corpus does not establish that the rule or decision does not exist.

## Development and validation

```sh
make test
make vet
make build
LEGIDOR_LIVE=1 go test ./internal/acceptance -run TestLive -v
```

Offline tests use invented responses and local HTTP servers. Live checks are opt-in, use the user's private credentials and never write them into fixtures. See [docs/validation.md](docs/validation.md) for scenarios and known limitations. The [agent skill](skills/legidor/SKILL.md) provides the legal-research workflow, four core recipes and troubleshooting.

Official API references: [Legifrance FAQ](https://www.legifrance.gouv.fr/contenu/pied-de-page/foire-aux-questions-api), [Judilibre documentation](https://github.com/Cour-de-cassation/judilibre-search), [BOFiP current-doctrine dataset](https://data.economie.gouv.fr/explore/dataset/bofip-vigueur/).
