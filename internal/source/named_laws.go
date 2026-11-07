package source

import (
	"regexp"
	"strings"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

func namedLawTitle(title string, r reference.Reference) bool {
	phrase := "de finances pour "
	if r.Nature == "LFSS" {
		phrase = "de financement de la sécurité sociale pour "
	}
	pattern := `(?i)^loi\s+(?:organique\s+)?n[°o]\s*\d{4}-\d+\s+du\s+\d{1,2}(?:er)?\s+\S+\s+\d{4}\s+` + regexp.QuoteMeta(phrase+r.Number) + `(?:\s*\([^)]*\))*\s*$`
	return regexp.MustCompile(pattern).MatchString(strings.Join(strings.Fields(title), " "))
}

func namedLawCandidates(s model.Search, r reference.Reference) model.Search {
	allSeen := s.Total <= len(s.Hits) && s.Complete
	var hits []model.Hit
	for _, h := range s.Hits {
		if namedLawTitle(h.Title, r) {
			hits = append(hits, h)
		}
	}
	s.Hits = hits
	if allSeen {
		s.Total = len(hits)
	}
	return s
}

// Named laws are pinned only after production consultation verified their title,
// nature, number and JORF identifier. Other years use a bounded title search.
// Sources: the official JORF consultation for each identifier below.
var namedLaws = map[string]struct{ number, id string }{
	"LFI 2024":  {"2023-1322", "JORFTEXT000048727345"},
	"LFSS 2026": {"2025-1403", "JORFTEXT000053226384"},
}
