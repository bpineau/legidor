package source

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
	legaltext "github.com/bpineau/legidor/internal/text"
)

func (c *Client) judiSearch(ctx context.Context, query string, limit, page int, decisionDate string) (model.Search, error) {
	q := url.Values{"query": {query}, "operator": {"exact"}, "page_size": {strconv.Itoa(min(limit, 50))}, "page": {strconv.Itoa(page)}, "sort": {"score"}, "order": {"desc"}}
	if decisionDate != "" {
		q.Set("date_start", decisionDate)
		q.Set("date_end", decisionDate)
	}
	b, p, e := c.JSON(ctx, "GET", c.JudiURL+"/search?"+q.Encode(), nil, true, currentTTL)
	s := model.Search{Schema: model.Schema, Hits: []model.Hit{}, Sort: "score desc", Stability: "Judilibre defaults to Cour de cassation; rankings can change as the corpus is updated.", Provenance: p, Complete: true}
	s.Provenance.Source = "judilibre"
	s.Provenance.URL = "https://www.courdecassation.fr/recherche-judilibre"
	if e != nil {
		return s, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return s, unexpected("Judilibre search")
	}
	if n, ok := raw["total"].(float64); ok {
		s.Total = int(n)
	} else {
		return s, unexpected("Judilibre count")
	}
	if relaxed, _ := raw["relaxed"].(bool); relaxed {
		s.Stability += " Search was relaxed by the source."
	}
	for _, r := range array(raw, "results") {
		id := str(r, "id")
		s.Hits = append(s.Hits, model.Hit{ID: "JUDI:" + id, Title: str(r, "jurisdiction") + " " + str(r, "chamber") + " " + str(r, "decision_date"), Number: str(r, "number"), URL: "https://www.courdecassation.fr/decision/" + id})
	}
	return s, nil
}
func (c *Client) decision(ctx context.Context, r reference.Reference) (model.Document, error) {
	if r.Date != "" {
		return model.Document{}, model.Err("USAGE", 2, "decisions have a decision date, not a version-at-date; omit --date")
	}
	if r.Code == "JUDILIBRE" {
		return c.judi(ctx, r)
	}
	if r.ID == "" {
		fond := r.Code
		if fond == "" {
			fond = "CETAT"
		}
		filters := []object{}
		if r.DecisionDate != "" {
			filters = append(filters, object{"facette": "DATE_DECISION", "dates": object{"start": r.DecisionDate, "end": r.DecisionDate}})
		}
		s, e := c.lfSearch(ctx, fond, []object{field("ALL", r.Number, "EXACTE")}, filters, 1, 100, false)
		if e != nil {
			return model.Document{}, e
		}
		if s.Total > 100 {
			return model.Document{}, model.Err("AMBIGUOUS_REFERENCE", 3, "more than 100 decision candidates; refine the reference")
		}
		var candidates []model.Hit
		for _, h := range s.Hits {
			if strings.HasPrefix(r.Number, "ECLI:") || containsNumber(h.Title, r.Number) {
				candidates = append(candidates, h)
			}
		}
		s.Hits = candidates
		s.Total = len(candidates)
		h, e := unique(s)
		if e != nil {
			return model.Document{}, e
		}
		r.ID = h.ID
	}
	endpoint := "/consult/juri"
	b, p, e := c.JSON(ctx, "POST", c.LFURL+endpoint, object{"textId": r.ID}, true, historicalTTL)
	d := model.Document{Schema: model.Schema, Nature: "case_law", Provenance: p, Complete: true}
	d.Provenance.Source = "legifrance"
	if e != nil {
		return d, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return d, unexpected("decision")
	}
	a := obj(raw, "text")
	if a == nil {
		return d, model.Err("NOT_FOUND_IN_CORPUS", 1, "decision not found in corpus")
	}
	d.ID = str(a, "id")
	d.Title = str(a, "titre")
	d.Number = str(a, "num")
	d.Publication = date(a["dateTexte"])
	d.HTML = str(a, "texteHtml")
	d.Text, d.Unsupported = legaltext.HTML(d.HTML)
	if d.Text == "" {
		d.Text = str(a, "texte")
	}
	d.Provenance.URL = officialURL(d.ID, "")
	numberMatches := r.Number == "" || str(a, "ecli") == r.Number || containsNumber(d.Number, r.Number)
	if strings.HasSuffix(r.Number, " QPC") {
		numberMatches = d.Number == strings.TrimSuffix(r.Number, " QPC") && strings.Contains(d.Title, "QPC")
	}
	if d.ID != r.ID || !numberMatches || (r.DecisionDate != "" && d.Publication != r.DecisionDate) {
		return d, model.Err("IDENTITY_MISMATCH", 1, "decision number, ECLI or date differs")
	}
	if r.Nature == "CE" && !strings.HasPrefix(str(a, "juridiction"), "Conseil d'") {
		return d, model.Err("IDENTITY_MISMATCH", 1, "decision is not from the Conseil d'Etat")
	}
	d.Complete = d.Text != "" && len(d.Unsupported) == 0
	return d, nil
}
func containsNumber(s, n string) bool {
	return regexp.MustCompile(`(^|[^0-9A-Za-z])` + regexp.QuoteMeta(n) + `([^0-9A-Za-z]|$)`).MatchString(s)
}
func (c *Client) judi(ctx context.Context, r reference.Reference) (model.Document, error) {
	if r.ID == "" {
		s, e := c.judiSearch(ctx, r.Number, 50, 0, r.DecisionDate)
		if e != nil {
			return model.Document{}, e
		}
		if s.Total > 50 {
			return model.Document{}, model.Err("AMBIGUOUS_REFERENCE", 3, "too many decision candidates; specify date or identifier")
		}
		var hits []model.Hit
		for _, h := range s.Hits {
			if h.Number == r.Number || strings.HasPrefix(r.Number, "ECLI:") {
				hits = append(hits, h)
			}
		}
		s.Hits = hits
		s.Total = len(hits)
		h, e := unique(s)
		if e != nil {
			return model.Document{}, e
		}
		r.ID = strings.TrimPrefix(h.ID, "JUDI:")
	}
	b, p, e := c.JSON(ctx, "GET", c.JudiURL+"/decision?id="+url.QueryEscape(r.ID), nil, true, historicalTTL)
	d := model.Document{Schema: model.Schema, Nature: "case_law", Provenance: p, Complete: true}
	d.Provenance.Source = "judilibre"
	if e != nil {
		return d, e
	}
	var a object
	if json.Unmarshal(b, &a) != nil {
		return d, unexpected("Judilibre decision")
	}
	d.ID = "JUDI:" + str(a, "id")
	d.Number = str(a, "number")
	d.Publication = str(a, "decision_date")
	d.Title = str(a, "jurisdiction") + " " + str(a, "chamber") + " " + d.Publication + " n° " + d.Number
	d.Text = str(a, "text")
	d.Provenance.URL = "https://www.courdecassation.fr/decision/" + str(a, "id")
	d.Provenance.UpdatedAt = str(a, "update_date")
	if str(a, "id") != r.ID || (r.Number != "" && r.Number != d.Number && r.Number != str(a, "ecli")) || (r.DecisionDate != "" && r.DecisionDate != d.Publication) {
		return d, model.Err("IDENTITY_MISMATCH", 1, "Judilibre identity differs")
	}
	if strings.HasPrefix(r.Nature, "CASS") && str(a, "jurisdiction") != "cc" {
		return d, model.Err("IDENTITY_MISMATCH", 1, "decision is not from the Cour de cassation")
	}
	if r.Nature != "" && r.Nature != "CASS." {
		chambers := map[string]string{"CASS. COM.": "comm", "CASS. SOC.": "soc", "CASS. CRIM.": "cr", "CASS. 1RE CIV.": "civ1", "CASS. 1E CIV.": "civ1", "CASS. 2E CIV.": "civ2", "CASS. 3E CIV.": "civ3"}
		want, ok := chambers[r.Nature]
		if !ok {
			return d, model.Err("UNSUPPORTED_REFERENCE", 3, "unrecognized Cour de cassation chamber; use a returned identifier")
		}
		got := str(a, "chamber")
		if want == "comm" && got == "com" {
			got = "comm"
		}
		if want != got {
			return d, model.Err("IDENTITY_MISMATCH", 1, "decision chamber differs")
		}
	}
	partial, known := a["partial"].(bool)
	d.Complete = known && !partial && d.Text != ""
	if !d.Complete {
		d.Warnings = append(d.Warnings, "Source supplies a partial decision, or does not establish completeness.")
	}
	zones := obj(a, "zones")
	var names []string
	for name := range zones {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var segments []string
		for _, z := range array(zones, name) {
			start, ok1 := z["start"].(float64)
			end, ok2 := z["end"].(float64)
			if !ok1 || !ok2 {
				continue
			}
			units := utf16.Encode([]rune(d.Text))
			if start >= 0 && end >= start && int(start) < len(units) {
				segments = append(segments, string(utf16.Decode(units[int(start):min(int(end), len(units))])))
			}
		}
		if len(segments) > 0 {
			d.Parts = append(d.Parts, model.Part{Label: name, Text: strings.Join(segments, "\n"), Parent: -1})
		}
	}
	return d, nil
}
