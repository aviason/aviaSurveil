package agacandidatedemo

import (
	"context"
	"fmt"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

// Service enforces the exact CAA Admin boundary before a reader is called.
// The neutral errors are deliberately suitable for a label-free HTTP 404.
type Service struct{ reader Reader }

func NewService(reader Reader) *Service { return &Service{reader: reader} }

func (service *Service) Capability(ctx context.Context, actor identity.Principal) (Capability, error) {
	if !authorized(actor) || service.reader == nil {
		return Capability{}, ErrUnavailable
	}
	return service.reader.Capability(ctx)
}

func (service *Service) Summary(ctx context.Context, actor identity.Principal) (Summary, error) {
	if !authorized(actor) || service.reader == nil {
		return Summary{}, ErrNotFound
	}
	return service.reader.Summary(ctx)
}

func (service *Service) Forms(ctx context.Context, actor identity.Principal, cursor string, limit int) (Page[Form], error) {
	if !authorized(actor) || service.reader == nil {
		return Page[Form]{}, ErrNotFound
	}
	if limit < 1 || limit > 100 {
		return Page[Form]{}, fmt.Errorf("invalid AGA candidate demo page limit")
	}
	return service.reader.Forms(ctx, cursor, limit)
}

func (service *Service) Form(ctx context.Context, actor identity.Principal, code string) (Form, error) {
	if !authorized(actor) || service.reader == nil {
		return Form{}, ErrNotFound
	}
	return service.reader.Form(ctx, code)
}

func (service *Service) Questions(ctx context.Context, actor identity.Principal, cursor, formCode, sourceGapCategory, riskBand string, limit int) (Page[Question], error) {
	if !authorized(actor) || service.reader == nil {
		return Page[Question]{}, ErrNotFound
	}
	if limit < 1 || limit > 100 {
		return Page[Question]{}, fmt.Errorf("invalid AGA candidate demo page limit")
	}
	if sourceGapCategory != "" && sourceGapCategory != "PROPOSAL_PRESENT_REVIEW_REQUIRED" && sourceGapCategory != "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL" {
		return Page[Question]{}, fmt.Errorf("invalid AGA candidate demo source-gap filter")
	}
	if riskBand != "" && riskBand != "PROPOSED_CONTROL_ASSURANCE" && riskBand != "PROPOSED_HIGH_OPERATIONAL" && riskBand != "PROPOSED_REVIEW_REQUIRED" && riskBand != "PROPOSED_SAFETY_CRITICAL" {
		return Page[Question]{}, fmt.Errorf("invalid AGA candidate demo risk-band filter")
	}
	return service.reader.Questions(ctx, cursor, formCode, sourceGapCategory, riskBand, limit)
}

func authorized(actor identity.Principal) bool {
	return actor.OrganizationID == "CAA" && actor.HasRole(identity.RoleAdmin)
}
