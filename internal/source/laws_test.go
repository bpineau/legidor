package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

func TestLawDoesNotConfirmWrongNatureOrDate(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
			return
		}
		w.Write([]byte(`{"cid":"JORFTEXT000000000001","title":"Decret n° 2025-999","nature":"DECRET","dateParution":"2025-01-01","dateDebutVersion":"2025-01-01","articles":[{"id":"JORFARTI000000000001","num":"1","content":"<p>Invented content.</p>"}]}`))
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.LFURL = s.URL
	c.TokenURL = s.URL + "/token"
	for _, r := range []reference.Reference{{Kind: "text", ID: "JORFTEXT000000000001", Date: "2020-01-01"}, {Kind: "law", ID: "JORFTEXT000000000001", Number: "2025-999", Nature: "loi", Article: "1"}} {
		if d, e := c.Get(context.Background(), r); e == nil {
			t.Fatalf("false success %+v", d)
		}
	}
}
func TestEmptyBOIIncomplete(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"identifiant_juridique":"BOI-EXAMPLE-1","contenu_html":"<div></div>"}]}`))
	}))
	defer s.Close()
	c := NewClient(Config{Environment: "test"}, cache.New(t.TempDir()))
	c.BOFIURL = s.URL
	d, e := c.Get(context.Background(), reference.Reference{Kind: "bofip", Number: "BOI-EXAMPLE-1"})
	if e == nil && d.Complete {
		t.Fatal("empty doctrine complete")
	}
}
func TestMissingArticleIntervalIsIndeterminate(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
			return
		}
		w.Write([]byte(`{"id":"LEGITEXT000000000002_31-12-2025","cid":"JORFTEXT000000000001","title":"LOI n° 2025-999","nature":"LOI","dateDebutVersion":"2025-12-31","dateFinVersion":"2026-01-01","articles":[{"id":"LEGIARTI000000000001","num":"1","dateDebut":32472144000000,"dateFin":32472144000000,"content":"<p>Invented content.</p>"}]}`))
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.LFURL = s.URL
	c.TokenURL = s.URL + "/token"
	r := reference.Reference{Kind: "text", ID: "LEGITEXT000000000002", Date: "2025-12-31", Article: "1"}
	_, e := c.Get(context.Background(), r)
	if e == nil || !strings.Contains(e.Error(), "NO_VERSION_AT_DATE") {
		t.Fatalf("%v", e)
	}
}

func TestNamedLawCandidateFiltering(t *testing.T) {
	r := reference.Reference{Kind: "named_law", Nature: "LFI", Number: "2031"}
	s := model.Search{Total: 3, Complete: true, Hits: []model.Hit{
		{ID: "law", Title: "LOI n° 2031-123 du 3 janvier 2031 de finances pour 2031 (1)"},
		{ID: "order", Title: "Arrêté relatif à la loi de finances pour 2031"},
		{ID: "amendment", Title: "LOI n° 2031-456 du 4 mars 2031 modifiant la loi de finances pour 2031"},
	}}
	filtered := namedLawCandidates(s, r)
	if len(filtered.Hits) != 1 || filtered.Total != 1 || filtered.Hits[0].ID != "law" {
		t.Fatal(filtered)
	}
	s.Hits = append(s.Hits, model.Hit{ID: "correction", Title: "LOI n° 2031-123 du 3 janvier 2031 de finances pour 2031 (rectificatif)"})
	s.Total++
	if _, err := unique(namedLawCandidates(s, r)); err == nil {
		t.Fatal("rectification ambiguity hidden")
	}
	s.Total = 101
	if _, err := unique(namedLawCandidates(s, r)); err == nil {
		t.Fatal("unseen candidates ignored")
	}
}

func TestDecisionRetainsHTMLLinks(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
			return
		}
		w.Write([]byte(`{"text":{"id":"JURITEXT000000000001","titre":"Invented decision","texteHtml":"<p>Reasons.</p><a href=\"https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000000000001\">Article</a>"}}`))
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.LFURL = s.URL
	c.TokenURL = s.URL + "/token"
	d, err := c.Get(context.Background(), reference.Reference{Kind: "decision", ID: "JURITEXT000000000001"})
	if err != nil || d.HTML == "" || !strings.Contains(d.HTML, "href=") {
		t.Fatalf("%+v %v", d, err)
	}
}

func TestNumberedLawFiltersNatureBeforeUniqueness(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
		case "/search":
			w.Write([]byte(`{"totalResultNumber":2,"results":[{"titles":[{"cid":"JORFTEXT000000000001","title":"LOI n° <mark>2031-987</mark> du 1 janvier 2031"}]},{"titles":[{"cid":"JORFTEXT000000000002","title":"Décision n° 2031-987"}]}]}`))
		default:
			w.Write([]byte(`{"cid":"JORFTEXT000000000001","title":"LOI n° 2031-987 du 1 janvier 2031","nature":"LOI","articles":[{"id":"JORFARTI000000000001","num":"1","content":"<p>Invented.</p>"}]}`))
		}
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.LFURL = s.URL
	c.TokenURL = s.URL + "/token"
	r, _ := reference.Parse("loi 2031-987")
	d, err := c.Get(context.Background(), r)
	if err != nil || d.ID != "JORFTEXT000000000001" {
		t.Fatalf("%+v %v", d, err)
	}
}

func TestCodeAndSectionIndexDoesNotFetchArticles(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
		case "/search":
			w.Write([]byte(`{"totalResultNumber":1,"results":[{"titles":[{"id":"LEGITEXT000000000001","title":"Code civil"}]}]}`))
		case "/consult/legi/tableMatieres":
			w.Write([]byte(`{"id":"LEGITEXT000000000001","title":"Code civil","sections":[{"id":"LEGISCTA000000000001","title":"First section","intOrdre":1,"articles":[{"id":"LEGIARTI000000000001","num":"42","intOrdre":1}]}]}`))
		default:
			t.Errorf("unexpected article fetch: %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.LFURL = s.URL
	c.TokenURL = s.URL + "/token"
	for _, r := range []reference.Reference{{Kind: "code", Code: "CIV"}, {Kind: "section_index", Code: "CIV", ID: "LEGISCTA000000000001"}} {
		d, err := c.Get(context.Background(), r)
		if err != nil || d.ContentKind != "table_of_contents" || len(d.Items) != 1 {
			t.Fatalf("%+v %v", d, err)
		}
	}
}
