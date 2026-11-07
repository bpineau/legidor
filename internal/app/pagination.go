package app

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

const outputLimit = 20000

type pageCursor struct {
	Raw         string              `json:"raw"`
	Reference   reference.Reference `json:"reference"`
	Hash        string              `json:"hash"`
	Offset      int                 `json:"offset"`
	Environment string              `json:"environment"`
}

func decodeCursor(s string) (pageCursor, error) {
	var c pageCursor
	b, e := base64.RawURLEncoding.DecodeString(s)
	if e != nil || json.Unmarshal(b, &c) != nil || c.Offset < 0 || c.Hash == "" {
		return c, model.Err("USAGE", 2, "invalid content cursor")
	}
	return c, nil
}
func paginate(d model.Document, r reference.Reference, raw string, o options) (model.Document, error) {
	if d.ContentKind == "table_of_contents" {
		return paginateItems(d, r, raw, o)
	}
	rr := []rune(d.Text)
	sum := sha256.Sum256([]byte(d.Text))
	hash := hex.EncodeToString(sum[:])
	start := 0
	if o.cursor != "" {
		cur, e := decodeCursor(o.cursor)
		if e != nil {
			return d, e
		}
		if cur.Hash != hash || cur.Offset > len(rr) {
			return d, model.Err("CURSOR_CHANGED", 3, "source content changed; restart the consultation")
		}
		start = cur.Offset
	}
	end := len(rr)
	if !o.full {
		end = min(end, start+outputLimit)
	}
	d.Text = string(rr[start:end])
	if end < len(rr) {
		d.Complete = false
		d.Omitted = len(rr) - end
		pin := r
		if pin.Date == "" {
			pin.Date = d.Date
		}
		if r.Kind == "article" {
			pin.ID = d.ID
			pin.Date = d.Date
		}
		b, _ := json.Marshal(pageCursor{raw, pin, hash, end, o.env})
		d.Cursor = base64.RawURLEncoding.EncodeToString(b)
		d.Next = continuation(raw, d.Cursor, o)
	}
	return d, nil
}

func continuation(raw, cursor string, o options) string {
	next := fmt.Sprintf("legidor get %q --cursor %s --env %s", raw, cursor, o.env)
	if o.json {
		next += " --json"
	}
	if o.offline {
		next += " --offline"
	}
	if o.fresh {
		next += " --fresh"
	}
	if o.maxAge > 0 {
		next += " --max-age " + o.maxAge.String()
	}
	return next
}

func paginateItems(d model.Document, r reference.Reference, raw string, o options) (model.Document, error) {
	all, _ := json.Marshal(d.Items)
	sum := sha256.Sum256(all)
	hash := hex.EncodeToString(sum[:])
	start := 0
	if o.cursor != "" {
		cur, e := decodeCursor(o.cursor)
		if e != nil {
			return d, e
		}
		if cur.Hash != hash || cur.Offset > len(d.Items) {
			return d, model.Err("CURSOR_CHANGED", 3, "source table changed; restart consultation")
		}
		start = cur.Offset
	}
	end := len(d.Items)
	if !o.full {
		size := 0
		end = start
		for end < len(d.Items) {
			b, _ := json.Marshal(d.Items[end])
			if size+len(b) > outputLimit && end > start {
				break
			}
			size += len(b)
			end++
		}
	}
	if end < len(d.Items) {
		d.Complete = false
		d.OmittedItems = len(d.Items) - end
		pin := r
		pin.ID = d.ID
		pin.Date = d.Date
		b, _ := json.Marshal(pageCursor{raw, pin, hash, end, o.env})
		d.Cursor = base64.RawURLEncoding.EncodeToString(b)
		d.Next = continuation(raw, d.Cursor, o)
	}
	d.Items = d.Items[start:end]
	return d, nil
}
