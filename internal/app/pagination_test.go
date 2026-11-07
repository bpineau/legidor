package app

import "github.com/bpineau/legidor/internal/reference"

func referenceForPagination() reference.Reference {
	return reference.Reference{Kind: "article", Raw: "CIV 1", Code: "CIV", Number: "1", Date: "2025-06-01"}
}
