package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/model"
)

func seedResponse(t *testing.T, disk *cache.Cache, endpoint string, payload any, raw string) {
	t.Helper()
	b, _ := json.Marshal(payload)
	if e := disk.Write("production:POST:https://api.piste.gouv.fr/dila/legifrance/lf-engine-app"+endpoint+":"+string(b), []byte(raw), time.Now()); e != nil {
		t.Fatal(e)
	}
}
func TestThirtyCitationAuditAndTOC(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LEGIDOR_CACHE_DIR", "")
	disk := cache.New(filepath.Join(root, "legidor/production"))
	seedResponse(t, disk, "/consult/getArticle", map[string]string{"id": "LEGIARTI000000000001"}, `{"article":{"id":"LEGIARTI000000000001","num":"1","dateDebut":1735689600000,"dateFin":1767225600000,"texteHtml":"<p>I.- It does not apply.</p>","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"Code civil"}]}}`)
	seedResponse(t, disk, "/consult/jorf", map[string]string{"textCid": "JORFTEXT000000000001"}, `{"cid":"JORFTEXT000000000001","title":"LOI n° 2025-999","articles":[{"id":"JORFARTI000000000001","num":"1","content":"<p>Source content.</p>"}]}`)
	var input strings.Builder
	for i := 0; i < 30; i++ {
		quote := "It does not apply."
		if i == 29 {
			quote = "It does apply."
		}
		fmt.Fprintf(&input, "{\"ref\":\"LEGIARTI000000000001\",\"date\":\"2025-06-01\",\"quote\":%q,\"where\":%q}\n", quote, fmt.Sprintf("example:%d", i+1))
	}
	var out, errout bytes.Buffer
	code := Run(context.Background(), []string{"check", "-", "--offline", "--json"}, strings.NewReader(input.String()), &out, &errout)
	if code != 1 {
		t.Fatalf("exit %d %s", code, &out)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 30 {
		t.Fatal(len(lines))
	}
	for i, line := range lines {
		var result model.Check
		if e := json.Unmarshal([]byte(line), &result); e != nil {
			t.Fatal(e)
		}
		want := "exact"
		if i == 29 {
			want = "differs"
		}
		if result.QuoteStatus != want || result.Identity != "confirmed" || result.Version != "in_force_at_date" || result.Input.Where != fmt.Sprintf("example:%d", i+1) || result.Applicability != "not_evaluated" {
			t.Fatalf("row %d: %+v", i, result)
		}
	}
	out.Reset()
	code = Run(context.Background(), []string{"check", "-", "--offline", "--json"}, strings.NewReader(`{"ref":"JORFTEXT000000000001","quote":"Table of contents."}`), &out, &errout)
	if code != 3 || strings.Contains(out.String(), `"quote":"exact"`) {
		t.Fatalf("synthetic quote verified: %d %s", code, &out)
	}
}
func TestPaginationPinsContentAndDoesNotRepeat(t *testing.T) {
	text := strings.Repeat("é", outputLimit+7)
	d := model.Document{Schema: model.Schema, ID: "LEGIARTI000000000001", Text: text, Complete: true}
	r := referenceForPagination()
	first, e := paginate(d, r, r.Raw, options{env: "production"})
	if e != nil || first.Complete || len([]rune(first.Text)) != outputLimit {
		t.Fatalf("%v %+v", e, first)
	}
	second, e := paginate(d, r, r.Raw, options{env: "production", cursor: first.Cursor})
	if e != nil || !second.Complete || first.Text+second.Text != text {
		t.Fatal("cursor repeated or lost source text")
	}
	d.Text += "changed"
	if _, e = paginate(d, r, r.Raw, options{env: "production", cursor: first.Cursor}); e == nil {
		t.Fatal("changed content accepted")
	}
}

func TestSelectedEvidenceAndSourceLinksOffline(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("LEGIDOR_CACHE_DIR", "")
	disk := cache.New(filepath.Join(root, "legidor/production"))
	raw := `{"article":{"id":"LEGIARTI000000000001","num":"1","dateDebut":1735689600000,"dateFin":1767225600000,"nota":"An exception has a separate commencement date.","texteHtml":"<p>I.- Shared introduction.</p><p>1° Usable provision. <a href=\"https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000000000002\">Related article</a></p><p>2° See illustration.</p><img src=\"example.png\">","textTitles":[{"id":"LEGITEXT000000000001","titreLong":"Code civil"}]}}`
	seedResponse(t, disk, "/consult/getArticle", map[string]string{"id": "LEGIARTI000000000001"}, raw)
	var out, diag bytes.Buffer
	manifest := `{"ref":"LEGIARTI000000000001","date":"2025-06-01","part":"I 1°","quote":"Usable provision."}` + "\n" + `{"ref":"LEGIARTI000000000001","date":"2025-06-01","part":"I 2°"}`
	code := Run(context.Background(), []string{"check", "-", "--offline", "--json"}, strings.NewReader(manifest), &out, &diag)
	if code != 3 {
		t.Fatalf("%d %s %s", code, &out, &diag)
	}
	dec := json.NewDecoder(&out)
	var first, second model.Check
	if err := dec.Decode(&first); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(&second); err != nil {
		t.Fatal(err)
	}
	if first.Error != nil || first.QuoteStatus != "exact" || first.Document.Nota == "" || first.Document.SourceComplete == nil || *first.Document.SourceComplete {
		t.Fatalf("%+v", first)
	}
	if second.Error == nil || second.Error.Code != "UNSUPPORTED_CONTENT" {
		t.Fatalf("%+v", second)
	}
	out.Reset()
	diag.Reset()
	code = Run(context.Background(), []string{"get", "LEGIARTI000000000001", "--date", "2025-06-01", "--links", "--offline", "--json"}, strings.NewReader(""), &out, &diag)
	var d model.Document
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if code != 3 || len(d.Items) != 1 || d.Items[0].ID != "LEGIARTI000000000002" || d.Text != "" || d.ContentKind != "source_references" {
		t.Fatalf("%d %+v %s", code, d, &diag)
	}
}
