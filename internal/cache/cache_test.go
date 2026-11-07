package cache

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestExpiryAndAtomicWrites(t *testing.T) {
	c := New(t.TempDir())
	if _, _, ok := c.Read("missing", time.Hour); ok {
		t.Fatal("unexpected hit")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := c.Write("key", []byte(`{"text":"public"}`), time.Now()); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	b, _, ok := c.Read("key", time.Hour)
	if !ok || string(b) != `{"text":"public"}` {
		t.Fatal("bad cache")
	}
	if _, _, ok = c.Read("key", -time.Second); ok {
		t.Fatal("expired hit")
	}
	st, e := os.Stat(c.path("key"))
	if e != nil || st.Mode().Perm() != 0600 {
		t.Fatal("unsafe permissions")
	}
}
