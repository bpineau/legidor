# Output contract: legidor/1

Every JSON result contains `schema: "legidor/1"`. Errors contain `error.code`, `error.message`, and `error.next` when a useful next action exists. Standard output contains only results; diagnostics go to standard error.

Document results contain identity (source, id, title, code, number, parent), version dates, provenance (official URL, retrieval time, environment, cache/offline flags), content, recognized parts, NOTA and modification links. `complete` describes the rendered content, not a legal conclusion. Unsupported or truncated content is always explicit.

Check emits one JSON object per input line in input order. Each includes separate `identity`, `version`, `part`, `quote`, `changed_since`, and `applicability` fields. `applicability` is always `not_evaluated`. Operational failures leave unestablished checks `not_evaluated` and include an error, rather than fabricating a negative finding.

Exit codes: 0 complete processing; 1 established verification failure; 2 invalid input; 3 indeterminate or incomplete result. In a mixed batch, 3 takes priority over 2, which takes priority over 1.

Quote normalization only collapses whitespace and maps typographic apostrophes and quotation marks to their ASCII equivalents. Case, digits and negation remain unchanged. Literal `[...]` separates ordered quote fragments.

Date means the version in force on that calendar date, never applicability to income earned in that year. Validity intervals are start-inclusive and end-exclusive. Open-ended upstream dates are rendered as empty end dates.
