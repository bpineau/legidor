// Package app implements the CLI independently of process globals and standard streams.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bpineau/legidor/internal/cache"
	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
	"github.com/bpineau/legidor/internal/source"
	legaltext "github.com/bpineau/legidor/internal/text"
)

const Version = "0.1.0"

func asError(e error) *model.Error {
	var m *model.Error
	if errors.As(e, &m) {
		return m
	}
	return model.Err("INTERNAL", 3, "operation failed")
}
func fail(out, errout io.Writer, jsonMode bool, e error) int {
	m := asError(e)
	if jsonMode {
		json.NewEncoder(out).Encode(map[string]any{"schema": model.Schema, "error": m, "complete": false})
	} else {
		fmt.Fprintln(errout, m.Error())
		if m.Next != "" {
			fmt.Fprintln(errout, m.Next)
		}
		for _, v := range m.Candidates {
			fmt.Fprintln(errout, v)
		}
	}
	return m.Exit
}
func Run(ctx context.Context, args []string, in io.Reader, out, errout io.Writer) int {
	jsonMode := false
	for _, a := range args {
		if a == "--json" || a == "--json=true" {
			jsonMode = true
		}
	}
	if len(args) == 0 {
		fmt.Fprint(out, help(""))
		return 0
	}
	cmd := args[0]
	if cmd == "--version" || cmd == "version" {
		if jsonMode {
			json.NewEncoder(out).Encode(map[string]any{"schema": model.Schema, "version": Version})
		} else {
			fmt.Fprintln(out, "legidor "+Version)
		}
		return 0
	}
	if cmd == "--help" || cmd == "help" {
		target := ""
		if len(args) > 1 {
			target = args[1]
		}
		fmt.Fprint(out, help(target))
		return 0
	}
	o, e := parseOptions(args[1:])
	if e != nil {
		return fail(out, errout, jsonMode, e)
	}
	if o.help {
		fmt.Fprint(out, help(cmd))
		return 0
	}
	valid := map[string]bool{"get": true, "resolve": true, "check": true, "refs": true, "search": true, "versions": true, "diff": true, "auth": true, "cache": true}
	if !valid[cmd] {
		return fail(out, errout, o.json, model.Err("USAGE", 2, "unknown command %q; use legidor --help", cmd))
	}
	if len(o.args) != 1 {
		return fail(out, errout, o.json, model.Err("USAGE", 2, "%s expects one argument; quote references containing spaces", cmd))
	}
	if e = validateCommandOptions(cmd, o); e != nil {
		return fail(out, errout, o.json, e)
	}
	arg := o.args[0]
	if cmd == "refs" {
		return refs(arg, o, in, out, errout)
	}
	var r reference.Reference
	if cmd == "get" || cmd == "resolve" || cmd == "versions" || cmd == "diff" {
		r, e = reference.Parse(arg)
		if e == nil {
			r, e = r.Options(o.date, o.part)
		}
		if e != nil {
			return fail(out, errout, o.json, e)
		}
		if o.parts && r.Part != "" {
			return fail(out, errout, o.json, model.Err("CONFLICT", 2, "--parts requires a reference without a subdivision"))
		}
		if cmd == "resolve" && o.parseOnly {
			v := map[string]any{"schema": model.Schema, "reference": r, "resolved": false}
			emit(out, o.json, v)
			return 0
		}
	}
	cfg, e := source.LoadConfig(o.env)
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	disk := cache.New(cfg.CacheDir)
	c := source.NewClient(cfg, disk)
	c.Fresh = o.fresh
	c.MaxAge = o.maxAge
	c.Offline = o.offline
	switch cmd {
	case "auth":
		if arg != "status" {
			return fail(out, errout, o.json, model.Err("USAGE", 2, "use legidor auth status"))
		}
		if o.offline {
			return fail(out, errout, o.json, model.Err("USAGE", 2, "auth status requires network access"))
		}
		v, e := c.AuthStatus(ctx)
		if e != nil {
			return fail(out, errout, o.json, e)
		}
		emit(out, o.json, v)
		return 0
	case "cache":
		if arg == "clear" {
			if e = disk.Clear(); e != nil {
				return fail(out, errout, o.json, model.Err("CACHE", 3, "cannot clear cache"))
			}
			emit(out, o.json, map[string]any{"schema": model.Schema, "cleared": true, "environment": o.env})
			return 0
		}
		if arg != "stats" {
			return fail(out, errout, o.json, model.Err("USAGE", 2, "use cache stats or cache clear"))
		}
		n, size, e := disk.Stats()
		if e != nil {
			return fail(out, errout, o.json, model.Err("CACHE", 3, "cannot read cache"))
		}
		emit(out, o.json, map[string]any{"schema": model.Schema, "entries": n, "bytes": size, "environment": o.env})
		return 0
	case "check":
		return check(ctx, c, arg, o, in, out, errout)
	case "search":
		s, e := c.Search(ctx, arg, o.corpus, o.date, o.cursor, o.limit)
		if e != nil {
			return fail(out, errout, o.json, e)
		}
		if o.json {
			emit(out, true, s)
		} else {
			fmt.Fprintf(out, "source: %s | retrieved_at: %s | cached: %t | offline: %t | environment: %s\nsort: %s | total: %d\n%s\n", s.Provenance.Source, s.Provenance.RetrievedAt, s.Provenance.Cached, s.Provenance.Offline, s.Provenance.Environment, s.Sort, s.Total, s.Stability)
			for _, h := range s.Hits {
				fmt.Fprintf(out, "%s | %s | %s\n%s\n", h.ID, h.Title, h.Number, h.Snippet)
			}
			if s.Cursor != "" {
				fmt.Fprintln(out, "cursor:", s.Cursor)
			}
		}
		return 0
	case "versions":
		vs, e := c.Versions(ctx, r, o.since)
		if e != nil {
			return fail(out, errout, o.json, e)
		}
		emit(out, o.json, map[string]any{"schema": model.Schema, "versions": vs, "complete": true})
		return 0
	case "diff":
		return diff(ctx, c, r, o, out, errout)
	case "get", "resolve":
		if o.cursor != "" {
			var cur pageCursor
			cur, e = decodeCursor(o.cursor)
			if e != nil {
				return fail(out, errout, o.json, e)
			}
			if cur.Raw != arg || cur.Environment != o.env || (o.date != "" && cur.Reference.Date != o.date) {
				return fail(out, errout, o.json, model.Err("CONFLICT", 2, "cursor does not match reference, date or environment"))
			}
			r = cur.Reference
		}
		d, e := c.Get(ctx, r)
		if e != nil {
			return fail(out, errout, o.json, e)
		}
		if cmd == "resolve" {
			d.Text = ""
			d.HTML = ""
			d.Parts = nil
			d.Items = nil
			d.Versions = nil
			emit(out, o.json, d)
			if !d.Complete {
				return 3
			}
			return 0
		}
		if o.links && r.Part != "" {
			return fail(out, errout, o.json, model.Err("CONFLICT", 2, "--links lists document references; use an unselected reference"))
		}
		if o.links {
			if d.HTML == "" {
				return fail(out, errout, o.json, model.Err("UNSUPPORTED_CONTENT", 3, "source does not supply HTML links"))
			}
			d.Items = legaltext.Links(d.HTML, d.Provenance.URL)
			d.ContentKind = "source_references"
			d.Text = ""
			d.Parts = nil
			d, e = paginate(d, r, arg, o)
			if e != nil {
				return fail(out, errout, o.json, e)
			}
			renderDocument(out, o.json, d)
			if !d.Complete {
				return 3
			}
			return 0
		}
		if r.Part != "" {
			d, e = selectPart(d, r.Part)
			if e != nil {
				return fail(out, errout, o.json, e)
			}
		}
		if o.grep != "" {
			ms := legaltext.Grep(d.Text, o.grep, d.Parts)
			emit(out, o.json, map[string]any{"schema": model.Schema, "id": d.ID, "matches": ms, "complete": d.Complete, "provenance": d.Provenance})
			if !d.Complete {
				return 3
			}
			if len(ms) == 0 {
				return 1
			}
			return 0
		}
		if o.parts {
			d.Text = ""
			for i := range d.Parts {
				d.Parts[i].Text = ""
			}
			if d.ContentKind == "table_of_contents" {
				d, e = paginate(d, r, arg, o)
				if e != nil {
					return fail(out, errout, o.json, e)
				}
			}
		} else if r.Part == "" {
			d, e = paginate(d, r, arg, o)
			if e != nil {
				return fail(out, errout, o.json, e)
			}
		}
		d.Versions = nil
		renderDocument(out, o.json, d)
		if !d.Complete {
			return 3
		}
		return 0
	}
	return 2
}
func emit(out io.Writer, jsonMode bool, v any) {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if !jsonMode {
		enc.SetIndent("", "  ")
	}
	enc.Encode(v)
}
func renderDocument(out io.Writer, jsonMode bool, d model.Document) {
	if jsonMode {
		emit(out, true, d)
		return
	}
	fmt.Fprintf(out, "%s | %s -> %s | %s\nnature: %s | retrieved_at: %s | complete: %t\nsource: %s | environment: %s | cached: %t | offline: %t\n", d.Title, d.Start, d.End, d.ID, d.Nature, d.Provenance.RetrievedAt, d.Complete, d.Provenance.Source, d.Provenance.Environment, d.Provenance.Cached, d.Provenance.Offline)
	renderResolution(out, d)
	renderCaveats(out, d)
	fmt.Fprintln(out, d.Provenance.URL)
	fmt.Fprintln(out, "---")
	fmt.Fprintln(out, d.Text)
	if d.Text == "" || d.Omitted > 0 {
		var labels []string
		for _, p := range d.Parts {
			labels = append(labels, p.Label)
		}
		fmt.Fprintln(out, "parts:", strings.Join(labels, ", "))
	}
	for _, item := range d.Items {
		fmt.Fprintf(out, "%s | %s | %s\n", item.ID, item.Title, item.URL)
	}
	if d.Next != "" {
		fmt.Fprintln(out, "next:", d.Next)
	}
}

