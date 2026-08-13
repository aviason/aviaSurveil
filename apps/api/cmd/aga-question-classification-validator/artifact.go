package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aviason/aviaSurveil/internal/agaapplicability"
)

const (
	classificationBatchManifestSchema = "aga-hybrid-classification-batch-manifest/v2"
	candidateManifestSchema           = "aga-hybrid-classification-candidate/v1"
	passRunArtifactSchema             = "aga-hybrid-classification-pass-run/v1"
	passResultsArtifactSchema         = "aga-hybrid-classification-pass-results/v1"
	questionClassificationsSchema     = "aga-hybrid-classification-question-classifications/v1"
	cleanupReceiptSchema              = "aga-hybrid-classification-pass-cleanup/v1"
)

type packageDocument struct {
	PackageVersion string        `json:"packageVersion"`
	Forms          []packageForm `json:"forms"`
}

type packageForm struct {
	FormCode     string            `json:"formCode"`
	FormKind     string            `json:"formKind"`
	ProposedRisk packageRisk       `json:"proposedRisk"`
	Questions    []packageQuestion `json:"questions"`
}

type packageQuestion struct {
	Ordinal                 int               `json:"ordinal"`
	ProposalID              string            `json:"proposalId"`
	OriginalText            string            `json:"originalText"`
	TextDigest              string            `json:"textDigest"`
	ProposedRisk            packageRisk       `json:"proposedRisk"`
	SourceMappingState      string            `json:"sourceMappingState"`
	SourceAuthorityState    string            `json:"sourceAuthorityState"`
	ExtractionState         string            `json:"extractionState"`
	RiskClassificationState string            `json:"riskClassificationState"`
	DecisionState           string            `json:"decisionState"`
	SourceProposals         []json.RawMessage `json:"sourceProposals"`
	SourceRefs              []string          `json:"sourceRefs"`
}

type packageRisk struct {
	Band   string `json:"band"`
	Domain string `json:"domain"`
}

type batchManifestDocument struct {
	SchemaVersion         string                             `json:"schemaVersion"`
	BatchCount            int                                `json:"batchCount"`
	ItemCount             int                                `json:"itemCount"`
	ManifestDigest        string                             `json:"manifestDigest"`
	OrderedIdentityDigest string                             `json:"orderedIdentityDigest"`
	FixedInputDigests     agaapplicability.FixedInputDigests `json:"fixedInputDigests"`
	Batches               []batchManifestBatch               `json:"batches"`
}

type batchManifestBatch struct {
	BatchOrdinal          int                             `json:"batchOrdinal"`
	ItemCount             int                             `json:"itemCount"`
	OrderedIdentityDigest string                          `json:"orderedIdentityDigest"`
	Identities            []agaapplicability.BaseIdentity `json:"identities"`
}

type sourceItemMaterial struct {
	InputItem     agaapplicability.ClassificationPassInputItem
	Governance    agaapplicability.GovernanceState
	EvidenceFacts agaapplicability.EvidenceFacts
}

type fixedClassificationInputs struct {
	Fixed       agaapplicability.FixedInputDigests
	ManifestRaw []byte
	Manifest    batchManifestDocument
	Ordered     []agaapplicability.BaseIdentity
	Batches     [][]agaapplicability.BaseIdentity
	Items       map[string]sourceItemMaterial
}

type sourceArchiveRef struct {
	ByteLength int64  `json:"byteLength"`
	SHA256     string `json:"sha256"`
}

type privateValidationReceipt struct {
	SemanticEntries int    `json:"semanticEntries"`
	TransportNoise  int    `json:"transportNoise"`
	ExpandedBytes   uint64 `json:"expandedBytes"`
	Digest          string `json:"digest"`
	RootRemoved     bool   `json:"privateRootRemoved"`
}

type passRunArtifact struct {
	SchemaVersion            string                           `json:"schemaVersion"`
	Status                   string                           `json:"status"`
	PassRole                 string                           `json:"passRole"`
	ClassificationRunID      string                           `json:"classificationRunId"`
	PassRunID                string                           `json:"passRunId"`
	PromptDigest             string                           `json:"promptDigest"`
	ModelDescriptorDigest    string                           `json:"modelDescriptorDigest"`
	BatchManifestDigest      string                           `json:"batchManifestDigest"`
	BatchCount               int                              `json:"batchCount"`
	ItemCount                int                              `json:"itemCount"`
	ModelDescriptor          agaapplicability.ModelDescriptor `json:"modelDescriptor"`
	MetadataAcceptanceStatus *string                          `json:"metadataAcceptanceStatus,omitempty"`
	SourceArchive            sourceArchiveRef                 `json:"sourceArchive"`
	PrivateValidation        privateValidationReceipt         `json:"privateValidation"`
	PassSealReceipt          agaapplicability.PassSealReceipt `json:"passSealReceipt"`
}

type passResultsArtifact struct {
	SchemaVersion         string                                `json:"schemaVersion"`
	Status                string                                `json:"status"`
	PassRole              string                                `json:"passRole"`
	ClassificationRunID   string                                `json:"classificationRunId"`
	PassRunID             string                                `json:"passRunId"`
	PromptDigest          string                                `json:"promptDigest"`
	ModelDescriptorDigest string                                `json:"modelDescriptorDigest"`
	ItemCount             int                                   `json:"itemCount"`
	Records               []agaapplicability.PassProposalRecord `json:"records"`
}

type questionClassificationsArtifact struct {
	SchemaVersion       string                                      `json:"schemaVersion"`
	Status              string                                      `json:"status"`
	ClassificationRunID string                                      `json:"classificationRunId"`
	PromptDigest        string                                      `json:"promptDigest"`
	ItemCount           int                                         `json:"itemCount"`
	Items               []agaapplicability.SealedClassificationItem `json:"items"`
}

type candidateFileDigest struct {
	Filename   string `json:"filename"`
	ByteLength int64  `json:"byteLength"`
	SHA256     string `json:"sha256"`
}

type candidateManifest struct {
	SchemaVersion           string                `json:"schemaVersion"`
	Status                  string                `json:"status"`
	CandidateOnly           bool                  `json:"candidateOnly"`
	PackageVersion          string                `json:"packageVersion"`
	PackageJSONSHA256       string                `json:"packageJsonSha256"`
	ClassificationRunID     string                `json:"classificationRunId"`
	PromptDigest            string                `json:"promptDigest"`
	TaxonomyVersion         string                `json:"taxonomyVersion"`
	TaxonomyDigest          string                `json:"taxonomyDigest"`
	BatchManifestDigest     string                `json:"batchManifestDigest"`
	InputDigest             string                `json:"inputDigest"`
	BatchCount              int                   `json:"batchCount"`
	ItemCount               int                   `json:"itemCount"`
	PassProposalRecordCount int                   `json:"passProposalRecordCount"`
	AggregateDigest         string                `json:"aggregateDigest"`
	ClassificationRunDigest string                `json:"classificationRunDigest"`
	PassOneSealDigest       string                `json:"passOneSealDigest"`
	PassTwoSealDigest       string                `json:"passTwoSealDigest"`
	CandidateSource         sourceArchiveRef      `json:"candidateSource"`
	ChallengeSource         sourceArchiveRef      `json:"challengeSource"`
	Files                   []candidateFileDigest `json:"files"`
	ManifestDigest          string                `json:"manifestDigest"`
}

