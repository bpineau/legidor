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

func TestArticleIdentity(t *testing.T) {
	for _, wrong := range []bool{false, true} {
		t.Run(map[bool]string{false: "correct", true: "wrong code"}[wrong], func(t *testing.T) {
			code := "Code de la sécurité sociale"
			if wrong {
				code = "Code civil"
			}
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/token":
					w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
				case "/search":
					w.Write([]byte(`{"totalResultNumber":1,"results":[{"titles":[{"id":"LEGITEXT000000000001","cid":"LEGITEXT000000000001","title":"Code de la sécurité sociale"}],"sections":[{"extracts":[{"id":"LEGIARTI000000000001","num":"L136-8","title":"L136-8"}]}]}]}`))
				case "/consult/getArticle":
					w.Write([]byte(`{"article":{"id":"LEGIARTI000000000001","num":"L136-8","dateDebut":1735689600000,"dateFin":32472144000000,"texteHtml":"<p>I.- Invented test content.</p>","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"` + code + `"}]}}`))
				}
			}))
			defer s.Close()
			c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
			c.TokenURL = s.URL + "/token"
			c.LFURL = s.URL
			ref, _ := reference.Parse("CSS L136-8")
			ref.Date = "2025-12-31"
			d, e := c.Get(context.Background(), ref)
			if wrong {
				if e == nil || !strings.Contains(e.Error(), "IDENTITY_MISMATCH") {
					t.Fatalf("false identity: %+v %v", d, e)
				}
			} else if e != nil || d.Code != "CSS" {
				t.Fatalf("%+v %v", d, e)
			}
		})
	}
}
func TestLODASearchKeepsConsolidatedIdentity(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"fake","expires_in":3600}`))
			return
		}
		w.Write([]byte(`{"totalResultNumber":1,"results":[{"titles":[{"id":"LEGITEXT000000000002_31-12-2025","cid":"JORFTEXT000000000001","title":"LOI n° 2025-999"}]}]}`))
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.LFURL = s.URL
	c.TokenURL = s.URL + "/token"
	result, e := c.lfSearch(context.Background(), "LODA_DATE", nil, nil, 1, 10, false)
	if e != nil || result.Hits[0].ID != "LEGITEXT000000000002" {
		t.Fatalf("%+v %v", result, e)
	}
}
