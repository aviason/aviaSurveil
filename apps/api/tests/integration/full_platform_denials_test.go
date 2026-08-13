//go:build canonicaltest

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/risk"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestFullPlatformRoleOrganizationAndPrivacyDenials(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_denials")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, testprofile.CanonicalScenarioTime()); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	generator := testprofile.NewGenerator()
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool),
		Risk: risk.NewService(pool, risk.Dependencies{
			Clock: testprofile.CanonicalScenarioTime, IDGenerator: generator.Next,
		}),
		Administration: administration.NewProjectionService(
			pool,
			administration.ProjectionDependencies{Clock: testprofile.CanonicalScenarioTime},
		),
		Clock: testprofile.CanonicalScenarioTime,
	})
	handler := httpapi.NewCanonicalTestBoundary("full-platform-denial-token").Protect(api.Handler())
	cases := []struct {
		name      string
		path      string
		subjectID string
	}{
		{
			name:      "Inspector cannot read the Lead Potential Finding queue",
			path:      "/v1/potential-findings?status=PENDING_LEAD_REVIEW",
			subjectID: testprofile.CanonicalInspectorSubjectID,
		},
		{
			name:      "Auditee cannot read private management risk",
			path:      "/v1/risk/overview?organizationId=ORG-FLY-NAMIBIA",
			subjectID: "USR-AUDITEE-FLY",
		},
		{
			name:      "Manager cannot read the Admin Question Bank",
			path:      "/v1/admin/questions",
			subjectID: "USR-MANAGER-NORA",
		},
		{
			name:      "Auditee cannot read an unreleased Report version",
			path:      "/v1/auditee/report-versions/RPT-CAB-2026-001-V1",
			subjectID: "USR-AUDITEE-FLY",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			request.Header.Set(httpapi.CanonicalTestTokenHeader, "full-platform-denial-token")
			request.Header.Set(httpapi.CanonicalTestSubjectHeader, tc.subjectID)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			for _, forbidden := range []string{
				"Internal CAA Note", "internalCaaNote", "enforcementDeliberation",
				"privateRisk", "ORG-SKYCARGO",
			} {
				if strings.Contains(response.Body.String(), forbidden) {
					t.Errorf("denial response leaked %q: %s", forbidden, response.Body.String())
				}
			}
		})
	}

	var forbiddenMutations int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM potential_findings) +
			(SELECT count(*) FROM risk_projection_versions) +
			(SELECT count(*) FROM question_versions WHERE created_at > $1)
	`, time.Date(2026, time.June, 15, 9, 0, 0, 0, time.UTC)).Scan(&forbiddenMutations); err != nil {
		t.Fatalf("count denied side effects: %v", err)
	}
	if forbiddenMutations != 0 {
		t.Fatalf("denied requests created %d canonical mutations", forbiddenMutations)
	}
}
