package app

import "strings"

func help(cmd string) string {
	intro := `legidor: primary French legal sources for agents

Usage: legidor COMMAND ARG [OPTIONS]
Commands: get, resolve, search, check, refs, versions, diff, auth, cache
Global options: --json, --fresh, --max-age DURATION, --offline, --env production|sandbox
Defaults: production, compact text, fresh cache allowed, today's version for named code articles.
Exit codes: 0 complete, 1 established verification failure, 2 invalid usage, 3 incomplete/indeterminate.
--date means version in force, never legal applicability to income or an operation.
Quote references containing spaces. Options may precede or follow the argument.

Examples:
  legidor get 'CSS L136-8' --date 2025-12-31 --parts
  legidor check citations.jsonl --json
Use legidor COMMAND --help for command help.
`
	texts := map[string]string{
		"get": `Usage: legidor get REF [--date YYYY-MM-DD] [--part LABEL | --parts] [--grep LITERAL] [--full] [--cursor TOKEN]
Read a code article, BOI paragraph, law table/article, decision, range or section.
Default: today for named articles, the supplied version for direct identifiers, 20000 characters.
Requested subdivisions are never truncated. Ancestor introductory text is retained.
--parts lists recognized labels. --grep is literal and case-sensitive.
Long output returns complete=false, exit 3 and a version-pinned continuation cursor.
Examples:
  legidor get 'CMF L224-4' --part 'I 2°' --date 2025-12-31
  legidor get 'BOI-RFPI-DECLA-10 §60' --json
Errors: IDENTITY_MISMATCH, PART_NOT_FOUND, STRUCTURE_UNRECOGNIZED, VERSION_CONFLICT, AUTH.
`,
		"resolve": `Usage: legidor resolve REF [--date YYYY-MM-DD] [--parse-only]
Resolve and independently check identity. --parse-only performs no network calls and does not validate existence.
Examples:
  legidor resolve 'CGI 150-0 D, 11' --parse-only --json
  legidor resolve 'CSS L742-1' --date 2025-12-31
Errors: INVALID_REFERENCE, AMBIGUOUS_REFERENCE, NOT_FOUND_IN_CORPUS, IDENTITY_MISMATCH.
`,
		"search": `Usage: legidor search QUERY [--in code:ALIAS|loda|jorf|cetat|juri|constit|kali|bofip|judilibre] [--date D] [--limit N] [--cursor TOKEN]
Default: loda, 10 hits. Limit: 1..50. Results are candidates, never identity confirmations.
The cursor binds the query, date, environment and corpus; repeat the same limit.
Judilibre search defaults to Cour de cassation. Direct JUDI: identifiers can consult other jurisdictions.
Examples:
  legidor search 'responsabilité' --in code:CIV --limit 5 --json
  legidor search 'abus de droit' --in cetat
Errors: USAGE, QUOTA, AUTH, UPSTREAM_FORMAT, OFFLINE_MISS.
`,
		"check": `Usage: legidor check FILE.jsonl|- [--json]
One explicit citation per line: {"ref":"CGI 32","date":"2025-12-31","quote":"literal text","where":"file:12"}
Optional keys: part, change. Local files and quotes are not transmitted to APIs.
Reads are deduplicated; output preserves input order. JSON output is JSONL.
Statuses are separate: identity, version, part, quote, changed_since, applicability.
Applicability is always not_evaluated. Operational failures are not verification successes.
Quote normalization only changes whitespace and quote punctuation; [...] means ordered fragments.
Examples:
  legidor check citations.jsonl --json
  printf '%s\n' '{"ref":"BOI-RFPI-DECLA-10 §60"}' | legidor check -
Errors: INVALID_MANIFEST (line reported), source errors and per-citation verification failures.
`,
		"refs": `Usage: legidor refs FILE|- [--diff]
Extract detected references locally into a JSONL manifest. Extraction is assisted, never exhaustive.
--diff labels added, removed and context lines. Review, complete and filter before check.
Defaults: plain text/Markdown/JSON input; no network; one result per detected reference.
Examples:
  legidor refs notes.md > citations.jsonl
  git diff | legidor refs - --diff > changes.jsonl
Errors: INPUT; review false positives and missed references manually.
`,
		"versions": `Usage: legidor versions REF [--since YYYY-MM-DD]
List article versions with validity dates, modifier links, NOTA and retrieval provenance.
Default: all versions, fetched individually and cached. --since filters on version start.
A modifier may have changed another passage; use diff to locate the introduction of a provision.
Examples:
  legidor versions 'CSS L136-8' --since 2025-01-01 --json
  legidor versions 'CGI 32' --since 2024-01-01
Errors: UNSUPPORTED_CONTENT (no history, including BOFiP), AUTH, QUOTA.
`,
		"diff": `Usage: legidor diff REF --from D1 --to D2 [--part LABEL] [--full]
Compare version texts by words: [-removed-]{+added+}. Unchanged recognized subdivisions are omitted.
Default: 20000-character output; incomplete results request --full. Dates are required.
Examples:
  legidor diff 'CGI 32' --from 2024-01-01 --to 2025-12-31
  legidor diff 'CSS L136-8' --from 2025-01-01 --to 2025-12-31 --part I --json
Errors: CONFLICT (dated URL or fixed identifier), NO_VERSION_AT_DATE, UNSUPPORTED_CONTENT.
`,
		"auth": `Usage: legidor auth status
Validate OAuth without displaying credentials. Reports expiry and environment.
Config: $XDG_CONFIG_HOME/legidor/piste.env (default ~/.config/legidor/piste.env), mode 0600.
Keys: PISTE_CLIENT_ID, PISTE_CLIENT_SECRET. Environment variables take precedence.
Sandbox uses piste-sandbox.env and separate token/response caches.
Examples:
  legidor auth status
  legidor auth status --env sandbox --json
Errors: AUTH_CONFIG (permissions), AUTH (credentials, subscription, environment).
`,
		"cache": `Usage: legidor cache stats|clear
Report or clear the current environment's private token and response cache.
Default: $XDG_CACHE_HOME/legidor/production (or ~/.cache/legidor/production).
TTL: 30 days for current sources and BOFiP; 90 days for historical consultations and decisions.
--max-age 24h overrides response age; --fresh bypasses even recent responses.
Do not combine --max-age with --fresh. OAuth expiry follows the server.
--offline requires an unexpired entry; --fresh refreshes without stale fallback.
Examples:
  legidor cache stats --json
  legidor cache clear --env sandbox
Errors: CACHE (unreadable or unwritable directory).
`}
	if s, ok := texts[cmd]; ok {
		return s + "\nGlobal options: --json, --fresh, --max-age DURATION, --offline, --env production|sandbox.\nExit codes: 0 complete, 1 verification failure, 2 usage, 3 incomplete/indeterminate.\n"
	}
	return strings.TrimSpace(intro) + "\n"
}
