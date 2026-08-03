package agacandidatedemo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestServiceDoesNotCallReaderBeforeExactCAAAdminCheck(t *testing.T) {
	reader := &spyReader{}
	service := agacandidatedemo.NewService(reader)
	_, err := service.Summary(context.Background(), identity.Principal{OrganizationID: "OTHER", Roles: []identity.Role{identity.RoleAdmin}})
	if !errors.Is(err, agacandidatedemo.ErrNotFound) || reader.calls != 0 {
		t.Fatalf("wrong-org request must be neutral before reader lookup: %v/%d", err, reader.calls)
	}
	_, err = service.Summary(context.Background(), identity.Principal{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}})
	if !errors.Is(err, agacandidatedemo.ErrNotFound) || reader.calls != 0 {
		t.Fatalf("non-Admin request must be neutral before reader lookup: %v/%d", err, reader.calls)
	}
}

func TestServiceAllowsExactCAAAdminRead(t *testing.T) {
	reader := &spyReader{}
	summary, err := agacandidatedemo.NewService(reader).Summary(context.Background(), identity.Principal{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}})
	if err != nil || summary.QuestionCount != 1310 || reader.calls != 1 {
		t.Fatalf("admin result=%#v err=%v calls=%d", summary, err, reader.calls)
	}
}

type spyReader struct{ calls int }

func (reader *spyReader) Capability(context.Context) (agacandidatedemo.Capability, error) {
	reader.calls++
	return agacandidatedemo.Capability{Available: true}, nil
}
func (reader *spyReader) Summary(context.Context) (agacandidatedemo.Summary, error) {
	reader.calls++
	return agacandidatedemo.Summary{QuestionCount: 1310}, nil
}
func (reader *spyReader) Forms(context.Context, string, int) (agacandidatedemo.Page[agacandidatedemo.Form], error) {
	reader.calls++
	return agacandidatedemo.Page[agacandidatedemo.Form]{}, nil
}
func (reader *spyReader) Form(context.Context, string) (agacandidatedemo.Form, error) {
	reader.calls++
	return agacandidatedemo.Form{}, nil
}
func (reader *spyReader) Questions(context.Context, string, string, string, string, int) (agacandidatedemo.Page[agacandidatedemo.Question], error) {
	reader.calls++
	return agacandidatedemo.Page[agacandidatedemo.Question]{}, nil
}
