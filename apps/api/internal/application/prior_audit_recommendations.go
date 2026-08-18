package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	RecommendationSuggestedNow     = "SUGGESTED_NOW"
	RecommendationRecentlyVerified = "RECENTLY_VERIFIED"
	RecommendationUncertainSignal  = "UNCERTAIN_SIGNAL"
	RecommendationOutsideFocus     = "OUTSIDE_FOCUS"
	RecommendationMatchingOptional = "MATCHING_OPTIONAL"

	ClassificationMandatoryCore = "MANDATORY_CORE"
	ClassificationFocusedFull   = "FOCUSED_FULL"
	ClassificationRotational    = "ROTATIONAL_SAMPLE"
	ClassificationDeferEligible = "DEFER_ELIGIBLE"

	DefaultPriorAuditHistoryWindowMonths = 36
	MinimumValidatedCleanAuditCount      = 2
)

// ComparableAuditKey is the complete server-owned identity used to decide
// whether an historical Audit can participate in a recommendation. Labels are
// intentionally not part of this key; immutable IDs and the exact catalog
// version are.
type ComparableAuditKey struct {
	OrganizationID      string
	ProviderScopeRootID string
	ProviderScopeID     string
	RegulatedTargetID   string
	Location            string
	AuditType           string
	CatalogVersion      string
	UsageClass          string
}

func (key ComparableAuditKey) normalized() ComparableAuditKey {
	key.OrganizationID = strings.TrimSpace(key.OrganizationID)
	key.ProviderScopeRootID = strings.TrimSpace(key.ProviderScopeRootID)
	key.ProviderScopeID = strings.TrimSpace(key.ProviderScopeID)
	key.RegulatedTargetID = strings.TrimSpace(key.RegulatedTargetID)
	key.Location = strings.TrimSpace(key.Location)
	key.AuditType = strings.TrimSpace(key.AuditType)
	key.CatalogVersion = strings.TrimSpace(key.CatalogVersion)
	key.UsageClass = strings.TrimSpace(key.UsageClass)
	return key
}

func (key ComparableAuditKey) Validate() error {
	key = key.normalized()
	for name, value := range map[string]string{
		"organizationId": key.OrganizationID, "providerScopeId": key.ProviderScopeID,
		"regulatedTargetId": key.RegulatedTargetID, "location": key.Location,
		"auditType": key.AuditType, "catalogVersion": key.CatalogVersion,
	} {
		if value == "" {
			return fmt.Errorf("comparable audit key %s is required", name)
		}
	}
	return nil
}

// Equal is deliberately exact. A missing optional root/usage value does not
// widen comparison; it must match the other key exactly.
func (key ComparableAuditKey) Equal(other ComparableAuditKey) bool {
	left, right := key.normalized(), other.normalized()
	return left == right
}

func (key ComparableAuditKey) DigestInput() string {
	return strings.Join([]string{
		key.OrganizationID, key.ProviderScopeRootID, key.ProviderScopeID,
		key.RegulatedTargetID, key.Location, key.AuditType, key.CatalogVersion,
		key.UsageClass,
	}, "\x00")
}

type RecommendationQuestion struct {
	QuestionVersionID                  string
	FormCode                           string
	Prompt                             string
	SourceDigest                       string
	SourcePredecessorQuestionVersionID string
	Mandatory                          bool
	SafetyCritical                     bool
	RecurrenceMonths                   int
}

type PriorAuditQuestionObservation struct {
	QuestionVersionID   string
	Result              string
	AnswerPresent       bool
	EvidenceValidated   bool
	SourceDigest        string
	ResultAt            time.Time
	HasOpenFinding      bool
	HasRepeatFinding    bool
	HasOverdueCAP       bool
	SourceChanged       bool
	UnknownHistory      bool
	RemediationAccepted bool
}

type PriorAuditRecord struct {
	AuditID       string
	ComparableKey ComparableAuditKey
	ScopeStatus   string
	ReportKind    string
	ReportStatus  string
	CompletedAt   time.Time
	Observations  []PriorAuditQuestionObservation
}

