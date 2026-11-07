# Advanced recipes

## Resolve and search

```sh
legidor resolve 'CGI 150-0 D, 11' --parse-only --json
legidor resolve 'CGI 150-0 D' --date 2025-12-31 --json
legidor search 'abus de droit' --in cetat --limit 5 --json
legidor search 'revenus fonciers' --in bofip --limit 5 --json
```

Parse-only validates grammar, not existence. Search returns identifiers and bounded excerpts. Re-read the chosen identifier with `get`, checking identity independently. Never select a result solely because it ranks first. Pagination is not a stable database snapshot; repeat the query, corpus, date and page size with the returned cursor.

Aliases include CGI and annexes, CIBS, LPF, CSS, CMF/CoMoFi, C. civ., CCH, C. assur., C. trav., CPCE, C. conso., C. com., CGCT, CSP and C. urb. A suffix such as the I in CGI 990 I belongs to the article number. Use commas to separate uncertain subdivisions and inspect `resolve --parse-only`.

Mixed suffixes are part of the article number: `CGI 1649 bis C` and `CGI 199 decies EA`. To select a subdivision of the shorter article instead, use a comma (`CGI 1649 bis, C`) or `--part C`. `refs` recognizes explicit code names before or after the number, including `art. R* 196-1, a LPF`; it does not infer the code of a nearby bare citation.

Compound subdivisions such as `2°-0 ter` and dotted lowercase labels such as `I b` are supported. `refs` expands explicit `articles ...` lists and BOFiP paragraph lists joined by commas or `et`. Ambiguous singular numeric lists are reported on stderr with exit 3; use explicit plural wording or separate manifest entries with `part` rather than silently treating an article number as a subdivision.

## Laws and official URLs

```sh
legidor get 'LFI 2024 art. 110' --json
legidor resolve 'LFI 2024' --json
```

A numbered law without `--date` searches its published JORF version. Rectifications can create several candidates: use the returned identifiers and titles to identify the intended publication. A dated law requests the consolidated LODA version, which can differ from the original publication. A fixed JORF identifier does not support consolidated date selection.

Named-law searches filter by the law's principal title, excluding implementing orders and texts that merely mention it. Original publications and rectifications remain distinct candidates. An incomplete search never establishes uniqueness.

Whole texts return an article table of contents. Read an article by its returned identifier, or append `art. NUMBER` to a text identifier. A table of contents cannot verify a substantive quotation. Official URLs retain their date and nature; mismatches with explicit options are rejected.

## Ranges and sections

```sh
legidor get 'CPCE R322-39..R322-42' --date 2025-12-31 --json
```

The source's table of contents supplies order, including intercalated articles. Missing/reversed bounds fail explicitly. Collections are bounded at 200 articles. Sections use `CODE section SECTION_ID`, where SECTION_ID must come from an official result, never memory.

## Decisions

```sh
legidor get 'Cass. 3e civ. 2023-10-19 n° 22-15.536' --json
legidor search 'responsabilité' --in cetat --limit 5 --json
legidor search '22-15.536' --in judilibre --json
```

For tax litigation, search CETAT first. Legifrance also supports JURI and CONSTIT. Judilibre search defaults to the Cour de cassation; direct returned `JUDI:` identifiers can consult other courts. Decision dates belong in the reference, not `--date`.

Judilibre provides source zones such as `motivations` and `dispositif` when available. Use `get REF --parts`, then `--part dispositif`. Lack of a zone does not license inferring its boundaries. Check `complete`, the court, chamber, number and date.

## Literal verification, cache and uncertainty

```sh
legidor get 'CGI 32' --date 2025-12-31 --grep 'sur option' --json
legidor get 'CSS L136-8 IV' --date 2025-12-31 --offline --json
legidor cache stats --json
```

Grep is case-sensitive literal matching; a missing match returns exit 1. A match is not a legal conclusion. Quote checks preserve digits and negation; inspect surrounding context before relying on an extracted fragment.

Cached output retains its original retrieval time. Offline mode accepts only unexpired entries and does not contact sources. Missing/expired entries fail; they never silently become evidence. Sandbox data is marked and cached separately, and should not support a production legal assertion.

The default age budget is 30 days for current sources and 90 days for historical consultations, direct article versions and decisions. `--max-age 24h` overrides that budget for all response reads, including article resolution; `--max-age 1h` is stricter. `--fresh` bypasses response caches entirely and cannot be combined with `--max-age` or `--offline`. `--offline --max-age 24h` accepts only recent local evidence. OAuth token expiry is independent.

Warm a small topic collection with `legidor check topics.jsonl --json > results.jsonl`, then verify availability with `legidor check topics.jsonl --offline --json`. Inspect every row, including incomplete content. Re-running respects cache age; it does not fetch whole codes. Keep personal manifests and reports outside public repositories and out of agent context unless their content is needed. Cached source text is not a cached legal conclusion.

## Modifier placeholders

A consolidated modifying article can contain only "A modifie..." substitution placeholders. That is not the original amending wording. Use its returned JORF parent, then read the same article number in the original publication. Check exactly which amendment a transitional provision targets; do not transfer a rule about one numbered amendment to another.

Legifrance case-law text is returned whole without inferred numbered subdivisions. Use literal `--grep` and the full decision to establish context. Judilibre's named zones are supplied by that source.

For a large audit, save the full evidence and project just the statuses into the agent context:

```sh
legidor check citations.jsonl --json --full > results.jsonl
jq '{ref:.input.ref,where:.input.where,identity,version,part,quote,changed_since,applicability,error}' results.jsonl
```

Preserve the check command's exit status. Batch documents retain NOTA, modification links, warnings and source completeness. Read those fields for legally material entries; a projected status row does not replace the underlying text or temporal caveats.

## Targeted search, code indexes and recodification

Full-text search combines the supplied terms; long natural-language queries often miss relevant wording. Start with one or two distinctive words in the right corpus. BOFiP search also depends on the exact wording and current dataset. Try a shorter term, then a returned BOI identifier. A zero-result query is not proof of absence.

```sh
legidor search 'marge' --in code:CIBS --date 2027-01-01 --limit 5
legidor get 'CIBS' --date 2027-01-01 --parts
legidor get 'CIBS L231-17..L231-21' --date 2027-01-01
legidor versions 'CIBS L213-82'
```

A code alias alone returns its immediate table of contents. Follow a returned section URL with `--parts` to browse its immediate children without fetching article bodies. Read an article or a small range once identified. An index cannot verify a substantive quotation. Legifrance search totals can count source groups while one group supplies several article hits: inspect `total_unit` and the returned hits, not just `total`.

For recodification, first read the old article's `state`, end date, NOTA and modifiers. Use `get REF --links` to inspect explicit cross-references. Read the original modifying text, search the destination code at the announced future date, then consult the relevant range. `next_versions` concerns the same article's history; it is not an old/new article correspondence. No general automatic successor mapping is claimed. Compare scope, exceptions and commencement provisions before calling a transfer legally equivalent.

Numbered laws and decrees filter candidates by the requested nature before choosing a result; rectifications can still require explicit selection. Unusual suffixes such as `CGI 1649 quater-0 B bis` are supported. For article subdivisions in manifests, prefer a separate `part` field or a comma. `check` can retry a Roman subdivision after the exact article is absent from dated and all-state searches, preserving real letter-suffixed articles and recording the reinterpretation in a warning.
