package app

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bpineau/legidor/internal/model"
)

const modifierLimit = 5

// compactMetadata bounds repeated links separately from substantive text and
// NOTA. An explicit count makes the abridgement visible in both output modes.
func compactMetadata(d model.Document, full bool) model.Document {
	if full || len(d.Modifications) <= modifierLimit {
		return d
	}
	links := append([]model.Link{}, d.Modifications...)
	sort.SliceStable(links, func(i, j int) bool { return links[i].Date > links[j].Date })
	d.OmittedModifications = len(links) - modifierLimit
	d.Modifications = links[:modifierLimit]
	return d
}

func renderVersions(out io.Writer, o options, vs []model.Version) {
	if o.json {
		vs = append([]model.Version{}, vs...)
		for i := range vs {
			d := compactMetadata(model.Document{Modifications: vs[i].Modifications}, o.full)
			vs[i].Modifications, vs[i].OmittedModifications = d.Modifications, d.OmittedModifications
		}
		emit(out, true, map[string]any{"schema": model.Schema, "versions": vs, "complete": true})
		return
	}
	fmt.Fprintf(out, "versions: %d\nHistory index; use --json for NOTA and provenance, and --full for all modifier details.\n", len(vs))
	for _, v := range vs {
		fmt.Fprintf(out, "%s | %s -> %s | %s | modifiers: %d | NOTA: %t\n", v.ID, v.Start, v.End, v.State, len(v.Modifications), v.Nota != "")
		if o.full {
			renderCaveats(out, model.Document{Modifications: v.Modifications, Nota: v.Nota})
		} else if len(v.Modifications) > 0 {
			link := v.Modifications[0]
			for _, candidate := range v.Modifications {
				if candidate.Date > link.Date {
					link = candidate
				}
			}
			title := []rune(strings.Join(strings.Fields(link.Title), " "))
			if len(title) > 240 {
				title = append(title[:240], []rune("...")...)
			}
			fmt.Fprintf(out, "  modifier: %s | %s | %s | omitted_modifications: %d\n", link.ID, link.Date, string(title), len(v.Modifications)-1)
		}
	}
}
