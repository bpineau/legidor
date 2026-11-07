// Package text selects conservative legal subdivisions and checks literal quotes.
package text

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

var tagRE = regexp.MustCompile(`(?s)<[^>]*>`)
var blockRE = regexp.MustCompile(`(?i)</(?:p|div|h[1-6]|li)>|<br\s*/?>`)
var unsafeRE = regexp.MustCompile(`(?is)<(?:script|style)\b[^>]*>.*?</(?:script|style)\s*>`)
var unsupportedBlockRE = regexp.MustCompile(`(?is)<(svg|object|iframe)\b[^>]*>.*?</(?:svg|object|iframe)\s*>`)
var unsupportedRE = regexp.MustCompile(`(?i)<(img|svg|object|iframe|embed)\b[^>]*>`)

func HTML(s string) (string, []string) {
	s = unsafeRE.ReplaceAllString(s, "")
	s = tables(s)
	s = unsupportedBlockRE.ReplaceAllStringFunc(s, func(v string) string { return omitted(strings.ToLower(unsupportedBlockRE.FindStringSubmatch(v)[1])) })
	s = unsupportedRE.ReplaceAllStringFunc(s, func(v string) string { return omitted(strings.ToLower(unsupportedRE.FindStringSubmatch(v)[1])) })
	s = blockRE.ReplaceAllString(s, "\n")
	s = tagRE.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			lines = append(lines, line)
		}
	}
	result := strings.Join(lines, "\n")
	return result, Omissions(result)
}

var labelRE = regexp.MustCompile(`^((?:[IVX]+|[A-Z])(?:\s+` + reference.LatinSuffixPattern + `\b)?)[.\s]*[-.]\s*|^(\d+°(?:-\d+)?(?:\s+` + reference.LatinSuffixPattern + `\b)?)\s*|^(\d+(?:\s+` + reference.LatinSuffixPattern + `\b)?)[.)]\s*|^([a-z])[.)]\s*`)

