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

func TestExtractCompoundPartsAndLists(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  []string
		exit  int
	}{
		{"art. 83, 2°-0 ter CGI", []string{"CGI|83|2°-0 ter"}, 0},
		{"CGI 83, 2°-0 ter", []string{"CGI|83|2°-0 ter"}, 0},
		{"CGI 83, 2° quinquies", []string{"CGI|83|2° quinquies"}, 0},
		{"CGI articles 10, 20 et 30", []string{"CGI|10|", "CGI|20|", "CGI|30|"}, 0},
		{"CGI 10, 20 et 30", nil, 3},
		{"BOI-EXAMPLE-10 § 40 et 70", []string{"|BOI-EXAMPLE-10|40", "|BOI-EXAMPLE-10|70"}, 0},
		{"articles 219, 1426 et 1427 C. civ.", []string{"CIV|219|", "CIV|1426|", "CIV|1427|"}, 0},
		{"art. 219, 1426 C. civ.", nil, 3},
		{"CIBS L213-82", []string{"CIBS|L213-82|"}, 0},
	} {
		var out, diag bytes.Buffer
		code := refs("-", options{}, strings.NewReader(tc.input), &out, &diag)
		if code != tc.exit {
			t.Errorf("%s: code %d, %s", tc.input, code, diag.String())
		}
		var got []string
		dec := json.NewDecoder(&out)
		for dec.More() {
			var row model.Citation
			if err := dec.Decode(&row); err != nil {
				t.Fatal(err)
			}
			r, err := reference.Parse(row.Ref)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, r.Code+"|"+r.Number+"|"+r.Part)
		}
		if strings.Join(got, ";") != strings.Join(tc.want, ";") {
			t.Errorf("%s: %v want %v", tc.input, got, tc.want)
		}
	}
}

func TestExtractDoesNotDropLeadingList(t *testing.T) {
	for _, input := range []string{"art. 10, 20 et 30 CGI", "art. 10 et 30 CGI"} {
		var out, diag bytes.Buffer
		if code := refs("-", options{}, strings.NewReader(input), &out, &diag); code != 3 || out.Len() != 0 {
			t.Fatalf("%q: %d %s %s", input, code, &out, &diag)
		}
	}
}

func TestExtractExtendedReferences(t *testing.T) {
	var out, diag bytes.Buffer
	code := refs("-", options{}, strings.NewReader("CGI 1649 quater-0 B bis; BOI-EXAMPLE-10 § 90, § 120; art. 150-0 A"), &out, &diag)
	if code != 3 || !strings.Contains(out.String(), "1649 quater-0 B bis") || !strings.Contains(out.String(), "§120") || !strings.Contains(diag.String(), "150-0 A") {
		t.Fatalf("%d %s %s", code, &out, &diag)
	}
}

func TestExtractTwoDigitYearLaw(t *testing.T) {
	var out, diag bytes.Buffer
	refs("-", options{}, strings.NewReader("Voir ordonnance n° 96-50 art. 19."), &out, &diag)
	var row model.Citation
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &row); err != nil {
		t.Fatalf("%s %v", out.String(), err)
	}
	r, err := reference.Parse(row.Ref)
	if err != nil || r.Kind != "law" || r.Number != "96-50" || r.Article != "19" {
		t.Fatalf("%+v %v", r, err)
	}
}
