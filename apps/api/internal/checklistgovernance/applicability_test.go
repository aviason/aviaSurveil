package checklistgovernance

import (
	"errors"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/regulatory"
)

func TestResolveApplicablePublishedVersionsFiltersExactScopeTargetQualifiersAndDate(t *testing.T) {
	at := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	request := PublishedChecklistSelectionRequest{
		OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION",
		TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION",
		DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: at,
		OperationQualifiers: map[string]string{"operation": "COMMERCIAL"},
		ActivityQualifiers:  map[string]string{"ramp": "required"},
	}
	base := PublishedChecklistApplicability{
		TemplateVersionID: "CTV-GOV-APPLICABLE", TemplateID: "TPL-SYNTHETIC-AOC",
		Version: 1, CandidateContentDigest: "sha256:exact", OrganizationID: request.OrganizationID,
		ProviderScopeID: "SCOPE-SYNTHETIC-AOC", ProviderTypeID: "AIR_OPERATOR",
		InspectionType: request.InspectionType, TargetID: request.TargetID,
		TargetKind: request.TargetKind, DepartmentID: request.DepartmentID,
		EffectiveFrom: at.AddDate(0, -1, 0), OperationQualifiers: request.OperationQualifiers,
		ActivityQualifiers: request.ActivityQualifiers,
		Questions:          []regulatory.ChecklistQuestion{{QuestionID: "Q-OPS-1", Prompt: "Exact governed question."}},
	}
	wrongScope := base
	wrongScope.TemplateVersionID = "CTV-WRONG-SCOPE"
	wrongScope.ProviderScopeID = "SCOPE-EXPIRED"
	wrongScope.EffectiveTo = ptrTime(at)
	wrongInspection := base
	wrongInspection.TemplateVersionID = "CTV-WRONG-INSPECTION"
	wrongInspection.InspectionType = "BASE_INSPECTION"
	wrongTarget := base
	wrongTarget.TemplateVersionID = "CTV-WRONG-TARGET"
	wrongTarget.TargetKind = "DEVICE"
	wrongQualifier := base
	wrongQualifier.TemplateVersionID = "CTV-WRONG-QUALIFIER"
	wrongQualifier.ActivityQualifiers = map[string]string{"ramp": "optional"}
	wrongDepartment := base
	wrongDepartment.TemplateVersionID = "CTV-WRONG-DEPARTMENT"
	wrongDepartment.DepartmentID = "AIRWORTHINESS_INSPECTORATE"

	actual, err := ResolveApplicablePublishedVersions(request, []PublishedChecklistApplicability{
		wrongScope, wrongInspection, wrongTarget, wrongQualifier, wrongDepartment, base,
	})
	if err != nil {
		t.Fatalf("resolve applicable published versions: %v", err)
	}
	if len(actual) != 1 || actual[0].TemplateVersionID != base.TemplateVersionID ||
		actual[0].CandidateContentDigest != base.CandidateContentDigest {
		t.Fatalf("applicable versions=%+v, want exact applicable governed version", actual)
	}
}

func TestResolveApplicablePublishedVersionsRejectsMissingScopeQualifier(t *testing.T) {
	at := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	request := PublishedChecklistSelectionRequest{OrganizationID: "ORG", InspectionType: "RAMP", TargetID: "TARGET", TargetKind: "ORGANIZATION", DepartmentID: "FOI", At: at}
	version := PublishedChecklistApplicability{
		TemplateVersionID: "CTV-QUALIFIED", TemplateID: "TPL-QUALIFIED", Version: 1, CandidateContentDigest: "sha256:qualified",
		OrganizationID: "ORG", ProviderScopeID: "SCOPE", ProviderTypeID: "AIR_OPERATOR", InspectionType: "RAMP", TargetID: "TARGET", TargetKind: "ORGANIZATION", DepartmentID: "FOI", EffectiveFrom: at.AddDate(0, -1, 0),
		OperationQualifiers: map[string]string{"operation": "COMMERCIAL"}, Questions: []regulatory.ChecklistQuestion{{QuestionID: "Q-QUALIFIED", Prompt: "Qualified."}},
	}
	selected, err := ResolveApplicablePublishedVersions(request, []PublishedChecklistApplicability{version})
	if err != nil {
		t.Fatalf("resolve missing qualifier: %v", err)
	}
	if len(selected) != 0 {
		t.Fatalf("missing qualifier selected version=%+v", selected)
	}
}