type RecommendationEvaluationInput struct {
	ScopeKey            ComparableAuditKey
	Questions           []RecommendationQuestion
	Audits              []PriorAuditRecord
	EvaluationAsOf      time.Time
	HistoryWindowMonths int
}

type QuestionRecommendation struct {
	QuestionVersionID     string
	RecommendationState   string
	Classification        string
	IncludedByDefault     bool
	CanDefer              bool
	HistoryCount          int
	ComparableAuditCount  int
	LastComparableResult  string
	LastComparableAuditID string
	LastVerifiedAt        *time.Time
	RecurrenceDueAt       *time.Time
	SignalCodes           []string
	Rationale             string
	Guardrails            []string
}

type RecommendationEvaluation struct {
	EvaluationAsOf      time.Time
	HistoryWindowMonths int
	ComparableAuditIDs  []string
	Recommendations     []QuestionRecommendation
	SnapshotDigest      string
}

type QuestionDeviation struct {
	QuestionVersionID string
	Action            string
	Reason            string
}

type FrozenRecommendationSelection struct {
	EvaluationDigest       string
	RecommendationSnapshot string
	SelectedQuestionIDs    []string
	Deviations             []QuestionDeviation
	SelectionDigest        string
	FreezeDigest           string
}

type AuditeeQuestionRecommendation struct {
	QuestionVersionID string `json:"questionVersionId"`
	IncludedByDefault bool   `json:"includedByDefault"`
}

func validRecommendationState(value string) bool {
	switch value {
	case RecommendationSuggestedNow, RecommendationRecentlyVerified,
		RecommendationUncertainSignal, RecommendationOutsideFocus, RecommendationMatchingOptional:
		return true
	default:
		return false
	}
}

func validClassification(value string) bool {
	switch value {
	case ClassificationMandatoryCore, ClassificationFocusedFull, ClassificationRotational, ClassificationDeferEligible:
		return true
	default:
		return false
	}
}

func addSignal(signals *[]string, signal string) {
	if signal == "" {
		return
	}
	for _, existing := range *signals {
		if existing == signal {
			return
		}
	}
	*signals = append(*signals, signal)
}

func cleanObservation(observation PriorAuditQuestionObservation, question RecommendationQuestion) bool {
	// A missing answer, generic N/A, missing Evidence validation, accepted
	// remediation, source drift, or unknown state can never establish a clean
	// longitudinal fact on its own.
	return observation.AnswerPresent && observation.Result == "COMPLIANT" &&
		observation.EvidenceValidated && !observation.HasOpenFinding &&
		!observation.HasRepeatFinding && !observation.HasOverdueCAP &&
		!observation.SourceChanged && !observation.UnknownHistory &&
		!observation.RemediationAccepted && observation.SourceDigest == question.SourceDigest
}

func eligiblePriorAudits(input RecommendationEvaluationInput) ([]PriorAuditRecord, error) {
	if err := input.ScopeKey.Validate(); err != nil {
		return nil, err
	}
	if input.EvaluationAsOf.IsZero() {
		return nil, errors.New("evaluationAsOf is required")
	}
	windowMonths := input.HistoryWindowMonths
	if windowMonths == 0 {
		windowMonths = DefaultPriorAuditHistoryWindowMonths
	}
	if windowMonths < 1 || windowMonths > 120 {
		return nil, fmt.Errorf("history window months must be between 1 and 120")
	}
	windowStart := input.EvaluationAsOf.UTC().AddDate(0, -windowMonths, 0)
	seen := map[string]struct{}{}
	eligible := make([]PriorAuditRecord, 0, len(input.Audits))
	for _, audit := range input.Audits {
		if strings.TrimSpace(audit.AuditID) == "" {
			return nil, errors.New("prior Audit identity is required")
		}
		if _, already := seen[audit.AuditID]; already {
			continue
		}
		if audit.ScopeStatus != "RELEASED" || audit.ReportKind != "FINAL" || audit.ReportStatus != "LOCKED" {
			continue
		}
		completedAt := audit.CompletedAt.UTC()
		if completedAt.IsZero() || completedAt.After(input.EvaluationAsOf.UTC()) || completedAt.Before(windowStart) || !audit.ComparableKey.Equal(input.ScopeKey) {
			continue
		}
		seen[audit.AuditID] = struct{}{}
		eligible = append(eligible, audit)
	}
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].CompletedAt.Equal(eligible[j].CompletedAt) {
			return eligible[i].AuditID < eligible[j].AuditID
		}
		return eligible[i].CompletedAt.Before(eligible[j].CompletedAt)
	})
	return eligible, nil
}

