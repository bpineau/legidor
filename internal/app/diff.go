package app

import (
	"github.com/bpineau/legidor/internal/model"
	legaltext "github.com/bpineau/legidor/internal/text"
)

// diffChanges covers all source text, including text preceding recognized parts.
func diffChanges(a, b model.Document) []map[string]string {
	changes := []map[string]string{}
	split := func(d model.Document) (map[string]string, []string, bool) {
		pieces := map[string]string{}
		var labels []string
		roots := []model.Part{}
		for _, p := range d.Parts {
			if p.Parent < 0 && p.End > p.Start && p.End <= len(d.Text) {
				roots = append(roots, p)
			}
		}
		if len(roots) == 0 {
			return nil, nil, false
		}
		if roots[0].Start > 0 {
			pieces["preamble"] = d.Text[:roots[0].Start]
			labels = append(labels, "preamble")
		}
		for _, p := range roots {
			if _, ok := pieces[p.Label]; ok {
				return nil, nil, false
			}
			pieces[p.Label] = d.Text[p.Start:p.End]
			labels = append(labels, p.Label)
		}
		return pieces, labels, true
	}
	left, labels, ok1 := split(a)
	right, other, ok2 := split(b)
	if !ok1 || !ok2 {
		if a.Text != b.Text {
			changes = append(changes, map[string]string{"part": "", "diff": legaltext.Diff(a.Text, b.Text)})
		}
		return changes
	}
	for _, label := range other {
		if _, ok := left[label]; !ok {
			labels = append(labels, label)
		}
	}
	for _, label := range labels {
		if left[label] != right[label] {
			changes = append(changes, map[string]string{"part": label, "diff": legaltext.Diff(left[label], right[label])})
		}
	}
	return changes
}
