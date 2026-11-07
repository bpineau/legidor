package text

import (
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/bpineau/legidor/internal/model"
)

var anchorRE = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a\s*>`)
var hrefRE = regexp.MustCompile(`(?is)(?:^|\s)href\s*=\s*(?:"([^"]*)"|'([^']*)')`)
var linkIDRE = regexp.MustCompile(`(?:LEGIARTI|JORFARTI|LEGITEXT|JORFTEXT|CETATEXT|JURITEXT|LEGISCTA)\d{12}|BOI-[A-Z0-9]+(?:-[A-Z0-9]+)*`)

// Links returns explicit links to supported official sources, never inferred
// references or fetched targets. Relative URLs use the document's permalink.
func Links(raw, base string) []model.Link {
	b, err := url.Parse(base)
	if err != nil {
		return nil
	}
	raw = tagRE.ReplaceAllStringFunc(raw, func(tag string) string { return strings.ReplaceAll(tag, `""`, `"`) })
	var links []model.Link
	seen := map[string]bool{}
	for _, m := range anchorRE.FindAllStringSubmatch(raw, -1) {
		h := hrefRE.FindStringSubmatch(m[1])
		if h == nil {
			continue
		}
		value := h[1]
		if value == "" {
			value = h[2]
		}
		if strings.HasPrefix(value, "#") {
			continue
		}
		u, err := url.Parse(html.UnescapeString(value))
		if err != nil {
			continue
		}
		u = b.ResolveReference(u)
		if u.Scheme != "https" || u.User != nil || u.Port() != "" {
			continue
		}
		switch u.Hostname() {
		case "legifrance.gouv.fr", "www.legifrance.gouv.fr", "bofip.impots.gouv.fr", "www.courdecassation.fr":
		default:
			continue
		}
		if seen[u.String()] {
			continue
		}
		seen[u.String()] = true
		title, _ := HTML(m[2])
		id := linkIDRE.FindString(u.Path)
		if id == "" {
			// Legacy URLs can name both the target and its containing code.
			// Query order must not make the parent look like the target.
			for _, key := range []string{"idArticle", "idSection", "cidTexte"} {
				value := u.Query().Get(key)
				if value != "" && linkIDRE.FindString(value) == value {
					id = value
					break
				}
			}
		}
		links = append(links, model.Link{ID: id, Title: title, URL: u.String(), Type: "source_reference"})
	}
	return links
}