func EvaluatePriorAuditRecommendations(input RecommendationEvaluationInput) (RecommendationEvaluation, error) {
	eligible, err := eligiblePriorAudits(input)
	if err != nil {
		return RecommendationEvaluation{}, err
	}
	questions := append([]RecommendationQuestion(nil), input.Questions...)
	sort.SliceStable(questions, func(i, j int) bool { return questions[i].QuestionVersionID < questions[j].QuestionVersionID })
	if len(questions) == 0 {
		return RecommendationEvaluation{}, errors.New("at least one question is required")
	}

	byQuestion := make(map[string][]struct {
		audit PriorAuditRecord
		obs   PriorAuditQuestionObservation
	})
	for _, audit := range eligible {
		seenQuestion := map[string]struct{}{}
		for _, observation := range audit.Observations {
			if _, duplicate := seenQuestion[observation.QuestionVersionID]; duplicate {
				continue
			}
			seenQuestion[observation.QuestionVersionID] = struct{}{}
			byQuestion[observation.QuestionVersionID] = append(byQuestion[observation.QuestionVersionID], struct {
				audit PriorAuditRecord
				obs   PriorAuditQuestionObservation
			}{audit: audit, obs: observation})
		}
	}

	recommendations := make([]QuestionRecommendation, 0, len(questions))
	for _, question := range questions {
		entries := byQuestion[question.QuestionVersionID]
		recommendation := QuestionRecommendation{
			QuestionVersionID:    question.QuestionVersionID,
			IncludedByDefault:    true,
			CanDefer:             false,
			ComparableAuditCount: len(eligible),
			Guardrails:           []string{"MANDATORY_FLOOR_ENFORCED", "FULL_CATALOG_OVERRIDE_ALLOWED"},
		}
		recommendation.HistoryCount = len(entries)
		if question.Mandatory {
			recommendation.Classification = ClassificationMandatoryCore
			recommendation.RecommendationState = RecommendationSuggestedNow
			addSignal(&recommendation.SignalCodes, "MANDATORY_CONTROL")
			recommendation.Rationale = "Mandatory controls remain in every comparable Audit scope."
		} else if question.SafetyCritical {
			recommendation.Classification = ClassificationMandatoryCore
			recommendation.RecommendationState = RecommendationSuggestedNow
			addSignal(&recommendation.SignalCodes, "SAFETY_CRITICAL_CONTROL")
			recommendation.Rationale = "Safety-critical controls remain in every comparable Audit scope."
		} else {
			unknown, open, repeat, overdue, changed := false, false, false, false, false
			allClean := len(entries) > 0
			cleanTimes := make([]time.Time, 0, len(entries))
			lastResultAt := time.Time{}
			for _, entry := range entries {
				observation := entry.obs
				if observation.UnknownHistory || !observation.AnswerPresent || observation.Result == "" || observation.Result == "NOT_APPLICABLE" {
					unknown = true
				}
				open = open || observation.HasOpenFinding
				repeat = repeat || observation.HasRepeatFinding
				overdue = overdue || observation.HasOverdueCAP
				changed = changed || observation.SourceChanged || observation.SourceDigest != question.SourceDigest || observation.RemediationAccepted
				if !cleanObservation(observation, question) {
					allClean = false
				} else {
					cleanTimes = append(cleanTimes, observation.ResultAt.UTC())
				}
				if observation.ResultAt.After(lastResultAt) {
					lastResultAt = observation.ResultAt.UTC()
					recommendation.LastComparableResult = observation.Result
					recommendation.LastComparableAuditID = entry.audit.AuditID
				}
			}
			recommendation.HistoryCount = len(entries)
			switch {
			case open:
				recommendation.Classification, recommendation.RecommendationState = ClassificationFocusedFull, RecommendationSuggestedNow
				addSignal(&recommendation.SignalCodes, "OPEN_FINDING")
				recommendation.Rationale = "Open Finding work keeps this question in the suggested scope."
			case repeat:
				recommendation.Classification, recommendation.RecommendationState = ClassificationFocusedFull, RecommendationSuggestedNow
				addSignal(&recommendation.SignalCodes, "REPEAT_FINDING")
				recommendation.Rationale = "Repeat Finding history keeps this question in the suggested scope."
			case overdue:
				recommendation.Classification, recommendation.RecommendationState = ClassificationFocusedFull, RecommendationSuggestedNow
				addSignal(&recommendation.SignalCodes, "OVERDUE_CAP")
				recommendation.Rationale = "Overdue CAP work keeps this question in the suggested scope."
			case changed:
				recommendation.Classification, recommendation.RecommendationState = ClassificationFocusedFull, RecommendationSuggestedNow
				addSignal(&recommendation.SignalCodes, "SOURCE_OR_MAPPING_CHANGED")
				recommendation.Rationale = "A source, mapping, successor, or accepted-remediation change requires full review."
			case unknown:
				recommendation.Classification, recommendation.RecommendationState = ClassificationRotational, RecommendationUncertainSignal
				addSignal(&recommendation.SignalCodes, "UNKNOWN_HISTORY")
				recommendation.Rationale = "History is incomplete or non-validating; the question remains suggested."
			case !allClean || len(entries) != len(eligible):
				recommendation.Classification, recommendation.RecommendationState = ClassificationRotational, RecommendationUncertainSignal
				addSignal(&recommendation.SignalCodes, "INSUFFICIENT_VALIDATED_CLEAN_HISTORY")
				recommendation.Rationale = "Missing or non-clean history is not evidence for automatic omission."
			case len(entries) < MinimumValidatedCleanAuditCount:
				recommendation.Classification, recommendation.RecommendationState = ClassificationRotational, RecommendationUncertainSignal
				addSignal(&recommendation.SignalCodes, "INSUFFICIENT_LONGITUDINAL_HISTORY")
				recommendation.Rationale = "One clean Audit is not sufficient longitudinal evidence for omission."
			default:
				lastVerified := latestTime(cleanTimes)
				recommendation.LastVerifiedAt = timePtr(lastVerified)
				dueAt := lastVerified.AddDate(0, recurrenceMonths(question), 0)
				recommendation.RecurrenceDueAt = timePtr(dueAt)
				if dueAt.After(input.EvaluationAsOf.UTC()) {
					recommendation.Classification, recommendation.RecommendationState = ClassificationDeferEligible, RecommendationRecentlyVerified
					recommendation.IncludedByDefault, recommendation.CanDefer = false, true
					addSignal(&recommendation.SignalCodes, "RECENTLY_VERIFIED")
					addSignal(&recommendation.SignalCodes, "DEFER_ELIGIBLE")
					recommendation.Rationale = "Repeated validated-clean history is within its recurrence interval; the question is safe to defer by default."
					recommendation.Guardrails = append(recommendation.Guardrails, "EXPLICIT_MANAGER_DEVIATION_REQUIRED")
				} else {
					recommendation.Classification, recommendation.RecommendationState = ClassificationRotational, RecommendationSuggestedNow
					addSignal(&recommendation.SignalCodes, "RECURRENCE_DUE")
					recommendation.Rationale = "The configured recurrence interval is due; keep this optional control in the suggested scope."
				}
			}
		}
		if recommendation.HistoryCount == 0 && recommendation.ComparableAuditCount > 0 {
			recommendation.RecommendationState = RecommendationUncertainSignal
			recommendation.Classification = ClassificationRotational
			addSignal(&recommendation.SignalCodes, "UNKNOWN_HISTORY")
			recommendation.Rationale = "No answer is recorded for this exact question version in the comparable history."
		}
		if !validRecommendationState(recommendation.RecommendationState) || !validClassification(recommendation.Classification) {
			return RecommendationEvaluation{}, fmt.Errorf("invalid recommendation for %s", question.QuestionVersionID)
		}
		recommendations = append(recommendations, recommendation)
	}

	auditIDs := make([]string, 0, len(eligible))
	for _, audit := range eligible {
		auditIDs = append(auditIDs, audit.AuditID)
	}
	evaluation := RecommendationEvaluation{
		EvaluationAsOf: input.EvaluationAsOf.UTC(), HistoryWindowMonths: input.HistoryWindowMonths,
		ComparableAuditIDs: auditIDs, Recommendations: recommendations,
	}
	if evaluation.HistoryWindowMonths == 0 {
		evaluation.HistoryWindowMonths = DefaultPriorAuditHistoryWindowMonths
	}
	digest, err := recommendationDigest(evaluation)
	if err != nil {
		return RecommendationEvaluation{}, err
	}
	evaluation.SnapshotDigest = digest
	return evaluation, nil
}

