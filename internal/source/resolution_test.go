package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/reference"
)

func TestCurrentResolutionAcrossDays(t *testing.T) {
	for _, ending := range []bool{false, true} {
		t.Run(fmt.Sprint("known end=", ending), func(t *testing.T) {
			searches, articles := 0, 0
			day := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/token":
					fmt.Fprint(w, `{"access_token":"fake","expires_in":3600}`)
				case "/search":
					searches++
					id := "LEGIARTI000000000001"
					if ending && day.Day() >= 2 {
						id = "LEGIARTI000000000002"
					}
					fmt.Fprintf(w, `{"totalResultNumber":1,"results":[{"titles":[{"id":"LEGITEXT000000000001","title":"Code civil"}],"sections":[{"extracts":[{"id":%q,"num":"1240"}]}]}]}`, id)
				case "/consult/getArticle":
					articles++
					var req struct {
						ID string `json:"id"`
					}
					json.NewDecoder(r.Body).Decode(&req)
					end := "2999-01-01"
					if ending && req.ID == "LEGIARTI000000000001" {
						end = "2026-06-02"
					}
					fmt.Fprintf(w, `{"article":{"id":%q,"num":"1240","dateDebut":"2020-01-01","dateFin":%q,"texteHtml":"<p>Invented content.</p>","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"Code civil"}]}}`, req.ID, end)
				}
			}))
			defer s.Close()
			c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
			c.TokenURL, c.LFURL = s.URL+"/token", s.URL
			c.now = func() time.Time { return day }
			ref, _ := reference.Parse("CIV 1240")
			first, err := c.Get(context.Background(), ref)
			if err != nil {
				t.Fatal(err)
			}
			day = day.Add(24 * time.Hour)
			c.Offline = true
			next, err := c.Get(context.Background(), ref)
			if ending {
				if err == nil {
					t.Fatal("reused expired legal version")
				}
				c.Offline = false
				next, err = c.Get(context.Background(), ref)
				if err != nil || next.ID == first.ID || searches != 2 {
					t.Fatalf("%+v %v searches=%d", next, err, searches)
				}
			} else {
				if err != nil || searches != 1 || articles != 1 || next.Date != "2026-06-02" || next.Provenance.ResolutionDate != "2026-06-01" || !next.Provenance.ResolutionCached || next.Provenance.ResolutionRetrievedAt != first.Provenance.ResolutionRetrievedAt {
					t.Fatalf("%+v %v searches=%d articles=%d", next, err, searches, articles)
				}
				// An explicit date must not use the implicit-current alias.
				dated := ref
				dated.Date = "2026-06-02"
				if _, err := c.Get(context.Background(), dated); err == nil {
					t.Fatal("explicit date reused current alias")
				}
				c.Offline, c.Fresh = false, true
				if _, err := c.Get(context.Background(), ref); err != nil || searches != 2 || articles != 2 {
					t.Fatalf("fresh: %v searches=%d articles=%d", err, searches, articles)
				}
			}
		})
	}
}

func TestResolutionKeepsSearchAge(t *testing.T) {
	c := NewClient(Config{Environment: "test"}, cache.New(t.TempDir()))
	c.Offline = true
	c.now = func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) }
	ref, _ := reference.Parse("CIV 1240")
	at := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	body := object{"fond": "CODE_DATE", "recherche": object{"champs": []object{field("NUM_ARTICLE", ref.Number, "EXACTE")}, "filtres": []object{{"facette": "NOM_CODE", "valeurs": []string{reference.CodeName(ref.Code)}}, dateFilter(c.today())}, "pageNumber": 1, "pageSize": 100, "operateur": "ET", "sort": "PERTINENCE", "typePagination": "ARTICLE"}}
	encoded, _ := json.Marshal(body)
	searchKey := "test:POST:" + c.LFURL + "/search:" + string(encoded)
	c.Cache.Write(searchKey, []byte(`{"totalResultNumber":1,"results":[{"titles":[{"id":"LEGITEXT000000000001","title":"Code civil"}],"sections":[{"extracts":[{"id":"LEGIARTI000000000001","num":"1240"}]}]}]}`), at)
	c.Cache.Write("test:POST:"+c.LFURL+`/consult/getArticle:{"id":"LEGIARTI000000000001"}`, []byte(`{"article":{"id":"LEGIARTI000000000001","num":"1240","dateDebut":"2020-01-01","dateFin":"2999-01-01","texteHtml":"<p>Invented.</p>","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"Code civil"}]}}`), time.Now())
	d, err := c.Get(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if d.Provenance.ResolutionRetrievedAt != at.Format(time.RFC3339) {
		t.Fatal("search age changed")
	}
	_, savedAt, ok := c.Cache.Read("resolution/1:test:"+c.LFURL+":"+ref.Code+":"+ref.Number, currentTTL)
	if !ok || !savedAt.Equal(at) {
		t.Fatalf("alias rejuvenated: %v %t", savedAt, ok)
	}
	c.MaxAge = 24 * time.Hour
	if _, err := c.Get(context.Background(), ref); err == nil {
		t.Fatal("old resolution accepted with fresh article")
	}
}