func TestComposeApplicablePublishedVersionsPinsOrderedVersionsAndDeduplicatesQuestionIdentity(t *testing.T) {
	at := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	request := PublishedChecklistSelectionRequest{
		OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION",
		TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION",
		DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: at,
	}
	versions := []PublishedChecklistApplicability{
		{TemplateVersionID: "CTV-GOV-Z", TemplateID: "TPL-Z", Version: 2, CandidateContentDigest: "sha256:z", OrganizationID: request.OrganizationID, ProviderScopeID: "SCOPE-SYNTHETIC-AOC", ProviderTypeID: "AIR_OPERATOR", InspectionType: request.InspectionType, TargetID: request.TargetID, TargetKind: request.TargetKind, DepartmentID: request.DepartmentID, EffectiveFrom: at.AddDate(0, -1, 0), Questions: []regulatory.ChecklistQuestion{{QuestionID: "Q-Z"}, {QuestionID: "Q-SHARED"}}},
		{TemplateVersionID: "CTV-GOV-A", TemplateID: "TPL-A", Version: 1, CandidateContentDigest: "sha256:a", OrganizationID: request.OrganizationID, ProviderScopeID: "SCOPE-SYNTHETIC-AOC", ProviderTypeID: "AIR_OPERATOR", InspectionType: request.InspectionType, TargetID: request.TargetID, TargetKind: request.TargetKind, DepartmentID: request.DepartmentID, EffectiveFrom: at.AddDate(0, -1, 0), Questions: []regulatory.ChecklistQuestion{{QuestionID: "Q-A"}, {QuestionID: "Q-SHARED"}}},
	}

	packageSnapshot, err := ComposeApplicablePublishedPackage(request, versions)
	if err != nil {
		t.Fatalf("compose applicable package: %v", err)
	}
	if len(packageSnapshot.PublishedVersions) != 2 ||
		packageSnapshot.PublishedVersions[0].TemplateVersionID != "CTV-GOV-A" ||
		packageSnapshot.PublishedVersions[1].TemplateVersionID != "CTV-GOV-Z" {
		t.Fatalf("published version order=%+v", packageSnapshot.PublishedVersions)
	}
	if len(packageSnapshot.Questions) != 3 ||
		packageSnapshot.Questions[0].QuestionID != "Q-A" ||
		packageSnapshot.Questions[1].QuestionID != "Q-SHARED" ||
		packageSnapshot.Questions[2].QuestionID != "Q-Z" {
		t.Fatalf("composed ordered questions=%+v", packageSnapshot.Questions)
	}
	if packageSnapshot.Applicability.OrganizationID != request.OrganizationID ||
		packageSnapshot.Applicability.TargetID != request.TargetID ||
		packageSnapshot.Applicability.InspectionType != request.InspectionType {
		t.Fatalf("pinned applicability=%+v", packageSnapshot.Applicability)
	}
}

func TestComposeApplicablePublishedPackageRejectsConflictingDuplicateQuestionIdentity(t *testing.T) {
	at := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	request := PublishedChecklistSelectionRequest{OrganizationID: "ORG", InspectionType: "RAMP", TargetID: "TARGET", TargetKind: "ORGANIZATION", DepartmentID: "FOI", At: at}
	base := PublishedChecklistApplicability{OrganizationID: "ORG", ProviderScopeID: "SCOPE", ProviderTypeID: "AIR_OPERATOR", InspectionType: "RAMP", TargetID: "TARGET", TargetKind: "ORGANIZATION", DepartmentID: "FOI", EffectiveFrom: at.AddDate(0, -1, 0)}
	first := base
	first.TemplateVersionID, first.TemplateID, first.Version, first.CandidateContentDigest = "CTV-A", "TPL-A", 1, "sha256:a"
	first.Questions = []regulatory.ChecklistQuestion{{QuestionID: "Q-SHARED", Prompt: "First exact question."}}
	second := base
	second.TemplateVersionID, second.TemplateID, second.Version, second.CandidateContentDigest = "CTV-B", "TPL-B", 1, "sha256:b"
	second.Questions = []regulatory.ChecklistQuestion{{QuestionID: "Q-SHARED", Prompt: "Mutated question."}}
	if _, err := ComposeApplicablePublishedPackage(request, []PublishedChecklistApplicability{first, second}); !errors.Is(err, ErrPublishedChecklistQuestionConflict) {
		t.Fatalf("conflicting duplicate identity error=%v, want conflict", err)
	}
}

func TestRunnerPackageSnapshotRejectsMissingExtraAndDuplicateAssignments(t *testing.T) {
	composed := ComposedPublishedChecklistPackage{Questions: []regulatory.ChecklistQuestion{{QuestionID: "Q-1", Prompt: "One."}}}
	for _, assigned := range []map[string][]string{
		{},
		{"Q-1": {"USR-1"}, "Q-EXTRA": {"USR-1"}},
		{"Q-1": {"USR-1", "USR-1"}},
	} {
		if _, err := runnerPackageSnapshot(composed, assigned); !errors.Is(err, application.ErrInvalid) {
			t.Fatalf("assignment validation error=%v, want invalid for %+v", err, assigned)
		}
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
