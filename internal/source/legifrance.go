package source

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
	legaltext "github.com/bpineau/legidor/internal/text"
)

type object = map[string]any

func str(m object, k string) string { s, _ := m[k].(string); return s }
func obj(m object, k string) object { o, _ := m[k].(map[string]any); return o }
func array(m object, k string) []object {
	var out []object
	a, _ := m[k].([]any)
	for _, v := range a {
		if o, ok := v.(map[string]any); ok {
			out = append(out, o)
		}
	}
	return out
}
func date(v any) string {
	switch x := v.(type) {
	case float64:
		if x >= 0 || x < 0 {
			t := time.UnixMilli(int64(x)).UTC()
			if t.Year() >= 2999 {
				return ""
			}
			return t.Format("2006-01-02")
		}
	case string:
		if len(x) >= 10 && reference.ValidDate(x[:10]) {
			if x[:4] >= "2999" {
				return ""
			}
			return x[:10]
		}
	}
	return ""
}
func today() string                     { return time.Now().Format("2006-01-02") }
func millis(d string) int64             { t, _ := time.Parse("2006-01-02", d); return t.UnixMilli() }
func inForce(d, start, end string) bool { return start != "" && d >= start && (end == "" || d < end) }
func officialURL(id, d string) string {
	path := "loda/id/"
	switch {
	case strings.HasPrefix(id, "LEGIARTI"):
		path = "codes/article_lc/"
	case strings.HasPrefix(id, "JORF"):
		path = "jorf/id/"
	case strings.HasPrefix(id, "KALI"):
		path = "conv_coll/id/"
	case strings.HasPrefix(id, "CETAT"):
		path = "ceta/id/"
	case strings.HasPrefix(id, "JURI"):
		path = "juri/id/"
	case strings.HasPrefix(id, "CONS"):
		path = "cons/id/"
	case strings.HasPrefix(id, "LEGISCTA"):
		path = "codes/section_lc/"
	}
	u := "https://www.legifrance.gouv.fr/" + path + id
	if d != "" && strings.HasPrefix(id, "LEGI") {
		u += "/" + d
	}
	return u
}
func field(kind, value, mode string) object {
	return object{"typeChamp": kind, "criteres": []any{object{"typeRecherche": mode, "valeur": value, "operateur": "ET"}}, "operateur": "ET"}
}
func dateFilter(d string) object { return object{"facette": "DATE_VERSION", "singleDate": millis(d)} }
func (c *Client) lfSearch(ctx context.Context, fond string, fields, filters []object, page, limit int, article bool) (model.Search, error) {
	pagination := "DEFAUT"
	if article {
		pagination = "ARTICLE"
	}
	body := object{"fond": fond, "recherche": object{"champs": fields, "filtres": filters, "pageNumber": page, "pageSize": limit, "operateur": "ET", "sort": "PERTINENCE", "typePagination": pagination}}
	b, p, e := c.JSON(ctx, "POST", c.LFURL+"/search", body, true, 24*time.Hour)
	s := model.Search{Schema: model.Schema, Hits: []model.Hit{}, Sort: "PERTINENCE", Stability: "Upstream ranking and pagination may change as the corpus is updated.", Provenance: p, Complete: true}
	s.Provenance.Source = "legifrance"
	s.Provenance.URL = "https://www.legifrance.gouv.fr/search/"
	if e != nil {
		return s, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return s, model.Err("UPSTREAM_FORMAT", 3, "invalid search result")
	}
	total, ok := raw["totalResultNumber"].(float64)
	if !ok {
		return s, model.Err("UPSTREAM_FORMAT", 3, "search count missing")
	}
	s.Total = int(total)
	seen := map[string]bool{}
	for _, r := range array(raw, "results") {
		titles := array(r, "titles")
		parent, title := "", ""
		if len(titles) > 0 {
			parent = str(titles[0], "cid")
			if parent == "" {
				parent = str(titles[0], "id")
			}
			title = str(titles[0], "title")
			title, _ = legaltext.HTML(title)
		}
		add := func(x object) {
			id := str(x, "id")
			if id == "" || seen[id] {
				return
			}
			seen[id] = true
			number := str(x, "num")
			if number == "" {
				number = str(x, "title")
			}
			snippet := ""
			if vals, ok := x["values"].([]any); ok && len(vals) > 0 {
				snippet, _ = vals[0].(string)
				snippet, _ = legaltext.HTML(snippet)
			}
			rr := []rune(snippet)
			if len(rr) > 240 {
				snippet = string(rr[:240]) + "..."
			}
			s.Hits = append(s.Hits, model.Hit{ID: id, Title: title, Number: number, Parent: parent, Snippet: snippet, URL: officialURL(id, ""), Start: date(x["dateDebut"]), End: date(x["dateFin"])})
		}
		if article {
			for _, section := range array(r, "sections") {
				for _, x := range array(section, "extracts") {
					add(x)
				}
			}
			for _, x := range array(r, "extracts") {
				add(x)
			}
		} else {
			for _, t := range titles {
				id := str(t, "cid")
				if strings.HasPrefix(fond, "LODA") || strings.HasPrefix(fond, "CODE") {
					id = strings.Split(str(t, "id"), "_")[0]
				}
				if id == "" {
					id = str(t, "id")
				}
				t["id"] = id
				add(t)
			}
		}
	}
	return s, nil
}
func unique(s model.Search) (model.Hit, error) {
	if len(s.Hits) == 0 {
		return model.Hit{}, model.Err("NOT_FOUND_IN_CORPUS", 1, "reference not found in the selected corpus")
	}
	if len(s.Hits) > 1 || s.Total > 1 {
		e := model.Err("AMBIGUOUS_REFERENCE", 3, "multiple candidates; specify an official identifier")
		for _, h := range s.Hits {
			e.Candidates = append(e.Candidates, h.ID+" "+h.Title+" "+h.Number)
		}
		return model.Hit{}, e
	}
	return s.Hits[0], nil
}
func (c *Client) Get(ctx context.Context, r reference.Reference) (model.Document, error) {
	switch r.Kind {
	case "bofip":
		return c.bofip(ctx, r)
	case "decision":
		return c.decision(ctx, r)
	case "article":
		if r.ID == "" {
			if r.Date == "" {
				r.Date = today()
			}
			s, e := c.lfSearch(ctx, "CODE_DATE", []object{field("NUM_ARTICLE", r.Number, "EXACTE")}, []object{{"facette": "NOM_CODE", "valeurs": []string{reference.CodeName(r.Code)}}, dateFilter(r.Date)}, 1, 100, true)
			if e != nil {
				return model.Document{}, e
			}
			h, e := unique(s)
			if e != nil {
				return model.Document{}, e
			}
			if reference.NormalizeNumber(h.Number) != r.Number || strings.TrimSuffix(h.Title, ".") != reference.CodeName(r.Code) {
				return model.Document{}, model.Err("IDENTITY_MISMATCH", 1, "search returned a different code or article")
			}
			r.ID = h.ID
			return c.article(ctx, r, h.Parent)
		}
		return c.article(ctx, r, "")
	case "law", "named_law", "nor", "eli", "text":
		return c.law(ctx, r)
	case "range", "section":
		return c.collection(ctx, r)
	}
	return model.Document{}, model.Err("UNSUPPORTED_REFERENCE", 2, "unsupported reference kind")
}
func (c *Client) article(ctx context.Context, r reference.Reference, expectedParent string) (model.Document, error) {
	ttl := 24 * time.Hour
	if r.Date != "" && r.Date < today() {
		ttl = 30 * 24 * time.Hour
	}
	b, p, e := c.JSON(ctx, "POST", c.LFURL+"/consult/getArticle", object{"id": r.ID}, true, ttl)
	d := model.Document{Schema: model.Schema, Provenance: p, Complete: true}
	d.Provenance.Source = "legifrance"
	if e != nil {
		return d, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return d, model.Err("UPSTREAM_FORMAT", 3, "invalid article response")
	}
	a := obj(raw, "article")
	if a == nil {
		return d, model.Err("NOT_FOUND_IN_CORPUS", 1, "article not found in corpus")
	}
	d.ID = str(a, "id")
	d.Number = reference.NormalizeNumber(str(a, "num"))
	d.Start = date(a["dateDebut"])
	d.End = date(a["dateFin"])
	d.Date = r.Date
	d.State = str(a, "etat")
	d.Nature = "legislation"
	d.Nota = str(a, "nota")
	d.HTML = str(a, "texteHtml")
	d.Text, d.Unsupported = legaltext.HTML(d.HTML)
	if d.HTML == "" {
		d.Text = str(a, "texte")
	}
	titles := array(a, "textTitles")
	if len(titles) == 0 {
		titles = array(obj(a, "context"), "titreTxt")
	}
	for _, t := range titles {
		title := str(t, "titreLong")
		if title == "" {
			title = str(t, "titre")
		}
		title = strings.TrimSuffix(title, ".")
		key := reference.CodeKey(title)
		id := str(t, "cid")
		if id == "" {
			id = str(t, "id")
		}
		if d.Parent == "" || key != "" {
			d.Parent = id
			d.Title = title
			d.Code = key
			d.Publication = date(t["datePubli"])
		}
		if key == r.Code && key != "" {
			break
		}
	}
	if d.ID != r.ID || (r.Number != "" && d.Number != r.Number) || (r.Code != "" && d.Code != r.Code) || (expectedParent != "" && d.Parent != expectedParent) {
		return d, model.Err("IDENTITY_MISMATCH", 1, "consulted article identity differs from requested code, number or parent")
	}
	if d.Number == "" || d.Parent == "" {
		return d, model.Err("IDENTITY_UNVERIFIED", 3, "article number or parent missing")
	}
	if r.Date != "" && d.Start == "" {
		return d, model.Err("NO_VERSION_AT_DATE", 3, "source does not establish this article's validity interval")
	}
	if r.Date != "" && !inForce(r.Date, d.Start, d.End) {
		return d, model.Err("VERSION_CONFLICT", 1, "identifier is not in force at %s (version %s to %s)", r.Date, d.Start, d.End)
	}
	d.Title += " art. " + d.Number
	d.Provenance.URL = officialURL(d.ID, r.Date)
	for _, v := range array(a, "articleVersions") {
		d.Versions = append(d.Versions, model.Version{ID: str(v, "id"), Start: date(v["dateDebut"]), End: date(v["dateFin"]), State: str(v, "etat")})
	}
	sort.SliceStable(d.Versions, func(i, j int) bool { return d.Versions[i].Start < d.Versions[j].Start })
	for _, l := range array(a, "lienModifications") {
		id := str(l, "articleId")
		if id == "" {
			id = str(l, "textCid")
		}
		d.Modifications = append(d.Modifications, model.Link{ID: id, Title: str(l, "textTitle"), Type: str(l, "linkType"), URL: officialURL(id, ""), Date: str(l, "datePubliTexte")})
	}
	d.Parts = legaltext.Parts(d.Text)
	if len(d.Unsupported) > 0 || strings.TrimSpace(d.Text) == "" {
		d.Complete = false
		d.Warnings = append(d.Warnings, "unsupported_content: consult the official source")
	}
	return d, nil
}
func (c *Client) Versions(ctx context.Context, r reference.Reference, since string) ([]model.Version, error) {
	d, e := c.Get(ctx, r)
	if e != nil {
		return nil, e
	}
	if len(d.Versions) == 0 {
		return nil, model.Err("UNSUPPORTED_CONTENT", 3, "version history unavailable for this source")
	}
	var out []model.Version
	for _, v := range d.Versions {
		if since != "" && v.Start < since {
			continue
		}
		rr := reference.Reference{Kind: "article", ID: v.ID, Code: d.Code, Number: d.Number}
		vd, e := c.article(ctx, rr, d.Parent)
		if e != nil {
			return out, e
		}
		v.Nota = vd.Nota
		v.Modifications = vd.Modifications
		v.Provenance = &vd.Provenance
		out = append(out, v)
	}
	return out, nil
}
func (c *Client) AuthStatus(ctx context.Context) (object, error) {
	_, e := c.bearer(ctx, false)
	if e != nil {
		return nil, e
	}
	return object{"schema": model.Schema, "environment": c.Config.Environment, "oauth": "authenticated", "expires_at": c.expires.UTC().Format(time.RFC3339), "api_key_configured": c.Config.APIKey != ""}, nil
}
func unexpected(kind string) error {
	return model.Err("UPSTREAM_FORMAT", 3, "unexpected %s response", fmt.Sprint(kind))
}
