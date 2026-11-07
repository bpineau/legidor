# Advanced recipes

## Resolve and search

```sh
legidor resolve 'CGI 150-0 D, 11' --parse-only --json
legidor resolve 'CGI 150-0 D' --date 2025-12-31 --json
legidor search 'abus de droit' --in cetat --limit 5 --json
legidor search 'revenus fonciers' --in bofip --limit 5 --json
```

Parse-only validates grammar, not existence. Search returns identifiers and bounded excerpts. Re-read the chosen identifier with `get`, checking identity independently. Never select a result solely because it ranks first. Pagination is not a stable database snapshot; repeat the query, corpus, date and page size with the returned cursor.

Aliases include CGI and annexes, LPF, CSS, CMF/CoMoFi, C. civ., CCH, C. assur., C. trav., CPCE, C. conso., C. com., CGCT, CSP and C. urb. A suffix such as the I in CGI 990 I belongs to the article number. Use commas to separate uncertain subdivisions and inspect `resolve --parse-only`.

## Laws and official URLs

```sh
legidor get 'LFI 2024 art. 110' --json
legidor resolve 'LFI 2024' --json
```

A numbered law without `--date` searches its published JORF version. Rectifications can create several candidates: use the returned identifiers and titles to identify the intended publication. A dated law requests the consolidated LODA version, which can differ from the original publication. A fixed JORF identifier does not support consolidated date selection.

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

## Modifier placeholders

A consolidated modifying article can contain only "A modifie..." substitution placeholders. That is not the original amending wording. Use its returned JORF parent, then read the same article number in the original publication. Check exactly which amendment a transitional provision targets; do not transfer a rule about one numbered amendment to another.

Legifrance case-law text is returned whole without inferred numbered subdivisions. Use literal `--grep` and the full decision to establish context. Judilibre's named zones are supplied by that source.

For a large audit, save the full evidence and project just the statuses into the agent context:

```sh
legidor check citations.jsonl --json > results.jsonl
jq '{ref:.input.ref,where:.input.where,identity,version,part,quote,changed_since,applicability,error}' results.jsonl
```

Preserve the check command's exit status. Read source details for failed or legally material entries; a projected status row does not replace the underlying text.
