package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/session"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestBootstrapAdministratorCannotCreateApplicationAuthorityByLogin(t *testing.T) {
	pool := createTestDatabase(t, "first_caa_administrator")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	manager, err := session.NewManager(
		pool,
		[]byte("0123456789abcdef0123456789abcdef"),
		session.ManagerDependencies{},
	)
	if err != nil {
		t.Fatalf("new session manager: %v", err)
	}
	_, err = manager.Create(context.Background(), session.CreateInput{
		SubjectID:      "first-admin",
		Issuer:         "https://identity.example/realms/avia",
		DisplayName:    "First Administrator",
		Email:          "first.admin@example.test",
		OrganizationID: "CAA",
		Roles:          []identity.Role{identity.RoleAdmin},
	})
	if !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("bootstrap administrator session error = %v", err)
	}
	var organizationType string
	if err := pool.QueryRow(
		context.Background(),
		"SELECT organization_type FROM organizations WHERE id = 'CAA'",
	).Scan(&organizationType); err != nil {
		t.Fatalf("read authority organization: %v", err)
	}
	if organizationType != "AUTHORITY" {
		t.Fatalf("organization type = %q", organizationType)
	}
	var identityCount, sessionCount int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM identity_references WHERE subject_id = 'first-admin'",
	).Scan(&identityCount); err != nil {
		t.Fatalf("count bootstrap identity references: %v", err)
	}
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM session_references WHERE subject_id = 'first-admin'",
	).Scan(&sessionCount); err != nil {
		t.Fatalf("count bootstrap sessions: %v", err)
	}
	if identityCount != 0 || sessionCount != 0 {
		t.Fatalf(
			"bootstrap login wrote identity/session rows = %d/%d",
			identityCount,
			sessionCount,
		)
	}
}

func TestAuthorizedCleanStateCreationIsTransactionalAndIdempotent(t *testing.T) {
	pool := createTestDatabase(t, "authorized_clean_state_creation")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (
			subject_id, issuer, display_name, email
		) VALUES
			('admin', 'https://identity.example/realms/avia', 'Administrator',
				'admin@example.test'),
			('manager', 'https://identity.example/realms/avia', 'Manager',
				'manager@example.test')
	`); err != nil {
		t.Fatalf("insert authenticated identities: %v", err)
	}
	service := testService(pool)
	admin := principal("admin", "CAA", "session-admin", identity.RoleAdmin)
	manager := principal("manager", "CAA", "session-manager", identity.RoleDepartmentManager)

	created, err := service.CreateAdminOrganization(
		context.Background(),
		admin,
		application.CreateAdminOrganizationCommand{
			OperationID: "create-operator", IdempotencyKey: "create-operator",
			OrganizationID: "ORG-OPERATOR", LegalName: "Operator", OrganizationType: "OPERATOR",
		},
	)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	replayed, err := service.CreateAdminOrganization(
		context.Background(),
		admin,
		application.CreateAdminOrganizationCommand{
			OperationID: "create-operator", IdempotencyKey: "create-operator",
			OrganizationID: "ORG-OPERATOR", LegalName: "Operator", OrganizationType: "OPERATOR",
		},
	)
	if err != nil {
		t.Fatalf("replay organization: %v", err)
	}
	if created != replayed {
		t.Fatalf("idempotent replay changed response: %#v != %#v", created, replayed)
	}

	_, err = service.CreatePlanningIntakeDraft(
		context.Background(),
		manager,
		application.CreatePlanningIntakeDraftCommand{
			OperationID: "create-draft", IdempotencyKey: "create-draft",
			DraftID: "DRAFT-001", OrganizationID: "ORG-OPERATOR",
			Values: map[string]any{
				"organizationId": "ORG-OPERATOR", "organizationName": "Operator",
				"inspectionCategory": "Routine / Announced", "noticePolicy": "ADVANCE",
			},
		},
	)
	if !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("legacy planning draft creation error = %v, want canonical scope rejection", err)
	}
	var draftCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM planning_intake_drafts WHERE id = 'DRAFT-001'`).Scan(&draftCount); err != nil {
		t.Fatalf("read rejected draft: %v", err)
	}
	if draftCount != 0 {
		t.Fatalf("legacy planning draft left %d rows", draftCount)
	}
}
