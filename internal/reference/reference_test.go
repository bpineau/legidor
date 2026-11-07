package reference

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct{ in, kind, code, num, part string }{
		{"CGI 150-0 D", "article", "CGI", "150-0 D", ""}, {"art. 150-0 D CGI", "article", "CGI", "150-0 D", ""}, {"CGI art. 150-0 D, 11", "article", "CGI", "150-0 D", "11"}, {"L. 136-8 CSS", "article", "CSS", "L136-8", ""}, {"CSS L136-8 IV", "article", "CSS", "L136-8", "IV"}, {"CMF L224-4 I 2° bis", "article", "CMF", "L224-4", "I 2° bis"}, {"CoMoFi L. 224-4", "article", "CMF", "L224-4", ""}, {"CGI ann. II 41 DN", "article", "CGI-II", "41 DN", ""}, {"C. civ. 1240", "article", "CIV", "1240", ""}, {"CPCE R322-39..R322-42", "range", "CPCE", "R322-39", ""}, {"BOI-RFPI-DECLA-10 §60", "bofip", "", "BOI-RFPI-DECLA-10", "60"}, {"loi n° 2023-1322 art. 110", "law", "", "2023-1322", ""}, {"CE 2024-07-12 n° 476458", "decision", "CETAT", "476458", ""}, {"QPC 2017-659", "decision", "CONSTIT", "2017-659 QPC", ""}}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			r, e := Parse(c.in)
			if e != nil {
				t.Fatal(e)
			}
			if r.Kind != c.kind || r.Code != c.code || r.Number != c.num || r.Part != c.part {
				t.Fatalf("got %+v", r)
			}
		})
	}
}

func TestLetterAfterLatinArticleSuffix(t *testing.T) {
	for _, tc := range []struct{ raw, number, part string }{
		{"CGI 1649 bis C", "1649 bis C", ""},
		{"CGI 199 decies EA", "199 decies EA", ""},
		{"199 decies F CGI", "199 decies F", ""},
		{"CGI 1649 bis C, I", "1649 bis C", "I"},
		{"CGI 163 bis, C", "163 bis", "C"},
	} {
		got, err := Parse(tc.raw)
		if err != nil || got.Number != tc.number || got.Part != tc.part {
			t.Fatalf("%q: %+v %v", tc.raw, got, err)
		}
	}
}
func TestConflicts(t *testing.T) {
	r, _ := Parse("CSS L136-8 IV")
	if _, e := r.Options("2025-12-31", "III"); e == nil {
		t.Fatal("part conflict accepted")
	}
	r, e := Parse("https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000053278668/2025-12-31")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.Options("2026-01-01", ""); e == nil {
		t.Fatal("date conflict accepted")
	}
}
func TestReject(t *testing.T) {
	for _, s := range []string{"C. civ. 757 B nonsense", "https://evil.example/LEGIARTI000012345678", "CGI unknown", "LFR 2022 art. 1", "CGI 32 --oops"} {
		if _, e := Parse(s); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
}
func TestSectionAndRomanSuffix(t *testing.T) {
	r, e := Parse("CSS section LEGISCTA000000000001")
	if e != nil || r.Kind != "section" || r.Code != "CSS" {
		t.Fatalf("%+v %v", r, e)
	}
	r, e = Parse("CGI 150-0 D II bis")
	if e != nil || r.Number != "150-0 D" || r.Part != "II bis" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestRomanArticleSuffix(t *testing.T) {
	for _, s := range []string{"CGI 990 I", "CGI 150 V"} {
		r, e := Parse(s)
		if e != nil || r.Part != "" {
			t.Fatalf("%s: %+v %v", s, r, e)
		}
	}
}
func TestTextArticle(t *testing.T) {
	r, e := Parse("JORFTEXT000000000001 art. 110")
	if e != nil || r.ID != "JORFTEXT000000000001" || r.Article != "110" || r.Kind != "text" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestLongLatinSuffix(t *testing.T) {
	r, e := Parse("CGI 199 novovicies")
	if e != nil || r.Number != "199 novovicies" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestOfficialSectionURL(t *testing.T) {
	r, e := Parse("https://www.legifrance.gouv.fr/codes/section_lc/LEGITEXT000000000001/LEGISCTA000000000002/2025-12-31")
	if e != nil || r.Kind != "section" || r.ID != "LEGISCTA000000000002" || r.Parent != "LEGITEXT000000000001" || r.Date != "2025-12-31" {
		t.Fatalf("%+v %v", r, e)
	}
}

func TestLatinSuffixPrefixDoesNotTruncateArticle(t *testing.T) {
	for _, ref := range []string{"CGI 80 quaterdecies", "CGI 163 quatervicies", "CGI 163 terdecies"} {
		got, err := Parse(ref)
		if err != nil || got.Number != strings.TrimPrefix(ref, "CGI ") || got.Part != "" {
			t.Fatalf("%s: %+v, %v", ref, got, err)
		}
	}
}

func TestCompoundSubdivisionAndCIBS(t *testing.T) {
	for _, tc := range []struct{ raw, code, number, part string }{
		{"art. 83, 2°-0 ter CGI", "CGI", "83", "2°-0 ter"},
		{"CIBS L213-82", "CIBS", "L213-82", ""},
		{"Code des impositions sur les biens et services L213-82", "CIBS", "L213-82", ""},
	} {
		r, err := Parse(tc.raw)
		if err != nil || r.Code != tc.code || r.Number != tc.number || r.Part != tc.part {
			t.Fatalf("%s: %+v %v", tc.raw, r, err)
		}
	}
}

func TestCompoundArticleSuffix(t *testing.T) {
	r, err := Parse("CGI 1649 quater-0 B bis, I")
	if err != nil || r.Number != "1649 quater-0 B bis" || r.Part != "I" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestCodeIndexReference(t *testing.T) {
	r, err := Parse("CIBS")
	if err != nil || r.Kind != "code" || r.Code != "CIBS" {
		t.Fatalf("%+v %v", r, err)
	}
}
