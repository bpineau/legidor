package source

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
	legaltext "github.com/bpineau/legidor/internal/text"
)

func (c *Client) bofip(ctx context.Context, r reference.Reference) (model.Document, error) {
	q := url.Values{"where": {`identifiant_juridique = "` + r.Number + `"`}, "limit": {"2"}}
	b, p, e := c.JSON(ctx, "GET", c.BOFIURL+"?"+q.Encode(), nil, false, 24*time.Hour)
	d := model.Document{Schema: model.Schema, Nature: "administrative_doctrine", Provenance: p, Complete: true}
	d.Provenance.Source = "bofip"
	d.Warnings = []string{"Current doctrine corpus, updated weekly; historical versions and legal applicability are not established."}
	if e != nil {
		return d, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return d, unexpected("BOFiP")
	}
	rows := array(raw, "results")
	if len(rows) == 0 {
		return d, model.Err("NOT_FOUND_IN_CORPUS", 1, "BOI not found in current doctrine corpus")
	}
	if len(rows) != 1 {
		return d, model.Err("AMBIGUOUS_REFERENCE", 3, "multiple BOI records")
	}
	row := rows[0]
	d.ID = str(row, "identifiant_juridique")
	if d.ID != r.Number {
		return d, model.Err("IDENTITY_MISMATCH", 1, "BOI identifier differs")
	}
	d.Title = str(row, "titre")
	d.Start = str(row, "debut_de_validite")
	d.Provenance.URL = str(row, "permalien")
	d.HTML = str(row, "contenu_html")
	if d.HTML == "" {
		return d, model.Err("UNSUPPORTED_CONTENT", 3, "BOI has no HTML paragraph structure")
	}
	d.Text, d.Parts, d.Unsupported = legaltext.BOI(d.HTML)
	if r.Date != "" && r.Date != today() {
		return d, model.Err("NO_VERSION_AT_DATE", 3, "bofip-vigueur cannot establish historical doctrine; current version starts %s; official permalink: %s", d.Start, d.Provenance.URL)
	}
	d.Complete = len(d.Unsupported) == 0 && strings.TrimSpace(d.Text) != ""
	return d, nil
}
func (c *Client) bofipSearch(ctx context.Context, query string, limit, offset int) (model.Search, error) {
	q := url.Values{"where": {"search(" + strconv.Quote(query) + ")"}, "limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}, "select": {"identifiant_juridique,titre,permalien,debut_de_validite"}, "order_by": {"identifiant_juridique"}}
	b, p, e := c.JSON(ctx, "GET", c.BOFIURL+"?"+q.Encode(), nil, false, 24*time.Hour)
	s := model.Search{Schema: model.Schema, Hits: []model.Hit{}, Sort: "identifiant_juridique", Stability: "Weekly current-doctrine corpus; pagination can move after an update.", Provenance: p, Complete: true}
	s.Provenance.Source = "bofip"
	s.Provenance.URL = "https://data.economie.gouv.fr/explore/dataset/bofip-vigueur/"
	if e != nil {
		return s, e
	}
	var raw object
	if json.Unmarshal(b, &raw) != nil {
		return s, unexpected("BOFiP search")
	}
	total, ok := raw["total_count"].(float64)
	if !ok {
		return s, unexpected("BOFiP count")
	}
	s.Total = int(total)
	for _, row := range array(raw, "results") {
		s.Hits = append(s.Hits, model.Hit{ID: str(row, "identifiant_juridique"), Title: strings.TrimSpace(str(row, "titre")), URL: str(row, "permalien"), Start: str(row, "debut_de_validite")})
	}
	return s, nil
}
