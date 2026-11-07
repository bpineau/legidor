package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/source"
)

func TestCheckRomanSubdivisionFallback(t *testing.T) {
	for _, mode := range []string{"absent", "exact", "consultation missing"} {
		t.Run(mode, func(t *testing.T) {
			exact := mode != "absent"
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/token":
					fmt.Fprint(w, `{"access_token":"fake","expires_in":3600}`)
				case "/search":
					var req map[string]any
					json.NewDecoder(r.Body).Decode(&req)
					b, _ := json.Marshal(req)
					if mode == "consultation missing" && bytes.Contains(b, []byte(`"42"`)) {
						t.Error("retried base article after a positive exact search and failed consultation")
					}
					if !exact && bytes.Contains(b, []byte(`"42 IV"`)) {
						fmt.Fprint(w, `{"totalResultNumber":0,"results":[]}`)
						return
					}
					num := "42"
					if exact {
						num = "42 IV"
					}
					fmt.Fprintf(w, `{"totalResultNumber":1,"results":[{"titles":[{"id":"LEGITEXT000000000001","title":"Code civil"}],"extracts":[{"id":"LEGIARTI000000000001","num":%q}]}]}`, num)
				default:
					if mode == "consultation missing" {
						fmt.Fprint(w, `{}`)
						return
					}
					num := "42"
					if exact {
						num = "42 IV"
					}
					fmt.Fprintf(w, `{"article":{"id":"LEGIARTI000000000001","num":%q,"dateDebut":"2020-01-01","texteHtml":"<p>IV.- Invented clause.</p>","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"Code civil"}]}}`, num)
				}
			}))
			defer s.Close()
			c := source.NewClient(source.Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
			c.LFURL = s.URL
			c.TokenURL = s.URL + "/token"
			var out, diag bytes.Buffer
			code := check(context.Background(), c, "-", options{json: true}, strings.NewReader(`{"ref":"CIV 42 IV","date":"2025-01-01"}`), &out, &diag)
			var result model.Check
			json.Unmarshal(out.Bytes(), &result)
			if mode == "consultation missing" {
				if code != 1 || result.Identity != "not_found_in_corpus" {
					t.Fatalf("false fallback: %d %s", code, &out)
				}
				return
			}
			if code != 0 || result.Document == nil {
				t.Fatalf("%d %s %s", code, &out, &diag)
			}
			if exact {
				if result.Document.Number != "42 IV" || result.PartStatus != "not_requested" {
					t.Fatal(result)
				}
			} else {
				if result.Document.Number != "42" || result.PartStatus != "found" || len(result.Document.Warnings) == 0 {
					t.Fatal(result)
				}
			}
		})
	}
}
