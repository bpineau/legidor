# Review assertions and their use

Establish which documents and calculations are authoritative before auditing a dossier or repository. Treat archived studies, superseded assumptions and already documented uncertainties accordingly. Use the user's existing context instead of collecting it again.

Keep a local working matrix outside public repositories when it contains confidential information:

| Assertion and location | Period and facts assumed | Primary source and subdivision | Use in document or calculation | Result and remaining evidence |
|---|---|---|---|---|
| An invented provision applies to a transaction | Operation date; stated eligibility | Dated statute plus commencement rule | Current estimate | Source confirmed; eligibility evidence pending |

Extract explicit references with `refs`, then inspect uncited assertions that could change a decision. Read stderr and preserve the exit status: ambiguous lists can produce exit 3 while other references are emitted. Do not treat a partial manifest as exhaustive. Expand explicit lists using `articles ...`; when a number means a subdivision, use the manifest's `part` field to resolve ambiguity.

Run a reviewed manifest with `check`. Read the actual provisions for material assertions, including exceptions, NOTA and transitional rules. Preserve separate conclusions for identity, wording, applicability, factual evidence and calculation. A valid citation does not prove that it supports the surrounding claim.

Trace each confirmed rule into the active calculation or document. Check whether input updates, dates, assumptions and exceptions reached that consumer. A correct source beside an outdated model is still an unresolved issue. Report calculation effects only when recomputed with stated assumptions; legidor does not calculate them.

Classify findings as a new error, a resolved question, an existing uncertainty, or outside the review's scope. Prioritize decision impact. Do not manufacture findings or repeat a known reservation as a new defect. Keep the detailed evidence private and use synthetic examples for public bug reports and tests.

## Follow the source chain

```sh
legidor get 'CGI 261' --links --json
legidor get 'CGI 261' --parts --json
```

`--links` lists explicit links supplied in the document HTML to supported official sources. It is a document-wide index, cannot be combined with a subdivision, and does not fetch or verify targets. Inspect the source title and link, then consult the returned identifier or URL. Absence from the list is not proof that there are no other references. A dated BOFiP link still cannot establish historical doctrine through the current-only dataset.

For recodification, read NOTA and the original modifying text; distinguish full repeal, partial repeal and transitional maintenance. CIBS is supported. Never replace an old article number with a guessed successor, or assume that an open-ended article interval covers every subdivision.

## Complements outside legidor

| Missing evidence | Primary complement | Record |
|---|---|---|
| Return boxes, calculation instructions, administrative examples | impots.gouv.fr forms, notices and brochures | Income year, form edition, page and official URL |
| Operational explanation or documentation requirements | service-public.gouv.fr and the responsible administration | Update date, covered product/status and link to the governing provision |
| Social contribution base and collection practice | URSSAF and BOSS | Effective period, applicable status, source and limits |
| Unemployment conditions or implementation | Unedic and France Travail | Regulation/circular version, section and relevant claimant category |
| Unsupported table, image, annex or PDF-only content | The source's official publication | Page/table and whether visually checked |
| Contractual condition or proof of eligibility | The actual contract or private supporting document | Local evidence reference; do not send it as a public API query |

State which evidence came from legidor and which required another tool. Do not downgrade a legal uncertainty into an API outage, or upgrade an administrative summary into legislation. Retain unresolved points when the necessary source or factual evidence is absent.

Start with material assertions explicitly marked uncertain (for example `[?]`) or sourced only to a secondary reproduction. A targeted primary-source read often resolves these with little output. Do not treat every uncertainty marker as a fiscal error: classify what the source actually resolves and what still depends on missing facts.
