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
	s, u := HTML(`<p>I.- A &amp; B</p><p>2° Case</p><table><tr><td>Complex</td></tr></table>`)
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
