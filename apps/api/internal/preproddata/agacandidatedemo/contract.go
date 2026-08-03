// Package agacandidatedemo reads the separately versioned AGA review package.
// It intentionally has no dependency on the connected preprod loader or any
// provider client.
package agacandidatedemo

import "regexp"

const (
	packageJSONName = "AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json"
	manifestName    = "MANIFEST.sha256"
)

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

type ExpectedCounts struct {
	Forms                        int
	FormsWithCandidateBoundaries int
	Questions                    int
	QuestionsWithProposals       int
	UnmappedQuestions            int
	QuestionSourceProposalLinks  int
	FormSourceProposalLinks      int
	UniqueSourceReferences       int
	ExpertRiskReviewBlockers     int
}

type ExpectedPackage struct {
	ZipBytes            int64
	ZipSHA256           string
	JSONBytes           int64
	JSONSHA256          string
	ManifestSHA256      string
	SourceArchiveSHA256 string
	SourceArchiveBytes  int64
	SourceArchivePDFs   int
	RegisterSHA256      string
	PackageVersion      string
	PackageStatus       string
	ExpectedCounts      ExpectedCounts
	FormCodes           []string
	ZeroFormCodes       []string
	RiskBands           map[string]int
	FormRiskBands       map[string]int
	StrictPolicies      bool
}

// ExactAcceptedPackage is the only production input contract. It contains no
// candidate question text, source URL, or original-PDF bytes.
func ExactAcceptedPackage() ExpectedPackage {
	return ExpectedPackage{
		ZipBytes:            336524,
		ZipSHA256:           "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
		JSONBytes:           3370312,
		JSONSHA256:          "sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15",
		ManifestSHA256:      "sha256:1be7b37e78a320da51cf7069b033240f1ad032b045d3e3cd5746c4b2115c19dc",
		SourceArchiveSHA256: "sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32",
		SourceArchiveBytes:  12227415,
		SourceArchivePDFs:   53,
		RegisterSHA256:      "sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f",
		PackageVersion:      "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1",
		PackageStatus:       "PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW",
		ExpectedCounts: ExpectedCounts{
			Forms: 52, FormsWithCandidateBoundaries: 31, Questions: 1310,
			QuestionsWithProposals: 1261, UnmappedQuestions: 49,
			QuestionSourceProposalLinks: 2329, FormSourceProposalLinks: 274,
			UniqueSourceReferences: 174, ExpertRiskReviewBlockers: 14,
		},
		FormCodes: completeFormCodes(),
		ZeroFormCodes: []string{
			"FSS-AGA-FORM-001", "FSS-AGA-FORM-003", "FSS-AGA-FORM-004",
			"FSS-AGA-FORM-005", "FSS-AGA-FORM-007", "FSS-AGA-FORM-008",
			"FSS-AGA-FORM-025", "FSS-AGA-FORM-026", "FSS-AGA-FORM-029",
			"FSS-AGA-FORM-032", "FSS-AGA-FORM-033", "FSS-AGA-FORM-035A",
			"FSS-AGA-FORM-036", "FSS-AGA-FORM-038", "FSS-AGA-FORM-039",
			"FSS-AGA-FORM-042", "FSS-AGA-FORM-043", "FSS-AGA-FORM-044",
			"FSS-AGA-FORM-045", "FSS-AGA-FORM-046", "FSS-AGA-FORM-052",
		},
		RiskBands: map[string]int{
			"PROPOSED_CONTROL_ASSURANCE": 50, "PROPOSED_HIGH_OPERATIONAL": 457,
			"PROPOSED_REVIEW_REQUIRED": 14, "PROPOSED_SAFETY_CRITICAL": 789,
		},
		FormRiskBands: map[string]int{
			"PROPOSED_CONTROL_ASSURANCE": 11, "PROPOSED_HIGH_OPERATIONAL": 23,
			"PROPOSED_REVIEW_REQUIRED": 4, "PROPOSED_SAFETY_CRITICAL": 14,
		},
		StrictPolicies: true,
	}
}