func Parts(s string) []model.Part {
	var ps []model.Part
	var kinds []string
	offset := 0
	for _, line := range strings.Split(s, "\n") {
		remaining, local := line, 0
		for {
			m := labelRE.FindStringSubmatch(remaining)
			if m == nil {
				break
			}
			label, kind := "", ""
			for i := 1; i < len(m); i++ {
				if m[i] == "" {
					continue
				}
				label = m[i]
				switch i {
				case 1:
					kind = "upper"
					if regexp.MustCompile(`^[IVX]+(?: |$)`).MatchString(label) {
						kind = "roman"
					}
				case 2:
					kind = "degree"
				case 3:
					kind = "number"
				case 4:
					kind = "lower"
				}
				break
			}
			open := map[int]bool{-1: true}
			parent := -1
			found := false
			for i := len(ps) - 1; i >= 0; i = ps[i].Parent {
				open[i] = true
				if !found && kinds[i] == kind {
					parent = ps[i].Parent
					found = true
				}
			}
			if !found {
				for i := len(ps) - 1; i >= 0; i-- {
					if kinds[i] == kind && open[ps[i].Parent] {
						parent = ps[i].Parent
						found = true
						break
					}
				}
			}
			if !found && len(ps) > 0 {
				parent = len(ps) - 1
			}
			level, full := 1, label
			if parent >= 0 {
				level = ps[parent].Level + 1
				full = ps[parent].Label + " " + label
			}
			ps = append(ps, model.Part{Label: full, Start: offset + local, Level: level, Parent: parent})
			kinds = append(kinds, kind)
			local += len(m[0])
			remaining = remaining[len(m[0]):]
		}
		offset += len(line) + 1
	}
	for i := range ps {
		ps[i].End = len(s)
		for j := i + 1; j < len(ps); j++ {
			if ps[j].Level <= ps[i].Level {
				ps[i].End = ps[j].Start
				break
			}
		}
	}
	return ps
}
func Select(s string, ps []model.Part, label string) (string, error) {
	label = strings.Join(strings.Fields(strings.ReplaceAll(label, ",", " ")), " ")
	if strings.HasPrefix(label, "alinéa ") {
		n, e := strconv.Atoi(strings.TrimPrefix(label, "alinéa "))
		lines := strings.Split(s, "\n")
		if e != nil || n < 1 || n > len(lines) {
			return "", model.Err("PART_NOT_FOUND", 1, "paragraph position absent")
		}
		return lines[n-1], nil
	}
	var matches []int
	for i, p := range ps {
		if p.Label == label {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		for i, p := range ps {
			if strings.HasSuffix(p.Label, " "+label) {
				matches = append(matches, i)
			}
		}
	}
	if len(matches) != 1 {
		code := "PART_NOT_FOUND"
		exit := 1
		if len(ps) == 0 {
			code = "STRUCTURE_UNRECOGNIZED"
			exit = 3
		}
		if len(matches) > 1 {
			code = "AMBIGUOUS_PART"
			exit = 3
		}
		return "", &model.Error{Code: code, Exit: exit, Message: "cannot select subdivision " + strconv.Quote(label), Next: "legidor get REF --parts"}
	}
	idx := matches[0]
	p := ps[idx]
	var ancestors []int
	for parent := p.Parent; parent >= 0; parent = ps[parent].Parent {
		ancestors = append(ancestors, parent)
	}
	var out []string
	if len(ps) > 0 && ps[0].Start > 0 {
		out = append(out, strings.TrimSpace(s[:ps[0].Start]))
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		a := ps[ancestors[i]]
		end := a.End
		for _, child := range ps {
			if child.Parent == ancestors[i] {
				end = child.Start
				break
			}
		}
		if end > a.Start {
			out = append(out, strings.TrimSpace(s[a.Start:end]))
		}
	}
	out = append(out, strings.TrimSpace(s[p.Start:p.End]))
	return strings.Join(out, "\n"), nil
}

var markerRE = regexp.MustCompile(`(?is)<p\b[^>]*class\s*=\s*["'][^"']*numero-de-paragraphe-western[^"']*["'][^>]*>\s*(?:<[^>]+>\s*)*(\d+)\s*(?:</[^>]+>\s*)*</p>`)

func BOI(raw string) (string, []model.Part, []string) {
	raw = tagRE.ReplaceAllStringFunc(raw, func(tag string) string { return strings.ReplaceAll(tag, `""`, `"`) })
	markers := markerRE.FindAllStringSubmatchIndex(raw, -1)
	if len(markers) == 0 {
		s, u := HTML(raw)
		return s, nil, u
	}
	var b strings.Builder
	var ps []model.Part
	var unsupported []string
	intro, u := HTML(raw[:markers[0][0]])
	unsupported = append(unsupported, u...)
	if intro != "" {
		b.WriteString(intro + "\n")
	}
	for i, m := range markers {
		end := len(raw)
		if i+1 < len(markers) {
			end = markers[i+1][0]
		}
		label := raw[m[2]:m[3]]
		s, u := HTML(raw[m[1]:end])
		unsupported = append(unsupported, u...)
		start := b.Len()
		b.WriteString(label + "\n" + s + "\n")
		ps = append(ps, model.Part{Label: label, Start: start, End: b.Len() - 1, Parent: -1})
	}
	return strings.TrimSuffix(b.String(), "\n"), ps, unsupported
}
func normalizeQuote(s string) string {
	r := strings.NewReplacer("\u2019", "'", "\u2018", "'", "\u201c", "\"", "\u201d", "\"", "«", "\"", "»", "\"")
	return strings.Join(strings.Fields(r.Replace(s)), " ")
}
func ordered(s, q string) bool {
	any := false
	for _, piece := range strings.Split(q, "[...]") {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			continue
		}
		any = true
		i := -1
		for offset := 0; offset <= len(s); {
			j := strings.Index(s[offset:], piece)
			if j < 0 {
				break
			}
			j += offset
			if quoteBoundary(s, j, j+len(piece)) {
				i = j
				break
			}
			offset = j + 1
		}
		if i < 0 {
			return false
		}
		s = s[i+len(piece):]
	}
	return any
}