type cleanupPassReceipt struct {
	SourceArchiveSHA256 string `json:"sourceArchiveSha256"`
	SemanticEntries     int    `json:"semanticEntries"`
	TransportNoise      int    `json:"transportNoise"`
	ExpandedBytes       uint64 `json:"expandedBytes"`
	ReceiptDigest       string `json:"receiptDigest"`
	BatchCount          int    `json:"batchCount"`
	RecordCount         int    `json:"recordCount"`
	PrivateRootRemoved  bool   `json:"privateRootRemoved"`
}

type cleanupPlatformMetadata struct {
	Policy    string   `json:"policy"`
	Candidate []string `json:"candidate"`
	Challenge []string `json:"challenge"`
}

type cleanupReceipt struct {
	SchemaVersion                string                  `json:"schemaVersion"`
	Status                       string                  `json:"status"`
	Result                       string                  `json:"result"`
	Candidate                    cleanupPassReceipt      `json:"candidate"`
	Challenge                    cleanupPassReceipt      `json:"challenge"`
	PlatformMetadataAvailability cleanupPlatformMetadata `json:"platformMetadataAvailability"`
	Cleanup                      struct {
		PrivateRootRemoved   bool `json:"privateRootRemoved"`
		FilesRemaining       int  `json:"filesRemaining"`
		DirectoriesRemaining int  `json:"directoriesRemaining"`
		ProcessesRemaining   int  `json:"processesRemaining"`
	} `json:"cleanup"`
}

type reconcileOptions struct {
	PackagePath  string
	TaxonomyPath string
	ResearchZIP  string
	ManifestPath string
	CandidateDir string
	CandidateZIP string
	ChallengeZIP string
}

