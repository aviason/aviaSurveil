package application

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func recommendationByID(evaluation RecommendationEvaluation) map[string]QuestionRecommendation {
	items := make(map[string]QuestionRecommendation, len(evaluation.Recommendations))
	for _, recommendation := range evaluation.Recommendations {
		items[recommendation.QuestionVersionID] = recommendation
	}
	return items
}

func TestScopeRecommendation_MultiHistoryGolden(t *testing.T) {
	fixture := PriorAuditMultiHistoryFixture()
	evaluation, err := EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate multi-history fixture: %v", err)
	}
	if len(evaluation.ComparableAuditIDs) != 3 {
		t.Fatalf("comparable audits = %v, want exactly three", evaluation.ComparableAuditIDs)
	}
	byID := recommendationByID(evaluation)
	for questionID, expected := range fixture.ExpectedStates {
		got := byID[questionID]
		if got.RecommendationState != expected || got.Classification != fixture.ExpectedClassifications[questionID] || got.IncludedByDefault != fixture.ExpectedIncluded[questionID] || got.HistoryCount != fixture.ExpectedHistoryCounts[questionID] {
			t.Fatalf("%s recommendation = %+v; want state=%s classification=%s included=%t history=%d", questionID, got, expected, fixture.ExpectedClassifications[questionID], fixture.ExpectedIncluded[questionID], fixture.ExpectedHistoryCounts[questionID])
		}
	}
	clean := byID[priorAuditQuestionID(5)]
	if clean.CanDefer != true || !contains(clean.SignalCodes, "DEFER_ELIGIBLE") || !contains(clean.SignalCodes, "RECENTLY_VERIFIED") {
		t.Fatalf("recent clean optional recommendation lost deferral guardrails: %+v", clean)
	}
	if len(FilterQuestionRecommendations(evaluation, false)) != 7 || len(FilterQuestionRecommendations(evaluation, true)) != 8 {
		t.Fatalf("suggested/full catalog counts = %d/%d, want 7/8", len(FilterQuestionRecommendations(evaluation, false)), len(FilterQuestionRecommendations(evaluation, true)))
	}
}

func TestScopeRecommendation_SingleHistoryGolden(t *testing.T) {
	fixture := PriorAuditSingleHistoryFixture()
	evaluation, err := EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate single-history fixture: %v", err)
	}
	if len(evaluation.ComparableAuditIDs) != 1 {
		t.Fatalf("comparable audits = %v, want exactly one", evaluation.ComparableAuditIDs)
	}
	byID := recommendationByID(evaluation)
	for questionID, expected := range fixture.ExpectedStates {
		got := byID[questionID]
		if got.RecommendationState != expected || got.Classification != fixture.ExpectedClassifications[questionID] || got.IncludedByDefault != fixture.ExpectedIncluded[questionID] || got.HistoryCount != fixture.ExpectedHistoryCounts[questionID] {
			t.Fatalf("%s recommendation = %+v; want state=%s classification=%s included=%t history=%d", questionID, got, expected, fixture.ExpectedClassifications[questionID], fixture.ExpectedIncluded[questionID], fixture.ExpectedHistoryCounts[questionID])
		}
	}
	clean := byID[priorAuditQuestionID(15)]
	if clean.RecommendationState != RecommendationUncertainSignal || clean.IncludedByDefault != true || clean.CanDefer != false || contains(clean.SignalCodes, "DEFER_ELIGIBLE") {
		t.Fatalf("single clean optional was treated as deferrable: %+v", clean)
	}
}

func TestScopeRecommendation_ComparableAuditKeyIsolation(t *testing.T) {
	fixture := PriorAuditMultiHistoryFixture()
	foreign := fixture.Input.Audits[0]
	foreign.AuditID = "AUD-FOREIGN-SCOPE"
	foreign.ComparableKey.Location = "Walvis Bay Airport"
	foreign.ScopeStatus = "RELEASED"
	foreign.ReportKind = "FINAL"
	foreign.ReportStatus = "LOCKED"
	draft := fixture.Input.Audits[0]
	draft.AuditID = "AUD-DRAFT"
	draft.ScopeStatus = "DRAFT"
	duplicate := fixture.Input.Audits[0]
	duplicate.AuditID = fixture.Input.Audits[0].AuditID
	fixture.Input.Audits = append(fixture.Input.Audits, foreign, draft, duplicate)
	evaluation, err := EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate isolated history: %v", err)
	}
	if len(evaluation.ComparableAuditIDs) != 3 {
		t.Fatalf("foreign/draft/duplicate history widened comparable set: %v", evaluation.ComparableAuditIDs)
	}
}

