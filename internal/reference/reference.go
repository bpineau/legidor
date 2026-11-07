// Package reference parses legal references without network access or guessed IDs.
package reference

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bpineau/legidor/internal/model"
)

type Reference struct {
	Kind         string `json:"kind"`
	Raw          string `json:"raw"`
	Code         string `json:"code,omitempty"`
	Number       string `json:"number,omitempty"`
	Article      string `json:"article,omitempty"`
	Part         string `json:"part,omitempty"`
	ID           string `json:"id,omitempty"`
	Parent       string `json:"parent,omitempty"`
	End          string `json:"end,omitempty"`
	Date         string `json:"date,omitempty"`
	DecisionDate string `json:"decision_date,omitempty"`
	URL          string `json:"url,omitempty"`
	Nature       string `json:"nature,omitempty"`
}
type Code struct {
	Key, Name string
	Aliases   []string
}

var Codes = []Code{
	{"CGI-II", "Code général des impôts, annexe II", []string{"CGI ann. II", "CGI annexe II", "CGI-II"}},
	{"CGI-III", "Code général des impôts, annexe III", []string{"CGI ann. III", "CGI annexe III", "CGI-III"}},
	{"CGI-IV", "Code général des impôts, annexe IV", []string{"CGI ann. IV", "CGI annexe IV", "CGI-IV"}},
	{"CGI-I", "Code général des impôts, annexe I", []string{"CGI ann. I", "CGI annexe I", "CGI-I"}},
	{"CGI", "Code général des impôts", []string{"CGI"}}, {"LPF", "Livre des procédures fiscales", []string{"LPF"}},
	{"CSS", "Code de la sécurité sociale", []string{"CSS"}}, {"CMF", "Code monétaire et financier", []string{"CMF", "CoMoFi"}},
	{"CIV", "Code civil", []string{"C. civ.", "C civ", "CIV"}}, {"CCH", "Code de la construction et de l'habitation", []string{"CCH"}},
	{"ASSUR", "Code des assurances", []string{"C. assur.", "ASSUR"}}, {"TRAV", "Code du travail", []string{"C. trav.", "TRAV"}},
	{"CPCE", "Code des procédures civiles d'exécution", []string{"CPCE"}}, {"CONSO", "Code de la consommation", []string{"C. conso.", "CONSO"}},
	{"COM", "Code de commerce", []string{"C. com.", "COM"}}, {"CGCT", "Code général des collectivités territoriales", []string{"CGCT"}},
	{"CSP", "Code de la santé publique", []string{"CSP"}}, {"URB", "Code de l'urbanisme", []string{"C. urb.", "URB"}},
}

func CodeName(key string) string {
	for _, c := range Codes {
		if c.Key == key {
			return c.Name
		}
	}
	return ""
}
func CodeKey(name string) string {
	for _, c := range Codes {
		if strings.EqualFold(name, c.Name) {
			return c.Key
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(name, a) {
				return c.Key
			}
		}
	}
	return ""
}

var articleRE = regexp.MustCompile(`^([LRDA]?\*?\s*\.?\s*\d+(?:-\d+)*(?:\s+[A-Z]{1,3})?(?:\s+(?:bis|ter|quater|quinquies|sexies|septies|octies|nonies|decies|undecies|duodecies|terdecies|quaterdecies|quindecies|sexdecies|septdecies|octodecies|novodecies|vicies|unvicies|duovicies|tervicies|quatervicies|quinvicies|sexvicies|septvicies|octovicies|novovicies|tricies))?)(.*)$`)
var idRE = regexp.MustCompile(`^(LEGIARTI|JORFARTI|KALIARTI|LEGITEXT|JORFTEXT|KALITEXT|LEGISCTA|KALISCTA|CETATEXT|JURITEXT|CONSTEXT)\d{12}$`)
var boiRE = regexp.MustCompile(`^(BOI-[A-Z0-9]+(?:-[A-Z0-9]+)*?)(?:\s*§\s*(\d+))?$`)
var lawRE = regexp.MustCompile(`(?i)^(loi|décret|decret|ordonnance|arrêté|arrete)\s+(?:n[°o]\s*)?(\d{4}-\d+)(?:\s+art(?:icle)?\.?\s+([\w.-]+))?$`)
var decisionRE = regexp.MustCompile(`(?i)^(CE|CAA|TA|Cass\.(?:\s+[a-z0-9èé]+\.?)*)\s+(\d{4}-\d{2}-\d{2})\s+n[°o]\s*([\d.-]+)$`)
var partRE = regexp.MustCompile(`^(?:(?:[IVX]+|[A-Z]|\d+°?|[a-z])(?:\s+(?:bis|ter|quater|quinquies|sexies|septies|octies|nonies|decies|undecies|duodecies|terdecies|quaterdecies|quindecies|sexdecies|septdecies|octodecies|novodecies|vicies|unvicies|duovicies|tervicies|quatervicies|quinvicies|sexvicies|septvicies|octovicies|novovicies|tricies))?|alinéa\s+\d+)(?:\s+(?:(?:[IVX]+|[A-Z]|\d+°?|[a-z])(?:\s+(?:bis|ter|quater|quinquies|sexies|septies|octies|nonies|decies|undecies|duodecies|terdecies|quaterdecies|quindecies|sexdecies|septdecies|octodecies|novodecies|vicies|unvicies|duovicies|tervicies|quatervicies|quinvicies|sexvicies|septvicies|octovicies|novovicies|tricies))?))*$`)

