package app

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

const extractionCode = `(?:(?:CGI(?: ann\. (?:II|III|IV|I))?|CSS|CMF|CoMoFi|LPF|CCH|CPCE|CGCT|CSP|CIBS)\b|C\. (?:civ|trav|assur|conso|com|urb)\.)`
const extractionArticle = `(?:[Aa]rt(?:icle)?\.?\s+)?` + reference.ArticleNumberPattern
const extractionParts = `(?:[,\s]+(?:[IVX]+\b|[a-z]\b|\d+°(?:-\d+)?(?:\s+` + reference.LatinSuffixPattern + `)?\b|\d+°|\d+\b))*`

var pluralCodeRE = regexp.MustCompile(`(?i)^articles?\.?\s+`)
var listSeparatorRE = regexp.MustCompile(`\s*(?:,|\bet\b)\s*`)
var trailingCodeRE = regexp.MustCompile(`\s+(` + extractionCode + `)$`)
var leadingCodeRE = regexp.MustCompile(`^(` + extractionCode + `)\s+`)
var ambiguousNumericListRE = regexp.MustCompile(`,\s*\d+(?:\s|$)`)
var bareArticleRE = regexp.MustCompile(`(?i)\bart(?:icle)?\.?\s+` + reference.ArticleNumberPattern)

var extractionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b` + extractionCode + `\s+(?:[Aa]rticles?\.?\s+)?` + reference.ArticleNumberPattern + `(?:\s*(?:,|et)\s*` + reference.ArticleNumberPattern + `)+`),
	regexp.MustCompile(`\b[Aa]rticles\s+` + reference.ArticleNumberPattern + `(?:\s*(?:,|et)\s*` + reference.ArticleNumberPattern + `)+\s+` + extractionCode),
	regexp.MustCompile(`\b` + extractionArticle + `(?:\s*(?:,|et)\s*` + reference.ArticleNumberPattern + `)+\s+` + extractionCode),
	regexp.MustCompile(`BOI-[A-Z0-9]+(?:-[A-Z0-9]+)*(?:\s*§\s*\d+(?:\s*(?:,|et)\s*(?:§\s*)?\d+)*)?`),
	regexp.MustCompile(`(?:LEGIARTI|JORFARTI|JORFTEXT|LEGITEXT|KALIARTI|CETATEXT|JURITEXT|CONSTEXT)\d{12}`),
	regexp.MustCompile(`\b` + extractionCode + `\s+` + extractionArticle + extractionParts),
	regexp.MustCompile(`\b` + extractionArticle + extractionParts + `\s+` + extractionCode),
	regexp.MustCompile(`(?i)(?:loi|décret|ordonnance)\s+(?:n[°o]\s*)?\d{4}-\d+(?:\s+art\.\s+\d+)?`),
	regexp.MustCompile(`(?:LFI|LFSS|LFR) \d{4}(?: art\. \d+)?`),
	regexp.MustCompile(`(?:CE|Cass\. (?:com\.|[123]e civ\.)) \d{4}-\d{2}-\d{2} n° [\d.-]+`),
}

func refs(name string, o options, in io.Reader, out, errout io.Writer) int {
	reader, close, e := inputReader(name, in)
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	defer close()
	s := bufio.NewScanner(reader)
	s.Buffer(make([]byte, 4096), 2<<20)
	line := 0
	count := 0
	exit := 0
	for s.Scan() {
		line++
		text := s.Text()
		change := "context"
		if o.diff {
			if strings.HasPrefix(text, "+++") || strings.HasPrefix(text, "---") || strings.HasPrefix(text, "@@") {
				continue
			}
			if strings.HasPrefix(text, "+") {
				change = "added"
				text = text[1:]
			} else if strings.HasPrefix(text, "-") {
				change = "removed"
				text = text[1:]
			}
		}
		seen := map[string]bool{}
		var covered [][2]int
		for _, pat := range extractionPatterns {
			for _, loc := range pat.FindAllStringIndex(text, -1) {
				// A numeric token followed by a degree sign is a subdivision,
				// not a complete article number. Let the part pattern consume it.
				if strings.HasPrefix(text[loc[1]:], "°") {
					continue
				}
				overlap := false
				for _, span := range covered {
					if loc[0] < span[1] && loc[1] > span[0] {
						overlap = true
						break
					}
				}
				if overlap {
					continue
				}
				match := strings.TrimSpace(text[loc[0]:loc[1]])
				covered = append(covered, [2]int{loc[0], loc[1]})
				matches, ambiguous := expandReferences(match)
				if ambiguous {
					fmt.Fprintf(errout, "%s:%d: ambiguous citation %q; write 'articles ...' for a list or use an explicit part in the manifest.\n", name, line, match)
					exit = 3
					continue
				}
				for _, match := range matches {
					if seen[match] {
						continue
					}
					seen[match] = true
					if _, e := reference.Parse(match); e != nil {
						continue
					}
					row := model.Citation{Ref: match, Where: fmt.Sprintf("%s:%d", name, line)}
					if o.diff {
						row.Change = change
					}
					emit(out, true, row)
					count++
				}
			}
		}
		for _, loc := range bareArticleRE.FindAllStringIndex(text, -1) {
			coveredByReference := false
			for _, span := range covered {
				if loc[0] < span[1] && loc[1] > span[0] {
					coveredByReference = true
					break
				}
			}
			if !coveredByReference {
				fmt.Fprintf(errout, "%s:%d: citation candidate %q has no recognized code; complete the manifest manually.\n", name, line, text[loc[0]:loc[1]])
				exit = 3
			}
		}
	}
	if s.Err() != nil {
		return fail(out, errout, o.json, model.Err("INPUT", 2, "cannot read input or line exceeds 2 MiB"))
	}
	fmt.Fprintf(errout, "Detected %d references. This is not exhaustive. Review the manifest, dates and subdivisions before check; removed lines are not automatically excluded.\n", count)
	return exit
}

func expandReferences(s string) ([]string, bool) {
	if m := leadingCodeRE.FindStringSubmatchIndex(s); m != nil {
		body := s[m[1]:]
		if strings.HasPrefix(strings.ToLower(body), "articles ") {
			var refs []string
			for _, article := range listSeparatorRE.Split(pluralCodeRE.ReplaceAllString(body, ""), -1) {
				refs = append(refs, s[m[2]:m[3]]+" "+article)
			}
			return refs, false
		}
	}
	if strings.HasPrefix(s, "BOI-") {
		if base, list, ok := strings.Cut(s, "§"); ok {
			var refs []string
			for _, part := range listSeparatorRE.Split(strings.TrimSpace(list), -1) {
				refs = append(refs, strings.TrimSpace(base)+" §"+strings.TrimSpace(strings.TrimPrefix(part, "§")))
			}
			return refs, false
		}
	}
	if strings.HasPrefix(strings.ToLower(s), "articles ") {
		m := trailingCodeRE.FindStringSubmatchIndex(s)
		if m != nil {
			var refs []string
			code := s[m[2]:m[3]]
			list := pluralCodeRE.ReplaceAllString(s[:m[0]], "")
			for _, article := range listSeparatorRE.Split(list, -1) {
				refs = append(refs, code+" "+article)
			}
			return refs, false
		}
	}
	if ambiguousNumericListRE.MatchString(s) || strings.Contains(s, " et ") {
		return nil, true
	}
	return []string{s}, false
}
