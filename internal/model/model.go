// Package model defines the versioned, source-independent CLI contract.
package model

import "fmt"

const Schema = "legidor/1"

type Error struct {
	ExactArticleAbsent bool     `json:"-"`
	Code               string   `json:"code"`
	Message            string   `json:"message"`
	Next               string   `json:"next,omitempty"`
	Candidates         []string `json:"candidates,omitempty"`
	Commands           []string `json:"commands,omitempty"`
	Hints              []string `json:"hints,omitempty"`
	Exit               int      `json:"-"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func Err(code string, exit int, format string, args ...any) *Error {
	return &Error{Code: code, Exit: exit, Message: fmt.Sprintf(format, args...)}
}

type Provenance struct {
	MaxAgeSeconds         int64  `json:"max_age_seconds"`
	ResolutionDate        string `json:"resolution_date,omitempty"`
	ResolutionRetrievedAt string `json:"resolution_retrieved_at,omitempty"`
	ResolutionCached      bool   `json:"resolution_cached,omitempty"`
	Source                string `json:"source"`
	URL                   string `json:"url"`
	RetrievedAt           string `json:"retrieved_at"`
	UpdatedAt             string `json:"source_updated_at,omitempty"`
	Environment           string `json:"environment"`
	Cached                bool   `json:"cached"`
	Offline               bool   `json:"offline"`
}
type Link struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	Type  string `json:"type,omitempty"`
	Date  string `json:"date,omitempty"`
}
type Part struct {
	Context    *string `json:"-"`
	Label      string  `json:"label"`
	Text       string  `json:"text,omitempty"`
	Preview    string  `json:"preview,omitempty"`
	Selectable *bool   `json:"selectable,omitempty"`
	Start      int     `json:"-"`
	End        int     `json:"-"`
	Level      int     `json:"-"`
	Parent     int     `json:"-"`
}
type Version struct {
	OmittedModifications int         `json:"omitted_modifications,omitempty"`
	ID                   string      `json:"id"`
	Start                string      `json:"start"`
	End                  string      `json:"end"`
	State                string      `json:"state,omitempty"`
	Nota                 string      `json:"nota,omitempty"`
	Modifications        []Link      `json:"modifications,omitempty"`
	Provenance           *Provenance `json:"provenance,omitempty"`
}
type Document struct {
	OutlineNavigationOnly bool       `json:"outline_navigation_only,omitempty"`
	OmittedPartPreviews   int        `json:"omitted_part_previews,omitempty"`
	Schema                string     `json:"schema"`
	ID                    string     `json:"id"`
	Title                 string     `json:"title"`
	Code                  string     `json:"code,omitempty"`
	Number                string     `json:"number,omitempty"`
	Parent                string     `json:"parent,omitempty"`
	Nature                string     `json:"nature"`
	ContentKind           string     `json:"content_kind,omitempty"`
	Start                 string     `json:"start,omitempty"`
	End                   string     `json:"end,omitempty"`
	Date                  string     `json:"date,omitempty"`
	Publication           string     `json:"publication_date,omitempty"`
	State                 string     `json:"state,omitempty"`
	Text                  string     `json:"text"`
	HTML                  string     `json:"-"`
	Nota                  string     `json:"nota,omitempty"`
	Modifications         []Link     `json:"modifications,omitempty"`
	Versions              []Version  `json:"versions,omitempty"`
	NextVersions          []Version  `json:"next_versions,omitempty"`
	OmittedModifications  int        `json:"omitted_modifications,omitempty"`
	Parts                 []Part     `json:"parts,omitempty"`
	Items                 []Link     `json:"items,omitempty"`
	Provenance            Provenance `json:"provenance"`
	Complete              bool       `json:"complete"`
	SourceComplete        *bool      `json:"source_complete,omitempty"`
	Unsupported           []string   `json:"unsupported_content,omitempty"`
	Omitted               int        `json:"omitted_characters,omitempty"`
	OmittedItems          int        `json:"omitted_items,omitempty"`
	Cursor                string     `json:"cursor,omitempty"`
	Next                  string     `json:"next,omitempty"`
	Warnings              []string   `json:"warnings,omitempty"`
}
type Hit struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Number  string `json:"number,omitempty"`
	Parent  string `json:"parent,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	URL     string `json:"url,omitempty"`
	Start   string `json:"start,omitempty"`
	End     string `json:"end,omitempty"`
}
type Search struct {
	Corpus         string     `json:"corpus"`
	Hints          []string   `json:"hints,omitempty"`
	ReturnedGroups int        `json:"returned_groups,omitempty"`
	Schema         string     `json:"schema"`
	Hits           []Hit      `json:"hits"`
	Total          int        `json:"total"`
	TotalUnit      string     `json:"total_unit,omitempty"`
	Cursor         string     `json:"cursor,omitempty"`
	Sort           string     `json:"sort"`
	Stability      string     `json:"stability"`
	Provenance     Provenance `json:"provenance"`
	Complete       bool       `json:"complete"`
}
type Citation struct {
	Ref    string `json:"ref"`
	Date   string `json:"date,omitempty"`
	Part   string `json:"part,omitempty"`
	Quote  string `json:"quote,omitempty"`
	Where  string `json:"where,omitempty"`
	Change string `json:"change,omitempty"`
}
type Check struct {
	Schema        string    `json:"schema"`
	Input         Citation  `json:"input"`
	Identity      string    `json:"identity"`
	Version       string    `json:"version"`
	PartStatus    string    `json:"part"`
	QuoteStatus   string    `json:"quote"`
	ChangedSince  string    `json:"changed_since"`
	Applicability string    `json:"applicability"`
	Document      *Document `json:"document,omitempty"`
	Error         *Error    `json:"error,omitempty"`
}