func TestScopeRecommendation_ValidatedCleanTruthTable(t *testing.T) {
	question := priorAuditQuestion(99, false, false, "sha256:truth-table")
	base := priorAuditObservation(question.QuestionVersionID, "COMPLIANT", question.SourceDigest, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	cases := []struct {
		name   string
		mutate func(*PriorAuditQuestionObservation)
		clean  bool
	}{
		{name: "validated compliant", clean: true},
		{name: "generic not applicable", mutate: func(value *PriorAuditQuestionObservation) { value.Result = "NOT_APPLICABLE" }, clean: false},
		{name: "missing answer", mutate: func(value *PriorAuditQuestionObservation) { value.AnswerPresent = false }, clean: false},
		{name: "unknown history", mutate: func(value *PriorAuditQuestionObservation) { value.UnknownHistory = true }, clean: false},
		{name: "accepted remediation", mutate: func(value *PriorAuditQuestionObservation) { value.RemediationAccepted = true }, clean: false},
		{name: "source changed", mutate: func(value *PriorAuditQuestionObservation) { value.SourceChanged = true }, clean: false},
		{name: "missing evidence validation", mutate: func(value *PriorAuditQuestionObservation) { value.EvidenceValidated = false }, clean: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			observation := base
			if testCase.mutate != nil {
				testCase.mutate(&observation)
			}
			if got := cleanObservation(observation, question); got != testCase.clean {
				t.Fatalf("cleanObservation = %t, want %t", got, testCase.clean)
			}
		})
	}
}

func TestScopeRecommendation_PrecedenceAndMandatoryFloor(t *testing.T) {
	fixture := PriorAuditMultiHistoryFixture()
	evaluation, err := EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate precedence fixture: %v", err)
	}
	byID := recommendationByID(evaluation)
	for _, questionID := range []string{priorAuditQuestionID(1), priorAuditQuestionID(2), priorAuditQuestionID(3), priorAuditQuestionID(4), priorAuditQuestionID(6), priorAuditQuestionID(7), priorAuditQuestionID(8)} {
		if !byID[questionID].IncludedByDefault || byID[questionID].CanDefer {
			t.Fatalf("precedence allowed deferral for %s: %+v", questionID, byID[questionID])
		}
	}
	if err := ValidateMandatoryFloor([]string{priorAuditQuestionID(5)}, evaluation, nil); err == nil {
		t.Fatal("mandatory/non-deferrable floor accepted a selection containing only the omitted optional question")
	}
	selected := []string{priorAuditQuestionID(1), priorAuditQuestionID(2), priorAuditQuestionID(3), priorAuditQuestionID(4), priorAuditQuestionID(6), priorAuditQuestionID(7), priorAuditQuestionID(8)}
	if err := ValidateMandatoryFloor(selected, evaluation, []QuestionDeviation{{QuestionVersionID: priorAuditQuestionID(5), Action: "DEFER", Reason: "Repeated validated-clean history is within the fixed recurrence interval."}}); err != nil {
		t.Fatalf("explicit deferral reason rejected: %v", err)
	}
}

func TestScopeRecommendation_FixedClockBoundaries(t *testing.T) {
	fixture := PriorAuditMultiHistoryFixture()
	fixture.Input.Questions = []RecommendationQuestion{fixture.Input.Questions[4]}
	fixture.Input.Audits = fixture.Input.Audits[:2]
	for _, audit := range fixture.Input.Audits {
		audit.Observations = []PriorAuditQuestionObservation{audit.Observations[4]}
	}
	fixture.Input.EvaluationAsOf = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 12, 0).Add(-time.Nanosecond)
	beforeDue, err := EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate before fixed recurrence boundary: %v", err)
	}
	if beforeDue.Recommendations[0].RecommendationState != RecommendationRecentlyVerified {
		t.Fatalf("before recurrence boundary = %+v, want recently verified", beforeDue.Recommendations[0])
	}
	fixture.Input.EvaluationAsOf = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 12, 0)
	atDue, err := EvaluatePriorAuditRecommendations(fixture.Input)
	if err != nil {
		t.Fatalf("evaluate at fixed recurrence boundary: %v", err)
	}
	if atDue.Recommendations[0].RecommendationState != RecommendationSuggestedNow {
		t.Fatalf("at recurrence boundary = %+v, want suggested now", atDue.Recommendations[0])
	}
}

func TestScopeRecommendation_AuditeeProjectionExcludesInternalFields(t *testing.T) {
	evaluation, err := EvaluatePriorAuditRecommendations(PriorAuditMultiHistoryFixture().Input)
	if err != nil {
		t.Fatalf("evaluate privacy fixture: %v", err)
	}
	projection := ProjectAuditeeQuestionRecommendations(evaluation)
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal Auditee projection: %v", err)
	}
	for _, forbidden := range []string{"recommendationState", "signalCodes", "rationale", "guardrails", "snapshotDigest", "managerReason", "internalRisk"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("Auditee projection leaked %q: %s", forbidden, encoded)
		}
	}
	if !strings.Contains(string(encoded), "includedByDefault") {
		t.Fatalf("Auditee projection lost public inclusion state: %s", encoded)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
