package text

import (
	"strings"
	"testing"
)

func TestSelectParentsAndDuplicates(t *testing.T) {
	s := "I.- The rate is set:\n1° First case.\n2° bis Second case.\nII.- Other cases:\n1° Different case."
	p := Parts(s)
	got, e := Select(s, p, "I 2° bis")
	if e != nil || !strings.Contains(got, "I.- The rate") || strings.Contains(got, "First case") {
		t.Fatalf("%q %v %+v", got, e, p)
	}
	if _, e = Select(s, p, "1°"); e == nil {
		t.Fatal("ambiguous part accepted")
	}
	if _, e = Select(s, p, "III"); e == nil {
		t.Fatal("absent part accepted")
	}
}
func TestQuote(t *testing.T) {
	for _, c := range []struct{ s, q, w string }{{"A 42 euros, not 43.", "42 euros", "exact"}, {"A 42 euros, not 43.", "43 euros", "differs"}, {"L\u2019article dit : oui.", "L'article dit : oui.", "matches_after_normalization"}, {"Alpha beta gamma delta", "Alpha [...] delta", "exact"}, {"Alpha beta gamma delta", "delta [...] Alpha", "differs"}, {"NO", "no", "differs"}, {"a b", "[...]", "differs"}} {
		if g := Quote(c.s, c.q); g != c.w {
			t.Errorf("%q got %s want %s", c.q, g, c.w)
		}
	}
}
func TestHTML(t *testing.T) {
	s, u := HTML(`<p>I.- A &amp; B</p><p>2° Case</p><table><tr><td colspan="2">Complex</td></tr></table>`)
	if !strings.Contains(s, "A & B") || len(u) == 0 {
		t.Fatal(s, u)
	}
}
func TestBOIParts(t *testing.T) {
	s, p, u := BOI(`<h2>Heading</h2><p class="numero-de-paragraphe-western">60</p><p>First paragraph.</p><p>Continuation.</p><p class="numero-de-paragraphe-western">70</p><p>Next.</p>`)
	g, e := Select(s, p, "60")
	if len(u) > 0 || e != nil || !strings.Contains(g, "Continuation") || strings.Contains(g, "Next") {
		t.Fatal(g, e, s, p)
	}
}
func TestDiff(t *testing.T) {
	d := Diff("alpha old omega", "alpha new omega")
	if d != "alpha [-old-]{+new+} omega" {
		t.Fatal(d)
	}
}
func TestBOIDoubledAttributeQuotes(t *testing.T) {
	s, p, _ := BOI(`<p class=""numero-de-paragraphe-western"" id=""60_0"">60</p><p>Public invented text.</p><p class=""numero-de-paragraphe-western"">70</p><p>Other.</p>`)
	g, e := Select(s, p, "60")
	if e != nil || !strings.Contains(g, "Public invented") || strings.Contains(g, "Other") {
		t.Fatalf("%q %v", g, e)
	}
}
func TestNumericRootAndInlineLetter(t *testing.T) {
	s := "1. First.\n1 ter. A. Intro:\n1° First case.\nB. Second group.\n2. Next.\n11. Target.\n12. End."
	p := Parts(s)
	g, e := Select(s, p, "11")
	if e != nil || g != "11. Target." {
		t.Fatalf("%q %v %+v", g, e, p)
	}
	g, e = Select(s, p, "1 ter A 1°")
	if e != nil || !strings.Contains(g, "Intro:") || strings.Contains(g, "Second group") {
		t.Fatalf("%q %v %+v", g, e, p)
	}
}
func TestNestedNumericAndDegree(t *testing.T) {
	s := "I.- Header\n1. Group\na) Nested:\n1° First\n2° Second\nb) Other\n2. Next"
	p := Parts(s)
	g, e := Select(s, p, "I 1 a")
	if e != nil || !strings.Contains(g, "2° Second") || strings.Contains(g, "b) Other") {
		t.Fatalf("%q %v %+v", g, e, p)
	}
}
func TestQuoteNumericBoundaries(t *testing.T) {
	for _, q := range []string{"threshold 1", "threshold 10"} {
		if g := Quote("threshold 10.5 applies", q); g != "differs" {
			t.Fatalf("%s: %s", q, g)
		}
	}
}

func TestLongLatinSubdivisionSuffix(t *testing.T) {
	s := "I.- Introduction\n1° quaterdecies First case.\n2° Other case."
	parts := Parts(s)
	got, err := Select(s, parts, "I 1° quaterdecies")
	if err != nil || !strings.Contains(got, "First case.") || strings.Contains(got, "Other case.") {
		t.Fatalf("%q %v %+v", got, err, parts)
	}
}

func TestSimpleTablesPreserveCellsAndNotes(t *testing.T) {
	raw := `<table><caption>Example schedule</caption><thead><tr><th>Band</th><th>Amount</th></tr></thead><tbody><tr><td>10 &lt; x &lt; 20</td><td>3 % x - (20 - x) / 4</td></tr><tr><td>A | B</td><td><p>First</p><p>Second</p></td></tr></tbody></table><p>Note: invented values.</p>`
	s, u := HTML(raw)
	for _, want := range []string{"Example schedule", "| Band | Amount |", "| 10 < x < 20 | 3 % x - (20 - x) / 4 |", `A \| B`, "First Second", "Note: invented values."} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %q", want, s)
		}
	}
	if len(u) != 0 {
		t.Fatal(u)
	}
}

