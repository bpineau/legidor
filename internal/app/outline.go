package app

import (
	"strings"

	"github.com/bpineau/legidor/internal/model"
)

const partPreviewLimit = 120
const outlinePreviewBudget = 6000

// outline uses only source text and existing parser boundaries. All labels stay
// visible in source order even when the preview budget is exhausted.
func outline(d model.Document, full bool) model.Document {
	d.Parts = append([]model.Part{}, d.Parts...)
	d.OutlineNavigationOnly = true
	counts := map[string]int{}
	for _, p := range d.Parts {
		counts[p.Label]++
	}
	remaining := outlinePreviewBudget
	for i := range d.Parts {
		p := &d.Parts[i]
		usable := counts[p.Label] == 1
		p.Selectable = &usable
		source := p.Text
		if source == "" && p.Start >= 0 && p.End >= p.Start && p.End <= len(d.Text) {
			end := p.End
			// Prefer the current heading/passage over the body of its first child.
			if i+1 < len(d.Parts) && d.Parts[i+1].Start > p.Start && d.Parts[i+1].Start < end {
				end = d.Parts[i+1].Start
			}
			source = d.Text[p.Start:end]
		}
		preview := []rune(strings.Join(strings.Fields(source), " "))
		limit := partPreviewLimit
		if !full {
			limit = min(limit, remaining)
		}
		if limit < 4 {
			limit = 0
		}
		if len(preview) > 0 && limit == 0 {
			d.OmittedPartPreviews++
		}
		if len(preview) > limit && limit > 0 {
			p.Preview = string(preview[:limit-3]) + "..."
		} else {
			p.Preview = string(preview[:min(len(preview), limit)])
		}
		remaining -= len([]rune(p.Preview))
		p.Text = ""
	}
	if len(d.Parts) == 0 && len(d.Items) == 0 {
		d.Warnings = append(d.Warnings, "No recognized subdivision labels. Use literal --grep or inspect the document; no usable --part selector is established.")
	}
	d.Text = ""
	return d
}
