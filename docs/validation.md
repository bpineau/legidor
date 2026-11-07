# Validation

## Offline tests

```sh
go test -race ./...
go vet ./...
```

Tests require no credentials or external network. Fixtures use invented content
and local HTTP servers. They cover reference parsing, strict source identity,
version intervals, structured selections, quotation checks, output completeness,
OAuth renewal, cache expiry/isolation, concurrent writes and CLI error handling.
Evaluation-runner tests use recorded synthetic events or stub executables, never
paid model calls.

## Live acceptance

```sh
LEGIDOR_LIVE=1 LEGIDOR_CACHE_DIR=/tmp/legidor-acceptance \
  go test ./internal/acceptance -run TestLive -count=1 -v
```

This opt-in suite retrieves public legislation, BOFiP and decisions using private
PISTE credentials. The dedicated cache root keeps responses and OAuth tokens
separate from everyday usage. Existing valid entries can be reused, so repeated
runs are not cold-cache performance measurements. See the cases in
[internal/acceptance/live_test.go](../internal/acceptance/live_test.go) for the
exact references and expected identities.

## Source fidelity

Regression fixtures cover unusual article suffixes, nested and repeated labels,
BOFiP paragraph boundaries, HTML tables and omitted content. Rectangular tables
and supported full-width notes retain source cells and formulas, including
upstream defects. Unsupported merged/nested tables and images remain explicitly
incomplete. Do not repair source wording by inference.

Date checks distinguish a requested version from legal applicability. Missing
historical intervals, ambiguous publications, absent subdivisions and upstream
errors must remain explicit. A confirmed identity or quotation does not establish
that a rule applies to a particular taxpayer or event.

## Agent evaluations

Use the [Go evaluation guide](../evals/README.md) for isolated model comparisons,
quota controls, grading and reporting. Keep raw traces, one-off reports and
completed plans outside this repository. Publish conclusions in the requesting
conversation, with measured coverage and limitations.

## Coverage limits

Historical BOFiP, tax notices/forms, PDF-only text, bulk mirrors, automatic
recodification correspondence and legal-applicability inference are outside the
CLI's scope. Membership in the current BOFiP corpus does not guarantee that every
quoted rule is up to date. Verify material changes against dated legislation and
commencement provisions.
