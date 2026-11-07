# Output contract: legidor/1

Every JSON result contains `schema: "legidor/1"`. Errors contain `error.code`, `error.message`, and `error.next` when a useful next action exists. Standard output contains only results; diagnostics go to standard error.

Document results contain identity (source, id, title, code, number, parent), version dates, provenance (official URL, retrieval time, environment, cache/offline flags), content, recognized parts, NOTA and modification links. `complete` describes the rendered content, not a legal conclusion. Unsupported or truncated content is always explicit.

Rectangular HTML tables are rendered as rows with cell separators and headers when supplied. Full-width notes after a regular row establishes the width are labeled `Table-wide:` in their original sequence. Other merged/nested or unsupported tables remain `[[legidor:omitted:table]]` markers. Other unsupported embedded content is marked similarly at its original location. Superscripts and subscripts in cells use `^(...)` and `_(...)`; this is a rendering convention, not original typography.

Selection can be complete even when unsupported content exists elsewhere. In this case `source_complete` preserves the original completeness and `unsupported_content` describes the selected text. Introductions and their omissions are retained. Other upstream incompleteness is not cleared by selection. Completeness does not establish that the selected provision has no relevant external exceptions.

`get --links` returns `content_kind: "source_references"` and `items` containing explicit HTML links to supported official sources. These are candidates, not consulted targets. The index is document-wide, paginated and incompatible with `--part`, `--parts` or `--grep`. Source incompleteness remains visible. No HTML representation produces `UNSUPPORTED_CONTENT`.

Check emits one JSON object per input line in input order. Each includes separate `identity`, `version`, `part`, `quote`, `changed_since`, and `applicability` fields. `applicability` is always `not_evaluated`. Operational failures leave unestablished checks `not_evaluated` and include an error, rather than fabricating a negative finding.

Batch documents retain NOTA, modification links and warnings, also shown in text mode. The article interval alone does not establish the effective period of every subdivision. `refs` is assisted extraction: ambiguous numeric lists produce diagnostics on stderr and exit 3, while independently recognized citations are still emitted. Explicit plural article lists and BOFiP paragraph lists are expanded.

Article `state` is source-supplied. `next_versions` contains the closest later valid intervals from the same article's history, not successor article numbers after recodification. A known article without a version at the requested date returns `NO_VERSION_AT_DATE` and available interval information. All-state search candidates are checked against consulted code identity; annex titles in search are not authoritative.

`check` may reinterpret a Roman article suffix as a subdivision only after dated and complete all-state searches establish that the exact article is absent. It records the interpretation in document warnings and preserves the input citation. Existing but unavailable, withdrawn or unconsultable exact articles do not authorize a retry. Code-less `refs` candidates produce stderr diagnostics and exit 3 rather than invented code names.

`omitted_modifications` counts links hidden by output compaction. Documents and JSON version histories keep at most five recent links unless `--full` is requested; NOTA stays available. Text histories are indexes showing one modifier with a count of the remaining links, and expand with `--full`. `complete` is independent of this explicit metadata abridgement. `versions --since` retains intervals whose exclusive end is later than the filter date, including versions starting earlier and open-ended intervals; empty results are `[]`.

Code aliases alone return immediate table-of-contents entries. A section with `--parts` does likewise, without fetching descendants' article content. Article searches can have `total_unit: "source_groups"`: `total` counts upstream groups, `returned_groups` counts received groups and `hits` contains flattened article candidates. These counts are not interchangeable.

BOFiP paragraph selection recognizes nonbreaking-space markers and retains the relevant heading hierarchy and introductory conditions. Following section headings stay with their own paragraphs. Editorial news notices remain in full text but are omitted from selected paragraph context.

Exit codes: 0 complete processing; 1 established verification failure; 2 invalid input; 3 indeterminate or incomplete result. In a mixed batch, 3 takes priority over 2, which takes priority over 1.

Quote normalization only collapses whitespace and maps typographic apostrophes and quotation marks to their ASCII equivalents. Case, digits and negation remain unchanged. Literal `[...]` separates ordered quote fragments.

Date means the version in force on that calendar date, never applicability to income earned in that year. Validity intervals are start-inclusive and end-exclusive. Open-ended upstream dates are rendered as empty end dates.

Provenance includes `max_age_seconds`, the selected response reuse budget. Current responses default to 30 days; historical consultations/searches, direct article versions and decision consultations default to 90 days. `--max-age` overrides this duration; `--fresh` bypasses response reads. Token expiry is independent.

Named code article resolution adds `resolution_date` (the legal date used by the source search), `resolution_retrieved_at` (when that search response was retrieved), and `resolution_cached` (true when reused). Implicit-current resolutions may be reused across days within the age budget and known validity interval. The document's `date` remains the requested day, and its own `retrieved_at` describes the article response. Neither timestamp is rejuvenated by a cache hit. `version=in_force_at_date` is an assessment from this dated evidence, not a claim of live source verification. Explicit-date requests never use an implicit-current resolution alias.
