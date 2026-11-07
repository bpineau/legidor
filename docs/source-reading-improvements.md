# Source reading improvements

Improve source fidelity and citation workflows without inferring legal applicability.

Implementation order:

1. Render rectangular HTML tables, preserving captions, headers, cells and full-width notes following regular rows. Reject other merged, nested or malformed tables explicitly. Mark omitted content at its original position.
2. Scope selection completeness to the selected content and retained introductions. Preserve the original completeness separately; do not upgrade other forms of incomplete source data.
3. Recognize dotted lowercase subdivisions and compound numbered subdivisions. Extract lists of explicit citations without silently interpreting article lists as subdivisions.
4. Support CIBS references through the existing code resolution and identity checks. Filter named-law candidates by their principal title while retaining ambiguity between originals and rectifications.
5. Expose source cross-references and retain temporal caveats in batch results. Document a workflow that links assertions, sources, assumptions and consuming calculations, with explicit external-source fallbacks.

Validation uses invented fixtures, integration tests with local HTTP servers, and bounded live reads of public legal sources. No personal dossier, paths or financial examples belong in this repository. Keep the standard-library-only Go implementation. Run the race tests and vet, review the change, install the binary and skill, then commit and push master.
