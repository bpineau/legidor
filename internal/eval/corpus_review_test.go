package eval

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestReviewedFiscalCorpusRetainsEveryUpstreamScenario(t *testing.T) {
	c, err := LoadCorpus(filepath.Join("..", "..", "evals", "full.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 25, 27, 28, 29, 30, 41, 42, 43, 44, 45, 46, 47, 48, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66}
	var got []int
	for _, v := range c.Cases {
		got = append(got, v.ID)
		if v.ExpectedOutput == "" || len(v.Amendments) == 0 || len(v.Files) != 0 || v.metadata["reviewed_on"] == nil {
			t.Fatalf("case %d is not a self-contained reviewed fixture", v.ID)
		}
		for _, text := range append(append([]string{v.Prompt, v.ExpectedOutput}, v.Assertions...), v.Amendments...) {
			if ASCII(text) != text || strings.Contains(text, "/Users/") {
				t.Fatalf("case %d contains nonportable authoring", v.ID)
			}
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("lost upstream coverage: %v", got)
	}
	if len(c.SourceSHA256) != 64 || c.Commit == "" || c.Source == "" {
		t.Fatal("missing upstream provenance")
	}
}
