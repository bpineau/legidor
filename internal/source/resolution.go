package source

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

// currentResolution preserves the date and age of the search that selected an
// article. Its key deliberately excludes today's date, unlike dated searches.
type currentResolution struct {
	ID, Parent, Start, End, Date, RetrievedAt string
}

func (c *Client) resolveArticle(ctx context.Context, r reference.Reference) (model.Document, error) {
	current := r.Date == ""
	key := "resolution/1:" + c.Config.Environment + ":" + c.LFURL + ":" + r.Code + ":" + r.Number
	if current {
		r.Date = c.today()
		if !c.Fresh {
			if b, _, ok := c.Cache.Read(key, c.cacheAge(currentTTL)); ok {
				var saved currentResolution
				if json.Unmarshal(b, &saved) == nil && saved.ID != "" && saved.Date <= r.Date && inForce(r.Date, saved.Start, saved.End) {
					pinned := r
					pinned.ID = saved.ID
					d, err := c.article(ctx, pinned, saved.Parent)
					if err == nil {
						resolutionProvenance(&d, saved.Date, saved.RetrievedAt, true)
						return d, nil
					}
					// Re-resolve when newer article metadata contradicts the alias.
					if e, ok := err.(*model.Error); !ok || e.Code != "VERSION_CONFLICT" {
						return d, err
					}
				}
			}
		}
	}
	s, err := c.lfSearch(ctx, "CODE_DATE", []object{field("NUM_ARTICLE", r.Number, "EXACTE")}, []object{{"facette": "NOM_CODE", "valeurs": []string{reference.CodeName(r.Code)}}, dateFilter(r.Date)}, 1, 100, true)
	if err != nil {
		return model.Document{}, err
	}
	h, err := unique(s)
	if err != nil {
		return model.Document{}, err
	}
	if reference.NormalizeNumber(h.Number) != r.Number || strings.TrimSuffix(h.Title, ".") != reference.CodeName(r.Code) {
		return model.Document{}, model.Err("IDENTITY_MISMATCH", 1, "search returned a different code or article")
	}
	r.ID = h.ID
	d, err := c.article(ctx, r, h.Parent)
	if err != nil {
		return d, err
	}
	resolutionProvenance(&d, r.Date, s.Provenance.RetrievedAt, s.Provenance.Cached)
	if current {
		saved := currentResolution{d.ID, d.Parent, d.Start, d.End, r.Date, s.Provenance.RetrievedAt}
		b, _ := json.Marshal(saved)
		// Never rejuvenate an alias built from an already cached search.
		if at, err := time.Parse(time.RFC3339, saved.RetrievedAt); err == nil {
			_ = c.Cache.Write(key, b, at)
		}
	}
	return d, nil
}

func resolutionProvenance(d *model.Document, date, retrieved string, cached bool) {
	d.Provenance.ResolutionDate = date
	d.Provenance.ResolutionRetrievedAt = retrieved
	d.Provenance.ResolutionCached = cached
}
