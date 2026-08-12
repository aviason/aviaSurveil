package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/aviason/aviaSurveil/internal/identity"
	organizationstore "github.com/aviason/aviaSurveil/internal/organizations/store/postgres"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type Record struct {
	ID               string `json:"id"`
	LegalName        string `json:"legalName"`
	OrganizationType string `json:"organizationType"`
	Status           string `json:"status"`
	Revision         int64  `json:"revision"`
	OpenFindingCount int64  `json:"openFindingCount,omitempty"`
	LastAuditDate    string `json:"lastAuditDate,omitempty"`
	NextAuditDate    string `json:"nextAuditDate,omitempty"`
}

type Reader interface {
	ListRegistry(context.Context, string, int32) ([]Record, error)
	Get(context.Context, string) (Record, error)
}

type Service struct {
	reader Reader
}

func NewService(reader Reader) *Service {
	return &Service{reader: reader}
}

func NewPostgresService(pool *database.Pool) *Service {
	return NewService(postgresReader{queries: organizationstore.New(pool)})
}

func (service *Service) ListRegistry(ctx context.Context, actor identity.Principal, limit int32) ([]Record, error) {
	if !CanListRegistry(actor) {
		return nil, ErrForbidden
	}
	if service == nil || service.reader == nil {
		return nil, fmt.Errorf("organization reader is required")
	}
	organizationScope := ""
	if actor.HasRole(identity.RoleAuditee) {
		organizationScope = actor.OrganizationID
	}
	return service.reader.ListRegistry(ctx, organizationScope, boundedLimit(limit))
}

func (service *Service) Get(ctx context.Context, actor identity.Principal, organizationID string) (Record, error) {
	if !CanView(actor, organizationID) {
		// Cross-organization direct identifiers are deliberately indistinguishable
		// from records that do not exist, and the guard runs before storage.
		return Record{}, ErrNotFound
	}
	if service == nil || service.reader == nil {
		return Record{}, fmt.Errorf("organization reader is required")
	}
	record, err := service.reader.Get(ctx, organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	return record, err
}

func boundedLimit(limit int32) int32 {
	if limit <= 0 || limit > 100 {
		return 100
	}
	return limit
}

type postgresReader struct {
	queries *organizationstore.Queries
}

func (reader postgresReader) ListRegistry(
	ctx context.Context,
	organizationScope string,
	limit int32,
) ([]Record, error) {
	rows, err := reader.queries.ListOrganizationRegistry(ctx, organizationstore.ListOrganizationRegistryParams{
		OrganizationScope: organizationScope,
		ResultLimit:       limit,
	})
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(rows))
	for _, row := range rows {
		records = append(records, Record{
			ID: row.ID, LegalName: row.LegalName, OrganizationType: row.OrganizationType,
			Status: row.Status, Revision: row.Revision, OpenFindingCount: row.OpenFindingCount,
			LastAuditDate: row.LastAuditDate, NextAuditDate: row.NextAuditDate,
		})
	}
	return records, nil
}

func (reader postgresReader) Get(ctx context.Context, id string) (Record, error) {
	row, err := reader.queries.GetOrganization(ctx, id)
	if err != nil {
		return Record{}, err
	}
	return Record{
		ID: row.ID, LegalName: row.LegalName, OrganizationType: row.OrganizationType,
		Status: row.Status, Revision: row.Revision,
	}, nil
}