func completeFormCodes() []string {
	codes := make([]string, 0, 52)
	for number := 1; number <= 34; number++ {
		codes = append(codes, formCode(number))
	}
	codes = append(codes, "FSS-AGA-FORM-035A")
	for number := 36; number <= 48; number++ {
		codes = append(codes, formCode(number))
	}
	for number := 50; number <= 53; number++ {
		codes = append(codes, formCode(number))
	}
	return codes
}

func formCode(number int) string {
	if number < 10 {
		return "FSS-AGA-FORM-00" + string(rune('0'+number))
	}
	if number < 100 {
		return "FSS-AGA-FORM-0" + string(rune('0'+number/10)) + string(rune('0'+number%10))
	}
	return ""
}

type PackageIdentity struct {
	ZipSHA256      string `json:"zipSha256"`
	ZipBytes       int64  `json:"zipBytes"`
	JSONSHA256     string `json:"jsonSha256"`
	JSONBytes      int64  `json:"jsonBytes"`
	ManifestSHA256 string `json:"manifestSha256"`
	PackageVersion string `json:"packageVersion"`
	PackageStatus  string `json:"packageStatus"`
}

type AcceptedPackage struct {
	Identity       PackageIdentity
	Forms          []FormCandidate
	SourceCoverage []SourceProposal
}

type FormCandidate struct {
	FormCode                string              `json:"formCode"`
	FormSHA256              string              `json:"formSha256"`
	ArchiveSHA256           string              `json:"archiveSha256"`
	ArchivePath             string              `json:"archivePath"`
	DocumentTitle           string              `json:"documentTitle"`
	FormKind                string              `json:"formKind"`
	PageCount               int                 `json:"pageCount"`
	RegisterTitleCandidate  string              `json:"registerTitleCandidate"`
	QuestionCount           int                 `json:"questionCount"`
	QuestionExtractionState string              `json:"questionExtractionState"`
	QuestionBoundaryWarning *string             `json:"questionBoundaryWarning"`
	CandidateState          string              `json:"candidateState"`
	SourceMappingState      string              `json:"sourceMappingState"`
	PublicationState        string              `json:"publicationState"`
	FormSourceRefs          []string            `json:"formSourceRefs"`
	FormSourceProposals     []SourceProposal    `json:"formSourceProposals"`
	ProposedRisk            RiskProposal        `json:"proposedRisk"`
	Questions               []QuestionCandidate `json:"questions"`
}

type QuestionCandidate struct {
	ProposalID              string           `json:"proposalId"`
	Ordinal                 int              `json:"ordinal"`
	ProtocolCode            *string          `json:"protocolCode"`
	Page                    int              `json:"page"`
	SourceLocator           string           `json:"sourceLocator"`
	OriginalText            string           `json:"originalText"`
	TextDigest              string           `json:"textDigest"`
	SourceRefs              []string         `json:"sourceRefs"`
	SourceProposals         []SourceProposal `json:"sourceProposals"`
	SourceMappingState      string           `json:"sourceMappingState"`
	SourceAuthorityState    string           `json:"sourceAuthorityState"`
	ExtractionState         string           `json:"extractionState"`
	RiskClassificationState string           `json:"riskClassificationState"`
	DecisionState           string           `json:"decisionState"`
	ProposedRisk            RiskProposal     `json:"proposedRisk"`
}

type SourceProposal struct {
	Ref              string  `json:"ref"`
	SourceDocumentID string  `json:"sourceDocumentId"`
	SourceTitle      string  `json:"sourceTitle"`
	SourceURL        *string `json:"sourceUrl"`
	SourceSHA256     *string `json:"sourceSha256"`
	SourcePage       *int    `json:"sourcePage"`
	ClauseLocator    *string `json:"clauseLocator"`
	Status           string  `json:"status"`
	AuthorityState   string  `json:"authorityState"`
}

type RiskProposal struct {
	Band           string `json:"band"`
	Domain         string `json:"domain"`
	Rationale      string `json:"rationale"`
	SafetyCritical bool   `json:"safetyCritical"`
}
