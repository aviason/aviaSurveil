package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
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

func TestScopeRecommendation_NoHistoryScopeFilterGolden(t *testing.T) {
	fixture := application.PriorAuditNoHistoryScopeFilterFixture()
	evaluation, err := application.EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("no-history scope-filter evaluation: %v", err)
	}
	allIDs := make([]string, 0, len(evaluation.Recommendations))
	suggestedIDs := make([]string, 0, len(evaluation.Recommendations))
	for _, recommendation := range evaluation.Recommendations {
		allIDs = append(allIDs, recommendation.QuestionVersionID)
		if recommendation.IncludedByDefault {
			suggestedIDs = append(suggestedIDs, recommendation.QuestionVersionID)
		}
	}
	sort.Strings(allIDs)
	sort.Strings(suggestedIDs)
	if want := []string{application.NoHistoryInFocusOptionalQuestionID, application.NoHistoryOutsideFocusMandatoryQuestionID, application.NoHistoryOutsideFocusOptionalQuestionID}; !reflect.DeepEqual(allIDs, func() []string { sort.Strings(want); return want }()) {
		t.Fatalf("full applicable IDs = %v, want exact three-ID oracle", allIDs)
	}
	if want := []string{application.NoHistoryInFocusOptionalQuestionID, application.NoHistoryOutsideFocusMandatoryQuestionID}; !reflect.DeepEqual(suggestedIDs, func() []string { sort.Strings(want); return want }()) {
		t.Fatalf("suggested IDs = %v, want exact two-ID oracle", suggestedIDs)
	}
	if len(suggestedIDs) >= len(allIDs) {
		t.Fatalf("suggested count %d must be less than full applicable count %d", len(suggestedIDs), len(allIDs))
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve no-history fixture path")
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "..", "..", "tests", "fixtures", "prior-audit-recommendations", "prior-audit-no-history-scope-filter.json"))
	if err != nil {
		t.Fatalf("read no-history fixture oracle: %v", err)
	}
	var fixtureOracle struct {
		QuestionVersionIDs               []string `json:"questionVersionIds"`
		SuggestedQuestionVersionIDs      []string `json:"suggestedQuestionVersionIds"`
		FullApplicableQuestionVersionIDs []string `json:"fullApplicableQuestionVersionIds"`
		ExcludedQuestionVersionIDs       []string `json:"excludedQuestionVersionIds"`
		SuggestedCount                   int      `json:"suggestedCount"`
		FullApplicableCount              int      `json:"fullApplicableCount"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixtureOracle); err != nil {
		t.Fatalf("decode no-history fixture oracle: %v", err)
	}
	sort.Strings(fixtureOracle.QuestionVersionIDs)
	sort.Strings(fixtureOracle.SuggestedQuestionVersionIDs)
	sort.Strings(fixtureOracle.FullApplicableQuestionVersionIDs)
	sort.Strings(fixtureOracle.ExcludedQuestionVersionIDs)
	if !reflect.DeepEqual(fixtureOracle.QuestionVersionIDs, append(append([]string{}, allIDs...), fixtureOracle.ExcludedQuestionVersionIDs...)) || !reflect.DeepEqual(fixtureOracle.SuggestedQuestionVersionIDs, suggestedIDs) || !reflect.DeepEqual(fixtureOracle.FullApplicableQuestionVersionIDs, allIDs) || fixtureOracle.SuggestedCount != len(suggestedIDs) || fixtureOracle.FullApplicableCount != len(allIDs) {
		t.Fatalf("JSON no-history fixture drifted: %+v; policy all=%v suggested=%v", fixtureOracle, allIDs, suggestedIDs)
	}
	for _, excluded := range []string{application.NoHistoryWrongProviderQuestionID, application.NoHistoryWrongTargetQuestionID, application.NoHistoryWrongGeneralTypeQuestionID} {
		if containsIntegration(allIDs, excluded) || containsIntegration(suggestedIDs, excluded) {
			t.Fatalf("scope-inapplicable question leaked into integration oracle: %s", excluded)
		}
	}
}

func containsIntegration(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
