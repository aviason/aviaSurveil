//go:build canonicaltest

package httpapi

import (
	"net/http"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/go-chi/chi/v5"
)

type CanonicalTestAdmin struct {
	pool      *database.Pool
	objects   objectstore.TestResetter
	buckets   []string
	generator *testprofile.Generator
	clock     func() time.Time
}

func NewCanonicalTestAdmin(
	pool *database.Pool,
	objects objectstore.TestResetter,
	buckets []string,
	generator *testprofile.Generator,
	clock func() time.Time,
) http.Handler {
	admin := &CanonicalTestAdmin{
		pool: pool, objects: objects, buckets: buckets,
		generator: generator, clock: clock,
	}
	router := chi.NewRouter()
	router.Post("/__test/reset", admin.reset)
	router.Post("/__test/governed-checklist/materialize-synthetic", admin.materializeSyntheticGovernedPackage)
	return router
}

// materializeSyntheticGovernedPackage exists only behind the canonical-test
// admin boundary. It exercises the production applicability materializer with
// the explicit internal synthetic source profile; it cannot exist in a normal
// API artifact and never accepts real OPS/AOC authority.
func (admin *CanonicalTestAdmin) materializeSyntheticGovernedPackage(writer http.ResponseWriter, request *http.Request) {
	manager, ok := testprofile.Principal("USR-MANAGER-NORA")
	if !ok {
		writeProblem(writer, http.StatusInternalServerError, "Test materialization failed", "canonical manager is unavailable", "TEST_PROFILE_INVALID")
		return
	}
	const inspectionID = "AUD-SYNTHETIC-OPS-AOC-001"
	if _, err := admin.pool.Exec(request.Context(), `
		INSERT INTO inspections (id,organization_id,assigned_inspector_subject_id,title,inspection_type,status,due_date,revision,created_at,updated_at)
		VALUES ($1,'ORG-SYNTHETIC-AOC',$2,'Synthetic governed ramp inspection','RAMP_INSPECTION','PREPARATION','2026-07-30',1,$3,$3)
		ON CONFLICT (id) DO NOTHING
	`, inspectionID, testprofile.CanonicalInspectorSubjectID, admin.clock().UTC()); err != nil {
		writeProblem(writer, http.StatusInternalServerError, "Test materialization failed", err.Error(), "TEST_MATERIALIZATION_FAILED")
		return
	}
	service := checklistgovernance.NewService(admin.pool, admin.clock)
	result, err := service.MaterializeApplicablePublishedPackage(request.Context(), manager, checklistgovernance.MaterializeApplicablePublishedPackageCommand{
		OperationID: "TASK9-MATERIALIZE-SYNTHETIC", IdempotencyKey: "TASK9-MATERIALIZE-SYNTHETIC", CorrelationID: "TASK9-MATERIALIZE-SYNTHETIC",
		InspectionID: inspectionID, PackageID: "PKG-SYNTHETIC-OPS-AOC-001", PackageVersion: 1,
		ExpiresAt: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC),
		Selection: checklistgovernance.PublishedChecklistSelectionRequest{
			OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION", TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION",
			DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC),
		},
		AssignedInspectorSubjectIDs: map[string][]string{"Q-SYNTHETIC-OPS-AOC-001": {testprofile.CanonicalInspectorSubjectID}},
	})
	if err != nil {
		apiError := "TEST_MATERIALIZATION_FAILED"
		writeProblem(writer, http.StatusUnprocessableEntity, "Test materialization failed", err.Error(), apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, result)
}

func (admin *CanonicalTestAdmin) reset(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if admin.objects != nil {
		if err := admin.objects.ResetPrivateBuckets(
			request.Context(),
			admin.buckets,
		); err != nil {
			writeProblem(
				writer,
				http.StatusInternalServerError,
				"Test reset failed",
				err.Error(),
				"TEST_RESET_FAILED",
			)
			return
		}
	}
	if err := testprofile.Reset(
		request.Context(),
		admin.pool,
		admin.clock().UTC(),
	); err != nil {
		writeProblem(
			writer,
			http.StatusInternalServerError,
			"Test reset failed",
			err.Error(),
			"TEST_RESET_FAILED",
		)
		return
	}
	admin.generator.Reset()
	writeJSON(writer, http.StatusOK, map[string]string{"status": "reset"})
}
