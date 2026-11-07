package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bpineau/legidor/internal/cache"
)

func TestTransportRefreshAndCache(t *testing.T) {
	tokens, calls := 0, 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			tokens++
			w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
			return
		}
		calls++
		if calls == 1 {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"text":"public"}`))
	}))
	defer s.Close()
	c := NewClient(Config{ClientID: "fake", ClientSecret: "fake", Environment: "test"}, cache.New(t.TempDir()))
	c.TokenURL = s.URL + "/token"
	for i := 0; i < 2; i++ {
		b, p, e := c.JSON(context.Background(), "POST", s.URL+"/data", map[string]string{"id": "example"}, true, time.Hour)
		if e != nil || !strings.Contains(string(b), "public") {
			t.Fatalf("%s %v", b, e)
		}
		if i == 1 && !p.Cached {
			t.Fatal("cache miss")
		}
	}
	if tokens != 2 || calls != 2 {
		t.Fatalf("tokens=%d calls=%d", tokens, calls)
	}
}
func TestTransportFailures(t *testing.T) {
	for _, kind := range []string{"html", "quota", "offline"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if kind == "quota" {
					w.Header().Set("Retry-After", "120")
					w.WriteHeader(429)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte("<html>not JSON</html>"))
			}))
			defer s.Close()
			c := NewClient(Config{Environment: "test"}, cache.New(t.TempDir()))
			c.Offline = kind == "offline"
			_, _, e := c.JSON(context.Background(), "GET", s.URL, nil, false, time.Hour)
			if e == nil {
				t.Fatal("false success")
			}
			if kind == "offline" && calls != 0 {
				t.Fatal("offline network access")
			}
		})
	}
}
func TestSearchCursorBindsPageSize(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total_count":10,"results":[{"identifiant_juridique":"BOI-EXAMPLE-1","titre":"Example"}]}`))
	}))
	defer s.Close()
	c := NewClient(Config{Environment: "test"}, cache.New(t.TempDir()))
	c.BOFIURL = s.URL
	first, e := c.Search(context.Background(), "example", "bofip", "", "", 1)
	if e != nil || first.Cursor == "" {
		t.Fatal(e)
	}
	if _, e = c.Search(context.Background(), "example", "bofip", "", first.Cursor, 5); e == nil {
		t.Fatal("cursor skipped results after page-size change")
	}
}