func recurrenceMonths(question RecommendationQuestion) int {
	if question.RecurrenceMonths > 0 && question.RecurrenceMonths <= 120 {
		return question.RecurrenceMonths
	}
	return 12
}

func latestTime(values []time.Time) time.Time {
	latest := time.Time{}
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest.UTC()
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	value = value.UTC()
	return &value
}

func recommendationDigest(evaluation RecommendationEvaluation) (string, error) {
	payload := struct {
		EvaluationAsOf      time.Time                `json:"evaluationAsOf"`
		HistoryWindowMonths int                      `json:"historyWindowMonths"`
		ComparableAuditIDs  []string                 `json:"comparableAuditIds"`
		Recommendations     []QuestionRecommendation `json:"recommendations"`
	}{evaluation.EvaluationAsOf.UTC(), evaluation.HistoryWindowMonths, append([]string(nil), evaluation.ComparableAuditIDs...), evaluation.Recommendations}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func FilterQuestionRecommendations(evaluation RecommendationEvaluation, fullCatalog bool) []QuestionRecommendation {
	items := make([]QuestionRecommendation, 0, len(evaluation.Recommendations))
	for _, recommendation := range evaluation.Recommendations {
		if fullCatalog || recommendation.IncludedByDefault {
			items = append(items, recommendation)
		}
	}
	return items
}

func ValidateMandatoryFloor(selected []string, evaluation RecommendationEvaluation, deviations []QuestionDeviation) error {
	selectedSet := make(map[string]struct{}, len(selected))
	for _, questionID := range selected {
		selectedSet[questionID] = struct{}{}
	}
	deviationReasons := map[string]string{}
	for _, deviation := range deviations {
		if deviation.Action == "DEFER" && strings.TrimSpace(deviation.Reason) != "" {
			deviationReasons[deviation.QuestionVersionID] = strings.TrimSpace(deviation.Reason)
		}
	}
	for _, recommendation := range evaluation.Recommendations {
		if recommendation.IncludedByDefault && !recommendation.CanDefer {
			if _, ok := selectedSet[recommendation.QuestionVersionID]; !ok {
				return fmt.Errorf("mandatory recommendation floor missing %s", recommendation.QuestionVersionID)
			}
		}
		if _, ok := selectedSet[recommendation.QuestionVersionID]; !ok && recommendation.CanDefer && deviationReasons[recommendation.QuestionVersionID] == "" {
			return fmt.Errorf("deferred question %s requires a manager reason", recommendation.QuestionVersionID)
		}
	}
	return nil
}

func FreezeRecommendationSelection(evaluation RecommendationEvaluation, selected []string, deviations []QuestionDeviation) (FrozenRecommendationSelection, error) {
	if err := ValidateMandatoryFloor(selected, evaluation, deviations); err != nil {
		return FrozenRecommendationSelection{}, err
	}
	selectedIDs := append([]string(nil), selected...)
	sort.Strings(selectedIDs)
	deviationCopy := append([]QuestionDeviation(nil), deviations...)
	sort.Slice(deviationCopy, func(i, j int) bool { return deviationCopy[i].QuestionVersionID < deviationCopy[j].QuestionVersionID })
	selectionPayload := struct {
		EvaluationDigest string              `json:"evaluationDigest"`
		SelectedIDs      []string            `json:"selectedQuestionVersionIds"`
		Deviations       []QuestionDeviation `json:"deviations"`
	}{evaluation.SnapshotDigest, selectedIDs, deviationCopy}
	bytes, err := json.Marshal(selectionPayload)
	if err != nil {
		return FrozenRecommendationSelection{}, err
	}
	selectionDigestBytes := sha256.Sum256(bytes)
	selectionDigest := "sha256:" + hex.EncodeToString(selectionDigestBytes[:])
	freezePayload := struct {
		EvaluationDigest       string `json:"evaluationDigest"`
		RecommendationSnapshot string `json:"recommendationSnapshot"`
		SelectionDigest        string `json:"selectionDigest"`
	}{evaluation.SnapshotDigest, evaluation.SnapshotDigest, selectionDigest}
	freezeBytes, err := json.Marshal(freezePayload)
	if err != nil {
		return FrozenRecommendationSelection{}, err
	}
	freezeDigestBytes := sha256.Sum256(freezeBytes)
	return FrozenRecommendationSelection{
		EvaluationDigest: evaluation.SnapshotDigest, RecommendationSnapshot: evaluation.SnapshotDigest,
		SelectedQuestionIDs: selectedIDs, Deviations: deviationCopy, SelectionDigest: selectionDigest,
		FreezeDigest: "sha256:" + hex.EncodeToString(freezeDigestBytes[:]),
	}, nil
}

func ProjectAuditeeQuestionRecommendations(evaluation RecommendationEvaluation) []AuditeeQuestionRecommendation {
	items := make([]AuditeeQuestionRecommendation, 0, len(evaluation.Recommendations))
	for _, recommendation := range evaluation.Recommendations {
		items = append(items, AuditeeQuestionRecommendation{QuestionVersionID: recommendation.QuestionVersionID, IncludedByDefault: recommendation.IncludedByDefault})
	}
	return items
}

// PriorAuditFixture is the deterministic golden input shared by Go tests,
// HTTP/mocked adapters, and the qualification helper.
type PriorAuditFixture struct {
	Name                    string
	Input                   RecommendationEvaluationInput
	ExpectedStates          map[string]string
	ExpectedClassifications map[string]string
	ExpectedIncluded        map[string]bool
	ExpectedHistoryCounts   map[string]int
}

const priorAuditFixtureCatalogVersion = "aga-approved-source@2.0.0"

func priorAuditQuestionID(ordinal int) string {
	return fmt.Sprintf("qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-%04d", ordinal)
}

func priorAuditFixtureScope() ComparableAuditKey {
	return ComparableAuditKey{
		OrganizationID: "ORG-PRIOR-AUDIT-QUALIFICATION", ProviderScopeRootID: "SCOPE-PRIOR-AUDIT-ROOT",
		ProviderScopeID: "SCOPE-PRIOR-AUDIT-001", RegulatedTargetID: "TARGET-PRIOR-AUDIT-001",
		Location: "Windhoek International Airport", AuditType: "RAMP_INSPECTION",
		CatalogVersion: priorAuditFixtureCatalogVersion, UsageClass: "GOVERNED_OPERATIONAL",
	}
}

func priorAuditQuestion(ordinal int, mandatory, safety bool, sourceDigest string) RecommendationQuestion {
	return RecommendationQuestion{QuestionVersionID: priorAuditQuestionID(ordinal), FormCode: "FSS-AGA-FORM-002", Prompt: fmt.Sprintf("Prior-audit fixture question %02d", ordinal), Mandatory: mandatory, SafetyCritical: safety, SourceDigest: sourceDigest, RecurrenceMonths: 12}
}

func priorAuditObservation(questionID, result, sourceDigest string, at time.Time) PriorAuditQuestionObservation {
	return PriorAuditQuestionObservation{QuestionVersionID: questionID, Result: result, AnswerPresent: true, EvidenceValidated: true, SourceDigest: sourceDigest, ResultAt: at}
}

func PriorAuditMultiHistoryFixture() PriorAuditFixture {
	scope := priorAuditFixtureScope()
	evaluationAsOf := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	questions := []RecommendationQuestion{
		priorAuditQuestion(1, true, false, "sha256:fixture-mandatory"),
		priorAuditQuestion(2, false, true, "sha256:fixture-safety"),
		priorAuditQuestion(3, false, false, "sha256:fixture-open"),
		priorAuditQuestion(4, false, false, "sha256:fixture-repeat"),
		priorAuditQuestion(5, false, false, "sha256:fixture-clean"),
		priorAuditQuestion(6, false, false, "sha256:fixture-due"),
		priorAuditQuestion(7, false, false, "sha256:fixture-successor"),
		priorAuditQuestion(8, false, false, "sha256:fixture-unknown"),
	}
	questions[6].SourcePredecessorQuestionVersionID = questions[5].QuestionVersionID
	audit := func(id string, completed time.Time) PriorAuditRecord {
		return PriorAuditRecord{AuditID: id, ComparableKey: scope, ScopeStatus: "RELEASED", ReportKind: "FINAL", ReportStatus: "LOCKED", CompletedAt: completed}
	}
	audits := []PriorAuditRecord{
		audit("AUD-PRIOR-MULTI-001", time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)),
		audit("AUD-PRIOR-MULTI-002", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)),
		audit("AUD-PRIOR-MULTI-003", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
	}
	for index := range audits {
		at := audits[index].CompletedAt
		for _, question := range questions {
			observation := priorAuditObservation(question.QuestionVersionID, "COMPLIANT", question.SourceDigest, at)
			switch question.QuestionVersionID {
			case priorAuditQuestionID(3):
				observation.Result = "NON_COMPLIANT"
				observation.HasOpenFinding = index == 0
			case priorAuditQuestionID(4):
				observation.HasRepeatFinding = index == 1
			case priorAuditQuestionID(6):
				observation.ResultAt = time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
			case priorAuditQuestionID(7):
				observation.SourceDigest = "sha256:fixture-predecessor"
				observation.SourceChanged = true
			case priorAuditQuestionID(8):
				observation.Result = ""
				observation.AnswerPresent = false
				observation.UnknownHistory = true
			}
			audits[index].Observations = append(audits[index].Observations, observation)
		}
	}
	return PriorAuditFixture{
		Name:                    "prior-audit-multi-history",
		Input:                   RecommendationEvaluationInput{ScopeKey: scope, Questions: questions, Audits: audits, EvaluationAsOf: evaluationAsOf, HistoryWindowMonths: DefaultPriorAuditHistoryWindowMonths},
		ExpectedStates:          map[string]string{priorAuditQuestionID(1): RecommendationSuggestedNow, priorAuditQuestionID(2): RecommendationSuggestedNow, priorAuditQuestionID(3): RecommendationSuggestedNow, priorAuditQuestionID(4): RecommendationSuggestedNow, priorAuditQuestionID(5): RecommendationRecentlyVerified, priorAuditQuestionID(6): RecommendationSuggestedNow, priorAuditQuestionID(7): RecommendationSuggestedNow, priorAuditQuestionID(8): RecommendationUncertainSignal},
		ExpectedClassifications: map[string]string{priorAuditQuestionID(1): ClassificationMandatoryCore, priorAuditQuestionID(2): ClassificationMandatoryCore, priorAuditQuestionID(3): ClassificationFocusedFull, priorAuditQuestionID(4): ClassificationFocusedFull, priorAuditQuestionID(5): ClassificationDeferEligible, priorAuditQuestionID(6): ClassificationRotational, priorAuditQuestionID(7): ClassificationFocusedFull, priorAuditQuestionID(8): ClassificationRotational},
		ExpectedIncluded:        map[string]bool{priorAuditQuestionID(1): true, priorAuditQuestionID(2): true, priorAuditQuestionID(3): true, priorAuditQuestionID(4): true, priorAuditQuestionID(5): false, priorAuditQuestionID(6): true, priorAuditQuestionID(7): true, priorAuditQuestionID(8): true},
		ExpectedHistoryCounts:   map[string]int{priorAuditQuestionID(1): 3, priorAuditQuestionID(2): 3, priorAuditQuestionID(3): 3, priorAuditQuestionID(4): 3, priorAuditQuestionID(5): 3, priorAuditQuestionID(6): 3, priorAuditQuestionID(7): 3, priorAuditQuestionID(8): 3},
	}
}

