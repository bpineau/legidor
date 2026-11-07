package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Case struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Prompt     string   `json:"prompt"`
	Assertions []string `json:"assertions"`
	Amendments []string `json:"amendments,omitempty"`
}
type Corpus struct {
	Source       string `json:"source"`
	Commit       string `json:"commit"`
	SourceSHA256 string `json:"source_sha256"`
	AsOf         string `json:"as_of"`
	Cases        []Case `json:"cases"`
}

func LoadCorpus(path string) (Corpus, error) {
	var c Corpus
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	seen := map[int]bool{}
	if len(c.Cases) == 0 || c.AsOf == "" {
		return c, fmt.Errorf("empty corpus or as_of")
	}
	for _, s := range c.Cases {
		if s.ID < 1 || seen[s.ID] || s.Name == "" || s.Prompt == "" || len(s.Assertions) == 0 {
			return c, fmt.Errorf("invalid or duplicate case %d", s.ID)
		}
		seen[s.ID] = true
	}
	return c, nil
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func HashFile(path string) (string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	return digest(b), nil
}

// HashTree includes names and contents, in stable order; symlinks are rejected.
func HashTree(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in snapshot: %s", p)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(b))
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func ASCII(s string) string {
	return strings.NewReplacer("\u2019", "'", "\u2018", "'", "\u201c", "\"", "\u201d", "\"", "\u2014", "-", "\u2013", "-").Replace(s)
}

// Snapshot copies documentation/data only, resolving internal data symlinks.
// Evaluation answers, scripts and hidden files are deliberately excluded.
func Snapshot(source, dest string) error {
	source, err := filepath.EvalSymlinks(source)
	if err != nil {
		return err
	}
	return filepath.WalkDir(source, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(source, p)
		parts := strings.Split(rel, string(filepath.Separator))
		for _, part := range parts {
			if part == "evals" || part == "scripts" || (strings.HasPrefix(part, ".") && part != ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dest, rel), 0700)
		}
		switch filepath.Ext(p) {
		case ".md", ".json":
		default:
			return nil
		}
		actual, e := filepath.EvalSymlinks(p)
		if e != nil {
			return e
		}
		r, e := filepath.Rel(source, actual)
		if e != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			return fmt.Errorf("external resource symlink %s", rel)
		}
		data, e := os.ReadFile(actual)
		if e != nil {
			return e
		}
		return os.WriteFile(filepath.Join(dest, rel), data, 0600)
	})
}
func ResourcePaths(root string) []string {
	var out []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			r, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(r))
		}
		return err
	})
	sort.Strings(out)
	return out
}
