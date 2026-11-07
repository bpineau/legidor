package source

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/bpineau/legidor/internal/model"
)

type Config struct{ ClientID, ClientSecret, APIKey, Environment, CacheDir string }

// LoadConfig reads a simple env file as data, never as shell code.
func LoadConfig(environment string) (Config, error) {
	c := Config{Environment: environment}
	if environment != "production" && environment != "sandbox" {
		return c, model.Err("USAGE", 2, "environment must be production or sandbox")
	}
	home, _ := os.UserHomeDir()
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		root = filepath.Join(home, ".config")
	}
	name := "piste.env"
	if environment == "sandbox" {
		name = "piste-sandbox.env"
	}
	values := map[string]string{}
	f, e := os.Open(filepath.Join(root, "legidor", name))
	if e == nil {
		defer f.Close()
		st, _ := f.Stat()
		if st != nil && st.Mode().Perm()&0077 != 0 {
			return c, model.Err("AUTH_CONFIG", 3, "credential file permissions must be 0600")
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			values[strings.TrimSpace(strings.TrimPrefix(k, "export "))] = strings.Trim(strings.TrimSpace(v), "\"'")
		}
		if s.Err() != nil {
			return c, model.Err("AUTH_CONFIG", 3, "cannot read credential file")
		}
	} else if !os.IsNotExist(e) {
		return c, model.Err("AUTH_CONFIG", 3, "cannot open credential file")
	}
	get := func(key string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return values[key]
	}
	c.ClientID = get("PISTE_CLIENT_ID")
	c.ClientSecret = get("PISTE_CLIENT_SECRET")
	c.APIKey = get("PISTE_API_KEY")
	root = os.Getenv("LEGIDOR_CACHE_DIR")
	if root == "" {
		root = os.Getenv("XDG_CACHE_HOME")
		if root == "" {
			root = filepath.Join(home, ".cache")
		}
		root = filepath.Join(root, "legidor")
	}
	c.CacheDir = filepath.Join(root, environment)
	return c, nil
}