func PriorAuditSingleHistoryFixture() PriorAuditFixture {
	scope := priorAuditFixtureScope()
	evaluationAsOf := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	questions := []RecommendationQuestion{
		priorAuditQuestion(11, true, false, "sha256:fixture-single-mandatory"),
		priorAuditQuestion(12, false, false, "sha256:fixture-single-open"),
		priorAuditQuestion(13, false, false, "sha256:fixture-single-unknown"),
		priorAuditQuestion(14, false, false, "sha256:fixture-single-successor"),
		priorAuditQuestion(15, false, false, "sha256:fixture-single-clean"),
	}
	questions[3].SourcePredecessorQuestionVersionID = questions[2].QuestionVersionID
	audit := PriorAuditRecord{AuditID: "AUD-PRIOR-SINGLE-001", ComparableKey: scope, ScopeStatus: "RELEASED", ReportKind: "FINAL", ReportStatus: "LOCKED", CompletedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)}
	for _, question := range questions {
		observation := priorAuditObservation(question.QuestionVersionID, "COMPLIANT", question.SourceDigest, audit.CompletedAt)
		switch question.QuestionVersionID {
		case priorAuditQuestionID(12):
			observation.Result = "NON_COMPLIANT"
			observation.HasOpenFinding = true
		case priorAuditQuestionID(13):
			observation.Result = ""
			observation.AnswerPresent = false
			observation.UnknownHistory = true
		case priorAuditQuestionID(14):
			observation.SourceDigest = "sha256:fixture-single-predecessor"
			observation.SourceChanged = true
		}
		audit.Observations = append(audit.Observations, observation)
	}
	return PriorAuditFixture{
		Name:                    "prior-audit-single-history",
		Input:                   RecommendationEvaluationInput{ScopeKey: scope, Questions: questions, Audits: []PriorAuditRecord{audit}, EvaluationAsOf: evaluationAsOf, HistoryWindowMonths: DefaultPriorAuditHistoryWindowMonths},
		ExpectedStates:          map[string]string{priorAuditQuestionID(11): RecommendationSuggestedNow, priorAuditQuestionID(12): RecommendationSuggestedNow, priorAuditQuestionID(13): RecommendationUncertainSignal, priorAuditQuestionID(14): RecommendationSuggestedNow, priorAuditQuestionID(15): RecommendationUncertainSignal},
		ExpectedClassifications: map[string]string{priorAuditQuestionID(11): ClassificationMandatoryCore, priorAuditQuestionID(12): ClassificationFocusedFull, priorAuditQuestionID(13): ClassificationRotational, priorAuditQuestionID(14): ClassificationFocusedFull, priorAuditQuestionID(15): ClassificationRotational},
		ExpectedIncluded:        map[string]bool{priorAuditQuestionID(11): true, priorAuditQuestionID(12): true, priorAuditQuestionID(13): true, priorAuditQuestionID(14): true, priorAuditQuestionID(15): true},
		ExpectedHistoryCounts:   map[string]int{priorAuditQuestionID(11): 1, priorAuditQuestionID(12): 1, priorAuditQuestionID(13): 1, priorAuditQuestionID(14): 1, priorAuditQuestionID(15): 1},
	}
}
