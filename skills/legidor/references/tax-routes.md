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
| PER | Contribution or withdrawal; available ceiling and prior-year income; deducted or non-deducted contributions; capital or annuity | Contribution ceiling: `get 'CGI 163 quatervicies' --part 'I'`; II defines the professional-income base if needed. Capital withdrawal: `get 'CGI 158' --part '5 b'` (includes b quinquies); follow linked exemptions when relevant. Annuities require their own branch. |
| Dividends/capital gains | Product and income year; eligible allowance; global flat-rate/scale option; payment date of social contributions | `get 'CGI 200 A' --parts`; for eligible dividends, `get 'CGI 158' --part '3'`. Read 154 quinquies when deductible CSG changes the comparison. |
| RSUs/free shares | Qualified plan or not; authorization, acquisition and disposal dates; acquisition gain versus sale gain; recurring or exceptional income | `get 'CGI 80 quaterdecies'`, then `get 'CGI 200 A' --part '3'`. Match the plan's authorization regime before calculating. |
| CEHR | Reference income, household status and year; prior-year income if smoothing is possible | `get 'CGI 223 sexies' --part 'I'`. Follow smoothing provisions only when relevant. |
| Rental deficit | Unfurnished/furnished; eligible works, interest versus other charges; ordinary or special regime; income year | `get 'CGI 156' --part 'I 3°'`; for the allocation method, `get 'BOI-RFPI-BASE-30-20 §100..110'`. |
| PEA | First-payment date, withdrawal date, full/partial withdrawal, gain periods, applicable exception | `get 'CMF L221-32'`; before the holding-period threshold, `get 'CGI 150-0 A' --part 'II 2'`. For the exemption after the threshold: `get 'CGI 157' --part '5° bis'`. |
| Life insurance | Contract age; premium dates and aggregate premiums; gains within the withdrawal; household allowance; prior social levies | `get 'CGI 125-0 A' --parts`, then select the applicable branch. Avoid reading the entire article by default. |
| Property dealing | Business qualification, intent/activity; transfer-tax commitment; VAT status/type of property and acquisition | Qualification: `get 'CGI 35' --part 'I 1°'`. Transfer-tax commitment: CGI 1115. VAT: locate the relevant branch in CGI 257, then 268 if the margin basis is at issue. These are separate questions. |

Add a source only for an unresolved material point:

- **Social contributions:** identify the product, assessment category and event/income year. Inspect CSS L136-6 or L136-7 as appropriate, then L136-8 and any relevant commencement provision. Do not transfer a rate or exception between products. Use the other contribution texts only when their component rules need checking.
- **Exceptional-income quotient:** when asked, read CGI 163-0 A. If exceptionality or the comparison period remains unclear, consult the relevant numbered paragraphs of BOI-IR-LIQ-20-30-20. A large amount alone does not establish eligibility.
- **Missing operational details:** an official tax notice, form or administration page can answer a point outside Legidor. Do not duplicate all statute reads on the web just to obtain another citation.

Keep the scope proportional: explain the directly useful alternative or caveat,
then stop when the requested conclusion is supported. For example, distinguish
current tax from withdrawal taxation for a PER, or explain a holding-period
consequence for a PEA. Do not assume unprovided facts or turn every answer into a
review of every possible exception.
