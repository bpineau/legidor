package text

import (
	"encoding/xml"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var tableTagRE = regexp.MustCompile(`(?is)<\s*(/?)table\b[^>]*>`)
var omissionRE = regexp.MustCompile(`\[\[legidor:omitted:([a-z]+)\]\]`)

func omitted(kind string) string { return "\n[[legidor:omitted:" + kind + "]]\n" }

// Omissions identifies placeholders emitted by HTML. Keeping their positions in
// the text lets selections retain omissions in both the body and introductions.
func Omissions(s string) []string {
	var kinds []string
	for _, m := range omissionRE.FindAllStringSubmatch(s, -1) {
		kinds = append(kinds, m[1])
	}
	return kinds
}

func tables(raw string) string {
	var out strings.Builder
	depth, start, last := 0, 0, 0
	for _, m := range tableTagRE.FindAllStringSubmatchIndex(raw, -1) {
		closing := raw[m[2]:m[3]] == "/"
		if !closing {
			if depth == 0 {
				start = m[0]
			}
			depth++
		} else if depth > 0 {
			depth--
			if depth == 0 {
				out.WriteString(raw[last:start])
				if rendered, ok := table(raw[start:m[1]]); ok {
					out.WriteString("\n" + html.EscapeString(rendered) + "\n")
				} else {
					out.WriteString(omitted("table"))
				}
				last = m[1]
			}
		}
	}
	if depth > 0 {
		out.WriteString(raw[last:start])
		out.WriteString(omitted("table"))
	} else {
		out.WriteString(raw[last:])
	}
	return out.String()
}

// table handles rectangular data tables and full-width notes after a regular
// row. Other merged cells and nested structures fail rather than changing a
// formula's row/column association.
func table(raw string) (string, bool) {
	d := xml.NewDecoder(strings.NewReader(raw))
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	var rows [][]string
	var row []string
	wideRows := map[int]bool{}
	width, span := 0, 1
	var cell, caption strings.Builder
	inRow, inCell, inCaption, seenTable, closedTable := false, false, false, false, false
	header, firstRow := true, true
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false
		}
		switch v := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(v.Name.Local)
			if closedTable {
				return "", false
			}
			switch name {
			case "table":
				if seenTable {
					return "", false
				}
				seenTable = true
			case "thead", "tbody", "tfoot", "colgroup", "col":
				if inRow || inCell || inCaption {
					return "", false
				}
			case "caption":
				if inRow || inCaption {
					return "", false
				}
				inCaption = true
			case "tr":
				if inRow || inCaption {
					return "", false
				}
				inRow = true
				row = nil
				span = 1
			case "td", "th":
				if !inRow || inCell || span > 1 {
					return "", false
				}
				for _, a := range v.Attr {
					key := strings.ToLower(a.Name.Local)
					if key == "rowspan" && strings.TrimSpace(a.Value) != "1" {
						return "", false
					}
					if key == "colspan" {
						n, err := strconv.Atoi(strings.TrimSpace(a.Value))
						if err != nil || n < 1 || n > 128 || (n > 1 && len(row) > 0) {
							return "", false
						}
						span = n
					}
				}
				inCell = true
				cell.Reset()
				if firstRow && name != "th" {
					header = false
				}
			case "p", "div", "br", "li", "ul", "ol", "span", "a", "b", "i", "em", "strong", "sup", "sub", "u", "small":
				if !inCell && !inCaption {
					return "", false
				}
				if name == "br" || name == "p" || name == "div" || name == "li" {
					cell.WriteByte(' ')
					if inCaption {
						caption.WriteByte(' ')
					}
				}
				if name == "sup" || name == "sub" {
					prefix := "^("
					if name == "sub" {
						prefix = "_("
					}
					if inCell {
						cell.WriteString(prefix)
					} else {
						caption.WriteString(prefix)
					}
				}
			default:
				return "", false
			}
		case xml.EndElement:
			switch strings.ToLower(v.Name.Local) {
			case "table":
				closedTable = true
			case "caption":
				inCaption = false
			case "td", "th":
				if !inCell {
					return "", false
				}
				row = append(row, strings.Join(strings.Fields(cell.String()), " "))
				inCell = false
				if len(row) > 128 {
					return "", false
				}
			case "tr":
				if !inRow || inCell || len(row) == 0 {
					return "", false
				}
				if span > 1 {
					if len(row) != 1 || span != width {
						return "", false
					}
					wideRows[len(rows)] = true
				} else if width == 0 {
					width = len(row)
				} else if len(row) != width {
					return "", false
				}
				rows = append(rows, row)
				inRow = false
				firstRow = false
				if len(rows) > 4096 {
					return "", false
				}
			case "p", "div", "li":
				if inCell {
					cell.WriteByte(' ')
				}
				if inCaption {
					caption.WriteByte(' ')
				}
			case "sup", "sub":
				if inCell {
					cell.WriteByte(')')
				} else if inCaption {
					caption.WriteByte(')')
				}
			}
		case xml.CharData:
			if inCell {
				cell.Write(v)
			} else if inCaption {
				caption.Write(v)
			} else if strings.TrimSpace(string(v)) != "" {
				return "", false
			}
		case xml.Comment:
		default:
			return "", false
		}
	}
	if !closedTable || len(rows) == 0 || inRow || inCell {
		return "", false
	}
	var out strings.Builder
	if s := strings.Join(strings.Fields(caption.String()), " "); s != "" {
		out.WriteString(s + "\n")
	}
	for i, row := range rows {
		if wideRows[i] {
			out.WriteString("Table-wide: " + row[0] + "\n")
			continue
		}
		out.WriteString("|")
		for _, s := range row {
			out.WriteString(" " + strings.ReplaceAll(s, "|", `\|`) + " |")
		}
		out.WriteByte('\n')
		if i == 0 && header {
			out.WriteString("|" + strings.Repeat(" --- |", len(row)) + "\n")
		}
	}
	return strings.TrimSuffix(out.String(), "\n"), true
}
