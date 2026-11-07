package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bpineau/legidor/internal/cache"
)

func TestMaxAgeRefreshDoesNotRefreshOAuth(t *testing.T) {
	tokens, calls := 0, 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			tokens++
			fmt.Fprint(w, `{"access_token":"fake","expires_in":3600}`)
			return
		}
		calls++
		fmt.Fprint(w, `{"value":"updated"}`)
	}))
	defer s.Close()
	c := NewClient(Config{Environment: "test", ClientID: "fake", ClientSecret: "fake"}, cache.New(t.TempDir()))
	c.TokenURL = s.URL + "/token"
	ctx := context.Background()
	if _, err := c.bearer(ctx, false); err != nil {
		t.Fatal(err)
	}
	// Exercise the disk token cache, not only the in-memory copy.
	c.token = ""
	if err := c.Cache.Write("test:GET:"+s.URL+"/data:", []byte(`{"value":"old"}`), time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	c.MaxAge = 24 * time.Hour
	_, p, err := c.JSON(ctx, "GET", s.URL+"/data", nil, true, currentTTL)
	if err != nil || p.Cached || calls != 1 || tokens != 1 {
		t.Fatalf("%+v %v calls=%d tokens=%d", p, err, calls, tokens)
	}
	_, p, err = c.JSON(ctx, "GET", s.URL+"/data", nil, true, currentTTL)
	if err != nil || !p.Cached || calls != 1 {
		t.Fatalf("recent response: %+v %v calls=%d", p, err, calls)
	}
	c.MaxAge, c.Fresh = 0, true
	_, p, err = c.JSON(ctx, "GET", s.URL+"/data", nil, true, currentTTL)
	if err != nil || p.Cached || calls != 2 || tokens != 1 {
		t.Fatalf("fresh: %+v %v calls=%d tokens=%d", p, err, calls, tokens)
	}
}

func TestResponseCacheAge(t *testing.T) {
	for _, tc := range []struct {
		name          string
		age, ttl, max time.Duration
		hit           bool
	}{
		{"current", 29 * 24 * time.Hour, currentTTL, 0, true},
		{"expired current", 31 * 24 * time.Hour, currentTTL, 0, false},
		{"historical", 89 * 24 * time.Hour, historicalTTL, 0, true},
		{"expired historical", 91 * 24 * time.Hour, historicalTTL, 0, false},
		{"strict", 48 * time.Hour, currentTTL, 24 * time.Hour, false},
		{"recent", time.Hour, currentTTL, 24 * time.Hour, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(Config{Environment: "test"}, cache.New(t.TempDir()))
			c.Offline, c.MaxAge = true, tc.max
			at := time.Now().Add(-tc.age)
			if err := c.Cache.Write("test:GET:https://example.invalid:", []byte(`{}`), at); err != nil {
				t.Fatal(err)
			}
			_, p, err := c.JSON(context.Background(), "GET", "https://example.invalid", nil, false, tc.ttl)
			if (err == nil) != tc.hit || p.Cached != tc.hit {
				t.Fatalf("%+v %v", p, err)
			}
			if tc.hit && p.RetrievedAt != at.UTC().Format(time.RFC3339) {
				t.Fatal("rejuvenated cache")
			}
		})
	}
}
