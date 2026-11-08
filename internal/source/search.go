package source

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/bpineau/legidor/internal/model"
	"github.com/bpineau/legidor/internal/reference"
)

type searchCursor struct {
	Key  string `json:"key"`
	Page int    `json:"page"`
}

func (c *Client) Search(ctx context.Context, query, corpus, date, cursor string, limit int) (model.Search, error) {
	if limit < 1 || limit > 50 {
		return model.Search{}, model.Err("USAGE", 2, "limit must be 1..50")
	}
	if date == "" && (corpus == "loda" || strings.HasPrefix(corpus, "code:")) {
		date = c.today()
	}
	keyBytes := sha256.Sum256([]byte(query + "\x00" + corpus + "\x00" + date + "\x00" + c.Config.Environment + "\x00" + strconv.Itoa(limit)))
	key := hex.EncodeToString(keyBytes[:])
	page := 0
	if cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(cursor)
		var cur searchCursor
		if e != nil || json.Unmarshal(b, &cur) != nil || cur.Key != key || cur.Page < 0 {
			return model.Search{}, model.Err("CONFLICT", 2, "search cursor does not match query, date, corpus or environment")
		}
		page = cur.Page
	}
	var s model.Search
	var e error
	switch corpus {
	case "bofip":
		if date != "" {
			return s, model.Err("NO_VERSION_AT_DATE", 3, "BOFiP search only covers current doctrine")
		}
		s, e = c.bofipSearch(ctx, query, limit, page*limit)
	case "judilibre":
		if date != "" {
			return s, model.Err("USAGE", 2, "--date is not a decision-date filter")
		}
		s, e = c.judiSearch(ctx, query, limit, page, "")
	default:
		fond := strings.ToUpper(corpus)
		filters := []object{}
		article := false
		if strings.HasPrefix(corpus, "code:") {
			code := reference.CodeKey(strings.TrimPrefix(corpus, "code:"))
			if code == "" {
				return s, model.Err("USAGE", 2, "unknown code alias")
			}
			fond = "CODE_DATE"
			filters = append(filters, object{"facette": "NOM_CODE", "valeurs": []string{reference.CodeName(code)}})
			article = true
		}
		if fond == "LODA" {
			fond = "LODA_DATE"
		}
		if fond == "CODE_DATE" || fond == "LODA_DATE" {
			if date == "" {
				date = c.today()
			}
			filters = append(filters, dateFilter(date))
		} else if date != "" {
			return s, model.Err("USAGE", 2, "--date applies only to consolidated codes and LODA")
		}
		switch fond {
		case "CODE_DATE", "LODA_DATE", "JORF", "CETAT", "JURI", "CONSTIT", "KALI":
		default:
			return s, model.Err("USAGE", 2, "unknown corpus")
		}
		s, e = c.lfSearch(ctx, fond, []object{field("ALL", query, "TOUS_LES_MOTS_DANS_UN_CHAMP")}, filters, page+1, limit, article)
	}
	if e != nil {
		return s, e
	}
	s.Corpus = corpus
	if (page+1)*limit < s.Total {
		b, _ := json.Marshal(searchCursor{key, page + 1})
		s.Cursor = base64.RawURLEncoding.EncodeToString(b)
	}
	return s, nil
}
