package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/model"
	legaltext "github.com/bpineau/legidor/internal/text"
)

func TestOutlineKeepsLabelsAndSourceCaveatsWithinBudget(t *testing.T) {
	var body strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&body, "%d. %s\n", i, strings.Repeat("évidence ", 50))
	}
	raw := body.String()
	parts := legaltext.Parts(raw)
	d := model.Document{Text: raw, Parts: parts, Complete: false, Unsupported: []string{"image"}, Nota: "Source commencement caveat", Warnings: []string{"Existing warning"}}
	got := outline(d, false)
	if got.Text != "" || got.Complete || !got.OutlineNavigationOnly || got.OmittedPartPreviews == 0 || got.Nota != d.Nota || len(got.Unsupported) != 1 || len(got.Warnings) != 1 || len(got.Parts) != len(parts) {
		t.Fatalf("lost metadata: %+v", got)
	}
	total := 0
	for i, p := range got.Parts {
		if p.Label != parts[i].Label || p.Text != "" || p.Selectable == nil || !*p.Selectable {
			t.Fatalf("lost label: %+v", p)
		}
		total += len([]rune(p.Preview))
		if len([]rune(p.Preview)) > partPreviewLimit {
			t.Fatal("long preview")
		}
	}
	if total > outlinePreviewBudget {
		t.Fatalf("unbounded %d", total)
	}
	full := outline(model.Document{Text: raw, Parts: legaltext.Parts(raw), Complete: true}, true)
	if full.OmittedPartPreviews != 0 || !full.Complete {
		t.Fatal("full outline incomplete")
	}
}

func TestOutlinePreservesAmbiguousAndUnrecognizedStructure(t *testing.T) {
	got := outline(model.Document{Text: "No recognized labels.", Complete: true}, false)
	if len(got.Parts) != 0 || len(got.Warnings) == 0 || !got.Complete {
		t.Fatalf("%+v", got)
	}
	got = outline(model.Document{Complete: true, Parts: []model.Part{{Label: "same", Text: "First source passage"}, {Label: "same", Text: "Second source passage"}}}, false)
	for _, p := range got.Parts {
		if p.Selectable == nil || *p.Selectable || p.Preview == "" {
			t.Fatalf("%+v", p)
		}
	}
	raw, parts, unsupported := legaltext.BOI(`<h2>Actual source heading</h2><p class="numero-de-paragraphe-western">10</p><p>Some source content.</p><p class="numero-de-paragraphe-western">20</p><img src="example.png">`)
	got = outline(model.Document{Text: raw, Parts: parts, Unsupported: unsupported, Complete: false}, false)
	if len(got.Parts) != 2 || got.Parts[0].Label != "10" || !strings.Contains(got.Parts[0].Preview, "source content") || len(got.Unsupported) == 0 || got.Complete {
		t.Fatalf("%+v", got)
	}
}
