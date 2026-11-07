# Validation

## Offline suite

Run `go test -race ./...` and `go vet ./...`. Tests require no credentials or external network. Fake source schemas follow observed production responses; all test content and credentials are invented.

Coverage includes reference aliases, official URLs, numeric/Roman/Latin suffixes, conflicting options, code and parent identity mismatches, source nature, historical intervals, absent intervals, HTML errors, OAuth renewal after 401, quota handling, offline misses, cache expiry and concurrent writes. Text checks cover nested clauses and introductory text, ambiguous labels, doubled BOFiP attribute quotes, absent paragraphs, altered numbers/negation, ordered ellipses, numeric boundaries and word diffs. CLI checks exercise 30 manifest rows in input order, error precedence, table-of-contents quote refusal, complete preamble diffs, content-pinned pagination and bounded table output.

Cache-policy regressions cover the 30/90-day boundaries, positive `--max-age` validation and conflicts, online refresh versus offline refusal, unchanged OAuth token reuse, and forced refresh of recent entries. A simulated next day verifies implicit-current resolution reuse without network calls, preserved source timestamps, explicit-date separation, and invalidation at a known version end. An old search with a newly retrieved article cannot bypass a stricter age budget or rejuvenate the resolution alias. Pagination retains the requested freshness flags.

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

The revised cache policy was also checked with a live `CIV 1240 --fresh` consultation followed by `--offline --max-age 24h`: the latter reused both response and resolution while preserving their retrieval times. The cache changes received a separate read-only code review with no material findings.

## Independent review and skill evaluation

An independent code review found issues in date handling, text nature, nested clauses, preamble diffs, empty doctrine, ignored flags, synthetic table content and page-size cursor binding. Material findings were corrected and covered by offline regression tests. Source excerpts from live calls were not copied into test fixtures.

A Codex subagent used the skill for five scenarios: a 30-entry citation audit with intentional failures, a dated subdivision, tracking an amendment, a range with an intercalated article, and a Conseil d'Etat citation. It also exercised an uncached offline miss. No false verification success was observed. The evaluation used 12 CLI calls (one help call and 11 scenario calls); measured scenario output was 81421 bytes before subsequent audit-output compaction. These are CLI output measurements, not API-call counts or latency results.

The evaluation prompted improvements for consolidated amendment placeholders, avoiding inferred case-law subdivisions, and compact audit status projection. A separate Claude Code evaluation was attempted but could not run because its account's five-hour usage limit was reached. Cross-provider behavioral acceptance therefore remains unverified for Claude Code; no successful Claude test is claimed.

## Citation-review exercise

Legidor was exercised against the substantive tax changes in Paperasse PRs [126](https://github.com/romainsimon/paperasse/pull/126), [127](https://github.com/romainsimon/paperasse/pull/127) and [129](https://github.com/romainsimon/paperasse/pull/129), at heads `565b27f`, `17f1b83` and `9f2fbe2` respectively. This was a local source-validation exercise, not a posted GitHub approval or a review of the whole repository.

The workflow used `refs --diff`, a manually reviewed 23-entry manifest, `check`, targeted `get` calls, and a dated `diff`. Primary texts supported the material changes: distinct complaint/assessment deadlines, the 2023 Besson repeal and the 2026 private-landlord provision, and separate micro-BIC limits for mixed furnished rentals. Applicability was checked from the amending law as well as the consolidated code; identity confirmation alone was not treated as legal validation. No blocking issue was identified in those changes.

The exercise exposed two defects, now covered by regression tests: extraction missed code-last references such as `art. L. 169 LPF`, and parsing treated the final letters in `1649 bis C` and `199 decies EA` as subdivisions. After correction, extraction on PR 126 found 10 occurrences instead of two (including context lines); this is not an exhaustive-recall metric. Twenty of the 23 initial manifest entries then completed without errors. Follow-up consultations selected the original 2026 finance law from explicit candidates and read an older version of article 199 decies F (marked PERIME by the source); neither was silently substituted for the original request.

BOI-BAREME-000044 contains tables and remained explicitly incomplete. Its thresholds were checked in the official web publication, while article 50-0 and the textual BOFiP paragraphs were available through the CLI. Table rendering, inferred references without a code name, historical BOFiP and tax-form box verification remain coverage limits. This exercise measured functionality, not comparative token savings or agent-provider performance.

## Calculation-review exercise

Paperasse [PR 97](https://github.com/romainsimon/paperasse/pull/97) was reviewed at head `3e70c1f`, against base `4fd0a05`. The review covered the changed capital-income options, deductible CSG, furnished and unfurnished rental income, carried-forward property losses, net capital losses and reference-income adjustments. Its existing deterministic calculation suite passed in an isolated export of that commit; the user's working checkout was unchanged. No blocking issue was identified within the calculator's documented scope.

A manually reviewed 16-entry manifest completed through `legidor check`, then again offline, with all identities confirmed and no incomplete content. Targeted consultations included commencement provisions: the article 200 C version available at the end of 2022 already described the crypto option, but its NOTA made it applicable only to disposals from 2023. The 2026 social-security finance law likewise distinguished income-year applicability from the commencement date for investment proceeds. The tool supplied the evidence; applicability remained a separate review step.

This exercise also exposed a source-maintenance caveat rather than a retrieval defect: BOI-IR-BASE-20-20 is present in the current-doctrine dataset but its 2017 publication still quotes older CSG rates. The dated CGI and the [2026 official income-tax brochure](https://www.impots.gouv.fr/www2/fichiers/documentation/brochure/ir_2026/pdf_integral/Brochure-IR-2026.pdf) supported the review instead. The skill now explicitly distinguishes current-corpus membership from up-to-date legal content. The brochure and tax-form fields required a web/PDF fallback; they were not retrieved by legidor. No PR comment or approval was posted.

## Explicit limits

Unsupported tables, images and embedded content are not flattened into supposedly complete legal text. Historical BOFiP, notices/forms, PDF-only text, bulk mirrors and legal-applicability inference remain outside this version. A missing historical interval, ambiguous publication (including rectifications), absent subdivision or network failure is explicit. BOFiP is current administrative doctrine; it does not establish historical law.
