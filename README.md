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

PISTE setup has three separate parts: a user account, consent to each API's terms, and an application authorized to use those APIs. Creating an account or obtaining credentials alone does not enable API access. BOFiP needs no PISTE credentials.

### 1. Create and activate your account

Open [piste.gouv.fr](https://piste.gouv.fr/) and choose the account creation option. Complete the [registration form](https://piste.gouv.fr/component/apiportal/registration), accept the PISTE platform terms (CGU, "Conditions générales d'utilisation"), then activate the account through the email link and sign in. The organization code field is optional; an individual user can leave it empty.

The steps below use French menu labels. Labels may vary with the portal language or interface version.

### 2. Accept the terms for both APIs

Open **API > Consentement CGU API**. Find **Légifrance** and **Judilibre**, read their respective terms and record your consent for production. If you also want sandbox access, accept the terms for those entries too. Platform registration consent does not replace these API-specific consents. See the official [Legifrance access FAQ](https://www.legifrance.gouv.fr/contenu/pied-de-page/foire-aux-questions-api) and [Judilibre onboarding guide](https://www.courdecassation.fr/acces-rapide-judilibre/donnees-ouvertes-open-data-et-api).

### 3. Create the production application and enable the APIs

Open **Applications > Créer une application** and create a production application, using `legidor` or another available name. PISTE automatically creates a sandbox application at registration; create the production application separately for legidor's default environment.

In the application's API selection, enable **both Légifrance and Judilibre**, then save. For an existing application, open **Modifier l'application**, select the APIs near the bottom of the form, and apply the changes. Reopen the application and verify that both APIs appear as selected/subscribed. Accepting terms and attaching APIs to the application are separate actions. The [PISTE application guide](https://portail.chorus-pro.gouv.fr/aife_documentation?id=kb_article_view&sysparm_article=KB0012883) illustrates these controls; its PPF-specific organization instructions do not apply to legidor.

### 4. Copy the application's OAuth credentials

Open the application's authentication/OAuth credentials section and obtain its **Client ID** and **Client Secret**. These belong to the application, not your PISTE login. If creating an OAuth credential explicitly, use a confidential client: legidor authenticates with `client_credentials` and a client secret, without a browser callback.

Use the pair from the **production** application. Legidor obtains and renews access tokens automatically; no manually generated bearer token is needed. A KeyId/API key shown elsewhere in PISTE is not a substitute for this OAuth pair.

### 5. Save credentials locally

Create the private configuration file outside the checkout:

```sh
legidor_config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/legidor"
mkdir -p "$legidor_config_dir"
chmod 700 "$legidor_config_dir"
touch "$legidor_config_dir/piste.env"
chmod 600 "$legidor_config_dir/piste.env"
```

Open that file in your editor and replace the placeholders below with the application's credentials. Keep them out of shell history, screenshots, issues and commits.

```text
PISTE_CLIENT_ID=your-client-id
PISTE_CLIENT_SECRET=your-client-secret
```

Use directory mode 0700 and file mode 0600. The file is parsed as data, never sourced as shell code. Environment variables override file values. `PISTE_API_KEY` is recognized for status reporting; OAuth is the supported authentication method. No credential value is printed.

### 6. Verify authentication and access to each API

```sh
legidor auth status
legidor get 'CIV 1240' --fresh
legidor search 'bail' --in judilibre --limit 1 --fresh
```

`auth status` checks OAuth authentication only. The next two commands exercise Legifrance and Judilibre independently; `--fresh` ensures an existing response cache cannot hide a missing API permission.

If authentication succeeds but an API request fails with access denied, return to PISTE and check that its terms were accepted and that the API was enabled on the same application whose credentials you copied. If authentication itself fails, check the credential pair and environment. Also check for exported `PISTE_CLIENT_ID` or `PISTE_CLIENT_SECRET` variables overriding the file, without printing their values.

For sandbox use, repeat the consent and API-selection steps on the sandbox application, store its own OAuth pair in `piste-sandbox.env` with mode 0600, and pass `--env sandbox` to every command. Production and sandbox have distinct credentials, endpoints and caches; a sandbox token cannot authenticate to production.

OAuth expiry comes from the server; tokens renew automatically, including one renewal after an HTTP 401. Network timeouts and retry counts are bounded, and long Retry-After delays are returned to the caller without early retries.

## Use

```sh
legidor get 'CSS L136-8' --date 2025-12-31 --parts
legidor get 'CSS L136-8 IV' --date 2025-12-31 --json
legidor get 'CGI 150-0 D, 11' --date 2025-12-31
legidor get 'BOI-RFPI-DECLA-10 §60'
legidor get 'CGI 219' --part 'I b'
legidor get 'CGI 261' --links --json
legidor get 'LFI 2024 art. 110'
legidor get 'CPCE R322-39..R322-42' --date 2025-12-31
legidor get 'Cass. 3e civ. 2023-10-19 n° 22-15.536' --json
legidor search 'abus de droit' --in cetat --limit 5
legidor versions 'CGI 32' --since 2024-01-01
legidor diff 'CGI 32' --from 2025-01-01 --to 2026-03-01
```

Quote references containing spaces. Options may appear before or after the reference. Every command has standalone `--help` with examples, defaults and common errors.

`--date` means the text version in force on that calendar date. It is not the law applicable to income earned during that year: publication, version validity and applicability are different facts. Read NOTA, modifying provisions and transitional rules separately.

Direct identifiers select their specific version. A date that contradicts that version is an error. Official article URLs preserve their date. Article and code numbers are never silently corrected. Recognized aliases include CGI and annexes I-IV, CIBS, LPF, CSS, CMF/CoMoFi, civil, construction, insurance, labor, enforcement, consumer, commercial, local-government, public-health and urban-planning codes.

Numbered laws without a date resolve their JORF publication; with a date they resolve consolidated LODA. Rectifications can create several candidates. Use a returned text identifier with `art. NUMBER` to select an article. Named LFI/LFSS references use verified entries where available, otherwise a title search with identity checks. LFR requires a specific law number because several rectifying laws can exist in one year.

Named-law searches exclude implementing texts that merely mention the law; original publications and rectifications remain distinct candidates. `get --links` provides a paginated index of explicit links in the source HTML to supported official sources, without fetching or validating the targets. It cannot be combined with text selection.

The [agent skill](skills/legidor/SKILL.md) covers dated citations and source limitations. Its [audit workflow](skills/legidor/references/audits.md) connects assertions, primary sources, assumptions and the calculations that consume them, including complements such as official tax notices and social-security instructions.

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

Explicit plural article lists and BOFiP paragraph lists are expanded. Ambiguous numeric lists produce a diagnostic and exit 3 while other citations are still emitted. Resolve them with explicit plural wording or the manifest's `part` field. Batch output retains NOTA, modifier links and warnings in both JSON and text modes.

## Output and reliability

Text is compact by default; `--json` uses the stable `legidor/1` contract described in [docs/contract.md](docs/contract.md). Source URLs, retrieval time, environment and cache/offline status accompany documents.

| Exit | Meaning |
|---|---|
| 0 | Complete processing, not an overall legal endorsement |
| 1 | Established verification failure |
| 2 | Invalid input or conflicting options |
| 3 | Incomplete or indeterminate result |

Batch precedence is 3, then 2, then 1. Diagnostics go to stderr. JSON stdout contains results or structured errors only.

The supported structural subset includes Roman numerals, uppercase letters, numbered clauses, compound degree clauses, dotted or parenthesized lowercase letters and common suffixes. Ancestor introductory text is retained. `--parts` shows recognized labels; ambiguous labels fail. `alinéa N` is a rendered paragraph position, not an authoritative legal numbering.

Rectangular HTML tables preserve captions, headers, rows, cells and formulas. Full-width notes after regular rows are labeled `Table-wide:`. Exponents and indices use `^(...)` and `_(...)` notation. Other merged/nested tables, images and unsupported embedded content remain explicit omission markers. A selection can be complete when omissions occur elsewhere: `source_complete`, when present, preserves the original document's status. Omissions in retained introductions still block verification. Completeness describes rendering, not legal applicability or the absence of relevant exceptions in other provisions.

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
- BOFiP: current doctrine only, refreshed weekly by the publisher. Individual BOIs may still quote superseded rates or statutes; check their publication dates against the relevant legislation. It is administrative doctrine, potentially opposable under conditions, not legislation. Historical queries fail explicitly.
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
