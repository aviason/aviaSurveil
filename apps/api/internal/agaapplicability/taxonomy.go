package agaapplicability

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"sync"
)

const frozenTaxonomyAuthorityGzipBase64 = `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`

var frozenTaxonomyAuthorityDigest struct {
	once   sync.Once
	digest string
}

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
	Digest:  "sha256:40517b48d0820db221501f89ff7fe58b120c6674e905cd722231d0b35ba18222",
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
	if err := validateFrozenTaxonomyAuthority(); err != nil {
		return Taxonomy{}
	}
	return cloneJSON(frozenTaxonomy)
}

func ComputeFrozenTaxonomySelfDigest() string {
	frozenTaxonomyAuthorityDigest.once.Do(func() {
		encoded, err := base64.StdEncoding.DecodeString(frozenTaxonomyAuthorityGzipBase64)
		if err != nil {
			return
		}
		reader, err := gzip.NewReader(bytes.NewReader(encoded))
		if err != nil {
			return
		}
		payload, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoder.UseNumber()
		var taxonomy map[string]any
		if err := decoder.Decode(&taxonomy); err != nil || taxonomy == nil {
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return
		}
		if _, ok := taxonomy["taxonomyDigest"]; !ok {
			return
		}
		delete(taxonomy, "taxonomyDigest")
		frozenTaxonomyAuthorityDigest.digest = digestValue("AGA-QUESTION-CLASSIFICATION-TAXONOMY-V1", taxonomy)
	})
	return frozenTaxonomyAuthorityDigest.digest
}

func validateTaxonomyDigest(expected string) error {
	if expected == "" || ComputeFrozenTaxonomySelfDigest() == "" || ComputeFrozenTaxonomySelfDigest() != expected {
		return ErrDigestMismatch
	}
	return nil
}

func validateFrozenTaxonomyAuthority() error {
	return validateTaxonomyDigest(frozenTaxonomy.Digest)
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
		return fmt.Errorf("%w: controlled code", ErrUnknownCode)
	}
	return nil
}

func normalizeStrings(values []string, allowed []string, field string, required bool) ([]string, error) {
	if required && len(values) == 0 {
		return nil, fmt.Errorf("%w: required code collection", ErrUnknownCode)
	}
	seen := make(map[string]struct{}, len(values))
	result := append([]string{}, values...)
	for _, value := range result {
		if err := validateCode(allowed, value, field); err != nil {
			return nil, err
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("%w: duplicate set member", ErrDuplicateProposalValue)
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
			return nil, fmt.Errorf("%w: qualifier code", ErrUnknownCode)
		}
		if _, exists := seen[qualifier.Key]; exists {
			return nil, fmt.Errorf("%w: duplicate qualifier key", ErrDuplicateProposalValue)
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
		return ProposalProjection{}, fmt.Errorf("%w: target compatibility", ErrTargetProfileMismatch)
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
			return fmt.Errorf("%w: inspection profile code", ErrUnknownCode)
		}
		if !contains(profile.AllowedTargetKinds, projection.CanonicalTargetKind) || !contains(profile.AllowedTargetProfileCodes, projection.TargetProfileCode) {
			return fmt.Errorf("%w: inspection profile target", ErrTargetProfileMismatch)
		}
		for _, inspectionType := range projection.InspectionTypeCodes {
			if !contains(profile.AllowedInspectionTypeCodes, inspectionType) {
				return fmt.Errorf("%w: inspection compatibility", ErrTargetProfileMismatch)
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
			return nil, fmt.Errorf("%w: proposal field", ErrEvidenceBinding)
		}
		if !validDigest(tuple.ProposalValueDigest) || !validDigest(tuple.InputFactValueDigest) {
			return nil, fmt.Errorf("%w: malformed evidence digest", ErrDigestMismatch)
		}
		if err := validateCode(taxonomy.RationaleCodes, tuple.RationaleCode, "confidenceEvidence.rationaleCode"); err != nil {
			return nil, err
		}
		if err := validateCode(taxonomy.InputFactSelectors, tuple.InputFactSelector, "confidenceEvidence.inputFactSelector"); err != nil {
			return nil, fmt.Errorf("%w: input fact selector", ErrUnknownInputFactSelector)
		}
		profileName, exists := taxonomy.EvidenceFieldProfiles[tuple.ProposalField]
		if !exists {
			return nil, fmt.Errorf("%w: evidence field profile", ErrEvidenceBinding)
		}
		profile := taxonomy.EvidenceProfiles[profileName]
		if !contains(profile.AllowedRationaleCodes, tuple.RationaleCode) || !contains(profile.AllowedInputFactSelectors, tuple.InputFactSelector) {
			return nil, fmt.Errorf("%w: evidence combination", ErrEvidenceBinding)
		}
		if tuple.SignalRuleID != "" {
			if !contains(taxonomy.SignalRuleIDs, tuple.SignalRuleID) {
				return nil, fmt.Errorf("%w: signal rule", ErrUnknownSignalRule)
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
			return fmt.Errorf("%w: proposal binding", ErrEvidenceBinding)
		}
		if err := validateSignalRuleBinding(taxonomy, binding, tuple); err != nil {
			return err
		}
		if !trustedEvidenceFact(facts, tuple) {
			return fmt.Errorf("%w: evidence fact", ErrEvidenceFactMismatch)
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
	return fmt.Errorf("%w: signal rule binding", ErrEvidenceBinding)
}

func ProjectionFieldEqual(left, right ProposalProjection, field string) bool {
	return projectionFieldEqual(FrozenTaxonomy(), left, right, field)
}

func projectionFieldEqual(taxonomy Taxonomy, left, right ProposalProjection, field string) bool {
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
