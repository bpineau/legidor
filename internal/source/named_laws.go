package source

// Named laws are pinned only after production consultation verified their title,
// nature, number and JORF identifier. Other years use a bounded title search.
// Sources: the official JORF consultation for each identifier below.
var namedLaws = map[string]struct{ number, id string }{
	"LFI 2024":  {"2023-1322", "JORFTEXT000048727345"},
	"LFSS 2026": {"2025-1403", "JORFTEXT000053226384"},
}
