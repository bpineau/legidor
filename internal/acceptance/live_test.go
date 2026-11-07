// Package acceptance holds opt-in live checks against official sources.
package acceptance

import (
	"context"
	"os"
	"strings"
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

func TestLiveSourceReading(t *testing.T) {
	if os.Getenv("LEGIDOR_LIVE") != "1" {
		t.Skip("set LEGIDOR_LIVE=1 to call official APIs")
	}
	cfg, err := source.LoadConfig("production")
	if err != nil {
		t.Fatal(err)
	}
	c := source.NewClient(cfg, cache.New(cfg.CacheDir))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, tc := range []struct{ ref, date, part string }{
		{"CGI 219", "2026-11-07", "I b"},
		{"CGI 1609 nonies G", "2026-11-07", ""},
		{"BOI-RFPI-TPVIE-20", "", "70"},
		{"BOI-IR-CDHR-10", "", "250"},
		{"CIBS L213-82", "2027-01-01", ""},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			r, err := reference.Parse(tc.ref)
			if err != nil {
				t.Fatal(err)
			}
			r.Date = tc.date
			d, err := c.Get(ctx, r)
			if err != nil {
				t.Fatal(err)
			}
			if !d.Complete {
				t.Fatalf("incomplete: %v", d.Unsupported)
			}
			if tc.part != "" {
				if _, err := legaltext.Select(d.Text, d.Parts, tc.part); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLiveResearchRegressions(t *testing.T) {
	if os.Getenv("LEGIDOR_LIVE") != "1" {
		t.Skip("set LEGIDOR_LIVE=1 to call official APIs")
	}
	cfg, err := source.LoadConfig("production")
	if err != nil {
		t.Fatal(err)
	}
	c := source.NewClient(cfg, cache.New(cfg.CacheDir))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, ref := range []string{"CGI 1649 quater-0 B bis", "loi 2026-534", "loi 2026-470", "décret 2017-372", "CIBS"} {
		t.Run(ref, func(t *testing.T) {
			r, err := reference.Parse(ref)
			if err != nil {
				t.Fatal(err)
			}
			d, err := c.Get(ctx, r)
			if err != nil || d.ID == "" {
				t.Fatalf("%+v %v", d, err)
			}
		})
	}
	t.Run("future article history", func(t *testing.T) {
		r, _ := reference.Parse("CIBS L213-82")
		r.Date = "2026-11-07"
		_, err := c.Get(ctx, r)
		if err == nil || !strings.Contains(err.Error(), "NO_VERSION_AT_DATE") {
			t.Fatalf("%v", err)
		}
		vs, err := c.Versions(ctx, r, "")
		if err != nil || len(vs) == 0 {
			t.Fatalf("%v", err)
		}
	})
	t.Run("BOFiP paragraph after table", func(t *testing.T) {
		r, _ := reference.Parse("BOI-IR-CHR")
		d, err := c.Get(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		s, err := legaltext.Select(d.Text, d.Parts, "220")
		if err != nil || len(legaltext.Omissions(s)) > 0 {
			t.Fatalf("selection: %v", err)
		}
	})
}