func runReconcile(arguments []string) (string, int) {
	options, ok := parseReconcileOptions(arguments, true)
	if !ok {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	bundle, err := loadFixedClassificationInputs(options.PackagePath, options.TaxonomyPath, options.ResearchZIP, options.ManifestPath)
	if err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	candidate, candidateArchive, candidatePrivate, err := validatePassArchiveForReconciliation(options.CandidateZIP, "CANDIDATE")
	if err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	challenge, challengeArchive, challengePrivate, err := validatePassArchiveForReconciliation(options.ChallengeZIP, "CHALLENGE")
	if err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	result, details, err := reconcileValidatedPasses(bundle, candidate, challenge, "")
	if err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	if err := writeCandidateArtifacts(options.CandidateDir, bundle, result, details, candidateArchive, challengeArchive, candidatePrivate, challengePrivate); err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	return "AGA_CANDIDATE_RECONCILED", 0
}

func runValidateCandidate(arguments []string) (string, int) {
	options, ok := parseReconcileOptions(arguments, false)
	if !ok {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	if err := validateCandidateDirectory(options.CandidateDir); err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	bundle, err := loadFixedClassificationInputs(options.PackagePath, options.TaxonomyPath, options.ResearchZIP, options.ManifestPath)
	if err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	if err := validateCandidateArtifacts(options.CandidateDir, bundle); err != nil {
		return diagnostic("ERR_AGA_CANDIDATE_INVALID", ""), 1
	}
	return "AGA_CANDIDATE_VALIDATED", 0
}

func parseReconcileOptions(arguments []string, requirePassZIPs bool) (reconcileOptions, bool) {
	flags := flag.NewFlagSet("aga-classification-reconcile", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := reconcileOptions{}
	flags.StringVar(&options.PackagePath, "package", "", "accepted package JSON")
	flags.StringVar(&options.TaxonomyPath, "taxonomy", "", "taxonomy JSON")
	flags.StringVar(&options.ResearchZIP, "research-zip", "", "fixed research deliverables ZIP")
	flags.StringVar(&options.ManifestPath, "batch-manifest", "", "fixed classification batch manifest")
	flags.StringVar(&options.CandidateDir, "candidate", "", "tracked candidate artifact directory")
	flags.StringVar(&options.CandidateZIP, "candidate-pass", "", "candidate sealed pass ZIP")
	flags.StringVar(&options.ChallengeZIP, "challenge-pass", "", "challenge sealed pass ZIP")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 {
		return reconcileOptions{}, false
	}
	if options.PackagePath == "" || options.TaxonomyPath == "" || options.ResearchZIP == "" || options.ManifestPath == "" || options.CandidateDir == "" {
		return reconcileOptions{}, false
	}
	if requirePassZIPs && (options.CandidateZIP == "" || options.ChallengeZIP == "") {
		return reconcileOptions{}, false
	}
	if !requirePassZIPs && (options.CandidateZIP != "" || options.ChallengeZIP != "") {
		return reconcileOptions{}, false
	}
	return options, true
}

func validatePassArchiveForReconciliation(sourcePath, expectedRole string) (passArchiveValidation, sourceArchiveRef, privateValidationReceipt, error) {
	absoluteSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return passArchiveValidation{}, sourceArchiveRef{}, privateValidationReceipt{}, errCandidateInvalid
	}
	archive, err := sourceArchiveInfo(absoluteSource)
	if err != nil {
		return passArchiveValidation{}, sourceArchiveRef{}, privateValidationReceipt{}, errCandidateInvalid
	}
	parent, err := os.MkdirTemp("", "aga-classification-private-parent-")
	if err != nil {
		return passArchiveValidation{}, sourceArchiveRef{}, privateValidationReceipt{}, errIsolation
	}
	defer os.RemoveAll(parent)
	root := filepath.Join(parent, "private-root")
	validation, receipt, err := validatePassZIPInPrivateRoot(absoluteSource, root, expectedRole)
	private := privateValidationReceipt{SemanticEntries: receipt.SemanticEntries, TransportNoise: receipt.TransportNoise, ExpandedBytes: receipt.ExpandedBytes, Digest: receipt.Digest}
	if _, statErr := os.Lstat(root); os.IsNotExist(statErr) {
		private.RootRemoved = true
	}
	if err != nil || !private.RootRemoved {
		return passArchiveValidation{}, archive, private, errCandidateInvalid
	}
	return validation, archive, private, nil
}

func sourceArchiveInfo(filename string) (sourceArchiveRef, error) {
	info, err := os.Stat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 {
		return sourceArchiveRef{}, errCandidateInvalid
	}
	digest, err := sha256File(filename)
	if err != nil {
		return sourceArchiveRef{}, errCandidateInvalid
	}
	return sourceArchiveRef{ByteLength: info.Size(), SHA256: digest}, nil
}

func sha256File(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func loadFixedClassificationInputs(packagePath, taxonomyPath, researchZIPPath, manifestPath string) (fixedClassificationInputs, error) {
	fixed := agaapplicability.FrozenFixedInputDigests()
	if err := validateTaxonomyFile(taxonomyPath); err != nil {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	packageBytes, err := os.ReadFile(packagePath)
	if err != nil || digestBytes(packageBytes) != fixed.PackageJSONSHA256 {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	var document packageDocument
	if err := json.Unmarshal(packageBytes, &document); err != nil || document.PackageVersion != agaapplicability.FrozenPackageVersion {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	research, err := loadResearchFacts(researchZIPPath, fixed)
	if err != nil {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	manifest, ordered, batches, err := parseBatchManifest(manifestBytes, fixed)
	if err != nil {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	items, err := buildSourceItems(document, research)
	if err != nil {
		return fixedClassificationInputs{}, errCandidateInvalid
	}
	for _, identity := range ordered {
		if _, exists := items[identity.Key()]; !exists {
			return fixedClassificationInputs{}, errCandidateInvalid
		}
	}
	return fixedClassificationInputs{Fixed: fixed, ManifestRaw: manifestBytes, Manifest: manifest, Ordered: ordered, Batches: batches, Items: items}, nil
}

func validateTaxonomyFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	var root struct {
		TaxonomyVersion string `json:"taxonomyVersion"`
		TaxonomyDigest  string `json:"taxonomyDigest"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	taxonomy := agaapplicability.FrozenTaxonomy()
	if root.TaxonomyVersion != taxonomy.Version || root.TaxonomyDigest != taxonomy.Digest {
		return errCandidateInvalid
	}
	return nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func loadResearchFacts(filename string, fixed agaapplicability.FixedInputDigests) (map[string]agaapplicability.ClassificationResearchCandidateFacts, error) {
	digest, err := sha256File(filename)
	if err != nil || digest != fixed.ResearchZIPSHA256 {
		return nil, errCandidateInvalid
	}
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return nil, errCandidateInvalid
	}
	defer archive.Close()
	entries := map[string]*zip.File{}
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() || strings.Contains(entry.Name, "/") {
			continue
		}
		entries[entry.Name] = entry
	}
	questionCSV := entries["question_level_review.csv"]
	providerCSV := entries["provider_classification_matrix.csv"]
	ambiguityCSV := entries["ambiguous_unmapped_inventory.csv"]
	if questionCSV == nil || providerCSV == nil || ambiguityCSV == nil {
		return nil, errCandidateInvalid
	}
	questionBytes, err := readZipFile(questionCSV)
	if err != nil || digestBytes(questionBytes) != fixed.ResearchQuestionCSVSHA256 {
		return nil, errCandidateInvalid
	}
	providerBytes, err := readZipFile(providerCSV)
	if err != nil || digestBytes(providerBytes) != fixed.ProviderClassificationCSVSHA256 {
		return nil, errCandidateInvalid
	}
	ambiguityBytes, err := readZipFile(ambiguityCSV)
	if err != nil || digestBytes(ambiguityBytes) != fixed.AmbiguityCSVSHA256 {
		return nil, errCandidateInvalid
	}
	reader := csv.NewReader(bytes.NewReader(questionBytes))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, errCandidateInvalid
	}
	expectedHeader := []string{"form_code", "proposal_id", "ordinal", "text_digest", "target_kind", "operation_activity_qualifier", "primary_subject_proposal", "operational_interface_candidates", "evidence_contributor_candidates", "provider_applicability_unresolved", "unresolved_reasons", "source_refs"}
	if !reflect.DeepEqual(records[0], expectedHeader) {
		return nil, errCandidateInvalid
	}
	facts := make(map[string]agaapplicability.ClassificationResearchCandidateFacts, len(records)-1)
	for _, row := range records[1:] {
		if len(row) != len(expectedHeader) {
			return nil, errCandidateInvalid
		}
		fact := agaapplicability.ClassificationResearchCandidateFacts{FormCode: row[0], ProposalID: row[1], Ordinal: row[2], TextDigest: row[3], TargetKind: row[4], OperationActivityQualifier: row[5], PrimarySubjectProposal: row[6], OperationalInterfaceCandidates: row[7], EvidenceContributorCandidates: row[8], ProviderApplicabilityUnresolved: row[9], UnresolvedReasons: row[10], SourceRefs: row[11]}
		key := strings.Join([]string{fact.FormCode, fact.ProposalID, fact.Ordinal, fact.TextDigest}, "\x1f")
		if _, exists := facts[key]; exists {
			return nil, errCandidateInvalid
		}
		facts[key] = fact
	}
	if len(facts) != agaapplicability.FrozenBaseQuestionCount {
		return nil, errCandidateInvalid
	}
	return facts, nil
}

func parseBatchManifest(data []byte, fixed agaapplicability.FixedInputDigests) (batchManifestDocument, []agaapplicability.BaseIdentity, [][]agaapplicability.BaseIdentity, error) {
	var manifest batchManifestDocument
	if err := json.Unmarshal(data, &manifest); err != nil {
		return batchManifestDocument{}, nil, nil, err
	}
	if manifest.SchemaVersion != classificationBatchManifestSchema || manifest.BatchCount != 25 || manifest.ItemCount != agaapplicability.FrozenBaseQuestionCount || manifest.ManifestDigest != agaapplicability.FrozenBatchManifestDigest || manifest.OrderedIdentityDigest != agaapplicability.FrozenOrderedIdentityDigest || !reflect.DeepEqual(manifest.FixedInputDigests, fixed) || len(manifest.Batches) != 25 {
		return batchManifestDocument{}, nil, nil, errCandidateInvalid
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return batchManifestDocument{}, nil, nil, err
	}
	delete(object, "manifestDigest")
	manifestDigest, err := agaapplicability.DigestValue("AGA-CLASSIFICATION-BATCH-MANIFEST-SET-V2", object)
	if err != nil || manifestDigest != manifest.ManifestDigest {
		return batchManifestDocument{}, nil, nil, errCandidateInvalid
	}
	ordered := make([]agaapplicability.BaseIdentity, 0, manifest.ItemCount)
	batches := make([][]agaapplicability.BaseIdentity, 0, manifest.BatchCount)
	for index, batch := range manifest.Batches {
		if batch.BatchOrdinal != index+1 || batch.ItemCount != len(batch.Identities) || batch.ItemCount < 1 {
			return batchManifestDocument{}, nil, nil, errCandidateInvalid
		}
		if digest, err := agaapplicability.DigestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", batch.Identities); err != nil || digest != batch.OrderedIdentityDigest {
			return batchManifestDocument{}, nil, nil, errCandidateInvalid
		}
		batchCopy := append([]agaapplicability.BaseIdentity(nil), batch.Identities...)
		for _, identity := range batchCopy {
			if err := identity.Validate(); err != nil {
				return batchManifestDocument{}, nil, nil, errCandidateInvalid
			}
		}
		batches = append(batches, batchCopy)
		ordered = append(ordered, batchCopy...)
	}
	if len(ordered) != agaapplicability.FrozenBaseQuestionCount {
		return batchManifestDocument{}, nil, nil, errCandidateInvalid
	}
	if digest, err := agaapplicability.DigestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", ordered); err != nil || digest != agaapplicability.FrozenOrderedIdentityDigest {
		return batchManifestDocument{}, nil, nil, errCandidateInvalid
	}
	seen := make(map[string]struct{}, len(ordered))
	for _, identity := range ordered {
		if _, exists := seen[identity.Key()]; exists {
			return batchManifestDocument{}, nil, nil, errCandidateInvalid
		}
		seen[identity.Key()] = struct{}{}
	}
	return manifest, ordered, batches, nil
}

func buildSourceItems(document packageDocument, research map[string]agaapplicability.ClassificationResearchCandidateFacts) (map[string]sourceItemMaterial, error) {
	if document.PackageVersion != agaapplicability.FrozenPackageVersion {
		return nil, errCandidateInvalid
	}
	items := make(map[string]sourceItemMaterial, agaapplicability.FrozenBaseQuestionCount)
	for _, form := range document.Forms {
		for _, question := range form.Questions {
			identity := agaapplicability.BaseIdentity{PackageVersion: document.PackageVersion, PackageJSONSHA256: agaapplicability.FrozenPackageJSONSHA256, FormCode: form.FormCode, ProposalID: question.ProposalID, Ordinal: question.Ordinal, TextDigest: question.TextDigest}
			if err := identity.Validate(); err != nil || question.TextDigest != rawTextDigest(question.OriginalText) {
				return nil, errCandidateInvalid
			}
			sourceProposalDigests := make([]string, 0, len(question.SourceProposals))
			for _, raw := range question.SourceProposals {
				var value any
				decoder := json.NewDecoder(bytes.NewReader(raw))
				decoder.UseNumber()
				if err := decoder.Decode(&value); err != nil {
					return nil, errCandidateInvalid
				}
				digest, err := agaapplicability.DigestValue("AGA-SOURCE-PROPOSAL-FACT-V1", value)
				if err != nil {
					return nil, errCandidateInvalid
				}
				sourceProposalDigests = append(sourceProposalDigests, digest)
			}
			sourceReferenceDigests := make([]string, 0, len(question.SourceRefs))
			for _, reference := range question.SourceRefs {
				digest, err := agaapplicability.DigestValue("AGA-SOURCE-REFERENCE-FACT-V1", reference)
				if err != nil {
					return nil, errCandidateInvalid
				}
				sourceReferenceDigests = append(sourceReferenceDigests, digest)
			}
			researchFacts, exists := research[strings.Join([]string{form.FormCode, question.ProposalID, strconv.Itoa(question.Ordinal), question.TextDigest}, "\x1f")]
			if !exists {
				return nil, errCandidateInvalid
			}
			decisionState := question.DecisionState
			if decisionState == "NOT_SUPPLIED" {
				decisionState = agaapplicability.DecisionNotSupplied
			}
			packageFacts := agaapplicability.ClassificationPackageFacts{FormKind: form.FormKind, FormRiskBand: form.ProposedRisk.Band, QuestionRiskBand: question.ProposedRisk.Band, QuestionRiskDomain: question.ProposedRisk.Domain, SourceMappingState: question.SourceMappingState, SourceAuthorityState: question.SourceAuthorityState, ExtractionState: question.ExtractionState, RiskClassificationState: question.RiskClassificationState, DecisionState: question.DecisionState, SourceProposalDigests: sourceProposalDigests, SourceReferenceDigests: sourceReferenceDigests}
			inputItem := agaapplicability.ClassificationPassInputItem{Identity: identity, QuestionBody: question.OriginalText, PackageFacts: packageFacts, ResearchCandidateFacts: researchFacts}
			sourceGap := len(question.SourceProposals) == 0
			externalUnresolved := researchFacts.ProviderApplicabilityUnresolved == "true"
			blockers := make([]string, 0, 2)
			if sourceGap {
				blockers = append(blockers, "SOURCE_MAPPING_REQUIRED")
			}
			if externalUnresolved {
				blockers = append(blockers, "PROVIDER_APPLICABILITY_UNRESOLVED")
			}
			sort.Strings(blockers)
			governance := agaapplicability.GovernanceState{SourceMappingState: question.SourceMappingState, SourceAuthorityState: question.SourceAuthorityState, RiskClassificationState: question.RiskClassificationState, DecisionState: decisionState, ExtractionState: question.ExtractionState, QuestionSourceProposalGap: sourceGap, ExternalApplicabilityUnresolved: externalUnresolved, BlockerCodes: blockers}
			facts := agaapplicability.EvidenceFacts{
				"QUESTION_BODY_DIGEST":    {{Digest: rawTextDigest(question.OriginalText)}},
				"FORM_METADATA_DIGEST":    {{Digest: formMetadataDigest(form.FormKind, form.ProposedRisk.Band)}},
				"RESEARCH_ROW_DIGEST":     {{Digest: researchRowDigest(researchFacts)}},
				"SOURCE_PROPOSAL_DIGEST":  {{Digest: sourceProposalSetDigest(sourceProposalDigests)}},
				"SOURCE_REFERENCE_DIGEST": {{Digest: sourceReferenceSetDigest(sourceReferenceDigests)}},
			}
			key := identity.Key()
			if _, duplicate := items[key]; duplicate {
				return nil, errCandidateInvalid
			}
			items[key] = sourceItemMaterial{InputItem: inputItem, Governance: governance, EvidenceFacts: facts}
		}
	}
	if len(items) != agaapplicability.FrozenBaseQuestionCount {
		return nil, errCandidateInvalid
	}
	return items, nil
}

func rawTextDigest(value string) string {
	return digestBytes([]byte(value))
}

func formMetadataDigest(formKind, formRiskBand string) string {
	digest, _ := agaapplicability.DigestValue("AGA-FORM-METADATA-FACT-V1", map[string]any{"formKind": formKind, "formRiskBand": formRiskBand})
	return digest
}

func sourceProposalSetDigest(values []string) string {
	digest, _ := agaapplicability.DigestValue("AGA-SOURCE-PROPOSAL-SET-V1", values)
	return digest
}

func sourceReferenceSetDigest(values []string) string {
	digest, _ := agaapplicability.DigestValue("AGA-SOURCE-REFERENCE-SET-V1", values)
	return digest
}

func researchRowDigest(facts agaapplicability.ClassificationResearchCandidateFacts) string {
	digest, _ := agaapplicability.DigestValue("AGA-RESEARCH-ROW-FACT-V1", facts)
	return digest
}

func buildPassInputs(bundle fixedClassificationInputs, receipt agaapplicability.PassSealReceipt, role agaapplicability.PassRole) []agaapplicability.ClassificationPassInput {
	inputs := make([]agaapplicability.ClassificationPassInput, 0, len(bundle.Batches))
	for ordinal, identities := range bundle.Batches {
		items := make([]agaapplicability.ClassificationPassInputItem, 0, len(identities))
		for _, identity := range identities {
			items = append(items, bundle.Items[identity.Key()].InputItem)
		}
		inputs = append(inputs, agaapplicability.ClassificationPassInput{SchemaVersion: "aga-hybrid-classification-pass-input/v1", Purpose: "ROW_CLASSIFICATION_PRIVATE_INPUT", ClassificationRunID: receipt.ClassificationRunID, PassRole: role, PassRunID: receipt.PassRunID, BatchOrdinal: ordinal + 1, TaxonomyVersion: agaapplicability.FrozenTaxonomy().Version, TaxonomyDigest: agaapplicability.FrozenTaxonomy().Digest, PromptDigest: receipt.PromptDigest, ModelDescriptorDigest: receipt.ModelDescriptorDigest, BatchManifestDigest: receipt.BatchManifestDigest, FixedInputDigests: bundle.Fixed, Items: items})
	}
	return inputs
}

type reconciliationDetails struct {
	CandidateModel      agaapplicability.ModelDescriptor
	ChallengeModel      agaapplicability.ModelDescriptor
	CandidateValidation privateValidationReceipt
	ChallengeValidation privateValidationReceipt
}

func reconcileValidatedPasses(bundle fixedClassificationInputs, candidate, challenge passArchiveValidation, finalRunID string) (agaapplicability.ClassificationResult, reconciliationDetails, error) {
	candidateReceipt := toDomainPassSealReceipt(candidate.Receipt)
	challengeReceipt := toDomainPassSealReceipt(challenge.Receipt)
	if candidateReceipt.PassRole != agaapplicability.PassCandidate || challengeReceipt.PassRole != agaapplicability.PassChallenge || candidateReceipt.ClassificationRunID == challengeReceipt.ClassificationRunID || candidateReceipt.PromptDigest != challengeReceipt.PromptDigest || candidateReceipt.BatchManifestDigest != challengeReceipt.BatchManifestDigest {
		return agaapplicability.ClassificationResult{}, reconciliationDetails{}, errCandidateInvalid
	}
	candidateModel := modelDescriptorFromMetadata(candidate.Metadata)
	challengeModel := modelDescriptorFromMetadata(challenge.Metadata)
	if finalRunID == "" {
		seed := strings.Join([]string{candidateReceipt.PassSealDigest, challengeReceipt.PassSealDigest, candidateReceipt.PromptDigest}, "\x00")
		digest := sha256.Sum256([]byte(seed))
		finalRunID = "aga-classification-run-reconciliation-" + hex.EncodeToString(digest[:16])
	}
	fixed := bundle.Fixed
	runInputDigest := agaapplicability.ComputeRunInputDigestForPrompt(fixed, candidateReceipt.PromptDigest)
	input := agaapplicability.ClassificationInput{ClassificationRunID: finalRunID, CandidateClassificationRunID: candidateReceipt.ClassificationRunID, ChallengeClassificationRunID: challengeReceipt.ClassificationRunID, RunInputDigest: runInputDigest, PromptDigest: candidateReceipt.PromptDigest, TaxonomyDigest: agaapplicability.FrozenTaxonomy().Digest, FixedInputDigests: fixed, ModelDescriptors: []agaapplicability.ModelDescriptor{candidateModel, challengeModel}, SuppliedModelDescriptorDigests: []string{candidateReceipt.ModelDescriptorDigest, challengeReceipt.ModelDescriptorDigest}, AcceptSuppliedPassInputDigests: true, BatchManifestDigest: agaapplicability.FrozenBatchManifestDigest, PassOneSealDigest: candidateReceipt.PassSealDigest, PassTwoSealDigest: challengeReceipt.PassSealDigest, PassOneSealReceipt: candidateReceipt, PassTwoSealReceipt: challengeReceipt, OrderedBaseIdentities: bundle.Ordered, PassInputsByRole: map[agaapplicability.PassRole][]agaapplicability.ClassificationPassInput{agaapplicability.PassCandidate: buildPassInputs(bundle, candidateReceipt, agaapplicability.PassCandidate), agaapplicability.PassChallenge: buildPassInputs(bundle, challengeReceipt, agaapplicability.PassChallenge)}, CandidateRecords: candidate.Records, ChallengeRecords: challenge.Records, GovernanceByIdentity: make(map[string]agaapplicability.GovernanceState, len(bundle.Items)), EvidenceFactsByIdentity: make(map[string]agaapplicability.EvidenceFacts, len(bundle.Items))}
	for key, material := range bundle.Items {
		input.GovernanceByIdentity[key] = material.Governance
		input.EvidenceFactsByIdentity[key] = material.EvidenceFacts
	}
	result, err := agaapplicability.ReconcileClassification(input)
	if err != nil {
		return agaapplicability.ClassificationResult{}, reconciliationDetails{}, err
	}
	if result.State != "SEALED" || result.Aggregate.ItemCount != agaapplicability.FrozenBaseQuestionCount || len(result.Items) != agaapplicability.FrozenBaseQuestionCount {
		return agaapplicability.ClassificationResult{}, reconciliationDetails{}, errCandidateInvalid
	}
	return result, reconciliationDetails{CandidateModel: candidateModel, ChallengeModel: challengeModel}, nil
}

func modelDescriptorFromMetadata(metadata chatMetadata) agaapplicability.ModelDescriptor {
	source := "platform-unavailable"
	if metadata.ModelID != nil {
		source = "platform-reported-exact"
	}
	return agaapplicability.ModelDescriptor{ModelID: metadata.ModelID, ModelIDSource: source, DisplayedModelLabel: metadata.DisplayedModelLabel, Service: metadata.Service, Interface: metadata.Interface, RequestedReasoningEffort: metadata.RequestedReasoningEffort, ForkTurns: metadata.ForkTurns, SnapshotBuildLabel: metadata.SnapshotBuildLabel, UnavailableFields: append([]string(nil), metadata.UnavailableFields...)}
}

func toDomainPassSealReceipt(receipt passArchiveReceipt) agaapplicability.PassSealReceipt {
	return agaapplicability.PassSealReceipt{ClassificationRunID: receipt.ClassificationRunID, PassRole: agaapplicability.PassRole(receipt.PassRole), PassRunID: receipt.PassRunID, PromptDigest: receipt.PromptDigest, ModelDescriptorDigest: receipt.ModelDescriptorDigest, BatchManifestDigest: receipt.BatchManifestDigest, BatchCount: receipt.BatchCount, ItemCount: receipt.ItemCount, OrderedInputDigests: append([]string(nil), receipt.OrderedInputDigests...), PassInputSetDigest: receipt.PassInputSetDigest, OrderedBatchOutputDigests: append([]string(nil), receipt.OrderedBatchOutputDigests...), OrderedPassResultDigests: append([]string(nil), receipt.OrderedPassResultDigests...), PassSealDigest: receipt.PassSealDigest}
}

func writeCandidateArtifacts(directory string, bundle fixedClassificationInputs, result agaapplicability.ClassificationResult, details reconciliationDetails, candidateArchive, challengeArchive sourceArchiveRef, candidatePrivate, challengePrivate privateValidationReceipt) error {
	if directory == "" {
		return errCandidateInvalid
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return errCandidateInvalid
	}
	if err := writeRawArtifact(filepath.Join(directory, "batch-manifest.json"), bundle.ManifestRaw); err != nil {
		return err
	}
	passOneRun := passRunArtifact{SchemaVersion: passRunArtifactSchema, Status: "SEALED", PassRole: "CANDIDATE", ClassificationRunID: result.PassOneSealReceipt.ClassificationRunID, PassRunID: result.PassOneSealReceipt.PassRunID, PromptDigest: result.PassOneSealReceipt.PromptDigest, ModelDescriptorDigest: result.PassOneSealReceipt.ModelDescriptorDigest, BatchManifestDigest: result.PassOneSealReceipt.BatchManifestDigest, BatchCount: result.PassOneSealReceipt.BatchCount, ItemCount: result.PassOneSealReceipt.ItemCount, ModelDescriptor: details.CandidateModel, SourceArchive: candidateArchive, PrivateValidation: candidatePrivate, PassSealReceipt: result.PassOneSealReceipt}
	passTwoRun := passRunArtifact{SchemaVersion: passRunArtifactSchema, Status: "SEALED", PassRole: "CHALLENGE", ClassificationRunID: result.PassTwoSealReceipt.ClassificationRunID, PassRunID: result.PassTwoSealReceipt.PassRunID, PromptDigest: result.PassTwoSealReceipt.PromptDigest, ModelDescriptorDigest: result.PassTwoSealReceipt.ModelDescriptorDigest, BatchManifestDigest: result.PassTwoSealReceipt.BatchManifestDigest, BatchCount: result.PassTwoSealReceipt.BatchCount, ItemCount: result.PassTwoSealReceipt.ItemCount, ModelDescriptor: details.ChallengeModel, SourceArchive: challengeArchive, PrivateValidation: challengePrivate, PassSealReceipt: result.PassTwoSealReceipt}
	if err := writeJSONArtifact(filepath.Join(directory, "pass-one-run.json"), passOneRun); err != nil {
		return err
	}
	if err := writeJSONArtifact(filepath.Join(directory, "pass-two-run.json"), passTwoRun); err != nil {
		return err
	}
	passOneResults := passResultsArtifact{SchemaVersion: passResultsArtifactSchema, Status: "SEALED", PassRole: "CANDIDATE", ClassificationRunID: result.PassOneSealReceipt.ClassificationRunID, PassRunID: result.PassOneSealReceipt.PassRunID, PromptDigest: result.PassOneSealReceipt.PromptDigest, ModelDescriptorDigest: result.PassOneSealReceipt.ModelDescriptorDigest, ItemCount: len(result.CandidateRecords), Records: result.CandidateRecords}
	passTwoResults := passResultsArtifact{SchemaVersion: passResultsArtifactSchema, Status: "SEALED", PassRole: "CHALLENGE", ClassificationRunID: result.PassTwoSealReceipt.ClassificationRunID, PassRunID: result.PassTwoSealReceipt.PassRunID, PromptDigest: result.PassTwoSealReceipt.PromptDigest, ModelDescriptorDigest: result.PassTwoSealReceipt.ModelDescriptorDigest, ItemCount: len(result.ChallengeRecords), Records: result.ChallengeRecords}
	if err := writeJSONArtifact(filepath.Join(directory, "pass-one-results.json"), passOneResults); err != nil {
		return err
	}
	if err := writeJSONArtifact(filepath.Join(directory, "pass-two-results.json"), passTwoResults); err != nil {
		return err
	}
	if err := writeJSONArtifact(filepath.Join(directory, "reconciliation.json"), result); err != nil {
		return err
	}
	questionArtifact := questionClassificationsArtifact{SchemaVersion: questionClassificationsSchema, Status: "SEALED", ClassificationRunID: result.ClassificationRunID, PromptDigest: result.RunReceipt.PromptDigest, ItemCount: len(result.Items), Items: result.Items}
	if err := writeJSONArtifact(filepath.Join(directory, "question-classifications.json"), questionArtifact); err != nil {
		return err
	}
	if err := writeClassificationCSV(filepath.Join(directory, "question-classifications.csv"), result.Items); err != nil {
		return err
	}
	if err := writeAmbiguityCSV(filepath.Join(directory, "ambiguity-review.csv"), result.Items); err != nil {
		return err
	}
	if err := writeJSONArtifact(filepath.Join(directory, "aggregates.json"), result.Aggregate); err != nil {
		return err
	}
	cleanup := cleanupReceipt{SchemaVersion: cleanupReceiptSchema, Status: "SEALED", Result: "AGA_PASS_VALIDATED", Candidate: cleanupPassReceipt{SourceArchiveSHA256: candidateArchive.SHA256, SemanticEntries: candidatePrivate.SemanticEntries, TransportNoise: candidatePrivate.TransportNoise, ExpandedBytes: candidatePrivate.ExpandedBytes, ReceiptDigest: candidatePrivate.Digest, BatchCount: result.PassOneSealReceipt.BatchCount, RecordCount: len(result.CandidateRecords), PrivateRootRemoved: candidatePrivate.RootRemoved}, Challenge: cleanupPassReceipt{SourceArchiveSHA256: challengeArchive.SHA256, SemanticEntries: challengePrivate.SemanticEntries, TransportNoise: challengePrivate.TransportNoise, ExpandedBytes: challengePrivate.ExpandedBytes, ReceiptDigest: challengePrivate.Digest, BatchCount: result.PassTwoSealReceipt.BatchCount, RecordCount: len(result.ChallengeRecords), PrivateRootRemoved: challengePrivate.RootRemoved}, PlatformMetadataAvailability: cleanupPlatformMetadata{Policy: "TRUTHFUL_UNAVAILABLE_ACCEPTED_FOR_CANDIDATE_ONLY_DEMO", Candidate: append([]string(nil), details.CandidateModel.UnavailableFields...), Challenge: append([]string(nil), details.ChallengeModel.UnavailableFields...)}}
	cleanup.Cleanup.PrivateRootRemoved = candidatePrivate.RootRemoved && challengePrivate.RootRemoved
	if err := writeJSONArtifact(filepath.Join(directory, "pass-isolation-cleanup.json"), cleanup); err != nil {
		return err
	}
	return writeCandidateManifest(directory, bundle, result, candidateArchive, challengeArchive)
}

func writeCandidateManifest(directory string, bundle fixedClassificationInputs, result agaapplicability.ClassificationResult, candidateArchive, challengeArchive sourceArchiveRef) error {
	filenames := []string{"aggregates.json", "ambiguity-review.csv", "batch-manifest.json", "pass-isolation-cleanup.json", "pass-one-results.json", "pass-one-run.json", "pass-two-results.json", "pass-two-run.json", "question-classifications.csv", "question-classifications.json", "reconciliation.json"}
	files := make([]candidateFileDigest, 0, len(filenames))
	for _, filename := range filenames {
		fullPath := filepath.Join(directory, filename)
		info, err := os.Stat(fullPath)
		if err != nil || !info.Mode().IsRegular() {
			return errCandidateInvalid
		}
		digest, err := sha256File(fullPath)
		if err != nil {
			return errCandidateInvalid
		}
		files = append(files, candidateFileDigest{Filename: filename, ByteLength: info.Size(), SHA256: digest})
	}
	manifest := candidateManifest{SchemaVersion: candidateManifestSchema, Status: "SEALED", CandidateOnly: true, PackageVersion: agaapplicability.FrozenPackageVersion, PackageJSONSHA256: agaapplicability.FrozenPackageJSONSHA256, ClassificationRunID: result.ClassificationRunID, PromptDigest: result.RunReceipt.PromptDigest, TaxonomyVersion: result.RunReceipt.TaxonomyVersion, TaxonomyDigest: result.RunReceipt.TaxonomyDigest, BatchManifestDigest: bundle.Manifest.ManifestDigest, InputDigest: result.InputDigest, BatchCount: result.PassOneSealReceipt.BatchCount, ItemCount: len(result.Items), PassProposalRecordCount: len(result.CandidateRecords) + len(result.ChallengeRecords), AggregateDigest: result.AggregateDigest, ClassificationRunDigest: result.ClassificationRunDigest, PassOneSealDigest: result.PassOneSealReceipt.PassSealDigest, PassTwoSealDigest: result.PassTwoSealReceipt.PassSealDigest, CandidateSource: candidateArchive, ChallengeSource: challengeArchive, Files: files}
	manifest.ManifestDigest = candidateManifestDigest(manifest)
	return writeJSONArtifact(filepath.Join(directory, "manifest.json"), manifest)
}

func candidateManifestDigest(manifest candidateManifest) string {
	manifest.ManifestDigest = ""
	data, _ := json.Marshal(manifest)
	var object map[string]any
	_ = json.Unmarshal(data, &object)
	delete(object, "manifestDigest")
	digest, _ := agaapplicability.DigestValue("AGA-CLASSIFICATION-CANDIDATE-MANIFEST-V1", object)
	return digest
}

func writeJSONArtifact(filename string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errCandidateInvalid
	}
	return writeRawArtifact(filename, data)
}

func writeRawArtifact(filename string, data []byte) error {
	if len(data) == 0 {
		return errCandidateInvalid
	}
	return os.WriteFile(filename, data, 0644)
}

func writeClassificationCSV(filename string, items []agaapplicability.SealedClassificationItem) error {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	header := []string{"packageVersion", "packageJsonSha256", "formCode", "proposalId", "ordinal", "textDigest", "mainDomainCode", "topicCodes", "inspectionProfileCodes", "inspectionTypeCodes", "canonicalTargetKind", "targetProfileCode", "operationQualifiers", "activityQualifiers", "applicabilityDisposition", "evidenceExpectationCodes", "externalInvolvements", "agreementConfidence", "recommendationState", "rationaleCodes", "confidenceEvidence", "sourceRefs", "sourceMappingState", "sourceAuthorityState", "riskClassificationState", "decisionState", "extractionState", "questionSourceProposalGap", "externalApplicabilityUnresolved", "passDisagreementCodes", "passOneResultDigest", "passTwoResultDigest", "passOneRunId", "passTwoRunId", "promptDigest", "modelDescriptorDigests", "taxonomyDigest", "inputDigest", "itemSemanticDigest", "classificationRunDigest", "aggregateDigest"}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, item := range items {
		identity := item.Identity
		row := []string{identity.PackageVersion, identity.PackageJSONSHA256, identity.FormCode, identity.ProposalID, strconv.Itoa(identity.Ordinal), identity.TextDigest, item.Projection.MainDomainCode, jsonCell(item.Projection.TopicCodes), jsonCell(item.Projection.InspectionProfileCodes), jsonCell(item.Projection.InspectionTypeCodes), item.Projection.CanonicalTargetKind, item.Projection.TargetProfileCode, jsonCell(item.Projection.OperationQualifiers), jsonCell(item.Projection.ActivityQualifiers), item.Projection.ApplicabilityDisposition, jsonCell(item.Projection.EvidenceExpectationCodes), jsonCell(item.Projection.ExternalInvolvements), string(item.AgreementConfidence), item.RecommendationState, jsonCell(item.RationaleCodes), jsonCell(item.ConfidenceEvidence), jsonCell(item.SourceRefs), item.SourceMappingState, item.SourceAuthorityState, item.RiskClassificationState, item.DecisionState, item.ExtractionState, strconv.FormatBool(item.QuestionSourceProposalGap), strconv.FormatBool(item.ExternalApplicabilityUnresolved), jsonCell(item.PassDisagreementCodes), item.PassOneResultDigest, item.PassTwoResultDigest, item.PassOneRunID, item.PassTwoRunID, item.PromptDigest, jsonCell(item.ModelDescriptorDigests), item.TaxonomyDigest, item.InputDigest, item.ItemSemanticDigest, item.ClassificationRunDigest, item.AggregateDigest}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return writeRawArtifact(filename, buffer.Bytes())
}

func writeAmbiguityCSV(filename string, items []agaapplicability.SealedClassificationItem) error {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"packageVersion", "packageJsonSha256", "formCode", "proposalId", "ordinal", "textDigest", "agreementConfidence", "recommendationState", "questionSourceProposalGap", "externalApplicabilityUnresolved", "passDisagreementCodes", "blockerCodes", "reviewReason"}); err != nil {
		return err
	}
	for _, item := range items {
		if !item.QuestionSourceProposalGap && !item.ExternalApplicabilityUnresolved && len(item.PassDisagreementCodes) == 0 && item.RecommendationState == agaapplicability.RecommendationAutoProposed {
			continue
		}
		reason := "PASS_DISAGREEMENT_OR_GOVERNANCE_EXCEPTION"
		if item.RecommendationState == agaapplicability.RecommendationBlockedSourceGap {
			reason = "SOURCE_MAPPING_REQUIRED"
		} else if item.ExternalApplicabilityUnresolved {
			reason = "PROVIDER_APPLICABILITY_UNRESOLVED"
		}
		identity := item.Identity
		row := []string{identity.PackageVersion, identity.PackageJSONSHA256, identity.FormCode, identity.ProposalID, strconv.Itoa(identity.Ordinal), identity.TextDigest, string(item.AgreementConfidence), item.RecommendationState, strconv.FormatBool(item.QuestionSourceProposalGap), strconv.FormatBool(item.ExternalApplicabilityUnresolved), jsonCell(item.PassDisagreementCodes), jsonCell(item.BlockerCodes), reason}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return writeRawArtifact(filename, buffer.Bytes())
}

func jsonCell(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func validateCandidateArtifacts(directory string, bundle fixedClassificationInputs) error {
	manifestBytes, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return errCandidateInvalid
	}
	var manifest candidateManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil || manifest.SchemaVersion != candidateManifestSchema || manifest.Status != "SEALED" || !manifest.CandidateOnly || manifest.ManifestDigest != candidateManifestDigest(manifest) {
		return errCandidateInvalid
	}
	if manifest.PackageVersion != agaapplicability.FrozenPackageVersion || manifest.PackageJSONSHA256 != agaapplicability.FrozenPackageJSONSHA256 || manifest.BatchManifestDigest != bundle.Manifest.ManifestDigest || manifest.BatchCount != 25 || manifest.ItemCount != agaapplicability.FrozenBaseQuestionCount || manifest.PassProposalRecordCount != agaapplicability.FrozenPassProposalRecordCount {
		return errCandidateInvalid
	}
	expectedFiles := map[string]struct{}{"manifest.json": {}, "batch-manifest.json": {}, "pass-one-run.json": {}, "pass-one-results.json": {}, "pass-two-run.json": {}, "pass-two-results.json": {}, "reconciliation.json": {}, "question-classifications.json": {}, "question-classifications.csv": {}, "ambiguity-review.csv": {}, "aggregates.json": {}, "pass-isolation-cleanup.json": {}}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != len(expectedFiles) {
		return errCandidateInvalid
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return errCandidateInvalid
		}
		if _, exists := expectedFiles[entry.Name()]; !exists {
			return errCandidateInvalid
		}
	}
	if len(manifest.Files) != len(expectedFiles)-1 {
		return errCandidateInvalid
	}
	seenFiles := make(map[string]struct{}, len(manifest.Files))
	for _, file := range manifest.Files {
		if file.Filename == "manifest.json" {
			return errCandidateInvalid
		}
		if _, duplicate := seenFiles[file.Filename]; duplicate {
			return errCandidateInvalid
		}
		seenFiles[file.Filename] = struct{}{}
		info, err := os.Stat(filepath.Join(directory, file.Filename))
		if err != nil || info.Size() != file.ByteLength {
			return errCandidateInvalid
		}
		digest, err := sha256File(filepath.Join(directory, file.Filename))
		if err != nil || digest != file.SHA256 {
			return errCandidateInvalid
		}
	}
	manifestCopy, err := os.ReadFile(filepath.Join(directory, "batch-manifest.json"))
	if err != nil || !bytes.Equal(manifestCopy, bundle.ManifestRaw) {
		return errCandidateInvalid
	}
	var passOneRun, passTwoRun passRunArtifact
	if err := readJSONArtifact(filepath.Join(directory, "pass-one-run.json"), &passOneRun); err != nil {
		return errCandidateInvalid
	}
	if err := readJSONArtifact(filepath.Join(directory, "pass-two-run.json"), &passTwoRun); err != nil {
		return errCandidateInvalid
	}
	var passOneResults, passTwoResults passResultsArtifact
	if err := readJSONArtifact(filepath.Join(directory, "pass-one-results.json"), &passOneResults); err != nil {
		return errCandidateInvalid
	}
	if err := readJSONArtifact(filepath.Join(directory, "pass-two-results.json"), &passTwoResults); err != nil {
		return errCandidateInvalid
	}
	if passOneRun.Status != "SEALED" || passTwoRun.Status != "SEALED" || passOneResults.Status != "SEALED" || passTwoResults.Status != "SEALED" || passOneRun.PassRole != "CANDIDATE" || passTwoRun.PassRole != "CHALLENGE" || passOneResults.PassRole != "CANDIDATE" || passTwoResults.PassRole != "CHALLENGE" || len(passOneResults.Records) != agaapplicability.FrozenBaseQuestionCount || len(passTwoResults.Records) != agaapplicability.FrozenBaseQuestionCount {
		return errCandidateInvalid
	}
	candidateValidation := passArchiveValidation{BatchCount: passOneRun.PassSealReceipt.BatchCount, RecordCount: len(passOneResults.Records), Receipt: passArchiveReceiptFromDomain(passOneRun.PassSealReceipt), Records: passOneResults.Records, Metadata: metadataFromModelDescriptor(passOneRun.ModelDescriptor, passOneRun.MetadataAcceptanceStatus)}
	challengeValidation := passArchiveValidation{BatchCount: passTwoRun.PassSealReceipt.BatchCount, RecordCount: len(passTwoResults.Records), Receipt: passArchiveReceiptFromDomain(passTwoRun.PassSealReceipt), Records: passTwoResults.Records, Metadata: metadataFromModelDescriptor(passTwoRun.ModelDescriptor, passTwoRun.MetadataAcceptanceStatus)}
	var result agaapplicability.ClassificationResult
	reconciliationBytes, err := os.ReadFile(filepath.Join(directory, "reconciliation.json"))
	if err != nil || json.Unmarshal(reconciliationBytes, &result) != nil {
		return errCandidateInvalid
	}
	rebuilt, _, err := reconcileValidatedPasses(bundle, candidateValidation, challengeValidation, result.ClassificationRunID)
	if err != nil {
		return errCandidateInvalid
	}
	rebuiltBytes, err := json.Marshal(rebuilt)
	if err != nil || !bytes.Equal(rebuiltBytes, reconciliationBytes) {
		return errCandidateInvalid
	}
	var questions questionClassificationsArtifact
	if err := readJSONArtifact(filepath.Join(directory, "question-classifications.json"), &questions); err != nil || questions.Status != "SEALED" || len(questions.Items) != len(result.Items) || !reflect.DeepEqual(questions.Items, result.Items) {
		return errCandidateInvalid
	}
	var aggregate agaapplicability.ClassificationAggregate
	if err := readJSONArtifact(filepath.Join(directory, "aggregates.json"), &aggregate); err != nil || !reflect.DeepEqual(aggregate, result.Aggregate) {
		return errCandidateInvalid
	}
	var cleanup cleanupReceipt
	if err := readJSONArtifact(filepath.Join(directory, "pass-isolation-cleanup.json"), &cleanup); err != nil || cleanup.Status != "SEALED" || cleanup.Result != "AGA_PASS_VALIDATED" || !cleanup.Cleanup.PrivateRootRemoved || cleanup.Cleanup.FilesRemaining != 0 || cleanup.Cleanup.DirectoriesRemaining != 0 || cleanup.Cleanup.ProcessesRemaining != 0 {
		return errCandidateInvalid
	}
	if manifest.ClassificationRunID != result.ClassificationRunID || manifest.InputDigest != result.InputDigest || manifest.AggregateDigest != result.AggregateDigest || manifest.ClassificationRunDigest != result.ClassificationRunDigest || manifest.PassOneSealDigest != result.PassOneSealReceipt.PassSealDigest || manifest.PassTwoSealDigest != result.PassTwoSealReceipt.PassSealDigest {
		return errCandidateInvalid
	}
	return nil
}

func readJSONArtifact(filename string, target any) error {
	data, err := os.ReadFile(filename)
	if err != nil || len(data) == 0 {
		return errCandidateInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errCandidateInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errCandidateInvalid
	}
	return nil
}

func passArchiveReceiptFromDomain(receipt agaapplicability.PassSealReceipt) passArchiveReceipt {
	return passArchiveReceipt{ClassificationRunID: receipt.ClassificationRunID, PassRole: string(receipt.PassRole), PassRunID: receipt.PassRunID, PromptDigest: receipt.PromptDigest, ModelDescriptorDigest: receipt.ModelDescriptorDigest, BatchManifestDigest: receipt.BatchManifestDigest, BatchCount: receipt.BatchCount, ItemCount: receipt.ItemCount, OrderedInputDigests: receipt.OrderedInputDigests, PassInputSetDigest: receipt.PassInputSetDigest, OrderedBatchOutputDigests: receipt.OrderedBatchOutputDigests, OrderedPassResultDigests: receipt.OrderedPassResultDigests, PassSealDigest: receipt.PassSealDigest}
}

func metadataFromModelDescriptor(descriptor agaapplicability.ModelDescriptor, status *string) chatMetadata {
	return chatMetadata{ModelID: descriptor.ModelID, Service: descriptor.Service, Interface: descriptor.Interface, SnapshotBuildLabel: descriptor.SnapshotBuildLabel, DisplayedModelLabel: descriptor.DisplayedModelLabel, RequestedReasoningEffort: descriptor.RequestedReasoningEffort, ForkTurns: descriptor.ForkTurns, UnavailableFields: append([]string(nil), descriptor.UnavailableFields...), MetadataAcceptanceStatus: status}
}
