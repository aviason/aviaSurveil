package agaapplicability

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Taxonomy is the executable projection of the frozen Gate 0B vocabulary.
// Every collection is copied by FrozenTaxonomy so callers cannot mutate the
// process-wide contract.
type Taxonomy struct {
	Version                       string
	Digest                        string
	MainDomainCodes               []string
	TopicCodes                    []string
	InspectionProfileCodes        []string
	InspectionTypeCodes           []string
	CanonicalTargetKinds          []string
	EligibleTargetKinds           []string
	TargetProfileCodes            []string
	ApplicabilityDispositions     []string
	EvidenceExpectationCodes      []string
	ExternalProviderTypes         []string
	ExternalInvolvementRoles      []string
	ExternalInvolvementConditions []string
	RationaleCodes                []string
	InputFactSelectors            []string
	SignalRuleIDs                 []string
	SourceReferenceKinds          []string
	BlockerCodes                  []string
	ProposalFields                []string
	DisagreementCodes             []string
	TargetCompatibility           map[string][]string
	OperationQualifierValues      map[string][]string
	ActivityQualifierValues       map[string][]string
	InspectionProfiles            map[string]InspectionProfileDefinition
	EvidenceProfiles              map[string]EvidenceCombinationProfile
	EvidenceFieldProfiles         map[string]string
	SignalRuleFieldRules          map[string][]SignalRuleFieldRule
}

type InspectionProfileDefinition struct {
	AllowedTargetKinds             []string
	AllowedTargetProfileCodes      []string
	AllowedInspectionTypeCodes     []string
	RequiredOperationQualifierKeys []string
	RequiredActivityQualifierKeys  []string
}

type EvidenceCombinationProfile struct {
	AllowedRationaleCodes     []string
	AllowedInputFactSelectors []string
}

type SignalRuleFieldRule struct {
	ProposalField                string
	ValueShape                   string
	AllowedValues                []string
	AllowedRationaleCodes        []string
	SignalAloneSatisfiesEvidence bool
}

