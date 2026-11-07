// Package cache provides private, expiring, atomic disk storage.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Cache struct{ Dir string }
type entry struct {
	Retrieved time.Time       `json:"retrieved"`
	Data      json.RawMessage `json:"data"`
}

func New(dir string) *Cache { return &Cache{Dir: dir} }
func (c *Cache) path(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.Dir, hex.EncodeToString(h[:])+".json")
}
func (c *Cache) Read(key string, ttl time.Duration) ([]byte, time.Time, bool) {
	b, e := os.ReadFile(c.path(key))
	if e != nil {
		return nil, time.Time{}, false
	}
	var v entry
	if json.Unmarshal(b, &v) != nil || time.Since(v.Retrieved) > ttl || v.Retrieved.After(time.Now().Add(time.Minute)) {
		return nil, time.Time{}, false
	}
	return v.Data, v.Retrieved, true
}
func (c *Cache) Write(key string, data []byte, at time.Time) error {
	if !json.Valid(data) {
		return os.ErrInvalid
	}
	if e := os.MkdirAll(c.Dir, 0700); e != nil {
		return e
	}
	if e := os.Chmod(c.Dir, 0700); e != nil {
		return e
	}
	b, e := json.Marshal(entry{at, data})
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(c.Dir, ".write-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), c.path(key))
}
func (c *Cache) Stats() (int, int64, error) {
	es, e := os.ReadDir(c.Dir)
	if os.IsNotExist(e) {
		return 0, 0, nil
	}
	if e != nil {
		return 0, 0, e
	}
	n := 0
	var size int64
	for _, v := range es {
		if v.IsDir() || filepath.Ext(v.Name()) != ".json" {
			continue
		}
		i, e := v.Info()
		if e != nil {
			return n, size, e
		}
		n++
		size += i.Size()
	}
	return n, size, nil
}
func (c *Cache) Clear() error {
	es, e := os.ReadDir(c.Dir)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	for _, v := range es {
		if !v.IsDir() && filepath.Ext(v.Name()) == ".json" {
			if e = os.Remove(filepath.Join(c.Dir, v.Name())); e != nil {
				return e
			}
		}
	}
	return nil
}
