# Legidor implementation plan

Goal: provide reliable, source-backed French legal references through a standard-library Go CLI and an agent skill.

Architecture: reference parsing and output models are independent of network clients. Source adapters use bounded HTTP transport, OAuth and a private atomic disk cache. Text selection and citation checking operate locally. The CLI only coordinates these components.

Constraints: Go standard library only; public repository with invented test fixtures and no personal context; production by default; strict identity; no inference of legal applicability; explicit partial results; no silent stale-cache fallback.

## Tasks

- [x] 1. Define the versioned JSON contract and acceptance cases. Implement tolerant reference parsing with strict code, article, date and subdivision conflicts. Files: internal/model, internal/reference, docs/contract.md.
- [x] 2. Probe official APIs with credentials loaded privately. Implement OAuth, retries, timeouts, cache isolation and freshness. Files: internal/source/transport.go, internal/cache.
- [x] 3. Implement exact resolution and consultation for codes and BOFiP, structure selection, literal grep and batch checks. Files: internal/source/legifrance.go, bofip.go, internal/text, internal/app.
- [x] 4. Add history, word diffs, laws, official URLs, ranges and sections, search and decisions (CETAT, CONSTIT, KALI, Judilibre). Refuse unsupported content explicitly.
- [x] 5. Add local assisted reference extraction, comprehensive help, README and portable skill with troubleshooting and advanced recipes. Install binary and skill locally.
- [x] 6. Run offline race tests, vet, live acceptance checks and an independent review. Fix material findings, scan tracked files for private material and commit on master.

Each implementation task starts with behavioral tests, verifies a failure, implements the behavior, and runs the relevant tests. Offline tests never depend on credentials or live APIs. Live checks are separate and opt-in.

## Review focus

- Search results can include the wrong article or duplicate versions: independently validate returned identity and validity dates.
- Nested labels can repeat: retain ancestors and reject ambiguous boundaries.
- Missing quote text, removed negations and changed numbers must never become a success.
- Upstream HTML, quota, expired credentials and missing metadata are indeterminate failures.
- Dates in URLs, references, cursors and options cannot silently override each other.

## Decisions and progress

The supplied design is the implementation authority. It is intentionally not copied into this public repository because it contains private operational context. All documentation and fixtures here are impersonal.

Implemented all six tasks. The independent review's material findings have regression coverage. The Codex skill evaluation completed; the attempted Claude Code evaluation was blocked by an external usage quota and remains pending. See validation.md for the measured results and the exact coverage limits.

Rulings: use generic `/consult/juri` for constitutional decisions after production verification; preserve a consolidated LODA ID separately from its original publication CID; reject unavailable/sentinel article intervals rather than infer validity from the containing law. Article tables are complete as tables but cannot validate substantive quotations. Only source-provided decision zones are selectable; no heuristic case-law hierarchy is asserted.
