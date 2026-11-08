# Troubleshooting

| Error | Next action |
|---|---|
| `AUTH`, `AUTH_CONFIG` | Run `legidor auth status`. The operator checks credentials, private file permissions, PISTE subscriptions and accepted API terms. Do not inspect or display secret values. |
| `QUOTA` | Respect the server delay and retry later. The CLI already bounds retries; do not create a tight retry loop. |
| `NETWORK`, `UPSTREAM`, `UPSTREAM_FORMAT` | Report retrieval as unverified. Try an existing fresh cache with `--offline`, or retry the official API later. HTML is not accepted as legal API content. |
| `OFFLINE_MISS` | Retry without `--offline` when network access is available, retaining any required `--max-age`. Expiry follows that override or the default 30/90-day budget. Do not treat an absent/expired cache entry as a legal absence. |
| `AMBIGUOUS_REFERENCE` | Inspect candidates. Refine code, number, date or use a returned identifier; never guess one. |
| `IDENTITY_MISMATCH` | Stop using that result. Check the supplied reference and report the mismatch. |
| `VERSION_CONFLICT`, `CONFLICT` | Reconcile reference, URL, cursor and options. Resolve a human reference at the desired date rather than forcing another date onto a fixed version. |
| `PART_NOT_FOUND`, `AMBIGUOUS_PART`, `STRUCTURE_UNRECOGNIZED` | Read `get REF --parts`. Use a full recognized label or inspect the official source. Do not claim an inferred boundary is verified. |
| `NO_VERSION_AT_DATE` | No version is available at the requested date. Inspect `versions`, NOTA and explicit modifying links; only consult another date when it answers the question. Do not guess adjacent articles, successors or current applicability. Current BOFiP cannot replace historical doctrine. |
| `REINTERPRETATION_FAILED` | The exact article was absent, but checking an alternative Roman subdivision failed. Review the diagnostic and use an explicit manifest `part`; do not treat the original identity as confirmed. |
| `UNSUPPORTED_CONTENT`, `RESULT_TOO_LARGE` | Use a narrower article/subdivision or consult the official source. `--full` only removes output truncation, not format limitations. |
| `CURSOR_CHANGED` | Restart the consultation; do not concatenate pages from different content revisions. |

`refs` can also exit 3 with an ambiguous-list diagnostic on stderr while emitting other citations. Review the reported source line and complete the manifest. For `--links`, follow the returned continuation command including `--links`; an absent HTML representation cannot be replaced with inferred links.

Installation is described in the application's README. The binary must be on PATH. Credentials belong in the user's private configuration, never in a skill, repository, manifest or diagnostic attachment. Production and sandbox use different configurations and caches.
