package agademoworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

type lifecycleProjectionQueryStore struct {
	*serviceTestStore
	events []preprod.LifecycleEvent
}

func (store *lifecycleProjectionQueryStore) AppendLifecycleEvent(_ context.Context, event preprod.LifecycleEvent) (preprod.LifecycleEvent, error) {
	store.events = append(store.events, event)
	return event, nil
}

func (store *lifecycleProjectionQueryStore) GetLifecycleEvents(_ context.Context, generationID, inspectionID string) ([]preprod.LifecycleEvent, error) {
	if generationID != store.workspace.Generation.GenerationID || inspectionID == "" {
		return nil, nil
	}
	return append([]preprod.LifecycleEvent(nil), store.events...), nil
}

func TestLifecycleProjectionPreservesEmptyCollections(t *testing.T) {
	aggregate := LifecycleAggregate{}
	projection := ProjectLifecycle(aggregate, identity.Principal{})
	caa := ProjectCAALifecycle(aggregate, identity.Principal{})

	collections := []struct {
		name  string
		value any
	}{
		{"questions", projection.Questions},
		{"responses", projection.Responses},
		{"potentialFindings", projection.PotentialFindings},
		{"findings", projection.Findings},
		{"capRevisions", projection.CAPRevisions},
		{"evidenceVersions", projection.EvidenceVersions},
		{"verificationDecisions", projection.VerificationDecisions},
		{"caa.reasonHistory", caa.ReasonHistory},
		{"caa.roleHistory", caa.RoleHistory},
	}
	for _, collection := range collections {
		if reflect.ValueOf(collection.value).IsNil() {
			t.Fatalf("%s must be an empty, non-nil collection", collection.name)
		}
	}

	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"questions":[]`, `"responses":[]`, `"potentialFindings":[]`, `"findings":[]`, `"capRevisions":[]`, `"evidenceVersions":[]`, `"verificationDecisions":[]`} {
		if !strings.Contains(string(encoded), expected) {
			t.Fatalf("projection JSON omitted empty array %s: %s", expected, encoded)
		}
	}
}

func TestLifecycleProjectionKeepsCAAHistoryOutOfThePublicShape(t *testing.T) {
	aggregate, _, lead, auditee, _, finding := findingFixture(t, true, false, false)
	submitCAP(t, &aggregate, auditee, finding, "projection root cause")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	public := ProjectLifecycle(aggregate, auditee)
	caa := ProjectCAALifecycle(aggregate, lead)
	publicBytes, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	caaBytes, err := json.Marshal(caa)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(publicBytes), "internalCaaNote") || strings.Contains(string(publicBytes), "subjectId") {
		t.Fatalf("public projection contains CAA-only fields: %s", publicBytes)
	}
	if !strings.Contains(string(caaBytes), "internalCaaNote") || !strings.Contains(string(caaBytes), "recommendationId") || !strings.Contains(string(caaBytes), "roleHistory") {
		t.Fatalf("CAA projection omitted audit fields: %s", caaBytes)
	}
}

func TestAuditeeProjectionUsesPositiveAllowlist(t *testing.T) {
	aggregate, _, lead, auditee, _, finding := findingFixture(t, true, true, false)
	submitCAP(t, &aggregate, auditee, finding, "auditee-visible root cause")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	submitEvidenceForTest(t, &aggregate, auditee, finding, "auditee-visible.pdf")
	verifyEvidenceForTest(t, &aggregate, lead, finding, EvidenceRequestMoreInformation)

	projection, err := ProjectAuditeeLifecycle(aggregate, auditee)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Findings) != 1 || len(projection.CAPRevisions) == 0 || len(projection.EvidenceVersions) == 0 || len(projection.VerificationDecisions) != 1 {
		t.Fatalf("auditee allowlist omitted released CAP/Evidence context: %+v", projection)
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"questions", "responses", "potentialFindings", "providerScopeId", "currentOwnerRole", "internalCaaNote", "actorSubjectId", "subjectId", "questionKey", "roleHistory", "recommendationId"} {
		if strings.Contains(string(encoded), `"`+forbidden+`"`) {
			t.Fatalf("auditee positive allowlist leaked %s: %s", forbidden, encoded)
		}
	}
}

func TestLifecycleQueryValidatesSelectorsBeforeAuditeeProjection(t *testing.T) {
	aggregate, _, lead, auditee, _, finding := findingFixture(t, true, true, false)
	submitCAP(t, &aggregate, auditee, finding, "selector validation root cause")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	submitEvidenceForTest(t, &aggregate, auditee, finding, "selector-validation.pdf")
	cap := aggregate.CAPRevisions[len(aggregate.CAPRevisions)-1]
	evidence := aggregate.EvidenceVersions[len(aggregate.EvidenceVersions)-1]
	event, err := lifecycleEventFor(aggregate, OperationSubmitEvidence, "selector-validation", auditee.SubjectID, "", aggregate.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	workspace := testWorkspace()
	workspace.Generation.GenerationID = aggregate.GenerationID
	store := &lifecycleProjectionQueryStore{serviceTestStore: &serviceTestStore{workspace: workspace}, events: []preprod.LifecycleEvent{event}}
	binding := bindingFor(auditee.SubjectID, aggregate.OrganizationID, "auditee")
	binding.SubjectID = auditee.SubjectID
	binding.DepartmentID = aggregate.Inspector.DepartmentID
	binding.OrganizationalUnitID = aggregate.Inspector.OrganizationalUnitID
	binding.ProviderScopeID = aggregate.ProviderScopeID
	service := NewService(ServiceConfig{Store: store, Resolver: operationBindingResolverForTest{binding: binding}})

	response, err := service.Query(context.Background(), auditee, QueryRequest{OperationID: OperationGetCAPEvidence, InspectionID: aggregate.InspectionID, FindingID: finding.FindingID, CapID: cap.CAPID, EvidenceID: evidence.EvidenceID})
	if err != nil || response.LifecycleAuditee == nil {
		t.Fatalf("valid exact selectors = response=%+v err=%v", response, err)
	}

	for _, request := range []QueryRequest{
		{OperationID: OperationGetFinding, InspectionID: aggregate.InspectionID, FindingID: "missing-finding"},
		{OperationID: OperationGetCAPEvidence, InspectionID: aggregate.InspectionID, CapID: "missing-cap"},
		{OperationID: OperationGetCAPEvidence, InspectionID: aggregate.InspectionID, EvidenceID: "missing-evidence"},
		{OperationID: OperationGetCAPEvidence, InspectionID: aggregate.InspectionID, FindingID: "missing-finding", CapID: cap.CAPID},
		{OperationID: OperationGetInspection, InspectionID: aggregate.InspectionID, FindingID: finding.FindingID},
	} {
		if _, err := service.Query(context.Background(), auditee, request); !errors.Is(err, ErrNeutralDenied) {
			t.Fatalf("selector request %+v error = %v, want neutral denial", request, err)
		}
	}
}