var frozenTaxonomy = Taxonomy{
	Version: "AGA_QUESTION_CLASSIFICATION_V1",
	Digest:  "sha256:811cc22605499506e7f89058a86e0f7421445825dc1b6af975cea5cf4c5e2de6",
	MainDomainCodes: []string{
		"GOVERNANCE_ORGANIZATION_PERSONNEL", "CERTIFICATION_LICENSING_CHANGE",
		"AERODROME_MANUAL_DOCUMENT_CONTROL", "QUALITY_MANAGEMENT",
		"SAFETY_MANAGEMENT_RISK_ASSESSMENT", "AERODROME_DATA_INFORMATION_PUBLICATION",
		"PHYSICAL_CHARACTERISTICS_MOVEMENT_AREA", "OBSTACLES_OLS_WORKS",
		"VISUAL_AIDS_MARKINGS_SIGNS_LIGHTING", "ELECTRICAL_SYSTEMS_POWER",
		"APRON_GROUND_OPERATIONS", "RESCUE_FIRE_FIGHTING_FIRE_SAFETY",
		"EMERGENCY_PLANNING", "MAINTENANCE_OPERATIONAL_INSPECTION",
		"RUNWAY_SAFETY_FRICTION_SURFACE_CONDITIONS", "WILDLIFE_HAZARD_MANAGEMENT",
		"ENVIRONMENTAL_MANAGEMENT", "NIGHT_OPERATIONS_FACILITIES",
	},
	TopicCodes: []string{
		"ACCOUNTABLE_EXECUTIVE", "AERODROME_CERTIFICATE_APPLICATION", "AERODROME_DATA_QUALITY",
		"AERODROME_EMERGENCY_PLAN", "AERODROME_MANUAL_CONTROL", "AERODROME_ORGANIZATIONAL_CHANGE",
		"AERODROME_SECURITY", "APRON_MANAGEMENT", "CHANGE_MANAGEMENT", "CONTRACTED_SERVICE_OVERSIGHT",
		"DECLARED_DISTANCES", "ELECTRICAL_POWER_SUPPLY", "ENVIRONMENTAL_MANAGEMENT",
		"FOREIGN_OBJECT_DEBRIS_CONTROL", "LOW_VISIBILITY_OPERATIONS", "MOVEMENT_AREA_CONDITION",
		"OBSTACLE_LIMITATION_SURFACES", "PAVEMENT_STRENGTH_AND_FRICTION", "QUALITY_MANAGEMENT_SYSTEM",
		"RESCUE_AND_FIRE_FIGHTING_SERVICE", "RUNWAY_INCURSION_PREVENTION", "RUNWAY_SAFETY_PROGRAMME",
		"SAFETY_MANAGEMENT_SYSTEM", "STAFFING_AND_COMPETENCE", "VISUAL_AIDS_MARKINGS_AND_LIGHTING",
		"WILDLIFE_HAZARD_MANAGEMENT",
	},
	InspectionProfileCodes: []string{
		"AERODROME_CERTIFICATION", "AERODROME_DATA_QUALITY", "AERODROME_MANAGEMENT_SYSTEM",
		"EMERGENCY_AND_RFFS", "LOW_VISIBILITY_AND_NIGHT", "MOVEMENT_AREA_PHYSICAL_CHARACTERISTICS",
		"OBSTACLE_SAFEGUARDING", "RUNWAY_SAFETY", "VISUAL_AIDS_SYSTEM", "WILDLIFE_AND_ENVIRONMENT",
	},
	InspectionTypeCodes: []string{
		"CHANGE_APPROVAL", "DOCUMENT_AND_RECORD_REVIEW", "FOLLOW_UP", "INITIAL_CERTIFICATION",
		"ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE", "RENEWAL", "SPECIAL_PURPOSE",
	},
	CanonicalTargetKinds: []string{"ORGANIZATION", "PERSON", "FACILITY", "DEVICE", "SYSTEM", "ASSET", "LOCATION"},
	EligibleTargetKinds:  []string{"ORGANIZATION", "FACILITY", "DEVICE", "SYSTEM", "ASSET", "LOCATION"},
	TargetProfileCodes: []string{
		"AERODROME_DATA_SYSTEM", "AERODROME_MANAGEMENT_SYSTEM", "APRON_SYSTEM", "ELECTRICAL_SYSTEM",
		"MOVEMENT_AREA", "OBSTACLE_SAFEGUARDING_AREA", "RFFS_FUNCTION", "RUNWAY_SYSTEM",
		"TAXIWAY_SYSTEM", "VISUAL_AIDS_SYSTEM",
	},
	ApplicabilityDispositions: []string{
		"APPLICABLE", "CONDITIONAL_ON_CONFIGURATION", "CONDITIONAL_ON_FACILITY",
		"CONDITIONAL_ON_OPERATION", "CONDITIONAL_ON_PRIOR_RESPONSE",
		"CONDITIONAL_ON_SERVICE_ARRANGEMENT", "NOT_APPLICABLE_WITH_REASON",
		"REQUIRES_EXPERT_DETERMINATION",
	},
	EvidenceExpectationCodes: []string{
		"AERODROME_MANUAL", "APPROVED_DESIGN_DRAWING", "AUDIT_OR_INSPECTION_RECORD", "COMPETENCE_RECORD",
		"EMERGENCY_EXERCISE_RECORD", "FUNCTIONAL_TEST_RECORD", "MAINTENANCE_RECORD", "OBSTACLE_SURVEY",
		"PHOTOMETRIC_TEST_RECORD", "RISK_ASSESSMENT", "RUNWAY_CONDITION_RECORD",
		"SAFETY_MANAGEMENT_RECORD", "SOURCE_REFERENCE", "WILDLIFE_HAZARD_RECORD",
	},
	ExternalProviderTypes: []string{
		"ANSP", "CNS_PROVIDER", "AIS_AIM_PROVIDER", "MET_PROVIDER", "SAR_ORGANIZATION", "AVSEC_PROVIDER",
		"AIR_OPERATOR", "AMO", "ATO", "GROUND_HANDLING", "FUEL_PROVIDER", "CARGO_REGULATED_AGENT",
		"RPAS_UAS_OPERATOR",
	},
	ExternalInvolvementRoles: []string{
		"TECHNICAL_INTERFACE", "COORDINATION", "DATA_ORIGINATION", "DATA_PUBLICATION",
		"EVIDENCE_CONTRIBUTION", "OPERATIONAL_PARTICIPATION",
	},
	ExternalInvolvementConditions: []string{
		"AIS_AIM_PUBLICATION_REQUIRED", "ANSP_COORDINATION_REQUIRED", "CNS_SAFEGUARDING_REQUIRED",
		"CONTRACTED_SERVICE_ENGAGED", "EMERGENCY_AGENCY_PARTICIPATION_REQUIRED",
		"EVIDENCE_CONTRIBUTION_REQUIRED", "STAKEHOLDER_CONSULTATION_REQUIRED",
	},
	RationaleCodes: []string{
		"CONFIGURATION_CUE", "CONTRACTED_ACTIVITY_CUE", "DATA_QUALITY_CUE", "EXTERNAL_INTERFACE_CUE",
		"GOVERNANCE_CUE", "LOW_VISIBILITY_CUE", "MANAGEMENT_SYSTEM_CUE", "OPERATIONAL_SAFETY_CUE",
		"PHYSICAL_CHARACTERISTICS_CUE", "SOURCE_EVIDENCE_PRESENT", "SOURCE_GAP_CUE",
	},
	InputFactSelectors: []string{
		"FORM_METADATA_DIGEST", "QUESTION_BODY_DIGEST", "RESEARCH_ROW_DIGEST", "SOURCE_PROPOSAL_DIGEST",
		"SOURCE_REFERENCE_DIGEST", "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST",
	},
	SignalRuleIDs: []string{
		"CONTRACTED_PERSONNEL_V1", "CROSS_QUESTION_DEPENDENCY_V1", "EMBEDDED_FORM_TEMPLATE_TEXT_V1",
		"EMERGENCY_AGENCY_PARTICIPATION_V1", "EXPLICIT_EXTERNAL_ACTOR_V1",
		"EXTERNAL_APPROVAL_AUTHORITY_V1", "EXTERNAL_REVIEW_PROVIDER_V1", "EXTERNAL_STAKEHOLDER_V1",
		"PRIOR_RESPONSE_DEPENDENCY_V1", "PROVIDER_APPLICABILITY_UNRESOLVED_V1",
		"SOURCE_PROPOSAL_GAP_V1", "SPECIALIST_RESCUE_SERVICE_V1", "THIRD_PARTY_SIGNOFF_V1",
	},
	SourceReferenceKinds: []string{
		"PACKAGE_SOURCE_PROPOSAL", "PACKAGE_SOURCE_REFERENCE", "RESEARCH_ROW", "VALIDATOR_SIGNAL_RULE",
		"WORKBOOK_FORM_HINT",
	},
	BlockerCodes: []string{
		"CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW", "DECISION_NOT_SUPPLIED", "EXPERT_REVIEW_REQUIRED",
		"NOT_ATTESTED", "PROVIDER_APPLICABILITY_UNRESOLVED", "SOURCE_AUTHORITY_NOT_ATTESTED",
		"SOURCE_MAPPING_REQUIRED", "SOURCE_REFERENCE_MISSING",
	},
	ProposalFields: []string{
		"activityQualifiers", "applicabilityDisposition", "canonicalTargetKind", "evidenceExpectationCodes",
		"externalInvolvements", "inspectionProfileCodes", "inspectionTypeCodes", "mainDomainCode",
		"operationQualifiers", "targetProfileCode", "topicCodes",
	},
	DisagreementCodes: []string{
		"ACTIVITY_QUALIFIER_DISAGREEMENT", "APPLICABILITY_DISAGREEMENT", "CANONICAL_TARGET_KIND_DISAGREEMENT",
		"EVIDENCE_EXPECTATION_DISAGREEMENT", "EXTERNAL_INVOLVEMENT_DISAGREEMENT", "INSPECTION_PROFILE_DISAGREEMENT",
		"INSPECTION_TYPE_DISAGREEMENT", "MAIN_DOMAIN_DISAGREEMENT", "OPERATION_QUALIFIER_DISAGREEMENT",
		"TARGET_PROFILE_DISAGREEMENT", "TOPIC_SET_DISAGREEMENT",
	},
	TargetCompatibility: map[string][]string{
		"ORGANIZATION": {"AERODROME_MANAGEMENT_SYSTEM"},
		"PERSON":       {},
		"FACILITY":     {"APRON_SYSTEM", "ELECTRICAL_SYSTEM", "MOVEMENT_AREA", "RFFS_FUNCTION", "RUNWAY_SYSTEM", "TAXIWAY_SYSTEM", "VISUAL_AIDS_SYSTEM"},
		"DEVICE":       {"ELECTRICAL_SYSTEM", "VISUAL_AIDS_SYSTEM"},
		"SYSTEM":       {"AERODROME_DATA_SYSTEM", "AERODROME_MANAGEMENT_SYSTEM", "APRON_SYSTEM", "ELECTRICAL_SYSTEM", "OBSTACLE_SAFEGUARDING_AREA", "RFFS_FUNCTION", "RUNWAY_SYSTEM", "TAXIWAY_SYSTEM", "VISUAL_AIDS_SYSTEM"},
		"ASSET":        {"APRON_SYSTEM", "MOVEMENT_AREA", "RUNWAY_SYSTEM", "TAXIWAY_SYSTEM", "VISUAL_AIDS_SYSTEM"},
		"LOCATION":     {"MOVEMENT_AREA", "OBSTACLE_SAFEGUARDING_AREA", "RUNWAY_SYSTEM", "TAXIWAY_SYSTEM"},
	},
	OperationQualifierValues: map[string][]string{
		"APPROACH_CATEGORY":   {"CAT_I", "CAT_II", "CAT_III", "NON_PRECISION", "NOT_APPLICABLE"},
		"DAY_OR_NIGHT":        {"DAY", "DAY_AND_NIGHT", "NIGHT"},
		"LOW_VISIBILITY_BAND": {"LOW_VISIBILITY_PROCEDURES", "NORMAL_VISIBILITY", "VERY_LOW_VISIBILITY"},
		"OPERATION_STATUS":    {"ACTIVE", "CLOSED", "TEMPORARILY_RESTRICTED"},
		"RUNWAY_USE":          {"ARRIVAL", "DEPARTURE", "MIXED", "NOT_APPLICABLE"},
	},
	ActivityQualifierValues: map[string][]string{
		"ACTIVITY_TYPE": {"DATA_PROVISION", "EMERGENCY_RESPONSE", "LIGHTING_INSPECTION", "MAINTENANCE", "MARKING_INSPECTION", "OBSTACLE_SURVEY", "RISK_ASSESSMENT", "RUNWAY_CONDITION_ASSESSMENT", "WILDLIFE_HAZARD_ASSESSMENT"},
	},
	InspectionProfiles: map[string]InspectionProfileDefinition{
		"AERODROME_CERTIFICATION": {
			AllowedTargetKinds: []string{"ORGANIZATION", "SYSTEM"}, AllowedTargetProfileCodes: []string{"AERODROME_MANAGEMENT_SYSTEM", "AERODROME_DATA_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"INITIAL_CERTIFICATION", "RENEWAL", "CHANGE_APPROVAL", "DOCUMENT_AND_RECORD_REVIEW"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"AERODROME_DATA_QUALITY": {
			AllowedTargetKinds: []string{"SYSTEM"}, AllowedTargetProfileCodes: []string{"AERODROME_DATA_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"DOCUMENT_AND_RECORD_REVIEW", "PERIODIC_SURVEILLANCE", "SPECIAL_PURPOSE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"AERODROME_MANAGEMENT_SYSTEM": {
			AllowedTargetKinds: []string{"ORGANIZATION", "SYSTEM"}, AllowedTargetProfileCodes: []string{"AERODROME_MANAGEMENT_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"DOCUMENT_AND_RECORD_REVIEW", "FOLLOW_UP", "PERIODIC_SURVEILLANCE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"EMERGENCY_AND_RFFS": {
			AllowedTargetKinds: []string{"FACILITY", "SYSTEM"}, AllowedTargetProfileCodes: []string{"RFFS_FUNCTION", "AERODROME_MANAGEMENT_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"FOLLOW_UP", "ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE", "SPECIAL_PURPOSE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"LOW_VISIBILITY_AND_NIGHT": {
			AllowedTargetKinds: []string{"FACILITY", "SYSTEM", "LOCATION"}, AllowedTargetProfileCodes: []string{"RUNWAY_SYSTEM", "TAXIWAY_SYSTEM", "VISUAL_AIDS_SYSTEM", "ELECTRICAL_SYSTEM", "MOVEMENT_AREA"},
			AllowedInspectionTypeCodes: []string{"ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE", "SPECIAL_PURPOSE"}, RequiredOperationQualifierKeys: []string{"APPROACH_CATEGORY", "DAY_OR_NIGHT", "LOW_VISIBILITY_BAND", "OPERATION_STATUS", "RUNWAY_USE"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"MOVEMENT_AREA_PHYSICAL_CHARACTERISTICS": {
			AllowedTargetKinds: []string{"FACILITY", "ASSET", "LOCATION"}, AllowedTargetProfileCodes: []string{"MOVEMENT_AREA", "RUNWAY_SYSTEM", "TAXIWAY_SYSTEM", "APRON_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"CHANGE_APPROVAL", "FOLLOW_UP", "ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS", "RUNWAY_USE"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"OBSTACLE_SAFEGUARDING": {
			AllowedTargetKinds: []string{"LOCATION", "SYSTEM"}, AllowedTargetProfileCodes: []string{"OBSTACLE_SAFEGUARDING_AREA"},
			AllowedInspectionTypeCodes: []string{"CHANGE_APPROVAL", "DOCUMENT_AND_RECORD_REVIEW", "ON_SITE_INSPECTION", "SPECIAL_PURPOSE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"RUNWAY_SAFETY": {
			AllowedTargetKinds: []string{"FACILITY", "ASSET", "LOCATION", "SYSTEM"}, AllowedTargetProfileCodes: []string{"RUNWAY_SYSTEM", "TAXIWAY_SYSTEM", "MOVEMENT_AREA"},
			AllowedInspectionTypeCodes: []string{"FOLLOW_UP", "ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE", "SPECIAL_PURPOSE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS", "RUNWAY_USE"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"VISUAL_AIDS_SYSTEM": {
			AllowedTargetKinds: []string{"DEVICE", "FACILITY", "SYSTEM"}, AllowedTargetProfileCodes: []string{"VISUAL_AIDS_SYSTEM", "ELECTRICAL_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"FOLLOW_UP", "ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE"}, RequiredOperationQualifierKeys: []string{"DAY_OR_NIGHT", "OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
		"WILDLIFE_AND_ENVIRONMENT": {
			AllowedTargetKinds: []string{"LOCATION", "ORGANIZATION", "SYSTEM"}, AllowedTargetProfileCodes: []string{"MOVEMENT_AREA", "AERODROME_MANAGEMENT_SYSTEM"},
			AllowedInspectionTypeCodes: []string{"DOCUMENT_AND_RECORD_REVIEW", "ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE", "SPECIAL_PURPOSE"}, RequiredOperationQualifierKeys: []string{"OPERATION_STATUS"}, RequiredActivityQualifierKeys: []string{"ACTIVITY_TYPE"},
		},
	},
	EvidenceProfiles: map[string]EvidenceCombinationProfile{
		"SEMANTIC_CORE": {
			AllowedRationaleCodes:     []string{"CONFIGURATION_CUE", "DATA_QUALITY_CUE", "GOVERNANCE_CUE", "LOW_VISIBILITY_CUE", "MANAGEMENT_SYSTEM_CUE", "OPERATIONAL_SAFETY_CUE", "PHYSICAL_CHARACTERISTICS_CUE", "SOURCE_EVIDENCE_PRESENT", "SOURCE_GAP_CUE"},
			AllowedInputFactSelectors: []string{"FORM_METADATA_DIGEST", "QUESTION_BODY_DIGEST", "RESEARCH_ROW_DIGEST", "SOURCE_PROPOSAL_DIGEST", "SOURCE_REFERENCE_DIGEST", "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST"},
		},
		"SEMANTIC_AUXILIARY": {
			AllowedRationaleCodes:     []string{"CONFIGURATION_CUE", "CONTRACTED_ACTIVITY_CUE", "DATA_QUALITY_CUE", "EXTERNAL_INTERFACE_CUE", "GOVERNANCE_CUE", "LOW_VISIBILITY_CUE", "MANAGEMENT_SYSTEM_CUE", "OPERATIONAL_SAFETY_CUE", "PHYSICAL_CHARACTERISTICS_CUE", "SOURCE_EVIDENCE_PRESENT", "SOURCE_GAP_CUE"},
			AllowedInputFactSelectors: []string{"FORM_METADATA_DIGEST", "QUESTION_BODY_DIGEST", "RESEARCH_ROW_DIGEST", "SOURCE_PROPOSAL_DIGEST", "SOURCE_REFERENCE_DIGEST", "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST"},
		},
		"EXTERNAL_EDGE": {
			AllowedRationaleCodes:     []string{"CONTRACTED_ACTIVITY_CUE", "EXTERNAL_INTERFACE_CUE", "OPERATIONAL_SAFETY_CUE", "SOURCE_EVIDENCE_PRESENT", "SOURCE_GAP_CUE"},
			AllowedInputFactSelectors: []string{"QUESTION_BODY_DIGEST", "RESEARCH_ROW_DIGEST", "SOURCE_PROPOSAL_DIGEST", "SOURCE_REFERENCE_DIGEST", "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST"},
		},
	},
	EvidenceFieldProfiles: map[string]string{
		"mainDomainCode": "SEMANTIC_CORE", "canonicalTargetKind": "SEMANTIC_CORE", "targetProfileCode": "SEMANTIC_CORE", "applicabilityDisposition": "SEMANTIC_CORE", "inspectionProfileCodes": "SEMANTIC_CORE",
		"topicCodes": "SEMANTIC_AUXILIARY", "inspectionTypeCodes": "SEMANTIC_AUXILIARY", "operationQualifiers": "SEMANTIC_AUXILIARY", "activityQualifiers": "SEMANTIC_AUXILIARY", "evidenceExpectationCodes": "SEMANTIC_AUXILIARY", "externalInvolvements": "EXTERNAL_EDGE",
	},
	SignalRuleFieldRules: signalRuleFieldRules(),
}

func signalRuleFieldRules() map[string][]SignalRuleFieldRule {
	external := func(rationales ...string) SignalRuleFieldRule {
		return SignalRuleFieldRule{ProposalField: "externalInvolvements", ValueShape: "EXTERNAL_EDGE_TUPLE", AllowedValues: []string{"ANY_TAXONOMY_VALID_EXTERNAL_EDGE"}, AllowedRationaleCodes: rationales}
	}
	return map[string][]SignalRuleFieldRule{
		"CONTRACTED_PERSONNEL_V1": {
			{ProposalField: "topicCodes", ValueShape: "SET_MEMBER", AllowedValues: []string{"CONTRACTED_SERVICE_OVERSIGHT"}, AllowedRationaleCodes: []string{"CONTRACTED_ACTIVITY_CUE"}, SignalAloneSatisfiesEvidence: true},
			external("CONTRACTED_ACTIVITY_CUE", "EXTERNAL_INTERFACE_CUE"),
		},
		"CROSS_QUESTION_DEPENDENCY_V1":         {{ProposalField: "applicabilityDisposition", ValueShape: "SCALAR", AllowedValues: []string{"CONDITIONAL_ON_PRIOR_RESPONSE"}, AllowedRationaleCodes: []string{"CONFIGURATION_CUE"}, SignalAloneSatisfiesEvidence: true}},
		"EMBEDDED_FORM_TEMPLATE_TEXT_V1":       {{ProposalField: "applicabilityDisposition", ValueShape: "SCALAR", AllowedValues: []string{"REQUIRES_EXPERT_DETERMINATION"}, AllowedRationaleCodes: []string{"CONFIGURATION_CUE"}, SignalAloneSatisfiesEvidence: true}},
		"EMERGENCY_AGENCY_PARTICIPATION_V1":    {external("EXTERNAL_INTERFACE_CUE", "OPERATIONAL_SAFETY_CUE")},
		"EXPLICIT_EXTERNAL_ACTOR_V1":           {external("EXTERNAL_INTERFACE_CUE")},
		"EXTERNAL_APPROVAL_AUTHORITY_V1":       {external("EXTERNAL_INTERFACE_CUE")},
		"EXTERNAL_REVIEW_PROVIDER_V1":          {external("EXTERNAL_INTERFACE_CUE")},
		"EXTERNAL_STAKEHOLDER_V1":              {external("EXTERNAL_INTERFACE_CUE")},
		"PRIOR_RESPONSE_DEPENDENCY_V1":         {{ProposalField: "applicabilityDisposition", ValueShape: "SCALAR", AllowedValues: []string{"CONDITIONAL_ON_PRIOR_RESPONSE"}, AllowedRationaleCodes: []string{"CONFIGURATION_CUE"}, SignalAloneSatisfiesEvidence: true}},
		"PROVIDER_APPLICABILITY_UNRESOLVED_V1": {{ProposalField: "applicabilityDisposition", ValueShape: "SCALAR", AllowedValues: []string{"REQUIRES_EXPERT_DETERMINATION"}, AllowedRationaleCodes: []string{"SOURCE_GAP_CUE"}, SignalAloneSatisfiesEvidence: true}},
		"SOURCE_PROPOSAL_GAP_V1":               {{ProposalField: "evidenceExpectationCodes", ValueShape: "SET_MEMBER", AllowedValues: []string{"SOURCE_REFERENCE"}, AllowedRationaleCodes: []string{"SOURCE_GAP_CUE"}, SignalAloneSatisfiesEvidence: true}},
		"SPECIALIST_RESCUE_SERVICE_V1":         {external("EXTERNAL_INTERFACE_CUE", "OPERATIONAL_SAFETY_CUE")},
		"THIRD_PARTY_SIGNOFF_V1":               {external("EXTERNAL_INTERFACE_CUE")},
	}
}

func FrozenTaxonomy() Taxonomy {
	return cloneJSON(frozenTaxonomy)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func validateCode(values []string, value, field string) error {
	if !contains(values, value) {
		return fmt.Errorf("%w: %s=%q", ErrUnknownCode, field, value)
	}
	return nil
}

func normalizeStrings(values []string, allowed []string, field string, required bool) ([]string, error) {
	if required && len(values) == 0 {
		return nil, fmt.Errorf("%w: %s is empty", ErrUnknownCode, field)
	}
	seen := make(map[string]struct{}, len(values))
	result := append([]string{}, values...)
	for _, value := range result {
		if err := validateCode(allowed, value, field); err != nil {
			return nil, err
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("%w: %s=%q", ErrDuplicateProposalValue, field, value)
		}
		seen[value] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool { return strings.Compare(result[i], result[j]) < 0 })
	return result, nil
}

func normalizeQualifiers(values []Qualifier, allowed map[string][]string, field string) ([]Qualifier, error) {
	seen := make(map[string]struct{}, len(values))
	result := append([]Qualifier{}, values...)
	for _, qualifier := range result {
		allowedValues, ok := allowed[qualifier.Key]
		if !ok || !contains(allowedValues, qualifier.Value) {
			return nil, fmt.Errorf("%w: %s %s=%q", ErrUnknownCode, field, qualifier.Key, qualifier.Value)
		}
		if _, exists := seen[qualifier.Key]; exists {
			return nil, fmt.Errorf("%w: duplicate qualifier key %s", ErrDuplicateProposalValue, qualifier.Key)
		}
		seen[qualifier.Key] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Key == result[j].Key {
			return strings.Compare(result[i].Value, result[j].Value) < 0
		}
		return strings.Compare(result[i].Key, result[j].Key) < 0
	})
	return result, nil
}

func normalizeSourceRefs(taxonomy Taxonomy, values []SourceReference) ([]SourceReference, error) {
	seen := make(map[string]struct{}, len(values))
	result := append([]SourceReference{}, values...)
	for _, reference := range result {
		if err := validateCode(taxonomy.SourceReferenceKinds, reference.Kind, "sourceRefs.kind"); err != nil {
			return nil, err
		}
		if !validDigest(reference.ReferenceDigest) {
			return nil, fmt.Errorf("%w: source reference", ErrDigestMismatch)
		}
		key := reference.Kind + "\x00" + reference.ReferenceDigest
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: duplicate source reference", ErrDuplicateProposalValue)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind == result[j].Kind {
			return strings.Compare(result[i].ReferenceDigest, result[j].ReferenceDigest) < 0
		}
		return strings.Compare(result[i].Kind, result[j].Kind) < 0
	})
	return result, nil
}

func normalizeProjection(taxonomy Taxonomy, projection ProposalProjection) (ProposalProjection, error) {
	projection = cloneJSON(projection)
	if err := validateCode(taxonomy.MainDomainCodes, projection.MainDomainCode, "mainDomainCode"); err != nil {
		return ProposalProjection{}, err
	}
	if err := validateCode(taxonomy.EligibleTargetKinds, projection.CanonicalTargetKind, "canonicalTargetKind"); err != nil {
		return ProposalProjection{}, err
	}
	if err := validateCode(taxonomy.TargetProfileCodes, projection.TargetProfileCode, "targetProfileCode"); err != nil {
		return ProposalProjection{}, err
	}
	if !contains(taxonomy.TargetCompatibility[projection.CanonicalTargetKind], projection.TargetProfileCode) {
		return ProposalProjection{}, fmt.Errorf("%w: %s/%s", ErrTargetProfileMismatch, projection.CanonicalTargetKind, projection.TargetProfileCode)
	}
	if err := validateCode(taxonomy.ApplicabilityDispositions, projection.ApplicabilityDisposition, "applicabilityDisposition"); err != nil {
		return ProposalProjection{}, err
	}
	var err error
	if projection.TopicCodes, err = normalizeStrings(projection.TopicCodes, taxonomy.TopicCodes, "topicCodes", false); err != nil {
		return ProposalProjection{}, err
	}
	if projection.InspectionProfileCodes, err = normalizeStrings(projection.InspectionProfileCodes, taxonomy.InspectionProfileCodes, "inspectionProfileCodes", true); err != nil {
		return ProposalProjection{}, err
	}
	if projection.InspectionTypeCodes, err = normalizeStrings(projection.InspectionTypeCodes, taxonomy.InspectionTypeCodes, "inspectionTypeCodes", true); err != nil {
		return ProposalProjection{}, err
	}
	if projection.EvidenceExpectationCodes, err = normalizeStrings(projection.EvidenceExpectationCodes, taxonomy.EvidenceExpectationCodes, "evidenceExpectationCodes", false); err != nil {
		return ProposalProjection{}, err
	}
	if projection.OperationQualifiers, err = normalizeQualifiers(projection.OperationQualifiers, taxonomy.OperationQualifierValues, "operationQualifiers"); err != nil {
		return ProposalProjection{}, err
	}
	if projection.ActivityQualifiers, err = normalizeQualifiers(projection.ActivityQualifiers, taxonomy.ActivityQualifierValues, "activityQualifiers"); err != nil {
		return ProposalProjection{}, err
	}
	if len(projection.TopicCodes) > 26 || len(projection.InspectionProfileCodes) > 10 || len(projection.InspectionTypeCodes) > 8 || len(projection.OperationQualifiers) > 5 || len(projection.ActivityQualifiers) > 1 || len(projection.EvidenceExpectationCodes) > 14 || len(projection.ExternalInvolvements) > 13 {
		return ProposalProjection{}, fmt.Errorf("%w: proposal cardinality", ErrInvalidResolution)
	}
	if err := validateInspectionProfileCompatibility(taxonomy, projection); err != nil {
		return ProposalProjection{}, err
	}

	edgeKeys := make(map[string]struct{}, len(projection.ExternalInvolvements))
	for index := range projection.ExternalInvolvements {
		edge := &projection.ExternalInvolvements[index]
		if err := validateCode(taxonomy.ExternalProviderTypes, edge.ProviderTypeCode, "externalInvolvements.providerTypeCode"); err != nil {
			return ProposalProjection{}, err
		}
		if err := validateCode(taxonomy.ExternalInvolvementRoles, edge.InvolvementRoleCode, "externalInvolvements.involvementRoleCode"); err != nil {
			return ProposalProjection{}, err
		}
		if err := validateCode(taxonomy.ExternalInvolvementConditions, edge.ConditionCode, "externalInvolvements.conditionCode"); err != nil {
			return ProposalProjection{}, err
		}
		if err := validateCode(taxonomy.ApplicabilityDispositions, edge.ApplicabilityDisposition, "externalInvolvements.applicabilityDisposition"); err != nil {
			return ProposalProjection{}, err
		}
		if edge.RationaleCodes, err = normalizeStrings(edge.RationaleCodes, taxonomy.EvidenceProfiles["EXTERNAL_EDGE"].AllowedRationaleCodes, "externalInvolvements.rationaleCodes", false); err != nil {
			return ProposalProjection{}, err
		}
		if edge.BlockerCodes, err = normalizeStrings(edge.BlockerCodes, taxonomy.BlockerCodes, "externalInvolvements.blockerCodes", false); err != nil {
			return ProposalProjection{}, err
		}
		if edge.SourceRefs, err = normalizeSourceRefs(taxonomy, edge.SourceRefs); err != nil {
			return ProposalProjection{}, err
		}
		if edge.ConfidenceEvidence, err = normalizeEvidence(taxonomy, edge.ConfidenceEvidence); err != nil {
			return ProposalProjection{}, err
		}
		if len(edge.RationaleCodes) > 16 || len(edge.ConfidenceEvidence) > 32 || len(edge.SourceRefs) > 16 || len(edge.BlockerCodes) > 8 {
			return ProposalProjection{}, fmt.Errorf("%w: external involvement cardinality", ErrInvalidResolution)
		}
		key := externalInvolvementKey(*edge)
		if _, exists := edgeKeys[key]; exists {
			return ProposalProjection{}, fmt.Errorf("%w: duplicate external involvement", ErrDuplicateProposalValue)
		}
		edgeKeys[key] = struct{}{}
	}
	sort.Slice(projection.ExternalInvolvements, func(i, j int) bool {
		return strings.Compare(externalInvolvementKey(projection.ExternalInvolvements[i]), externalInvolvementKey(projection.ExternalInvolvements[j])) < 0
	})
	if projection.ExternalInvolvements == nil {
		projection.ExternalInvolvements = []ExternalInvolvement{}
	}
	return projection, nil
}

func validateInspectionProfileCompatibility(taxonomy Taxonomy, projection ProposalProjection) error {
	requiredOperationKeys := make(map[string]struct{})
	requiredActivityKeys := make(map[string]struct{})
	for _, profileCode := range projection.InspectionProfileCodes {
		profile, exists := taxonomy.InspectionProfiles[profileCode]
		if !exists {
			return fmt.Errorf("%w: inspection profile %s", ErrUnknownCode, profileCode)
		}
		if !contains(profile.AllowedTargetKinds, projection.CanonicalTargetKind) || !contains(profile.AllowedTargetProfileCodes, projection.TargetProfileCode) {
			return fmt.Errorf("%w: inspection profile %s target %s/%s", ErrTargetProfileMismatch, profileCode, projection.CanonicalTargetKind, projection.TargetProfileCode)
		}
		for _, inspectionType := range projection.InspectionTypeCodes {
			if !contains(profile.AllowedInspectionTypeCodes, inspectionType) {
				return fmt.Errorf("%w: inspection type %s is incompatible with profile %s", ErrTargetProfileMismatch, inspectionType, profileCode)
			}
		}
		for _, key := range profile.RequiredOperationQualifierKeys {
			requiredOperationKeys[key] = struct{}{}
		}
		for _, key := range profile.RequiredActivityQualifierKeys {
			requiredActivityKeys[key] = struct{}{}
		}
	}
	operationKeys := make(map[string]struct{}, len(projection.OperationQualifiers))
	for _, qualifier := range projection.OperationQualifiers {
		operationKeys[qualifier.Key] = struct{}{}
	}
	activityKeys := make(map[string]struct{}, len(projection.ActivityQualifiers))
	for _, qualifier := range projection.ActivityQualifiers {
		activityKeys[qualifier.Key] = struct{}{}
	}
	if !reflect.DeepEqual(operationKeys, requiredOperationKeys) || !reflect.DeepEqual(activityKeys, requiredActivityKeys) {
		return fmt.Errorf("%w: profile-required qualifier keys", ErrQualifierMismatch)
	}
	return nil
}

func ValidateProjection(taxonomy Taxonomy, projection ProposalProjection) error {
	_, err := normalizeProjection(taxonomy, projection)
	return err
}

func externalInvolvementKey(edge ExternalInvolvement) string {
	return strings.Join([]string{edge.ProviderTypeCode, edge.InvolvementRoleCode, edge.ConditionCode, edge.ApplicabilityDisposition}, "\x00")
}

func proposalBinding(domain, field string, value any, core bool) ProposalValueBinding {
	preimage := map[string]any{"proposalField": field}
	shape := "SCALAR"
	semanticValue := ""
	switch typed := value.(type) {
	case Qualifier:
		shape = "QUALIFIER_PAIR"
		semanticValue = typed.Key + "=" + typed.Value
		preimage["key"] = typed.Key
		preimage["value"] = typed.Value
	case ExternalInvolvement:
		shape = "EXTERNAL_EDGE_TUPLE"
		semanticValue = externalInvolvementKey(typed)
		preimage["providerTypeCode"] = typed.ProviderTypeCode
		preimage["involvementRoleCode"] = typed.InvolvementRoleCode
		preimage["conditionCode"] = typed.ConditionCode
		preimage["applicabilityDisposition"] = typed.ApplicabilityDisposition
	default:
		semanticValue = fmt.Sprint(typed)
		if domain == "AGA-PROPOSAL-VALUE-SET-MEMBER-V1" {
			shape = "SET_MEMBER"
		}
		preimage["value"] = typed
	}
	return ProposalValueBinding{ProposalField: field, ValueDigest: digestValue(domain, preimage), Core: core, ValueShape: shape, SemanticValue: semanticValue}
}

func ProposalValueBindings(taxonomy Taxonomy, projection ProposalProjection) []ProposalValueBinding {
	normalized, err := normalizeProjection(taxonomy, projection)
	if err != nil {
		return nil
	}
	bindings := []ProposalValueBinding{
		proposalBinding("AGA-PROPOSAL-VALUE-SCALAR-V1", "mainDomainCode", normalized.MainDomainCode, true),
		proposalBinding("AGA-PROPOSAL-VALUE-SCALAR-V1", "canonicalTargetKind", normalized.CanonicalTargetKind, true),
		proposalBinding("AGA-PROPOSAL-VALUE-SCALAR-V1", "targetProfileCode", normalized.TargetProfileCode, true),
		proposalBinding("AGA-PROPOSAL-VALUE-SCALAR-V1", "applicabilityDisposition", normalized.ApplicabilityDisposition, true),
	}
	for _, value := range normalized.TopicCodes {
		bindings = append(bindings, proposalBinding("AGA-PROPOSAL-VALUE-SET-MEMBER-V1", "topicCodes", value, false))
	}
	for _, value := range normalized.InspectionProfileCodes {
		bindings = append(bindings, proposalBinding("AGA-PROPOSAL-VALUE-SET-MEMBER-V1", "inspectionProfileCodes", value, true))
	}
	for _, value := range normalized.InspectionTypeCodes {
		bindings = append(bindings, proposalBinding("AGA-PROPOSAL-VALUE-SET-MEMBER-V1", "inspectionTypeCodes", value, false))
	}
	for _, value := range normalized.OperationQualifiers {
		bindings = append(bindings, proposalBinding("AGA-PROPOSAL-VALUE-QUALIFIER-V1", "operationQualifiers", value, false))
	}
	for _, value := range normalized.ActivityQualifiers {
		bindings = append(bindings, proposalBinding("AGA-PROPOSAL-VALUE-QUALIFIER-V1", "activityQualifiers", value, false))
	}
	for _, value := range normalized.EvidenceExpectationCodes {
		bindings = append(bindings, proposalBinding("AGA-PROPOSAL-VALUE-SET-MEMBER-V1", "evidenceExpectationCodes", value, false))
	}
	for _, edge := range normalized.ExternalInvolvements {
		bindings = append(bindings, ExternalInvolvementBinding(taxonomy, edge))
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].ProposalField == bindings[j].ProposalField {
			return strings.Compare(bindings[i].ValueDigest, bindings[j].ValueDigest) < 0
		}
		return strings.Compare(bindings[i].ProposalField, bindings[j].ProposalField) < 0
	})
	return bindings
}

func ExternalInvolvementBinding(_ Taxonomy, edge ExternalInvolvement) ProposalValueBinding {
	return proposalBinding("AGA-PROPOSAL-VALUE-EXTERNAL-INVOLVEMENT-V1", "externalInvolvements", edge, false)
}

func CoreProposalBindingKeys(taxonomy Taxonomy, projection ProposalProjection) map[string]bool {
	result := make(map[string]bool)
	for _, binding := range ProposalValueBindings(taxonomy, projection) {
		if binding.Core {
			result[binding.ProposalField+"\x00"+binding.ValueDigest] = true
		}
	}
	return result
}

func normalizeEvidence(taxonomy Taxonomy, evidence []ConfidenceEvidence) ([]ConfidenceEvidence, error) {
	seen := make(map[string]struct{}, len(evidence))
	result := append([]ConfidenceEvidence{}, evidence...)
	for _, tuple := range result {
		if !contains(taxonomy.ProposalFields, tuple.ProposalField) {
			return nil, fmt.Errorf("%w: proposalField=%q", ErrEvidenceBinding, tuple.ProposalField)
		}
		if !validDigest(tuple.ProposalValueDigest) || !validDigest(tuple.InputFactValueDigest) {
			return nil, fmt.Errorf("%w: malformed evidence digest", ErrDigestMismatch)
		}
		if err := validateCode(taxonomy.RationaleCodes, tuple.RationaleCode, "confidenceEvidence.rationaleCode"); err != nil {
			return nil, err
		}
		if err := validateCode(taxonomy.InputFactSelectors, tuple.InputFactSelector, "confidenceEvidence.inputFactSelector"); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnknownInputFactSelector, tuple.InputFactSelector)
		}
		profileName, exists := taxonomy.EvidenceFieldProfiles[tuple.ProposalField]
		if !exists {
			return nil, fmt.Errorf("%w: no field profile for %s", ErrEvidenceBinding, tuple.ProposalField)
		}
		profile := taxonomy.EvidenceProfiles[profileName]
		if !contains(profile.AllowedRationaleCodes, tuple.RationaleCode) || !contains(profile.AllowedInputFactSelectors, tuple.InputFactSelector) {
			return nil, fmt.Errorf("%w: forbidden evidence combination for %s", ErrEvidenceBinding, tuple.ProposalField)
		}
		if tuple.SignalRuleID != "" {
			if !contains(taxonomy.SignalRuleIDs, tuple.SignalRuleID) {
				return nil, fmt.Errorf("%w: %s", ErrUnknownSignalRule, tuple.SignalRuleID)
			}
			if tuple.InputFactSelector != "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST" {
				return nil, fmt.Errorf("%w: signal rule requires validator fact selector", ErrEvidenceFactMismatch)
			}
		} else if tuple.InputFactSelector == "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST" {
			return nil, fmt.Errorf("%w: validator fact selector requires signal rule", ErrUnknownSignalRule)
		}
		key := strings.Join([]string{tuple.ProposalField, tuple.ProposalValueDigest, tuple.RationaleCode, tuple.InputFactSelector, tuple.InputFactValueDigest, tuple.SignalRuleID}, "\x00")
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: duplicate evidence tuple", ErrDuplicateProposalValue)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool {
		left := strings.Join([]string{result[i].ProposalField, result[i].ProposalValueDigest, result[i].RationaleCode, result[i].InputFactSelector, result[i].InputFactValueDigest, result[i].SignalRuleID}, "\x00")
		right := strings.Join([]string{result[j].ProposalField, result[j].ProposalValueDigest, result[j].RationaleCode, result[j].InputFactSelector, result[j].InputFactValueDigest, result[j].SignalRuleID}, "\x00")
		return strings.Compare(left, right) < 0
	})
	return result, nil
}

func ValidateConfidenceEvidence(taxonomy Taxonomy, projection ProposalProjection, evidence []ConfidenceEvidence, facts EvidenceFacts) error {
	bindings := make(map[string]ProposalValueBinding)
	for _, binding := range ProposalValueBindings(taxonomy, projection) {
		if binding.ProposalField != "externalInvolvements" {
			bindings[binding.ProposalField+"\x00"+binding.ValueDigest] = binding
		}
	}
	normalized, err := normalizeEvidence(taxonomy, evidence)
	if err != nil {
		return err
	}
	for _, tuple := range normalized {
		binding, exists := bindings[tuple.ProposalField+"\x00"+tuple.ProposalValueDigest]
		if !exists {
			return fmt.Errorf("%w: %s", ErrEvidenceBinding, tuple.ProposalField)
		}
		if err := validateSignalRuleBinding(taxonomy, binding, tuple); err != nil {
			return err
		}
		if !trustedEvidenceFact(facts, tuple) {
			return fmt.Errorf("%w: %s", ErrEvidenceFactMismatch, tuple.InputFactSelector)
		}
	}
	return nil
}

func ValidateProjectionEvidence(taxonomy Taxonomy, projection ProposalProjection, facts EvidenceFacts) error {
	if err := ValidateProjection(taxonomy, projection); err != nil {
		return err
	}
	if err := ValidateConfidenceEvidence(taxonomy, projection, nil, facts); err != nil {
		return err
	}
	for _, edge := range projection.ExternalInvolvements {
		normalized, err := normalizeEvidence(taxonomy, edge.ConfidenceEvidence)
		if err != nil {
			return err
		}
		binding := ExternalInvolvementBinding(taxonomy, edge)
		for _, tuple := range normalized {
			if tuple.ProposalField != binding.ProposalField || tuple.ProposalValueDigest != binding.ValueDigest {
				return fmt.Errorf("%w: external involvement", ErrEvidenceBinding)
			}
			if err := validateSignalRuleBinding(taxonomy, binding, tuple); err != nil {
				return err
			}
			if !trustedEvidenceFact(facts, tuple) {
				return fmt.Errorf("%w: external involvement", ErrEvidenceFactMismatch)
			}
		}
	}
	return nil
}

func trustedEvidenceFact(facts EvidenceFacts, tuple ConfidenceEvidence) bool {
	for _, fact := range facts[tuple.InputFactSelector] {
		if fact.Digest == tuple.InputFactValueDigest && fact.SignalRuleID == tuple.SignalRuleID {
			return true
		}
	}
	return false
}

func validateSignalRuleBinding(taxonomy Taxonomy, binding ProposalValueBinding, tuple ConfidenceEvidence) error {
	if tuple.SignalRuleID == "" {
		return nil
	}
	for _, rule := range taxonomy.SignalRuleFieldRules[tuple.SignalRuleID] {
		valueAllowed := contains(rule.AllowedValues, binding.SemanticValue) || (binding.ValueShape == "EXTERNAL_EDGE_TUPLE" && contains(rule.AllowedValues, "ANY_TAXONOMY_VALID_EXTERNAL_EDGE"))
		if rule.ProposalField == binding.ProposalField && rule.ValueShape == binding.ValueShape && valueAllowed && contains(rule.AllowedRationaleCodes, tuple.RationaleCode) {
			return nil
		}
	}
	return fmt.Errorf("%w: signal rule %s cannot evidence %s/%s", ErrEvidenceBinding, tuple.SignalRuleID, binding.ProposalField, binding.SemanticValue)
}

func ProjectionFieldEqual(left, right ProposalProjection, field string) bool {
	taxonomy := FrozenTaxonomy()
	normalLeft, leftErr := normalizeProjection(taxonomy, left)
	normalRight, rightErr := normalizeProjection(taxonomy, right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	switch field {
	case "mainDomainCode":
		return normalLeft.MainDomainCode == normalRight.MainDomainCode
	case "topicCodes":
		return reflect.DeepEqual(normalLeft.TopicCodes, normalRight.TopicCodes)
	case "inspectionProfileCodes":
		return reflect.DeepEqual(normalLeft.InspectionProfileCodes, normalRight.InspectionProfileCodes)
	case "inspectionTypeCodes":
		return reflect.DeepEqual(normalLeft.InspectionTypeCodes, normalRight.InspectionTypeCodes)
	case "canonicalTargetKind":
		return normalLeft.CanonicalTargetKind == normalRight.CanonicalTargetKind
	case "targetProfileCode":
		return normalLeft.TargetProfileCode == normalRight.TargetProfileCode
	case "operationQualifiers":
		return reflect.DeepEqual(normalLeft.OperationQualifiers, normalRight.OperationQualifiers)
	case "activityQualifiers":
		return reflect.DeepEqual(normalLeft.ActivityQualifiers, normalRight.ActivityQualifiers)
	case "applicabilityDisposition":
		return normalLeft.ApplicabilityDisposition == normalRight.ApplicabilityDisposition
	case "evidenceExpectationCodes":
		return reflect.DeepEqual(normalLeft.EvidenceExpectationCodes, normalRight.EvidenceExpectationCodes)
	case "externalInvolvements":
		leftKeys := make([]string, len(normalLeft.ExternalInvolvements))
		rightKeys := make([]string, len(normalRight.ExternalInvolvements))
		for index, edge := range normalLeft.ExternalInvolvements {
			leftKeys[index] = externalInvolvementKey(edge)
		}
		for index, edge := range normalRight.ExternalInvolvements {
			rightKeys[index] = externalInvolvementKey(edge)
		}
		return reflect.DeepEqual(leftKeys, rightKeys)
	default:
		return false
	}
}
