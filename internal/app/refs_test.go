package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

func TestExtractCodeBeforeOrAfterArticle(t *testing.T) {
	for _, tc := range []struct{ text, code, number string }{
		{"+Reclamation (art. R* 196-1, a LPF).", "LPF", "R*196-1"},
		{"+Reprise (art. L. 169 LPF).", "LPF", "L169"},
		{"+Conservation (L. 102 B LPF).", "LPF", "L102 B"},
		{"+Renvoi a l'art. 1649 bis C CGI.", "CGI", "1649 bis C"},
		{"+CGI art. 199 decies EA : exemple.", "CGI", "199 decies EA"},
		{"-Article 163 quatervicies CGI.", "CGI", "163 quatervicies"},
		{"+LPF R*196-3 et son delai.", "LPF", "R*196-3"},
		{"+CGI article 1649 bis C.", "CGI", "1649 bis C"},
		{"+art. 1240 C. civ., responsabilite.", "CIV", "1240"},
		{"+art. 1240 C. civ. pour cet exemple.", "CIV", "1240"},
	} {
		var out, diagnostics bytes.Buffer
		if code := refs("-", options{diff: true}, strings.NewReader(tc.text), &out, &diagnostics); code != 0 {
			t.Fatal(code)
		}
		var row model.Citation
		if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &row); err != nil {
			t.Fatalf("%s: %s (%v)", tc.text, out.String(), err)
		}
		r, err := reference.Parse(row.Ref)
		if err != nil || r.Code != tc.code || r.Number != tc.number {
			t.Fatalf("%s: %+v %v", tc.text, r, err)
		}
		if row.Change != map[bool]string{true: "removed", false: "added"}[strings.HasPrefix(tc.text, "-")] {
			t.Fatal(row)
		}
	}
}

func TestExtractPreservesNumericSubdivision(t *testing.T) {
	var out, diagnostics bytes.Buffer
	refs("-", options{}, strings.NewReader("CGI 81 1° bis"), &out, &diagnostics)
	var row model.Citation
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &row); err != nil {
		t.Fatal(err)
	}
	r, err := reference.Parse(row.Ref)
	if err != nil || r.Number != "81" || r.Part != "1° bis" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestExtractRequiresWholeCodeAlias(t *testing.T) {
	var out, diagnostics bytes.Buffer
	refs("-", options{}, strings.NewReader("art. L169 LPFfoobar"), &out, &diagnostics)
	if out.Len() != 0 {
		t.Fatal(out.String())
	}
}
