package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/cache"
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
