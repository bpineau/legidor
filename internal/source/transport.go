package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/model"
)

const maxBody = 16 << 20

// Client centralizes bounded HTTP, OAuth renewal and response-cache policy.
type Client struct {
	Config                            Config
	HTTP                              *http.Client
	Cache                             *cache.Cache
	TokenURL, LFURL, JudiURL, BOFIURL string
	Fresh, Offline                    bool
	mu                                sync.Mutex
	token                             string
	expires                           time.Time
}

func NewClient(cfg Config, c *cache.Cache) *Client {
	prefix := ""
	if cfg.Environment == "sandbox" {
		prefix = "sandbox-"
	}
	return &Client{Config: cfg, Cache: c, HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, TokenURL: "https://" + prefix + "oauth.piste.gouv.fr/api/oauth/token", LFURL: "https://" + prefix + "api.piste.gouv.fr/dila/legifrance/lf-engine-app", JudiURL: "https://" + prefix + "api.piste.gouv.fr/cassation/judilibre/v1.0", BOFIURL: "https://data.economie.gouv.fr/api/explore/v2.1/catalog/datasets/bofip-vigueur/records"}
}

type tokenEntry struct {
	Token   string    `json:"access_token"`
	Expires time.Time `json:"expires"`
}

func (c *Client) tokenKey() string {
	h := sha256.Sum256([]byte(c.Config.ClientID + "\x00" + c.Config.ClientSecret))
	return "oauth:" + c.Config.Environment + ":" + hex.EncodeToString(h[:])
}
func (c *Client) bearer(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && c.token != "" && time.Until(c.expires) > 30*time.Second {
		return c.token, nil
	}
	if !force {
		if b, _, ok := c.Cache.Read(c.tokenKey(), 24*time.Hour); ok {
			var t tokenEntry
			if json.Unmarshal(b, &t) == nil && time.Until(t.Expires) > 30*time.Second {
				c.token = t.Token
				c.expires = t.Expires
				return c.token, nil
			}
		}
	}
	if c.Config.ClientID == "" || c.Config.ClientSecret == "" {
		return "", model.Err("AUTH", 3, "OAuth credentials missing; configure PISTE_CLIENT_ID and PISTE_CLIENT_SECRET, then run legidor auth status")
	}
	v := url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}, "client_id": {c.Config.ClientID}, "client_secret": {c.Config.ClientSecret}}
	req, _ := http.NewRequestWithContext(ctx, "POST", c.TokenURL, strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return "", model.Err("AUTH", 3, "OAuth request failed; check network and environment")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", model.Err("AUTH", 3, "OAuth returned HTTP %d; check credentials and PISTE subscription", resp.StatusCode)
	}
	var t struct {
		Token   string      `json:"access_token"`
		Expires json.Number `json:"expires_in"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t) != nil || t.Token == "" {
		return "", model.Err("AUTH", 3, "invalid OAuth response")
	}
	seconds, e := strconv.ParseFloat(string(t.Expires), 64)
	if e != nil || seconds <= 0 {
		return "", model.Err("AUTH", 3, "invalid OAuth expiry")
	}
	c.token = t.Token
	c.expires = time.Now().Add(time.Duration(seconds * float64(time.Second)))
	b, _ := json.Marshal(tokenEntry{c.token, c.expires})
	if e = c.Cache.Write(c.tokenKey(), b, time.Now()); e != nil {
		return "", model.Err("CACHE", 3, "cannot store private OAuth cache")
	}
	return c.token, nil
}
func (c *Client) JSON(ctx context.Context, method, endpoint string, payload any, auth bool, ttl time.Duration) ([]byte, model.Provenance, error) {
	p := model.Provenance{Environment: c.Config.Environment, Offline: c.Offline}
	var body []byte
	var e error
	if payload != nil {
		body, e = json.Marshal(payload)
		if e != nil {
			return nil, p, model.Err("USAGE", 2, "invalid request")
		}
	}
	key := c.Config.Environment + ":" + method + ":" + endpoint + ":" + string(body)
	if !c.Fresh {
		if b, at, ok := c.Cache.Read(key, ttl); ok {
			p.Cached = true
			p.RetrievedAt = at.UTC().Format(time.RFC3339)
			return b, p, nil
		}
	}
	if c.Offline {
		return nil, p, model.Err("OFFLINE_MISS", 3, "no fresh cached response; retry without --offline")
	}
	token := ""
	if auth {
		token, e = c.bearer(ctx, false)
		if e != nil {
			return nil, p, e
		}
	}
	refreshed := false
	for attempt := 0; attempt < 3; attempt++ {
		req, e := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
		if e != nil {
			return nil, p, model.Err("NETWORK", 3, "invalid source request")
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "legidor/1")
		if auth {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, e := c.HTTP.Do(req)
		if e != nil {
			return nil, p, model.Err("NETWORK", 3, "source request failed or timed out")
		}
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
		resp.Body.Close()
		if resp.StatusCode == 401 && auth && !refreshed {
			refreshed = true
			token, e = c.bearer(ctx, true)
			if e != nil {
				return nil, p, e
			}
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			delay := time.Duration(attempt+1) * time.Second
			if h := resp.Header.Get("Retry-After"); h != "" {
				if n, err := strconv.Atoi(h); err == nil {
					delay = time.Duration(n) * time.Second
				} else if d, err := http.ParseTime(h); err == nil {
					delay = time.Until(d)
				}
			}
			if attempt == 2 || delay > 10*time.Second {
				code := "UPSTREAM"
				if resp.StatusCode == 429 {
					code = "QUOTA"
				}
				return nil, p, model.Err(code, 3, "source returned HTTP %d; retry later (Retry-After respected)", resp.StatusCode)
			}
			if delay < 0 {
				delay = 0
			}
			select {
			case <-ctx.Done():
				return nil, p, model.Err("NETWORK", 3, "request cancelled")
			case <-time.After(delay):
			}
			continue
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, p, model.Err("AUTH", 3, "source returned HTTP %d; check API subscription and environment with legidor auth status", resp.StatusCode)
		}
		if resp.StatusCode == 404 {
			return nil, p, model.Err("NOT_FOUND_IN_CORPUS", 1, "reference not found in this source")
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, p, model.Err("UPSTREAM", 3, "source returned HTTP %d", resp.StatusCode)
		}
		if readErr != nil || len(b) > maxBody {
			return nil, p, model.Err("UNSUPPORTED_CONTENT", 3, "source response incomplete or larger than 16 MiB")
		}
		if !strings.Contains(resp.Header.Get("Content-Type"), "json") || !json.Valid(b) {
			return nil, p, model.Err("UPSTREAM_FORMAT", 3, "source did not return JSON; no content cached")
		}
		var envelope map[string]json.RawMessage
		if json.Unmarshal(b, &envelope) == nil {
			if v := envelope["error"]; len(v) > 0 && string(v) != "null" && string(v) != "false" && string(v) != `""` {
				return nil, p, model.Err("UPSTREAM", 3, "source returned an error envelope")
			}
		}
		at := time.Now().UTC()
		p.RetrievedAt = at.Format(time.RFC3339)
		if e = c.Cache.Write(key, b, at); e != nil {
			return nil, p, model.Err("CACHE", 3, "cannot write private response cache")
		}
		return b, p, nil
	}
	return nil, p, model.Err("AUTH", 3, "authentication retry limit reached")
}
