package app

import (
	"regexp"
	"strings"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

var doctrineQueryRE = regexp.MustCompile(`(?i)\b(?:BOI|BOFIP)\b`)
var textArticleSuffixRE = regexp.MustCompile(`(?i)\s+art(?:icle)?\.?\s+[0-9A-Za-z.-]+$`)

// shellQuote keeps source-derived references literal in copyable commands.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// recoveryCommand carries compatible consultation options, never a selection or
// cursor from the failed command. Environment variables remain inherited.
func recoveryCommand(command, ref string, o options, flags ...string) string {
	s := "legidor " + command + " " + shellQuote(ref)
	for _, flag := range flags {
		s += " " + flag
	}
	if command != "versions" && o.date != "" {
		s += " --date " + shellQuote(o.date)
	}
	if o.env != "" {
		s += " --env " + shellQuote(o.env)
	}
	if o.json {
		s += " --json"
	}
	if o.offline {
		s += " --offline"
	}
	if o.fresh {
		s += " --fresh"
	}
	if o.maxAge > 0 {
		s += " --max-age " + shellQuote(o.maxAge.String())
	}
	return s
}

func unselectedReference(r reference.Reference) string {
	switch {
	case r.Kind == "bofip":
		return r.Number
	case r.Kind == "article" && r.Code != "":
		return r.Code + " " + r.Number
	case r.Kind == "article" && r.ID != "":
		return r.ID
	default:
		return r.Raw
	}
}

func searchHints(query string, s model.Search, o options) model.Search {
	if len(s.Hits) == 0 {
		s.Hints = append(s.Hints, "No candidates were returned on this page. Shorten the terms and check the selected corpus and its coverage; this does not establish legal absence.")
	}
	if o.corpus == "loda" && !o.supplied["in"] && doctrineQueryRE.MatchString(query) {
		// BOFiP cannot accept a legislative date. Make that deliberate difference
		// explicit instead of silently executing a different request.
		next := o
		next.date = ""
		s.Hints = append(s.Hints, "LODA does not search tax doctrine. For current BOFiP (without a legislative --date): "+recoveryCommand("search", query, next, "--in bofip"))
	}
	return s
}

func recoveryError(err error, r reference.Reference, o options) error {
	original := asError(err)
	m := *original
	// Do not mutate source-owned errors reused by callers or tests.
	m.Hints = append([]string{}, original.Hints...)
	m.Commands = append([]string{}, original.Commands...)
	base := unselectedReference(r)
	next := o
	if next.date == "" {
		next.date = r.Date
	}
	switch m.Code {
	case "NO_VERSION_AT_DATE":
		history := r
		history.Date = ""
		if r.Kind == "article" {
			m.Next = recoveryCommand("versions", unselectedReference(history), next)
			m.Hints = append(m.Hints, "Inspect available intervals, NOTA and explicit modifying links. History is not evidence that a historical rule applies today; do not guess neighboring or successor article numbers.")
		}
	case "PART_NOT_FOUND", "STRUCTURE_UNRECOGNIZED", "AMBIGUOUS_PART":
		if r.Article != "" && strings.HasPrefix(m.Message, "article ") {
			base = textArticleSuffixRE.ReplaceAllString(base, "")
		}
		m.Next = recoveryCommand("get", base, next, "--parts")
		m.Hints = append(m.Hints, "Inspect source labels before selecting. Duplicate labels are ambiguous; an outline preview is navigation, not the full rule.")
	case "AMBIGUOUS_REFERENCE":
		if r.Kind != "law" && r.Kind != "named_law" && r.Kind != "text" && r.Kind != "nor" {
			break
		}
		for _, candidate := range m.Candidates {
			fields := strings.Fields(candidate)
			if len(fields) == 0 {
				continue
			}
			id, e := reference.Parse(fields[0])
			if e != nil || id.Kind != "text" || id.ID == "" {
				continue
			}
			if next.date != "" && strings.HasPrefix(id.ID, "JORFTEXT") {
				continue
			}
			ref := id.ID
			if r.Article != "" {
				ref += " art. " + r.Article
			}
			candidateRef, err := reference.Parse(ref)
			if err != nil || candidateRef.ID != id.ID || candidateRef.Article != r.Article {
				continue
			}
			m.Commands = append(m.Commands, recoveryCommand("get", ref, next))
		}
		if len(m.Commands) > 0 {
			m.Hints = append(m.Hints, "Identify the intended publication, including any rectificatif, before choosing a candidate command. 'art. NUMBER' selects an article of a whole text; --part selects a subdivision inside it. Candidates from an incomplete search do not establish uniqueness.")
		}
	}
	return &m
}
