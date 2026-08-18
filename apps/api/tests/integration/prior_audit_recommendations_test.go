package integration_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
)

func TestPriorAuditRecommendation_GoldenParity(t *testing.T) {
	for _, fixture := range []application.PriorAuditFixture{application.PriorAuditMultiHistoryFixture(), application.PriorAuditSingleHistoryFixture()} {
		evaluation, err := application.EvaluatePriorAuditRecommendations(fixture.Input)
		if err != nil {
			t.Fatalf("%s evaluation: %v", fixture.Name, err)
		}
		if len(evaluation.ComparableAuditIDs) != len(fixture.Input.Audits) {
			t.Fatalf("%s comparable audit count = %d, want %d", fixture.Name, len(evaluation.ComparableAuditIDs), len(fixture.Input.Audits))
		}
		for _, recommendation := range evaluation.Recommendations {
			if recommendation.RecommendationState != fixture.ExpectedStates[recommendation.QuestionVersionID] || recommendation.IncludedByDefault != fixture.ExpectedIncluded[recommendation.QuestionVersionID] {
				t.Fatalf("%s golden mismatch for %s: %+v", fixture.Name, recommendation.QuestionVersionID, recommendation)
			}
		}
	}
}

func TestPriorAuditRecommendation_MandatoryFloorDirectAPI(t *testing.T) {
	fixture := application.PriorAuditMultiHistoryFixture()
	evaluation, err := application.EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate fixture: %v", err)
	}
	selected := []string{fixture.Input.Questions[4].QuestionVersionID}
	if err := application.ValidateMandatoryFloor(selected, evaluation, nil); err == nil {
		t.Fatal("direct API floor accepted a selection that omitted protected questions")
	}
	deviation := []application.QuestionDeviation{{QuestionVersionID: fixture.Input.Questions[4].QuestionVersionID, Action: "DEFER", Reason: "Validated clean history and recurrence interval."}}
	selected = append([]string(nil), fixture.Input.Questions[0].QuestionVersionID, fixture.Input.Questions[1].QuestionVersionID, fixture.Input.Questions[2].QuestionVersionID, fixture.Input.Questions[3].QuestionVersionID, fixture.Input.Questions[5].QuestionVersionID, fixture.Input.Questions[6].QuestionVersionID, fixture.Input.Questions[7].QuestionVersionID)
	if err := application.ValidateMandatoryFloor(selected, evaluation, deviation); err != nil {
		t.Fatalf("direct API floor rejected explicit optional deviation: %v", err)
	}
}

func TestPriorAuditRecommendation_SnapshotFreeze(t *testing.T) {
	fixture := application.PriorAuditMultiHistoryFixture()
	evaluation, err := application.EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate fixture: %v", err)
	}
	selected := []string{
		fixture.Input.Questions[0].QuestionVersionID, fixture.Input.Questions[1].QuestionVersionID,
		fixture.Input.Questions[2].QuestionVersionID, fixture.Input.Questions[3].QuestionVersionID,
		fixture.Input.Questions[5].QuestionVersionID, fixture.Input.Questions[6].QuestionVersionID,
		fixture.Input.Questions[7].QuestionVersionID,
	}
	deviations := []application.QuestionDeviation{{QuestionVersionID: fixture.Input.Questions[4].QuestionVersionID, Action: "DEFER", Reason: "Validated clean history and recurrence interval."}}
	first, err := application.FreezeRecommendationSelection(evaluation, selected, deviations)
	if err != nil {
		t.Fatalf("freeze recommendation selection: %v", err)
	}
	second, err := application.FreezeRecommendationSelection(evaluation, selected, deviations)
	if err != nil {
		t.Fatalf("replay recommendation selection: %v", err)
	}
	if first.FreezeDigest != second.FreezeDigest || first.SelectionDigest != second.SelectionDigest {
		t.Fatalf("freeze replay drifted: first=%+v second=%+v", first, second)
	}
	if first.RecommendationSnapshot != evaluation.SnapshotDigest || first.EvaluationDigest != evaluation.SnapshotDigest {
		t.Fatal("freeze did not retain the exact recommendation snapshot digest")
	}
}

func TestPriorAuditRecommendation_ReplayImmutability(t *testing.T) {
	fixture := application.PriorAuditMultiHistoryFixture()
	before := fixture.Input
	if _, err := application.EvaluatePriorAuditRecommendations(fixture.Input); err != nil {
		t.Fatalf("evaluate fixture: %v", err)
	}
	if !reflect.DeepEqual(before, fixture.Input) {
		t.Fatal("recommendation evaluation mutated historical fixture input")
	}
}

func TestPriorAuditRecommendation_AuditeePrivacy(t *testing.T) {
	evaluation, err := application.EvaluatePriorAuditRecommendations(application.PriorAuditMultiHistoryFixture().Input)
	if err != nil {
		t.Fatalf("evaluate fixture: %v", err)
	}
	encoded, err := json.Marshal(application.ProjectAuditeeQuestionRecommendations(evaluation))
	if err != nil {
		t.Fatalf("marshal Auditee projection: %v", err)
	}
	for _, forbidden := range []string{"recommendationState", "signalCodes", "rationale", "guardrails", "historyCount", "managerReason", "internalRisk"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("Auditee projection leaked %q: %s", forbidden, encoded)
		}
	}
}
