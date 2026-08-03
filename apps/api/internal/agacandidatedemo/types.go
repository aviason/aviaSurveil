// Package agacandidatedemo is the read-only, tagged API boundary for the
// sealed preprod AGA candidate projection. It intentionally does not import
// the loader package or any governed-domain command service.
package agacandidatedemo

import (
	"context"
	"errors"
)

var (
	ErrUnavailable = errors.New("AGA candidate demo capability unavailable")
	ErrNotFound    = errors.New("AGA candidate demo record not found")
)

type Capability struct {
	Available bool     `json:"available"`
	Labels    []string `json:"labels"`
}

type Summary struct {
	PackageDigest      string   `json:"packageDigest"`
	FormCount          int      `json:"formCount"`
	QuestionCount      int      `json:"questionCount"`
	SourceRequirements []string `json:"sourceRequirements"`
	Labels             []string `json:"labels"`
}

type Form struct {
	Code                    string `json:"code"`
	Title                   string `json:"title"`
	QuestionCount           int    `json:"questionCount"`
	QuestionExtractionState string `json:"questionExtractionState"`
}

type Question struct {
	ProposalID        string `json:"proposalId"`
	FormCode          string `json:"formCode"`
	Ordinal           int    `json:"ordinal"`
	Text              string `json:"text"`
	TextDigest        string `json:"textDigest"`
	SourceGapCategory string `json:"sourceGapCategory"`
	RiskBand          string `json:"riskBand"`
}

type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// Reader may return only a committed, reconciled sealed projection. All
// methods are read-only; no write operation is present in this interface.
type Reader interface {
	Capability(context.Context) (Capability, error)
	Summary(context.Context) (Summary, error)
	Forms(context.Context, string, int) (Page[Form], error)
	Form(context.Context, string) (Form, error)
	Questions(context.Context, string, string, string, string, int) (Page[Question], error)
}
