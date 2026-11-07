# Validation

## Offline suite

Run `go test -race ./...` and `go vet ./...`. Tests require no credentials or external network. Fake source schemas follow observed production responses; all test content and credentials are invented.

Coverage includes reference aliases, official URLs, numeric/Roman/Latin suffixes, conflicting options, code and parent identity mismatches, source nature, historical intervals, absent intervals, HTML errors, OAuth renewal after 401, quota handling, offline misses, cache expiry and concurrent writes. Text checks cover nested clauses and introductory text, ambiguous labels, doubled BOFiP attribute quotes, absent paragraphs, altered numbers/negation, ordered ellipses, numeric boundaries and word diffs. CLI checks exercise 30 manifest rows in input order, error precedence, table-of-contents quote refusal, complete preamble diffs, content-pinned pagination and bounded table output.

## Live acceptance

```sh
LEGIDOR_LIVE=1 go test ./internal/acceptance -run TestLive -count=1 -v
```

The opt-in suite validates public-source identities and structure for:

- CSS L136-8 at 2025-12-31 and its IV; CSS L742-1.
- CGI 158, 200 A, 150-0 D subdivision 11, and 199 novovicies.
- LPF L169 and CMF L224-4 subdivision I 2°.
- BOI-RFPI-DECLA-10 paragraph 60 and BOI-RSA-PENS-30-10-20 paragraph 200.
- LFI 2024 article 110, an identified Conseil d'Etat decision and a Cour de cassation decision.
- CPCE R322-39 through R322-42 in official order, including R322-41-1.

These 14 cases passed against production sources during implementation. Additional manual checks exercised QPC/CONSTIT and JURI consultations, KALI search/consultation, dated section URLs, NOR-based ELI, history and diffs. Repeated runs can use the normal fresh cache; this is not a cold-cache performance benchmark.

Production probes established several non-obvious contracts: BOFiP HTML can have doubled attribute quotes; LODA search returns a consolidated identifier distinct from the original JORF CID; CONSTIT consultation uses the generic `/consult/juri` endpoint; some modifying LODA articles have sentinel validity dates even when their containing text has a dated version. These cases have explicit handling rather than fabricated metadata.

## Independent review and skill evaluation

An independent code review found issues in date handling, text nature, nested clauses, preamble diffs, empty doctrine, ignored flags, synthetic table content and page-size cursor binding. Material findings were corrected and covered by offline regression tests. Source excerpts from live calls were not copied into test fixtures.

A Codex subagent used the skill for five scenarios: a 30-entry citation audit with intentional failures, a dated subdivision, tracking an amendment, a range with an intercalated article, and a Conseil d'Etat citation. It also exercised an uncached offline miss. No false verification success was observed. The evaluation used 12 CLI calls (one help call and 11 scenario calls); measured scenario output was 81421 bytes before subsequent audit-output compaction. These are CLI output measurements, not API-call counts or latency results.

The evaluation prompted improvements for consolidated amendment placeholders, avoiding inferred case-law subdivisions, and compact audit status projection. A separate Claude Code evaluation was attempted but could not run because its account's five-hour usage limit was reached. Cross-provider behavioral acceptance therefore remains unverified for Claude Code; no successful Claude test is claimed.

## Explicit limits

Unsupported tables, images and embedded content are not flattened into supposedly complete legal text. Historical BOFiP, notices/forms, PDF-only text, bulk mirrors and legal-applicability inference remain outside this version. A missing historical interval, ambiguous publication (including rectifications), absent subdivision or network failure is explicit. BOFiP is current administrative doctrine; it does not establish historical law.
