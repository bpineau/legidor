package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/cache"
)

func TestCompactOutputAndFullMetadata(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LEGIDOR_CACHE_DIR", "")
	disk := cache.New(filepath.Join(root, "legidor/production"))
	mods := []map[string]string{}
	for i := 0; i < 30; i++ {
		mods = append(mods, map[string]string{"articleId": fmt.Sprintf("JORFARTI%012d", i+1), "textTitle": fmt.Sprintf("Invented modifier %d\n continuation", i), "datePubliTexte": fmt.Sprintf("20%02d-01-01", i)})
	}
	for i := 1; i <= 2; i++ {
		a := map[string]any{"id": fmt.Sprintf("LEGIARTI%012d", i), "num": "42", "dateDebut": "2020-01-01", "dateFin": "2027-01-01", "etat": "MODIFIE", "texteHtml": "<p>IV.- A matching provision.</p>", "textTitles": []map[string]string{{"id": "LEGITEXT000000000001", "titreLong": "Code civil"}}, "lienModifications": mods, "articleVersions": []map[string]string{{"id": "LEGIARTI000000000001", "dateDebut": "2020-01-01", "dateFin": "2027-01-01", "etat": "MODIFIE"}, {"id": "LEGIARTI000000000002", "dateDebut": "2027-01-01", "etat": "VIGUEUR_DIFF"}}}
		if i == 2 {
			a["dateDebut"] = "2027-01-01"
			a["dateFin"] = "2999-01-01"
		}
		b, _ := json.Marshal(map[string]any{"article": a})
		seedResponse(t, disk, "/consult/getArticle", map[string]string{"id": a["id"].(string)}, string(b))
	}
	for _, tc := range []struct {
		args     []string
		contains string
		not      string
	}{
		{[]string{"get", "LEGIARTI000000000001"}, "state: MODIFIE", "modified by: Invented modifier 0 "},
		{[]string{"get", "LEGIARTI000000000001"}, "next version: LEGIARTI000000000002", ""},
		{[]string{"get", "LEGIARTI000000000001", "--json"}, `"omitted_modifications":25`, ""},
		{[]string{"get", "LEGIARTI000000000001", "--json", "--full"}, "Invented modifier 0 continuation", `"omitted_modifications"`},
		{[]string{"get", "LEGIARTI000000000001", "--grep", "matching"}, "matching", "\"matches\""},
		{[]string{"versions", "LEGIARTI000000000001", "--since", "2026-01-01"}, "2020-01-01", "\"versions\""},
		{[]string{"versions", "LEGIARTI000000000001", "--json"}, `"omitted_modifications":25`, "Invented modifier 0 continuation"},
		{[]string{"versions", "LEGIARTI000000000001", "--json", "--full"}, "Invented modifier 0 continuation", `"omitted_modifications"`},
		{[]string{"versions", "LEGIARTI000000000001"}, "omitted_modifications: 29", "Invented modifier 0 continuation"},
	} {
		var out, diag bytes.Buffer
		code := Run(context.Background(), append(tc.args, "--offline"), strings.NewReader(""), &out, &diag)
		if code != 0 || !strings.Contains(out.String(), tc.contains) || (tc.not != "" && strings.Contains(out.String(), tc.not)) {
			t.Errorf("%v: %d %s %s", tc.args, code, &out, &diag)
		}
	}
}
