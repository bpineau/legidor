// Package acceptance holds opt-in live checks against official sources.
package acceptance

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/reference"
	"github.com/bpineau/legidor/internal/source"
	legaltext "github.com/bpineau/legidor/internal/text"
)

func TestLive(t *testing.T) {
	if os.Getenv("LEGIDOR_LIVE") != "1" {
		t.Skip("set LEGIDOR_LIVE=1 to call official APIs")
	}
	cfg, e := source.LoadConfig("production")
	if e != nil {
		t.Fatal(e)
	}
	c := source.NewClient(cfg, cache.New(cfg.CacheDir))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for _, tc := range []struct{ ref, date, code, num, part string }{{"CSS L136-8", "2025-12-31", "CSS", "L136-8", "IV"}, {"CSS L742-1", "2025-12-31", "CSS", "L742-1", ""}, {"CGI 158", "2025-12-31", "CGI", "158", ""}, {"CGI 200 A", "2025-12-31", "CGI", "200 A", ""}, {"LPF L169", "2025-12-31", "LPF", "L169", ""}, {"CGI 150-0 D", "2025-12-31", "CGI", "150-0 D", "11"}, {"CGI 199 novovicies", "2025-12-31", "CGI", "199 novovicies", ""}, {"CMF L224-4", "2025-12-31", "CMF", "L224-4", "I 2°"}, {"BOI-RFPI-DECLA-10", "", "", "", "60"}, {"BOI-RSA-PENS-30-10-20", "", "", "", "200"}, {"LFI 2024 art. 110", "", "", "110", ""}, {"CE 2025-10-09 n° 506041", "", "", "506041", ""}, {"Cass. 3e civ. 2023-10-19 n° 22-15.536", "", "", "22-15.536", ""}} {
		t.Run(tc.ref, func(t *testing.T) {
			r, e := reference.Parse(tc.ref)
			if e != nil {
				t.Fatal(e)
			}
			r.Date = tc.date
			d, e := c.Get(ctx, r)
			if e != nil {
				t.Fatal(e)
			}
			if d.ID == "" || d.Provenance.URL == "" || d.Text == "" {
				t.Fatal("missing evidence metadata or content")
			}
			if tc.code != "" && d.Code != tc.code || tc.num != "" && d.Number != tc.num {
				t.Fatal("wrong identity")
			}
			if tc.part != "" {
				s, e := legaltext.Select(d.Text, d.Parts, tc.part)
				if e != nil || s == "" {
					t.Fatalf("selection failed: %v", e)
				}
			}
		})
	}
	t.Run("official range order", func(t *testing.T) {
		r, _ := reference.Parse("CPCE R322-39..R322-42")
		r.Date = "2025-12-31"
		d, e := c.Get(ctx, r)
		if e != nil {
			t.Fatal(e)
		}
		if !d.Complete || len(d.Items) != 5 {
			t.Fatalf("expected 5 articles including intercalated article, got %d", len(d.Items))
		}
	})
}
