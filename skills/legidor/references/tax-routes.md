# Tax research routes

Choose the relevant row; these are starting points, not an exhaustive source list
or a tax calculator. Use the requested facts and dates. Do not load unrelated
routes or fetch every listed reference. This map stores no rates or thresholds:
read those from the applicable official text.

Commands below are subcommands of `legidor`: prepend `legidor` in a shell,
or pass the subcommand and arguments to the allocated CLI tool.

| Topic | Facts that select the branch | First targeted reads |
|---|---|---|
| Income tax | Income year, income category, household shares, deductions; distinguish salary declared from income after expenses | `get 'CGI 197' --part 'I 1'`; for salary expenses, `get 'CGI 83' --part '3°'`. Read other parts of 197 only for relevant adjustments. |
| PER | Contribution or withdrawal; each member's own ceiling versus optional couple aggregation; prior-year income and unused ceilings; deducted or non-deducted contributions | Contribution eligibility/ceiling: `get 'CGI 163 quatervicies' --part 'I'`; II defines professional income. Do not substitute a couple's combined ceiling for the member's own allowance without examining the option. Capital withdrawal: `get 'CGI 158' --part '5 b'` (includes b quinquies); annuities need their own branch. For residence changes or CEHR, use the interaction routes below. |
| Dividends/capital gains | Product and income year; eligible allowance; global flat-rate/scale option; payment date of social contributions | `get 'CGI 200 A' --parts`; for eligible dividends, `get 'CGI 158' --part '3'`. Read 154 quinquies when deductible CSG changes the comparison. |
| RSUs/free shares | Qualified plan or not; authorization, acquisition and disposal dates; acquisition gain versus sale gain; recurring or exceptional income | `get 'CGI 80 quaterdecies'`, then `get 'CGI 200 A' --part '3'`. Match the plan's authorization regime before calculating. |
| CEHR | RFR rather than taxable income, household status/year; deductible PER or other RFR adjustments; prior years if smoothing matters | `get 'CGI 223 sexies' --part 'I'` defines the basis with its adjustments. When a deduction affects the comparison, inspect `get 'CGI 1417' --part 'IV'`, including the reintegration of 163 quatervicies contributions. Do not assume an income-tax deduction reduces RFR or CEHR. Read smoothing provisions only when relevant. |
| Rental deficit | Unfurnished/furnished, professional status; rent, interest and other eligible charges separately; current deficit versus earlier carry-forwards | Unfurnished: `get 'CGI 156' --part 'I 3°'` and `get 'BOI-RFPI-BASE-30-20 §100..110'`. Allocate rent to interest before other charges to identify the deficit's origin; inspect the applicable global-income cap, rental-income carry-forward and continued-letting condition. Furnished nonprofessional deficits use the separate I 1° ter branch, not the unfurnished rule; distinguish deficits from deferred depreciation and check the regime of the utilization year. |
| PEA | First-payment date, withdrawal date, full/partial withdrawal, gain periods, applicable exception | `get 'CMF L221-32'`; before the holding-period threshold, `get 'CGI 150-0 A' --part 'II 2'`. For the exemption after the threshold: `get 'CGI 157' --part '5° bis'`. |
| Life insurance | Contract age; premium dates and aggregate premiums; gains within the withdrawal; household allowance; prior social levies | `get 'CGI 125-0 A' --parts`, then select the applicable branch. Avoid reading the entire article by default. |
| Property dealing | Business qualification, intent/activity; transfer-tax commitment; VAT status/type of property and acquisition | Qualification: `get 'CGI 35' --part 'I 1°'`. Transfer-tax commitment: CGI 1115. VAT: locate the relevant branch in CGI 257, then 268 if the margin basis is at issue. These are separate questions. |

## Interactions that can change the conclusion

Open only the relevant route. These provisions identify questions to resolve,
not automatic eligibility or complete tax advice. Statutory references below
mean the version applicable to the facts; current BOFiP is a doctrinal starting
point, not evidence of historical wording. Inspect its date and relevant exceptions.

| Topic | Branching facts and targeted reads |
|---|---|
| PER and nonresidence | Establish income year, tax/treaty residence, move date and contribution payment date before applying a resident ceiling. Read `get 'CGI 164 A'` (global-income charges of nonresidents) alongside `get 'CGI 163 quatervicies' --part 'I'`. If resident-equivalent treatment is claimed, `get 'BOI-IR-DOMIC-40 §30..90'` sets out Schumacker conditions and consequences; do not infer them merely from French income. For a move during the year, inspect the relevant period in `BOI-IR-DOMIC-20` before allocating a payment. New residents have a distinct ceiling branch: `get 'BOI-IR-BASE-20-50-20 §430..500'`. An old available ceiling alone does not establish current deductibility. |
| Nonresident property sale | Identify departure/sale dates, destination and convention coverage, occupation or availability to third parties after departure, nationality, prior residence and prior exemptions. Start the former-main-home branch at `get 'CGI 244 bis A' --part 'I'`, 1. Separately examine `get 'CGI 150 U' --part 'II 2°'` for the limited nonresident-housing exemption and 244 bis A II for their interaction. Do not transpose the ordinary resident main-home tolerance, combine the exemptions or transfer conditions between them. |
| Managers | Identify legal form, tax regime, role and ownership/control before classifying remuneration under `get 'CGI 62'`; a manager title alone is insufficient. For that category, distinguish gross remuneration, eligible contributions under `get 'CGI 154 bis'` and the subsequent professional-expense basis under CGI 83, 3°. `get 'BOI-RSA-GER-20 §10'` explains the sequence; `§110..140` addresses contributions and charges, with partial activity-CSG deductibility referring to CGI 154 quinquies. Identify who paid each item and whether already included/deducted; do not treat every social payment as deductible or subtract it twice. |
| Combined tax benefits | Identify each benefit, expense/payment year, reduction versus refundable credit, special limits and any carry-forward origin. Inspect `get 'CGI 200-0 A'` for included/excluded benefits and the combined cap, alongside the benefit's own article. Credit status alone does not exclude a benefit from the cap. Separate refundability, order of imputation and carry-forward eligibility; an excess is not automatically reportable. For a claimed investment carry-forward, read the precise computation in the benefit's article, not just the total combined excess. |
| Crypto election | Establish private versus professional activity, year and type of disposal. Read `get 'CGI 200 C'` for the election specific to gains under 150 VH bis, then the relevant basis in 150 VH bis if unresolved. Its scope is distinct from the other capital-income election in CGI 200 A; do not transfer the global 2OP rule to crypto. Use the year's official return/notice for the actual election box. Assess social-contribution category/date separately. |

Add a source only for an unresolved material point:

- **Social contributions:** identify the product, assessment category and event/income year. Inspect CSS L136-6 or L136-7 as appropriate, then L136-8 and any relevant commencement provision. Do not transfer a rate or exception between products. Use the other contribution texts only when their component rules need checking.
- **Exceptional-income quotient:** when asked, read CGI 163-0 A. If exceptionality or the comparison period remains unclear, consult the relevant numbered paragraphs of BOI-IR-LIQ-20-30-20. A large amount alone does not establish eligibility.
- **Missing operational details:** an official tax notice, form or administration page can answer a point outside Legidor. Do not duplicate all statute reads on the web just to obtain another citation.

Keep the scope proportional: explain the directly useful alternative or caveat,
then stop when the requested conclusion is supported. For example, distinguish
current tax from withdrawal taxation for a PER, or explain a holding-period
consequence for a PEA. Do not assume unprovided facts or turn every answer into a
review of every possible exception.