func renderCaveats(out io.Writer, d model.Document) {
	if d.SourceComplete != nil {
		fmt.Fprintf(out, "source_complete: %t\n", *d.SourceComplete)
	}
	for _, l := range d.Modifications {
		fmt.Fprintf(out, "modified by: %s | %s | %s\n", l.Title, l.Date, l.URL)
	}
	if d.Nota != "" {
		fmt.Fprintln(out, "nota:", d.Nota)
	}
	for _, w := range d.Warnings {
		fmt.Fprintln(out, "warning:", w)
	}
	if len(d.Unsupported) > 0 {
		fmt.Fprintln(out, "unsupported_content:", strings.Join(d.Unsupported, ", "))
	}
}
func renderResolution(out io.Writer, d model.Document) {
	if d.Provenance.ResolutionDate != "" {
		fmt.Fprintf(out, "requested_date: %s | resolution_date: %s | resolution_retrieved_at: %s | resolution_cached: %t\n", d.Date, d.Provenance.ResolutionDate, d.Provenance.ResolutionRetrievedAt, d.Provenance.ResolutionCached)
	}
}
func selectPart(d model.Document, label string) (model.Document, error) {
	for _, p := range d.Parts {
		if p.Label == label && p.Text != "" {
			d.Text = p.Text
			d.Parts = []model.Part{{Label: label, Start: 0, End: len(d.Text), Parent: -1}}
			return d, nil
		}
	}
	s, e := legaltext.Select(d.Text, d.Parts, label)
	if e != nil {
		return d, e
	}
	// Only omissions represented in the source text can be scoped. Missing
	// source zones, truncation and other upstream incompleteness stay failures.
	if d.HTML != "" && len(d.Unsupported) > 0 && len(legaltext.Omissions(d.Text)) == len(d.Unsupported) && d.Omitted == 0 {
		complete := d.Complete
		d.SourceComplete = &complete
		d.Unsupported = legaltext.Omissions(s)
		d.Complete = len(d.Unsupported) == 0 && strings.TrimSpace(s) != ""
	}
	d.Text = s
	d.Parts = []model.Part{{Label: label, Start: 0, End: len(d.Text), Parent: -1}}
	return d, nil
}
func validateCommandOptions(cmd string, o options) error {
	allowed := map[string]string{
		"get": "date part parts grep full cursor links", "resolve": "date parse-only",
		"search": "date in limit cursor", "check": "", "refs": "diff", "versions": "since",
		"diff": "from to part full", "auth": "", "cache": "",
	}
	for name := range o.supplied {
		if strings.Contains(" json help h fresh offline env max-age ", " "+name+" ") {
			continue
		}
		if !strings.Contains(" "+allowed[cmd]+" ", " "+name+" ") {
			return model.Err("USAGE", 2, "--%s is not supported by %s", name, cmd)
		}
	}
	bad := (o.parseOnly && cmd != "resolve") || (o.diff && cmd != "refs") || (o.parts && cmd != "get") || (o.grep != "" && cmd != "get") || (o.cursor != "" && cmd != "get" && cmd != "search") || (o.from != "" && cmd != "diff") || (o.to != "" && cmd != "diff") || (o.since != "" && cmd != "versions")
	if bad {
		return model.Err("USAGE", 2, "option is not supported by %s", cmd)
	}
	if o.parts && (o.part != "" || o.grep != "") {
		return model.Err("CONFLICT", 2, "--parts cannot be combined with --part or --grep")
	}
	if o.links && (o.part != "" || o.parts || o.grep != "") {
		return model.Err("CONFLICT", 2, "--links cannot be combined with text selection")
	}
	if o.cursor != "" && (o.part != "" || o.parts || o.grep != "") {
		return model.Err("CONFLICT", 2, "content cursor cannot be combined with selection")
	}
	return nil
}