func TestUnsupportedTablesRemainVisible(t *testing.T) {
	for _, raw := range []string{
		`<table><tr><td colspan="2">Merged</td></tr></table>`,
		`<table><tr><td><table><tr><td>Nested</td></tr></table></td></tr></table>`,
		`<table><tr><td>A</td><td>B</td></tr><tr><td>C</td></tr></table>`,
		`<table><tr><td>Unclosed`,
	} {
		s, u := HTML("<p>Before.</p>" + raw + "<p>After.</p>")
		if len(u) == 0 || !strings.Contains(s, "omitted:table") {
			t.Fatalf("%q: %q %v", raw, s, u)
		}
	}
}

func TestDottedLettersAndCompoundDegrees(t *testing.T) {
	s := "I.- Introduction\na. First group\n1° First case\n2° Second case\nb. Target group\n2°-0 ter Target case\n3° Next case\nII.- Other group"
	ps := Parts(s)
	got, err := Select(s, ps, "I b 2°-0 ter")
	if err != nil || !strings.Contains(got, "Target case") || strings.Contains(got, "Next case") || strings.Contains(got, "First case") {
		t.Fatalf("%q %v %+v", got, err, ps)
	}
	ms := Grep(s, "b. Target", ps)
	if len(ms) != 1 || ms[0].Part != "I b" {
		t.Fatal(ms)
	}
}

func TestSourceLinks(t *testing.T) {
	raw := `<a href="/codes/article_lc/LEGIARTI000000000001?dateVersion=20300101&amp;x=1">Article <b>example</b></a><a href="javascript:bad()">Bad</a><a href="https://example.net/">External</a><a href="https://user:secret@www.legifrance.gouv.fr/">Credentials</a>`
	links := Links(raw, "https://www.legifrance.gouv.fr/jorf/id/JORFTEXT000000000001")
	if len(links) != 1 || links[0].ID != "LEGIARTI000000000001" || links[0].Title != "Article example" || !strings.Contains(links[0].URL, "&x=1") {
		t.Fatal(links)
	}
}

func TestTableExponentsAndIndices(t *testing.T) {
	s, u := HTML(`<table><tr><td>x<sup>2</sup> + y<sub>n</sub></td></tr></table>`)
	if len(u) != 0 || !strings.Contains(s, "x^(2) + y_(n)") {
		t.Fatalf("%q %v", s, u)
	}
}

func TestSelectionRetainsOmittedIntroductions(t *testing.T) {
	s, ps, _ := BOI(`<img src="context.png"><p class="numero-de-paragraphe-western">10</p><p>Selected.</p>`)
	got, err := Select(s, ps, "10")
	if err != nil || len(Omissions(got)) != 1 {
		t.Fatalf("%q %v", got, err)
	}
}

func TestSourceLinksIgnoreDataAttributes(t *testing.T) {
	raw := `<a data-href="https://www.legifrance.gouv.fr/jorf/id/JORFTEXT000000000001" href="javascript:void(0)">Example</a>`
	if links := Links(raw, "https://www.legifrance.gouv.fr/"); len(links) != 0 {
		t.Fatal(links)
	}
}

func TestSourceLinksPreferTargetOverParent(t *testing.T) {
	links := Links(`<a href="https://www.legifrance.gouv.fr/affichCodeArticle.do?cidTexte=LEGITEXT000000000001&amp;idArticle=LEGIARTI000000000002">Article</a>`, "https://www.legifrance.gouv.fr/")
	if len(links) != 1 || links[0].ID != "LEGIARTI000000000002" {
		t.Fatalf("wrong link target: %+v", links)
	}
}

func TestTableFullWidthNote(t *testing.T) {
	s, u := HTML(`<table><tr><th>Band</th><th>Formula</th></tr><tr><td>A</td><td>x + 1</td></tr><tr><td colspan="2">x is the base.</td></tr></table>`)
	if len(u) != 0 || !strings.Contains(s, "| A | x + 1 |") || !strings.Contains(s, "Table-wide: x is the base.") {
		t.Fatalf("%q %v", s, u)
	}
}

func TestBOINonbreakingParagraphMarker(t *testing.T) {
	s, ps, _ := BOI(`<p class="numero-de-paragraphe-western">210</p><table><tr><td rowspan="2">Complex</td></tr></table><p class="numero-de-paragraphe-western">&nbsp;220</p><p>Example.</p>`)
	got, err := Select(s, ps, "220")
	if err != nil || !strings.Contains(got, "Example.") || strings.Contains(got, "omitted:table") {
		t.Fatalf("%q %v", got, err)
	}
}

func TestBOISectionsStayWithFollowingParagraphs(t *testing.T) {
	raw := `<p>Actualité liée : <a href="https://bofip.impots.gouv.fr/">Editorial notice</a></p><p>General condition.</p><h1>First section</h1><p class="numero-de-paragraphe-western">10</p><p>First rule.</p><h1>Second section</h1><p>Section condition.</p><p class="numero-de-paragraphe-western">20</p><p>Second rule.</p><p class="numero-de-paragraphe-western">30</p><p>Third rule.</p>`
	s, ps, _ := BOI(raw)
	first, e := Select(s, ps, "10")
	third, e3 := Select(s, ps, "30")
	if e != nil || e3 != nil || strings.Contains(first, "Second section") || strings.Contains(third, "First section") || strings.Contains(third, "Actualité liée") || !strings.Contains(third, "General condition.") || !strings.Contains(third, "Section condition.") {
		t.Fatalf("first=%q third=%q %v %v", first, third, e, e3)
	}
	if !strings.Contains(s, "Actualité liée") {
		t.Fatal("lost editorial notice from full source")
	}
}
