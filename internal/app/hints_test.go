package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

func commandArgs(t *testing.T, command string) []string {
	t.Helper()
	// A local shell function captures arguments without contacting sources.
	b, e := exec.Command("sh", "-c", "legidor() { printf '%s\\000' \"$@\"; }; "+command).Output()
	if e != nil {
		t.Fatal(e)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\x00"), "\x00")
}

func TestRecoveryCommandsKeepLiteralArguments(t *testing.T) {
	query := "l'article $(printf injected); `printf bad` \"quoted\""
	o := options{env: "sandbox", date: "2031-06-01", offline: true, json: true, maxAge: 24 * time.Hour}
	cmd := recoveryCommand("get", query, o, "--parts")
	args := commandArgs(t, cmd)
	if len(args) < 2 || args[1] != query {
		t.Fatalf("argument changed: %q", args)
	}
	parsed, e := parseOptions(args[1:])
	if e != nil || parsed.env != o.env || parsed.date != o.date || !parsed.parts || !parsed.offline || parsed.maxAge != o.maxAge {
		t.Fatalf("%+v %v", parsed, e)
	}
	if e = validateCommandOptions(args[0], parsed); e != nil {
		t.Fatal(e)
	}
}

func TestSearchHintsDoNotChangeScopeOrPagination(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		o := options{corpus: "loda", env: "production", date: "2031-01-01", supplied: map[string]bool{"in": explicit}}
		original := model.Search{Corpus: "loda", Total: 11, Cursor: "same-cursor", Complete: false}
		got := searchHints("BOI-EXAMPLE l'abattement", original, o)
		if got.Corpus != original.Corpus || got.Cursor != original.Cursor || got.Total != original.Total || got.Complete {
			t.Fatalf("semantics changed: %+v", got)
		}
		if len(got.Hints) != 1+boolInt(!explicit) {
			t.Fatalf("%+v", got)
		}
		if !explicit {
			hint := got.Hints[1]
			at := strings.Index(hint, "legidor ")
			args := commandArgs(t, hint[at:])
			parsed, e := parseOptions(args[1:])
			if e != nil || parsed.corpus != "bofip" || parsed.date != "" || parsed.args[0] != "BOI-EXAMPLE l'abattement" {
				t.Fatalf("bad suggestion: %+v %v", parsed, e)
			}
		}
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func TestRecoveryErrorsKeepStatusAndSelectWholeArticles(t *testing.T) {
	o := options{env: "sandbox", json: true, offline: true, maxAge: time.Hour}
	r, _ := reference.Parse("CGI 42, I")
	r.Date = "2031-01-01"
	original := model.Err("NO_VERSION_AT_DATE", 1, "not valid")
	got := asError(recoveryError(original, r, o))
	args := commandArgs(t, got.Next)
	parsed, e := parseOptions(args[1:])
	if got.Code != original.Code || got.Exit != original.Exit || original.Next != "" || len(got.Hints) == 0 || e != nil || args[0] != "versions" || parsed.date != "" || parsed.args[0] != "CGI 42" {
		t.Fatalf("%+v %+v %v", got, parsed, e)
	}
	if e = validateCommandOptions(args[0], parsed); e != nil {
		t.Fatal(e)
	}
	law, _ := reference.Parse("loi 2031-999 art. 7")
	law.Date = "2031-06-01"
	original = &model.Error{Code: "AMBIGUOUS_REFERENCE", Exit: 3, Candidates: []string{"LEGITEXT000000000001 Law", "LEGITEXT000000000002 Law rectificatif", "JORFTEXT000000000003 Original"}}
	got = asError(recoveryError(original, law, o))
	if len(got.Candidates) != 3 || len(got.Commands) != 2 || got.Exit != 3 {
		t.Fatalf("%+v", got)
	}
	for i, cmd := range got.Commands {
		args = commandArgs(t, cmd)
		parsed, e = parseOptions(args[1:])
		ref, pe := reference.Parse(parsed.args[0])
		if e != nil || pe != nil || ref.Article != "7" || parsed.part != "" || parsed.date != law.Date || parsed.env != "sandbox" || ref.ID != strings.Fields(original.Candidates[i])[0] {
			t.Fatalf("bad candidate %q: %+v %+v %v %v", cmd, parsed, ref, e, pe)
		}
	}
}

func TestZeroGrepMatchIsRetrievedAndRecoveryIsCompatible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LEGIDOR_CACHE_DIR", "")
	disk := cache.New(filepath.Join(root, "legidor/production"))
	seedResponse(t, disk, "/consult/getArticle", map[string]string{"id": "LEGIARTI000000000001"}, `{"article":{"id":"LEGIARTI000000000001","num":"42","dateDebut":"2030-01-01","texteHtml":"<p>I.- An invented condition.</p>","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"Code civil"}]}}`)
	for _, jsonMode := range []bool{false, true} {
		args := []string{"get", "LEGIARTI000000000001", "--part", "I", "--grep", "not present", "--date", "2031-01-01", "--offline"}
		if jsonMode {
			args = append(args, "--json")
		}
		var out, diag bytes.Buffer
		code := Run(context.Background(), args, strings.NewReader(""), &out, &diag)
		if code != 1 || !strings.Contains(out.String(), "Document retrieved") || !strings.Contains(out.String(), "--parts") || strings.Contains(out.String(), "--part ") || strings.Contains(out.String(), "--grep ") {
			t.Fatalf("%d %s %s", code, &out, &diag)
		}
		if jsonMode {
			var v struct {
				Complete bool
				ID       string
				Hints    []string
			}
			if e := json.Unmarshal(out.Bytes(), &v); e != nil {
				t.Fatal(e)
			}
			if !v.Complete || v.ID != "LEGIARTI000000000001" || len(v.Hints) != 1 {
				t.Fatalf("%+v", v)
			}
			at := strings.Index(v.Hints[0], "legidor ")
			recovery := commandArgs(t, v.Hints[0][at:])
			o, e := parseOptions(recovery[1:])
			if e != nil || o.date != "2031-01-01" || !o.parts || !o.offline || o.part != "" || o.grep != "" {
				t.Fatalf("%+v %v", o, e)
			}
		}
	}
	// A missing document must retain its retrieval error, not receive a no-match hint.
	var out, diag bytes.Buffer
	code := Run(context.Background(), []string{"get", "LEGIARTI000000000002", "--grep", "not present", "--offline", "--json"}, strings.NewReader(""), &out, &diag)
	if code != 3 || strings.Contains(out.String(), "Document retrieved") {
		t.Fatalf("%d %s", code, &out)
	}
}

func TestLawAndTextRecoveryReferenceRoundTrip(t *testing.T) {
	for _, raw := range []string{"loi 2031-999 art. 7", "loi 2031-999 du 2 janvier 2031 art. 7", "LEGITEXT000000000001 art. 7"} {
		r, e := reference.Parse(raw)
		if e != nil {
			t.Fatal(e)
		}
		r, e = r.Options("2031-06-01", "II")
		if e != nil {
			t.Fatal(e)
		}
		got := asError(recoveryError(model.Err("PART_NOT_FOUND", 1, "cannot select subdivision"), r, options{env: "production"}))
		args := commandArgs(t, got.Next)
		o, e := parseOptions(args[1:])
		if e != nil {
			t.Fatal(e)
		}
		recovered, e := reference.Parse(o.args[0])
		if e != nil {
			t.Fatal(e)
		}
		if recovered.Article != "7" || recovered.Part != "" || recovered.TextDate != r.TextDate || !o.parts || o.part != "" || o.date != r.Date {
			t.Fatalf("%s: %+v %+v", got.Next, recovered, o)
		}
		if e = validateCommandOptions(args[0], o); e != nil {
			t.Fatal(e)
		}
		missing := asError(recoveryError(model.Err("PART_NOT_FOUND", 1, "article 7 not found in this text"), r, options{env: "production"}))
		args = commandArgs(t, missing.Next)
		o, e = parseOptions(args[1:])
		if e != nil {
			t.Fatal(e)
		}
		recovered, e = reference.Parse(o.args[0])
		if e != nil || recovered.Article != "" || !o.parts || o.date != r.Date || recovered.TextDate != r.TextDate {
			t.Fatalf("%s: %+v %+v %v", missing.Next, recovered, o, e)
		}
	}
	// This syntax is intentionally unsupported, rather than silently reinterpreted.
	for _, raw := range []string{"loi 2031-999 art. 7, II", "LEGITEXT000000000001 art. 7, II"} {
		if _, e := reference.Parse(raw); e == nil {
			t.Fatalf("unexpected grammar acceptance: %s", raw)
		}
	}
	raw := "https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000000000001/2031-06-01"
	r, _ := reference.Parse(raw)
	r.Part = "I"
	got := asError(recoveryError(model.Err("PART_NOT_FOUND", 1, "cannot select subdivision"), r, options{env: "sandbox"}))
	args := commandArgs(t, got.Next)
	o, e := parseOptions(args[1:])
	if e != nil || o.date != "2031-06-01" || !o.parts || o.env != "sandbox" || o.args[0] != "LEGIARTI000000000001" {
		t.Fatalf("%s: %+v %v", got.Next, o, e)
	}
}

func TestOfflineSearchHintsUseOnlyTheRequestedCorpus(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LEGIDOR_CACHE_DIR", "")
	disk := cache.New(filepath.Join(root, "legidor/production"))
	date := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	body := map[string]any{"fond": "LODA_DATE", "recherche": map[string]any{"champs": []any{map[string]any{"typeChamp": "ALL", "criteres": []any{map[string]any{"typeRecherche": "TOUS_LES_MOTS_DANS_UN_CHAMP", "valeur": "BOI-EXAMPLE", "operateur": "ET"}}, "operateur": "ET"}}, "filtres": []any{map[string]any{"facette": "DATE_VERSION", "singleDate": date.UnixMilli()}}, "pageNumber": 1, "pageSize": 10, "operateur": "ET", "sort": "PERTINENCE", "typePagination": "DEFAUT"}}
	seedResponse(t, disk, "/search", body, `{"totalResultNumber":0,"results":[]}`)
	n, b, e := disk.Stats()
	if e != nil {
		t.Fatal(e)
	}
	for _, jsonMode := range []bool{false, true} {
		args := []string{"search", "BOI-EXAMPLE", "--date", "2031-01-01", "--offline"}
		if jsonMode {
			args = append(args, "--json")
		}
		var out, diag bytes.Buffer
		code := Run(context.Background(), args, strings.NewReader(""), &out, &diag)
		if code != 0 || !strings.Contains(out.String(), "loda") || !strings.Contains(out.String(), "--in bofip") {
			t.Fatalf("%d %s %s", code, &out, &diag)
		}
		if jsonMode {
			var s model.Search
			if e = json.Unmarshal(out.Bytes(), &s); e != nil {
				t.Fatal(e)
			}
			if s.Corpus != "loda" || s.Total != 0 || len(s.Hits) != 0 || !s.Complete || len(s.Hints) != 2 {
				t.Fatalf("%+v", s)
			}
		}
	}
	nn, bb, e := disk.Stats()
	if e != nil || n != nn || b != bb {
		t.Fatalf("hint modified cache: %d/%d %d/%d %v", n, nn, b, bb, e)
	}
}

func TestCandidateCommandsRejectIncompatibleArticleTokens(t *testing.T) {
	r, err := reference.Parse("loi 2031-999 art. 7_extra")
	if err != nil || r.Article != "7_extra" {
		t.Fatalf("fixture is not accepted: %+v %v", r, err)
	}
	if _, err = reference.Parse("JORFTEXT000000000001 art. " + r.Article); err == nil {
		t.Fatal("fixture must be rejected by text-ID grammar")
	}
	original := &model.Error{Code: "AMBIGUOUS_REFERENCE", Exit: 3, Candidates: []string{"JORFTEXT000000000001 Invented law", "JORFTEXT000000000002 Invented rectificatif"}}
	got := asError(recoveryError(original, r, options{env: "production"}))
	if len(got.Commands) != 0 || got.Code != original.Code || got.Exit != original.Exit || len(got.Candidates) != len(original.Candidates) {
		t.Fatalf("invalid command or changed semantics: %+v", got)
	}
	for i, candidate := range got.Candidates {
		if candidate != original.Candidates[i] {
			t.Fatal("candidate metadata changed")
		}
	}
	if r.Article != "7_extra" {
		t.Fatal("requested article changed")
	}
}