func NormalizeNumber(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`^([LRDA])\s*\.?\s*(\*?)\s*`).ReplaceAllString(s, "$1$2")
	return strings.Join(strings.Fields(s), " ")
}
func NormalizePart(s string) string {
	s = strings.Trim(s, " ,.")
	s = strings.ReplaceAll(s, ",", " ")
	return strings.Join(strings.Fields(s), " ")
}
func ValidDate(s string) bool { _, e := time.Parse("2006-01-02", s); return e == nil }
func (r Reference) Options(date, part string) (Reference, error) {
	if date != "" {
		if !ValidDate(date) {
			return r, model.Err("USAGE", 2, "date must be YYYY-MM-DD")
		}
		if r.Date != "" && r.Date != date {
			return r, model.Err("CONFLICT", 2, "reference date conflicts with --date")
		}
		r.Date = date
	}
	part = NormalizePart(part)
	if part != "" {
		if r.Part != "" && r.Part != part {
			return r, model.Err("CONFLICT", 2, "reference subdivision conflicts with --part")
		}
		r.Part = part
	}
	return r, nil
}
func Parse(input string) (Reference, error) {
	s := strings.Join(strings.Fields(input), " ")
	r := Reference{Raw: input}
	bad := func() (Reference, error) {
		return r, model.Err("INVALID_REFERENCE", 2, "unrecognized reference %q; use legidor resolve REF --parse-only", input)
	}
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		u, e := url.Parse(s)
		if e != nil || u.User != nil || u.Scheme != "https" || u.Port() != "" {
			return bad()
		}
		r.URL = s
		switch u.Hostname() {
		case "www.legifrance.gouv.fr", "legifrance.gouv.fr":
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			parent := ""
			for _, p := range parts {
				if strings.HasPrefix(p, "LEGITEXT") && idRE.MatchString(p) {
					parent = p
				}
			}
			for i := len(parts) - 1; i >= 0; i-- {
				p := parts[i]
				if idRE.MatchString(p) {
					rr, err := Parse(p)
					rr.Raw = input
					rr.URL = s
					if rr.Kind == "section" {
						rr.Parent = parent
					}
					for _, d := range parts {
						if ValidDate(d) {
							rr.Date = d
						}
					}
					for _, key := range []string{"dateTexte", "dateVersion"} {
						if d := u.Query().Get(key); d != "" {
							if len(d) == 8 {
								d = d[:4] + "-" + d[4:6] + "-" + d[6:]
							}
							if !ValidDate(d) || (rr.Date != "" && rr.Date != d) {
								return bad()
							}
							rr.Date = d
						}
					}
					return rr, err
				}
			}
			if strings.HasPrefix(u.Path, "/eli/") {
				r.Kind = "eli"
				return r, nil
			}
		case "bofip.impots.gouv.fr":
			i := strings.Index(s, "identifiant=")
			if i >= 0 {
				v := s[i+len("identifiant="):]
				v = strings.Split(v, "#")[0]
				dateRE := regexp.MustCompile(`-(\d{8})$`)
				m := dateRE.FindStringSubmatch(v)
				if len(m) > 0 {
					v = strings.TrimSuffix(v, "-"+m[1])
					r.Date = m[1][:4] + "-" + m[1][4:6] + "-" + m[1][6:]
					if !ValidDate(r.Date) {
						return bad()
					}
				}
				if boiRE.MatchString(v) {
					r.Kind = "bofip"
					r.Number = v
					return r, nil
				}
			}
		}
		return bad()
	}
	if idRE.MatchString(s) {
		r.ID = s
		r.Kind = "article"
		switch {
		case strings.HasPrefix(s, "LEGISCTA"), strings.HasPrefix(s, "KALISCTA"):
			r.Kind = "section"
		case strings.Contains(s, "TEXT"):
			r.Kind = "text"
			if strings.HasPrefix(s, "CETAT") || strings.HasPrefix(s, "JURI") || strings.HasPrefix(s, "CONS") {
				r.Kind = "decision"
			}
		}
		return r, nil
	}
	if m := regexp.MustCompile(`^((?:JORFTEXT|LEGITEXT|KALITEXT)\d{12})\s+art\.\s+([0-9A-Za-z.-]+)$`).FindStringSubmatch(s); m != nil {
		r.Kind = "text"
		r.ID = m[1]
		r.Article = m[2]
		return r, nil
	}
	if strings.HasPrefix(s, "JUDI:") {
		r.Kind = "decision"
		r.Code = "JUDILIBRE"
		r.ID = strings.TrimPrefix(s, "JUDI:")
		if !regexp.MustCompile(`^[a-f0-9]{24}$`).MatchString(r.ID) {
			return bad()
		}
		return r, nil
	}
	if m := boiRE.FindStringSubmatch(s); m != nil {
		r.Kind = "bofip"
		r.Number = m[1]
		r.Part = m[2]
		return r, nil
	}
	if strings.HasPrefix(s, "ECLI:") {
		if !regexp.MustCompile(`^ECLI:FR:[A-Z0-9]+:[0-9]{4}:[A-Z0-9.]+$`).MatchString(s) {
			return bad()
		}
		r.Kind = "decision"
		r.Number = s
		r.Code = "CETAT"
		if strings.HasPrefix(s, "ECLI:FR:CCASS:") {
			r.Code = "JUDILIBRE"
		}
		if strings.HasPrefix(s, "ECLI:FR:CC:") {
			r.Code = "CONSTIT"
		}
		return r, nil
	}
	if strings.HasPrefix(s, "NOR ") {
		r.Kind = "nor"
		r.Number = strings.TrimPrefix(s, "NOR ")
		if !regexp.MustCompile(`^[A-Z]{4}\d{7}[A-Z]$`).MatchString(r.Number) {
			return bad()
		}
		return r, nil
	}
	if m := lawRE.FindStringSubmatch(s); m != nil {
		r.Kind = "law"
		r.Nature = strings.ToLower(m[1])
		r.Number = m[2]
		r.Article = m[3]
		return r, nil
	}
	if m := regexp.MustCompile(`^(LFI|LFSS|LFR) (\d{4})(?: art\. (\d+))?$`).FindStringSubmatch(s); m != nil {
		r.Kind = "named_law"
		r.Nature = m[1]
		r.Number = m[2]
		r.Article = m[3]
		if m[1] == "LFR" {
			return r, model.Err("AMBIGUOUS_REFERENCE", 2, "LFR can name several laws; use the law number")
		}
		return r, nil
	}
	if m := decisionRE.FindStringSubmatch(s); m != nil {
		r.Kind = "decision"
		r.Code = "CETAT"
		r.Nature = strings.ToUpper(m[1])
		if strings.HasPrefix(strings.ToLower(m[1]), "cass") {
			r.Code = "JUDILIBRE"
		}
		r.Number = m[3]
		r.DecisionDate = m[2]
		if !ValidDate(r.DecisionDate) {
			return bad()
		}
		return r, nil
	}
	if m := regexp.MustCompile(`^QPC (\d{4}-\d+)$`).FindStringSubmatch(s); m != nil {
		r.Kind = "decision"
		r.Code = "CONSTIT"
		r.Number = m[1] + " QPC"
		return r, nil
	}
	s = regexp.MustCompile(`(?i)^art(?:icle)?\.? +`).ReplaceAllString(s, "")
	type alias struct{ value, key string }
	var aliases []alias
	for _, c := range Codes {
		for _, a := range append([]string{c.Name}, c.Aliases...) {
			aliases = append(aliases, alias{a, c.Key})
		}
	}
	sort.SliceStable(aliases, func(i, j int) bool { return len(aliases[i].value) > len(aliases[j].value) })
	for _, a := range aliases {
		low := strings.ToLower(s)
		al := strings.ToLower(a.value)
		if strings.HasPrefix(low, al+" ") {
			r.Code = a.key
			s = strings.TrimSpace(s[len(a.value):])
			break
		}
		if strings.HasSuffix(low, " "+al) {
			r.Code = a.key
			s = strings.TrimSpace(s[:len(s)-len(a.value)])
			break
		}
	}
	if r.Code == "" {
		return bad()
	}
	s = regexp.MustCompile(`(?i)^art(?:icle)?\.?\s+`).ReplaceAllString(s, "")
	if strings.HasPrefix(s, "section ") {
		id := strings.TrimPrefix(s, "section ")
		if !idRE.MatchString(id) || !strings.HasPrefix(id, "LEGISCTA") {
			return bad()
		}
		r.Kind = "section"
		r.ID = id
		return r, nil
	}
	if strings.Contains(s, "..") {
		a, b, _ := strings.Cut(s, "..")
		m := articleRE.FindStringSubmatch(strings.TrimSpace(a))
		n := articleRE.FindStringSubmatch(strings.TrimSpace(b))
		if m == nil || n == nil || m[2] != "" || n[2] != "" {
			return bad()
		}
		r.Kind = "range"
		r.Number = NormalizeNumber(m[1])
		r.End = NormalizeNumber(n[1])
		return r, nil
	}
	m := articleRE.FindStringSubmatch(s)
	if m == nil {
		return bad()
	}
	r.Kind = "article"
	r.Number = NormalizeNumber(m[1])
	r.Part = NormalizePart(m[2])
	// A roman numeral following a numbered article is a subdivision, not a suffix.
	fields := strings.Fields(r.Number)
	if len(fields) > 1 && regexp.MustCompile(`^[LRDA]`).MatchString(r.Number) && regexp.MustCompile(`^[IVX]+$`).MatchString(fields[len(fields)-1]) {
		r.Part = strings.TrimSpace(fields[len(fields)-1] + " " + r.Part)
		r.Number = strings.Join(fields[:len(fields)-1], " ")
	}
	if r.Part != "" && !partRE.MatchString(r.Part) {
		return bad()
	}
	return r, nil
}
