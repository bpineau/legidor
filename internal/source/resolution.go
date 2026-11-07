package source

import (
	"context"
	"encoding/json"
	"sort"
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
	if len(s.Hits) == 0 {
		if s.Total != 0 {
			return model.Document{}, model.Err("UPSTREAM_FORMAT", 3, "dated search reports results without usable article hits")
		}
		d, err := c.articleHistoryAnchor(ctx, r)
		if err != nil {
			return d, err
		}
		e := model.Err("NO_VERSION_AT_DATE", 1, "article exists but no version was found at %s; available version starts %s", r.Date, d.Start)
		e.Next = "Use legidor versions with the same reference, or get with an explicit --date."
		for _, v := range d.Versions {
			if v.Start != "" && (v.End == "" || v.End > v.Start) {
				e.Candidates = append(e.Candidates, v.ID+" "+v.Start+" -> "+v.End+" "+v.State)
			}
		}
		return d, e
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

// articleHistoryAnchor finds a known version without claiming it is in force.
// CODE_ETAT may return several versions and section IDs as parents. Identity is
// checked on the consultation and every candidate must belong to its history.
func (c *Client) articleHistoryAnchor(ctx context.Context, r reference.Reference) (model.Document, error) {
	s, err := c.lfSearch(ctx, "CODE_ETAT", []object{field("NUM_ARTICLE", r.Number, "EXACTE")}, []object{{"facette": "NOM_CODE", "valeurs": []string{reference.CodeName(r.Code)}}}, 1, 100, true)
	if err != nil {
		return model.Document{}, err
	}
	if len(s.Hits) == 0 {
		if s.Total != 0 {
			return model.Document{}, model.Err("UPSTREAM_FORMAT", 3, "history search reports results without usable article hits")
		}
		return model.Document{}, absentArticle("article not found in dated or all-state code searches")
	}
	if !searchExhausted(s) {
		return model.Document{}, model.Err("AMBIGUOUS_REFERENCE", 3, "article history search is incomplete")
	}
	for _, h := range s.Hits {
		if reference.NormalizeNumber(h.Number) != r.Number || strings.TrimSuffix(h.Title, ".") != reference.CodeName(r.Code) {
			return model.Document{}, model.Err("IDENTITY_MISMATCH", 1, "history search returned a different code or number")
		}
	}
	sort.SliceStable(s.Hits, func(i, j int) bool { return s.Hits[i].Start > s.Hits[j].Start })
	r.Date = ""
	var d model.Document
	known := map[string]bool{}
	for _, h := range s.Hits {
		// Withdrawn future versions can retain an end preceding their start.
		// They never supply a usable validity interval.
		if d.ID != "" && h.Start != "" && h.End != "" && h.End <= h.Start {
			continue
		}
		if known[h.ID] {
			continue
		}
		r.ID = h.ID
		candidate, err := c.article(ctx, r, "")
		if err != nil {
			// Search may label an annex article with the main code's title.
			// Reject that candidate using the consulted identity, not its label.
			if p, ok := err.(*model.Error); ok && p.Code == "IDENTITY_MISMATCH" && candidate.ID == r.ID && candidate.Number == r.Number && candidate.Code != "" && candidate.Code != r.Code {
				continue
			}
			return candidate, err
		}
		if d.ID != "" {
			return d, model.Err("AMBIGUOUS_REFERENCE", 3, "search candidates do not share a verified article history")
		}
		d = candidate
		known[d.ID] = true
		for _, v := range d.Versions {
			known[v.ID] = true
		}
	}
	if d.ID == "" {
		return d, absentArticle("no article with the requested code and number was found")
	}
	return d, nil
}

func absentArticle(message string) *model.Error {
	e := model.Err("NOT_FOUND_IN_CORPUS", 1, "%s", message)
	e.ExactArticleAbsent = true
	return e
}

func resolutionProvenance(d *model.Document, date, retrieved string, cached bool) {
	d.Provenance.ResolutionDate = date
	d.Provenance.ResolutionRetrievedAt = retrieved
	d.Provenance.ResolutionCached = cached
}
