package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
	"github.com/bpineau/legidor/internal/source"
	legaltext "github.com/bpineau/legidor/internal/text"
)

var romanArticleSuffixRE = regexp.MustCompile(`^(\d+(?:-\d+)*) ([IVX]+)$`)

func inputReader(name string, in io.Reader) (io.Reader, func(), error) {
	if name == "-" {
		return in, func() {}, nil
	}
	f, e := os.Open(name)
	if e != nil {
		return nil, func() {}, model.Err("INPUT", 2, "cannot read input file")
	}
	return f, func() { f.Close() }, nil
}
func check(ctx context.Context, c *source.Client, name string, o options, in io.Reader, out, errout io.Writer) int {
	reader, close, e := inputReader(name, in)
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	defer close()
	s := bufio.NewScanner(reader)
	s.Buffer(make([]byte, 4096), 2<<20)
	exit := 0
	type fetched struct {
		d model.Document
		e error
	}
	memo := map[string]fetched{}
	line := 0
	for s.Scan() {
		line++
		b := bytes.TrimSpace(s.Bytes())
		if len(b) == 0 {
			continue
		}
		result := model.Check{Schema: model.Schema, Identity: "not_evaluated", Version: "not_evaluated", PartStatus: "not_requested", QuoteStatus: "not_requested", Applicability: "not_evaluated"}
		var citation model.Citation
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		err := dec.Decode(&citation)
		var extra any
		if err == nil && dec.Decode(&extra) != io.EOF {
			err = fmt.Errorf("trailing data")
		}
		result.Input = citation
		if err != nil || citation.Ref == "" {
			result.Error = model.Err("INVALID_MANIFEST", 2, "line %d must contain one citation object with ref", line)
			exit = max(exit, 2)
			emitCheck(out, o.json, result)
			continue
		}
		if citation.Part != "" {
			result.PartStatus = "not_evaluated"
		}
		if citation.Quote != "" {
			result.QuoteStatus = "not_evaluated"
		}
		r, e := reference.Parse(citation.Ref)
		if e == nil {
			r, e = r.Options(citation.Date, citation.Part)
		}
		if e != nil {
			result.Error = asError(e)
			exit = max(exit, result.Error.Exit)
			emitCheck(out, o.json, result)
			continue
		}
		part := r.Part
		r.Part = ""
		r.Raw = ""
		key, _ := json.Marshal(r)
		v, ok := memo[string(key)]
		if !ok {
			v.d, v.e = c.Get(ctx, r)
			memo[string(key)] = v
		}
		d := v.d
		// Preserve real letter-suffixed articles. Only retry after both dated
		// and all-state lookup established that the exact number was absent.
		if problem, ok := v.e.(*model.Error); ok && problem.ExactArticleAbsent && r.Kind == "article" && r.ID == "" {
			if m := romanArticleSuffixRE.FindStringSubmatch(r.Number); m != nil {
				alternative := r
				alternative.Number = m[1]
				alternativeKey, _ := json.Marshal(alternative)
				retry, ok := memo[string(alternativeKey)]
				if !ok {
					retry.d, retry.e = c.Get(ctx, alternative)
					memo[string(alternativeKey)] = retry
				}
				if retry.e == nil {
					d, v.e = retry.d, nil
					part = strings.TrimSpace(m[2] + " " + part)
					d.Warnings = append(append([]string{}, d.Warnings...), fmt.Sprintf("Reinterpreted absent article %s as article %s, subdivision %s after exact lookup failed; verify this reading against the cited claim.", r.Number, alternative.Number, part))
				} else if failure, ok := retry.e.(*model.Error); !ok || failure.Code != "NOT_FOUND_IN_CORPUS" {
					// An attempted reinterpretation cannot hide an API failure or
					// pretend to confirm the original citation's identity.
					v.e = model.Err("REINTERPRETATION_FAILED", 3, "exact article absent; checking article %s with subdivision %s failed: %v", alternative.Number, m[2], retry.e)
				}
			}
		}
		if v.e != nil {
			result.Error = asError(v.e)
			switch result.Error.Code {
			case "IDENTITY_MISMATCH":
				result.Identity = "mismatch"
			case "AMBIGUOUS_REFERENCE":
				result.Identity = "ambiguous"
			case "NOT_FOUND_IN_CORPUS":
				result.Identity = "not_found_in_corpus"
			case "VERSION_CONFLICT":
				result.Identity = "confirmed"
				result.Version = "not_in_force_at_date"
			case "NO_VERSION_AT_DATE":
				result.Identity = "confirmed"
				result.Version = "no_version_at_date"
			}
			exit = max(exit, result.Error.Exit)
		} else {
			result.Identity = "confirmed"
			if d.Date != "" && d.Start != "" {
				result.Version = "in_force_at_date"
			}
			for _, version := range d.Versions {
				if version.Start > d.Start && (result.ChangedSince == "" || version.Start < result.ChangedSince) {
					result.ChangedSince = version.Start
				}
			}
			if part != "" {
				selected, e := selectPart(d, part)
				if e != nil {
					result.Error = asError(e)
					result.PartStatus = "part_not_found"
					if result.Error.Code == "STRUCTURE_UNRECOGNIZED" || result.Error.Code == "AMBIGUOUS_PART" {
						result.PartStatus = "structure_unrecognized"
					}
					exit = max(exit, result.Error.Exit)
				} else {
					d = selected
					result.PartStatus = "found"
				}
			}
			if result.Error == nil {
				if !d.Complete || (d.ContentKind == "table_of_contents" && citation.Quote != "") {
					result.Error = model.Err("UNSUPPORTED_CONTENT", 3, "source content is incomplete; citation cannot be established")
					exit = max(exit, 3)
				} else {
					result.QuoteStatus = legaltext.Quote(d.Text, citation.Quote)
					if result.QuoteStatus == "differs" {
						exit = max(exit, 1)
					}
				}
			}
		}
		d.Text = ""
		d.HTML = ""
		d.Parts = nil
		d.Versions = nil
		d.Items = nil
		if d.ID != "" {
			d = compactMetadata(d, o.full)
			result.Document = &d
		}
		emitCheck(out, o.json, result)
	}
	if s.Err() != nil {
		return max(exit, fail(out, errout, o.json, model.Err("INVALID_MANIFEST", 2, "cannot read manifest or line exceeds 2 MiB")))
	}
	return exit
}
func emitCheck(out io.Writer, jsonMode bool, r model.Check) {
	if jsonMode {
		emit(out, true, r)
		return
	}
	fmt.Fprintf(out, "%s | %s\nidentity: %s | version: %s | part: %s | quote: %s | applicability: %s\n", r.Input.Ref, r.Input.Where, r.Identity, r.Version, r.PartStatus, r.QuoteStatus, r.Applicability)
	if r.ChangedSince != "" {
		fmt.Fprintln(out, "changed_since:", r.ChangedSince)
	}
	if r.Error != nil {
		fmt.Fprintln(out, r.Error.Error())
	}
	if r.Document != nil {
		renderResolution(out, *r.Document)
		renderCaveats(out, *r.Document)
		fmt.Fprintf(out, "source: %s | version: %s -> %s | retrieved_at: %s | complete: %t\n", r.Document.Provenance.Source, r.Document.Start, r.Document.End, r.Document.Provenance.RetrievedAt, r.Document.Complete)
		fmt.Fprintln(out, r.Document.Provenance.URL)
	}
}
func diff(ctx context.Context, c *source.Client, r reference.Reference, o options, out, errout io.Writer) int {
	if o.from == "" || o.to == "" {
		return fail(out, errout, o.json, model.Err("USAGE", 2, "diff requires --from and --to"))
	}
	if r.Date != "" {
		return fail(out, errout, o.json, model.Err("CONFLICT", 2, "diff dates conflict with a dated reference"))
	}
	r1, e := r.Options(o.from, "")
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	r2, e := r.Options(o.to, "")
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	a, e := c.Get(ctx, r1)
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	b, e := c.Get(ctx, r2)
	if e != nil {
		return fail(out, errout, o.json, e)
	}
	if !a.Complete || !b.Complete {
		return fail(out, errout, o.json, model.Err("UNSUPPORTED_CONTENT", 3, "cannot compare incomplete source texts"))
	}
	if a.ContentKind == "table_of_contents" || b.ContentKind == "table_of_contents" {
		return fail(out, errout, o.json, model.Err("UNSUPPORTED_CONTENT", 3, "select an article before comparing text; a table of contents is not source content"))
	}
	if r.Part != "" {
		a, e = selectPart(a, r.Part)
		if e != nil {
			return fail(out, errout, o.json, e)
		}
		b, e = selectPart(b, r.Part)
		if e != nil {
			return fail(out, errout, o.json, e)
		}
	}
	changes := diffChanges(a, b)
	complete := true
	chars := 0
	for i, ch := range changes {
		n := len([]rune(ch["diff"]))
		chars += n
		if !o.full && chars > outputLimit {
			changes = changes[:i]
			complete = false
			break
		}
	}
	v := map[string]any{"schema": model.Schema, "from": a.Provenance, "from_id": a.ID, "to": b.Provenance, "to_id": b.ID, "changes": changes, "complete": complete, "applicability": "not_evaluated"}
	if !complete {
		v["next"] = "legidor diff " + fmt.Sprintf("%q", r.Raw) + " --from " + o.from + " --to " + o.to + " --full"
	}
	emit(out, o.json, v)
	if !complete {
		return 3
	}
	return 0
}