func quoteBoundary(s string, start, end int) bool {
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	first, _ := utf8.DecodeRuneInString(s[start:end])
	last, _ := utf8.DecodeLastRuneInString(s[start:end])
	if start > 0 {
		prev, _ := utf8.DecodeLastRuneInString(s[:start])
		if word(first) && word(prev) {
			return false
		}
		if unicode.IsDigit(first) && (prev == '.' || prev == ',') && start > 1 {
			before, _ := utf8.DecodeLastRuneInString(s[:start-1])
			if unicode.IsDigit(before) {
				return false
			}
		}
	}
	if end < len(s) {
		next, n := utf8.DecodeRuneInString(s[end:])
		if word(last) && word(next) {
			return false
		}
		if unicode.IsDigit(last) && (next == '.' || next == ',') && end+n < len(s) {
			after, _ := utf8.DecodeRuneInString(s[end+n:])
			if unicode.IsDigit(after) {
				return false
			}
		}
	}
	return true
}
func Quote(s, q string) string {
	if q == "" {
		return "not_requested"
	}
	if ordered(s, q) {
		return "exact"
	}
	if ordered(normalizeQuote(s), normalizeQuote(q)) {
		return "matches_after_normalization"
	}
	return "differs"
}

// Diff computes a bounded word LCS; large documents use common-prefix/suffix edits.
func Diff(a, b string) string {
	x, y := strings.Fields(a), strings.Fields(b)
	if len(x)*len(y) > 2000000 {
		return coarseDiff(x, y)
	}
	dp := make([][]int, len(x)+1)
	for i := range dp {
		dp[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	var out []string
	var old, new []string
	flush := func() {
		s := ""
		if len(old) > 0 {
			s += "[-" + strings.Join(old, " ") + "-]"
		}
		if len(new) > 0 {
			s += "{+" + strings.Join(new, " ") + "+}"
		}
		if s != "" {
			out = append(out, s)
		}
		old = nil
		new = nil
	}
	i, j := 0, 0
	for i < len(x) || j < len(y) {
		if i < len(x) && j < len(y) && x[i] == y[j] {
			flush()
			out = append(out, x[i])
			i++
			j++
		} else if i < len(x) && (j == len(y) || dp[i+1][j] >= dp[i][j+1]) {
			old = append(old, x[i])
			i++
		} else {
			new = append(new, y[j])
			j++
		}
	}
	flush()
	return strings.Join(out, " ")
}
func coarseDiff(x, y []string) string {
	i := 0
	for i < len(x) && i < len(y) && x[i] == y[i] {
		i++
	}
	j := 0
	for j < len(x)-i && j < len(y)-i && x[len(x)-1-j] == y[len(y)-1-j] {
		j++
	}
	out := append([]string{}, x[:i]...)
	if len(x)-j > i {
		out = append(out, "[-"+strings.Join(x[i:len(x)-j], " ")+"-]")
	}
	if len(y)-j > i {
		out = append(out, "{+"+strings.Join(y[i:len(y)-j], " ")+"+}")
	}
	out = append(out, x[len(x)-j:]...)
	return strings.Join(out, " ")
}

type Match struct {
	Part    string `json:"part,omitempty"`
	Context string `json:"context"`
	Offset  int    `json:"offset"`
}

func Grep(s, q string, ps []model.Part) []Match {
	var ms []Match
	if q == "" {
		return ms
	}
	at := 0
	for at < len(s) {
		i := strings.Index(s[at:], q)
		if i < 0 {
			break
		}
		i += at
		label := ""
		for _, p := range ps {
			if p.Start <= i && i < p.End {
				label = p.Label
			}
		}
		start, end := max(0, i-100), min(len(s), i+len(q)+100)
		for start > 0 && !unicode.IsSpace(rune(s[start-1])) {
			start--
		}
		for end < len(s) && s[end]&0xc0 == 0x80 {
			end++
		}
		ms = append(ms, Match{label, s[start:end], i})
		at = i + len(q)
	}
	return ms
}
