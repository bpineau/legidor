package source

import (
	"path/filepath"
	"testing"
)

func TestDedicatedCacheRootOverridesXDGAndSeparatesEnvironments(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg := t.TempDir()
	dedicated := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdg)
	t.Setenv("LEGIDOR_CACHE_DIR", dedicated)
	for _, env := range []string{"production", "sandbox"} {
		c, err := LoadConfig(env)
		if err != nil || c.CacheDir != filepath.Join(dedicated, env) {
			t.Fatalf("%s: %+v %v", env, c, err)
		}
	}
	t.Setenv("LEGIDOR_CACHE_DIR", "")
	c, err := LoadConfig("production")
	if err != nil || c.CacheDir != filepath.Join(xdg, "legidor", "production") {
		t.Fatalf("fallback: %+v %v", c, err)
	}
}
