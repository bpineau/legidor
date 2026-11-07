package app

import (
	"github.com/bpineau/legidor/internal/model"
	legaltext "github.com/bpineau/legidor/internal/text"
)

func documentForDiff(s string) model.Document {
	return model.Document{Text: s, Parts: legaltext.Parts(s)}
}
