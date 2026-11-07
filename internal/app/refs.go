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

var extractionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`BOI-[A-Z0-9]+(?:-[A-Z0-9]+)*(?:\s*§\s*\d+)?`),
	regexp.MustCompile(`(?:LEGIARTI|JORFARTI|JORFTEXT|LEGITEXT|KALIARTI|CETATEXT|JURITEXT|CONSTEXT)\d{12}`),
	regexp.MustCompile(`(?:CGI(?: ann\. (?:II|III|IV|I))?|CSS|CMF|CoMoFi|LPF|CCH|CPCE|CGCT|CSP|C\. (?:civ|trav|assur|conso|com|urb)\.)\s+(?:art\.\s+)?[LRDA]?\.?\s*\d+(?:-\d+)*(?:\s+[A-Z]{1,3}\b)?(?:\s+\d+°(?:\s+bis)?)?`),
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
		for _, pat := range extractionPatterns {
			for _, match := range pat.FindAllString(text, -1) {
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
	if s.Err() != nil {
		return fail(out, errout, o.json, model.Err("INPUT", 2, "cannot read input or line exceeds 2 MiB"))
	}
	fmt.Fprintf(errout, "Detected %d references. This is not exhaustive. Review the manifest, dates and subdivisions before check; removed lines are not automatically excluded.\n", count)
	return 0
}
