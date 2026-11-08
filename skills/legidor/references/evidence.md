# Evidence and output

## Dates and sources

`--date` means the version in force on that calendar date. A law enacted later can govern earlier income. Distinguish publication, version validity and applicability to the operation. Read NOTA, the original modifier and transitional provisions before concluding.

Code articles default to today's version. A fixed identifier selects that version; inspect its interval. Conflicting explicit dates fail. `NO_VERSION_AT_DATE` can identify a published future or past version without substituting it for the requested one. Use `versions` to inspect history. Search candidates are not confirmed identities.

An implicit current article can reuse an earlier resolution. Read `provenance.resolution_date` and `resolution_retrieved_at` alongside `retrieved_at`: a new requested date or article fetch does not prove that current-version selection was refreshed. A known version end triggers a new resolution.

BOFiP is administrative doctrine, potentially opposable under conditions, not legislation. The current-doctrine dataset is updated weekly. A current BOI, even with an old start date, cannot establish historical doctrine. `--fresh` refreshes the API response, not the dataset's publication schedule. Current corpus membership does not guarantee that every statutory reference or rate is up to date. Compare a conflicting passage with dated legislation and recent official instructions; report the discrepancy.

Decisions are case law. Read reasons and the operative part, check jurisdiction/date/number, and inspect completeness. Describe partial or pseudonymized text accordingly. Judilibre zones are source-supplied; do not invent numbered subdivisions in other decisions.

## Citation manifests and checks

Use one explicit citation per JSONL line:

```json
{"ref":"CGI 32","date":"2025-12-31","part":"2 c","where":"references/example.md:12"}
{"ref":"BOI-RFPI-DECLA-10 §60","where":"references/example.md:18"}
```

Add `quote` only for wording actually being checked. Review all `refs` output, stderr diagnostics, dates and subdivisions. With `refs --diff`, filter added/removed/context entries as appropriate; removed lines are not automatically excluded. Quotes and local documents are not transmitted to the APIs; keep personal facts out of free-text queries.

| Status | Meaning |
|---|---|
| `identity=confirmed` | Code, article or decision identity is established, not applicability. |
| `version=in_force_at_date` | The interval contains the requested date, not necessarily the relevant income year. |
| `part=found` | The subdivision was selected, including recognized introductory context. |
| `quote=exact` or `matches_after_normalization` | Wording matches; the surrounding claim still requires analysis. |
| `changed_since` | A later version starts then; the quoted passage may be unchanged. |
| `applicability=not_evaluated` | Always separate from retrieval and literal verification. |

`not_requested` is not a positive check. `not_evaluated`, ambiguity and an `error` leave the relevant claim unestablished. Read the independent statuses rather than reducing them to a single success flag. A Roman-suffix retry records a warning; confirm that the corrected subdivision corresponds to the author's intended citation.

Quote normalization changes whitespace and quotation punctuation only, never case, numbers or negation. `[...]` checks fragments in order, not omitted reasoning. A grep match is an excerpt, not a complete provision. `alinéa N` selects a rendered paragraph position, not an authoritative legal paragraph number.

## Completeness and output

Schema is `legidor/1`; `check --json` emits JSONL. Exit 0 means complete processing, 1 an established verification failure, 2 invalid input, and 3 an incomplete or indeterminate result. Read stdout even on nonzero exit.

For text pagination, follow the returned continuation or select a subdivision. Cursors pin content; do not edit them. Requested subdivisions are returned whole, with relevant introductory context. BOFiP selections retain section headings and introductions, omit the next section's heading and avoid repeating editorial news. Full text retains the news notice.

`get --parts` is a navigation outline. `preview` contains bounded source wording, not a summary or verified complete rule; `selectable=false` marks a duplicate label. All labels remain visible when the default preview budget runs out; `omitted_part_previews` counts those without previews, and `--full` restores them. No recognized labels means no usable subdivision was established. Source omissions and warnings still require inspection.

On a selection, `complete` describes the returned text and context; `source_complete`, when present, describes the original document. An omission in retained context still blocks verification. This describes rendering, not legal applicability or the absence of exceptions elsewhere.

Modifier lists in `get`, `resolve` and `check` show up to five recent entries by default. `omitted_modifications` counts hidden entries; repeat with `--full` for all links. The text-completeness flag does not certify metadata exhaustiveness. NOTA is never removed by this compaction. Read temporal caveats even after a successful check.

`versions` is a compact history index in text mode. Use `--json` for structured NOTA, modifier links and provenance, and `--full` for all modifier links in either format. `--since` retains intervals overlapping or following that date. Empty results are an empty list, not `null`.

Rectangular tables preserve cells, full-width notes and formulas; exponents and indices use `^(...)` and `_(...)`. Complex merged/nested tables, images and unsupported formats remain explicit omissions. Source wording and malformed source-cell boundaries are not repaired. Verify formatted quotations or suspicious formulas against the official publication.

Coverage excludes historical BOFiP, forms/notices, BOSS, URSSAF, Unedic, HCSF and PDF-only materials. Use the complements listed in [audits.md](audits.md), state what came from another tool, and retain unresolved questions when evidence is missing.
