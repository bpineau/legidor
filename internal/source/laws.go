package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
	legaltext "github.com/bpineau/legidor/internal/text"
)

func (c *Client) law(ctx context.Context, r reference.Reference) (model.Document, error) {
	original := r
	if r.Kind == "named_law" {
		if v, ok := namedLaws[r.Nature+" "+r.Number]; ok {
			r.Kind = "law"
			r.Nature = "loi"
			r.Number = v.number
			if r.Date == "" {
				r.ID = v.id
			}
			original = r
		}
	}
	if r.Date != "" && strings.HasPrefix(r.ID, "JORFTEXT") {
		return model.Document{}, model.Err("CONFLICT", 2, "a JORF identifier denotes the published text, not a consolidated version; use a law number with --date")
	}
	if r.Kind == "eli" {
		u, _ := url.Parse(r.URL)
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 8 {
			return model.Document{}, model.Err("INVALID_REFERENCE", 2, "incomplete ELI URL")
		}
		if regexp.MustCompile(`^[A-Z]{4}\d{7}[A-Z]$`).MatchString(parts[5]) {
			r.Kind = "nor"
			r.Number = parts[5]
		} else if regexp.MustCompile(`^\d{2,4}-\d+$`).MatchString(parts[5]) {
			r.Kind = "law"
			r.Number = parts[5]
			r.Nature = parts[1]
		} else {
			return model.Document{}, model.Err("UNSUPPORTED_REFERENCE", 3, "ELI must contain a NOR or numbered text")
		}
		if strings.HasPrefix(parts[len(parts)-1], "article_") {
			r.Article = strings.TrimPrefix(parts[len(parts)-1], "article_")
		}
	}
	if r.ID == "" {
		fond := "JORF"
		filters := []object{}
		fields := []object{}
		switch r.Kind {
		case "law":
			fields = append(fields, field("NUM", r.Number, "EXACTE"))
		case "nor":
			fields = append(fields, field("NOR", r.Number, "EXACTE"))
		case "named_law":
			q := "finances pour " + r.Number
			if r.Nature == "LFSS" {
				q = "financement de la sécurité sociale pour " + r.Number
			}
			fields = append(fields, field("TITLE", q, "EXACTE"))
		}
		if r.Date != "" {
			fond = "LODA_DATE"
			filters = append(filters, dateFilter(r.Date))
		}
		s, e := c.lfSearch(ctx, fond, fields, filters, 1, 100, false)
		if e != nil {
			return model.Document{}, e
		}
		h, e := unique(s)
		if e != nil {
			return model.Document{}, e
		}
		r.ID = h.ID
	}
	endpoint := "/consult/jorf"
	payload := object{"textCid": r.ID}
	if strings.HasPrefix(r.ID, "LEGITEXT") {
		endpoint = "/consult/legiPart"
		d := r.Date
		if d == "" {
			d = today()
		}
		payload = object{"textId": r.ID, "date": millis(d)}
		r.Date = d
	}
	if strings.HasPrefix(r.ID, "KALITEXT") {
		endpoint = "/consult/kaliText"
		payload = object{"id": r.ID}
	}
	b, p, e := c.JSON(ctx, "POST", c.LFURL+endpoint, payload, true, 24*time.Hour)
	d := model.Document{Schema: model.Schema, Nature: "legislation", Date: r.Date, Provenance: p, Complete: true}
	d.Provenance.Source = "legifrance"
	if e != nil {
		return d, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return d, unexpected("text")
	}
	d.ID = str(raw, "cid")
	if d.ID == "" || strings.HasPrefix(r.ID, "LEGITEXT") {
		d.ID = strings.Split(str(raw, "id"), "_")[0]
	}
	d.Title = str(raw, "title")
	d.Nota = str(raw, "nota")
	d.Publication = date(raw["dateParution"])
	d.Start = date(raw["dateDebutVersion"])
	d.End = date(raw["dateFinVersion"])
	d.Provenance.URL = officialURL(d.ID, r.Date)
	if d.ID != r.ID {
		return d, model.Err("IDENTITY_MISMATCH", 1, "text identifier differs")
	}
	if r.Date != "" {
		if d.Start != "" && !inForce(r.Date, d.Start, d.End) {
			return d, model.Err("VERSION_CONFLICT", 1, "consolidated text is not in force at requested date")
		}
		if d.Start == "" && r.Article == "" {
			d.Warnings = append(d.Warnings, "The source does not provide a validity interval for the table of contents; consult an individual article.")
		}
	}
	if original.Kind == "law" && !containsNumber(d.Title, original.Number) {
		return d, model.Err("IDENTITY_MISMATCH", 1, "law number differs")
	}
	if original.Kind == "law" && !sameNature(original.Nature, str(raw, "nature"), d.Title) {
		return d, model.Err("IDENTITY_MISMATCH", 1, "text nature differs from requested law or decree")
	}
	if original.Kind == "named_law" {
		phrase := "finances pour " + original.Number
		if original.Nature == "LFSS" {
			phrase = "financement de la sécurité sociale pour " + original.Number
		}
		if !sameNature("loi", str(raw, "nature"), d.Title) || !strings.Contains(strings.ToLower(d.Title), phrase) {
			return d, model.Err("IDENTITY_MISMATCH", 1, "named law title differs")
		}
	}
	if r.Kind == "nor" && str(raw, "nor") != r.Number {
		return d, model.Err("IDENTITY_MISMATCH", 1, "NOR differs")
	}
	if original.Kind == "eli" {
		path, _ := url.Parse(original.URL)
		eli := str(raw, "eli")
		if strings.Contains(path.Path, "/article_") {
			path.Path = path.Path[:strings.LastIndex(path.Path, "/")] + "/texte"
		}
		matches := eli == path.Path || str(raw, "alias") == path.Path
		if r.Kind == "law" {
			pieces := strings.Split(strings.Trim(path.Path, "/"), "/")
			year, _ := strconv.Atoi(pieces[2])
			month, _ := strconv.Atoi(pieces[3])
			day, _ := strconv.Atoi(pieces[4])
			matches = containsNumber(d.Title, r.Number) && sameNature(r.Nature, str(raw, "nature"), d.Title) && date(raw["dateTexte"]) == fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
		if !matches {
			return d, model.Err("IDENTITY_MISMATCH", 1, "ELI differs")
		}
	}
	articles := orderedArticles(raw)
	if len(articles) == 0 {
		d.Complete = false
		d.Unsupported = []string{"text_unavailable"}
		return d, nil
	}
	if r.Article != "" {
		var matches []object
		for _, a := range articles {
			if reference.NormalizeNumber(str(a, "num")) == r.Article {
				matches = append(matches, a)
			}
		}
		if len(matches) == 0 {
			return d, model.Err("PART_NOT_FOUND", 1, "article %s not found in this text", r.Article)
		}
		if len(matches) > 1 {
			return d, model.Err("AMBIGUOUS_REFERENCE", 3, "multiple versions of article %s", r.Article)
		}
		a := matches[0]
		d.Parent = d.ID
		d.ID = str(a, "id")
		d.Number = str(a, "num")
		d.Title += " art. " + d.Number
		d.HTML = str(a, "content")
		d.Text, d.Unsupported = legaltext.HTML(d.HTML)
		d.Nota = str(a, "nota")
		d.Provenance.URL = officialURL(d.ID, r.Date)
		d.Start = date(a["dateDebut"])
		d.End = date(a["dateFin"])
		if r.Date != "" && d.Start == "" {
			return d, model.Err("NO_VERSION_AT_DATE", 3, "source does not establish this article's validity interval; consult the original JORF publication for amending provisions")
		}
		if r.Date != "" && !inForce(r.Date, d.Start, d.End) {
			return d, model.Err("VERSION_CONFLICT", 1, "article is not in force at requested date")
		}
		d.Parts = legaltext.Parts(d.Text)
		d.Complete = d.Text != "" && len(d.Unsupported) == 0
		return d, nil
	}
	for _, a := range articles {
		id := str(a, "id")
		d.Items = append(d.Items, model.Link{ID: id, Title: "Article " + str(a, "num"), URL: officialURL(id, r.Date)})
	}
	d.ContentKind = "table_of_contents"
	d.Text = ""
	return d, nil
}

// orderedArticles preserves the source's ordering, including intercalated articles.
func orderedArticles(root object) []object {
	type child struct {
		order   float64
		article bool
		value   object
	}
	var children []child
	for _, a := range array(root, "articles") {
		n, _ := a["intOrdre"].(float64)
		children = append(children, child{n, true, a})
	}
	for _, s := range array(root, "sections") {
		n, _ := s["intOrdre"].(float64)
		children = append(children, child{n, false, s})
	}
	sort.SliceStable(children, func(i, j int) bool { return children[i].order < children[j].order })
	var out []object
	for _, v := range children {
		if v.article {
			out = append(out, v.value)
		} else {
			out = append(out, orderedArticles(v.value)...)
		}
	}
	return out
}
func (c *Client) collection(ctx context.Context, r reference.Reference) (model.Document, error) {
	if r.Date == "" {
		r.Date = today()
	}
	var parent string
	if r.Kind == "range" {
		first := r
		first.Kind = "article"
		first.End = ""
		d, e := c.Get(ctx, first)
		if e != nil {
			return d, e
		}
		parent = d.Parent
	} else {
		if r.Parent != "" {
			parent = r.Parent
		} else {
			if r.Code == "" {
				return model.Document{}, model.Err("USAGE", 2, "section requires its code: legidor get 'CSS section SECTION_ID'")
			}
			s, e := c.lfSearch(ctx, "CODE_DATE", []object{field("TITLE", reference.CodeName(r.Code), "EXACTE")}, []object{dateFilter(r.Date)}, 1, 100, false)
			if e != nil {
				return model.Document{}, e
			}
			h, e := unique(s)
			if e != nil {
				return model.Document{}, e
			}
			parent = h.ID
		}
	}
	b, p, e := c.JSON(ctx, "POST", c.LFURL+"/consult/legi/tableMatieres", object{"textId": parent, "nature": "CODE", "date": r.Date}, true, 24*time.Hour)
	d := model.Document{Schema: model.Schema, ID: parent, Title: reference.CodeName(r.Code), Code: r.Code, Nature: "legislation", Date: r.Date, Provenance: p, Complete: true}
	d.Provenance.Source = "legifrance"
	d.Provenance.URL = officialURL(parent, r.Date)
	if e != nil {
		return d, e
	}
	var root object
	if json.Unmarshal(b, &root) != nil {
		return d, unexpected("table of contents")
	}
	if r.Code == "" {
		r.Code = reference.CodeKey(str(root, "title"))
		d.Code = r.Code
		d.Title = str(root, "title")
		if r.Code == "" {
			return d, model.Err("IDENTITY_UNVERIFIED", 3, "cannot identify the section's parent code")
		}
	}
	if r.Kind == "section" {
		var find func(object) object
		find = func(o object) object {
			if str(o, "id") == r.ID || str(o, "cid") == r.ID {
				return o
			}
			for _, s := range array(o, "sections") {
				if v := find(s); v != nil {
					return v
				}
			}
			return nil
		}
		root = find(root)
		if root == nil {
			return d, model.Err("NOT_FOUND_IN_CORPUS", 1, "section not found in code at date")
		}
		d.ID = r.ID
		d.Title = str(root, "title")
		d.Provenance.URL = officialURL(r.ID, r.Date)
	}
	articles := orderedArticles(root)
	start, end := -1, -1
	if r.Kind == "section" {
		start = 0
		end = len(articles) - 1
	} else {
		for i, a := range articles {
			n := reference.NormalizeNumber(str(a, "num"))
			if n == r.Number {
				start = i
			}
			if n == r.End {
				end = i
			}
		}
	}
	if start < 0 || end < start {
		return d, model.Err("RANGE_BOUND_MISSING", 1, "range bounds absent or reversed in official order")
	}
	if end-start >= 200 {
		return d, model.Err("RESULT_TOO_LARGE", 3, "collection exceeds 200 articles; request a smaller section or range")
	}
	var texts []string
	for _, a := range articles[start : end+1] {
		id := str(a, "id")
		ar := reference.Reference{Kind: "article", ID: id, Code: r.Code, Number: reference.NormalizeNumber(str(a, "num")), Date: r.Date}
		item, e := c.article(ctx, ar, parent)
		if e != nil {
			return d, e
		}
		offset := len(strings.Join(texts, "\n\n"))
		if len(texts) > 0 {
			offset += 2
		}
		chunk := "Article " + item.Number + "\n" + item.Text
		texts = append(texts, chunk)
		d.Parts = append(d.Parts, model.Part{Label: item.Number, Start: offset, End: offset + len(chunk), Parent: -1})
		d.Items = append(d.Items, model.Link{ID: item.ID, Title: item.Title, URL: item.Provenance.URL})
		if !item.Complete {
			d.Complete = false
			d.Unsupported = append(d.Unsupported, item.Unsupported...)
		}
	}
	d.Text = strings.Join(texts, "\n\n")
	return d, nil
}

func sameNature(want, got, title string) bool {
	normalize := func(s string) string {
		return strings.NewReplacer("é", "e", "ê", "e", "É", "e", "Ê", "e").Replace(strings.ToLower(s))
	}
	want = normalize(want)
	got = normalize(got)
	if got != "" {
		return got == want
	}
	return strings.HasPrefix(normalize(title), want+" ")
}
