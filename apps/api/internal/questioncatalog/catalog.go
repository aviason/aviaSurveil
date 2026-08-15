// Package questioncatalog contains the small, canonical contract used by the
// preprod catalog importer and the planning-owned question selector.  It is
// deliberately independent from the AGA stakeholder workspace: catalog rows
// reference an immutable question_versions identity, while the catalog owns
// only lineage/selection metadata.
package questioncatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// UsageClass identifies the operational authority boundary of a catalog.
type UsageClass string

const (
	UsageClassGovernedOperational UsageClass = "GOVERNED_OPERATIONAL"
)

// SourceOrigin records why an operational catalog is available. It is
// intentionally separate from UsageClass: the imported source classification
// is not an internal Manager approval or publication event.
type SourceOrigin string

const (
	SourceOriginImportedApproved  SourceOrigin = "IMPORTED_APPROVED_SOURCE"
	SourceOriginInternalCandidate SourceOrigin = "INTERNAL_GENERATED_CANDIDATE"
)

// ImportRow is the import/provenance projection for one immutable
// question_versions row.  Body is accepted only so ValidateImport can fail
// closed if a caller attempts to introduce a second question-body authority;
// valid catalog rows leave it empty.
type ImportRow struct {
	CatalogVersion    string
	FormCode          string
	ProposalID        string
	Ordinal           int
	QuestionVersionID string
	QuestionDigest    string
	UsageClass        UsageClass
	SourceOrigin      SourceOrigin
	Body              string
}

// ImportPolicy describes the exact shape expected from a sealed package.  A
// zero value leaves that particular count unconstrained, which keeps the
// validator useful for bounded unit fixtures while callers importing the AGA
// profile pass 1,310 and 52 explicitly.
type ImportPolicy struct {
	ExpectedRows  int
	ExpectedForms int
}

var (
	errEmptyImport        = errors.New("question catalog import is empty")
	errCopiedQuestionBody = errors.New("question body must remain in question_versions")
	errDuplicateIdentity  = errors.New("duplicate catalog question identity")
	errDuplicateVersion   = errors.New("duplicate question version id")
	errInvalidSelection   = errors.New("invalid question selection")
	errSelectionCAS       = errors.New("question selection revision is stale")
	errSelectionNotFound  = errors.New("selected question version is not in catalog")
	errUsageClassMismatch = errors.New("catalog usage class mismatch")
)

// ValidateImport checks the importer boundary before any durable write.  It
// validates identity/lineage fields, uniqueness, and the no-body-copy rule;
// callers should run it before inserting question_versions and membership
// rows in one transaction.
func ValidateImport(rows []ImportRow, policy ImportPolicy) error {
	if len(rows) == 0 {
		return errEmptyImport
	}
	if policy.ExpectedRows > 0 && len(rows) != policy.ExpectedRows {
		return fmt.Errorf("expected %d import rows, got %d", policy.ExpectedRows, len(rows))
	}
	identitySeen := make(map[string]struct{}, len(rows))
	versionSeen := make(map[string]struct{}, len(rows))
	forms := make(map[string]struct{})
	for i, row := range rows {
		if strings.TrimSpace(row.CatalogVersion) == "" {
			return fmt.Errorf("row %d: catalog version is required", i)
		}
		if strings.TrimSpace(row.FormCode) == "" || strings.TrimSpace(row.ProposalID) == "" {
			return fmt.Errorf("row %d: form and proposal identity are required", i)
		}
		if row.Ordinal < 1 {
			return fmt.Errorf("row %d: ordinal must be positive", i)
		}
		if strings.TrimSpace(row.QuestionVersionID) == "" || strings.TrimSpace(row.QuestionDigest) == "" {
			return fmt.Errorf("row %d: question version and digest are required", i)
		}
		if row.Body != "" {
			return fmt.Errorf("row %d: %w", i, errCopiedQuestionBody)
		}
		identity := importIdentity(row)
		if _, ok := identitySeen[identity]; ok {
			return fmt.Errorf("row %d: %w: %s", i, errDuplicateIdentity, identity)
		}
		identitySeen[identity] = struct{}{}
		if _, ok := versionSeen[row.QuestionVersionID]; ok {
			return fmt.Errorf("row %d: %w: %s", i, errDuplicateVersion, row.QuestionVersionID)
		}
		versionSeen[row.QuestionVersionID] = struct{}{}
		forms[row.FormCode] = struct{}{}
	}
	if policy.ExpectedForms > 0 && len(forms) != policy.ExpectedForms {
		return fmt.Errorf("expected %d import forms, got %d", policy.ExpectedForms, len(forms))
	}
	return nil
}

func importIdentity(row ImportRow) string {
	return row.CatalogVersion + "\x00" + row.FormCode + "\x00" + row.ProposalID + "\x00" + fmt.Sprintf("%d", row.Ordinal)
}

// ImportDigest returns a deterministic aggregate digest over import lineage.
// Import packages are canonicalized by their immutable source identity, so
// retries cannot create a different catalog identity merely by returning rows
// in another order.
func ImportDigest(rows []ImportRow) string {
	canonical := append([]ImportRow(nil), rows...)
	sort.Slice(canonical, func(i, j int) bool {
		left, right := importIdentity(canonical[i]), importIdentity(canonical[j])
		if left != right {
			return left < right
		}
		return canonical[i].QuestionVersionID < canonical[j].QuestionVersionID
	})
	h := sha256.New()
	for _, row := range canonical {
		// Include every immutable lineage fact and usage boundary.  Delimiters
		// avoid ambiguous concatenations while keeping the digest portable.
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00%d\x00%s\x00%s\x00%s\x00%s\n",
			row.CatalogVersion, row.FormCode, row.ProposalID, row.Ordinal,
			row.QuestionVersionID, row.QuestionDigest, row.UsageClass, row.SourceOrigin)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// CatalogPolicy fixes the usage boundary for an in-memory catalog instance.
// A production importer should enforce the same boundary in the database
// profile and foreign keys; this type provides the command-level guard.
type CatalogPolicy struct {
	UsageClass   UsageClass
	SourceOrigin SourceOrigin
}

// SelectionRequest carries the desired exact selected set.  ExpectedDigest is
// the digest of the current set (CAS); an empty value is valid only for a new
// empty catalog.  MaxBatch bounds preview/commit work and defaults to 500.
type SelectionRequest struct {
	QuestionVersionIDs []string
	ExpectedDigest     string
	MaxBatch           int
}

// SelectionPreview is a server-derived, bounded preview.  The digest covers
// the proposed ordered set and is suitable for the UI to display; callers
// still provide ExpectedDigest on commit to protect the current revision.
type SelectionPreview struct {
	QuestionVersionIDs []string
	Count              int
	SelectionDigest    string
}

// SelectionReceipt is returned after an atomic selection commit.
type SelectionReceipt struct {
	QuestionVersionIDs []string
	Count              int
	SelectionDigest    string
}

// Catalog is a concurrency-safe, append-only catalog projection with a CAS
// protected selected set.  Persistence adapters can use the same validation,
// digest, and command semantics around a database transaction.
type Catalog struct {
	mu       sync.RWMutex
	policy   CatalogPolicy
	rows     map[string]ImportRow
	ordered  []string
	selected []string
}

// NewCatalog validates rows and creates a catalog projection.  It does not
// copy question bodies; only lineage rows and immutable question-version IDs
// are retained.
func NewCatalog(rows []ImportRow, policy CatalogPolicy) (*Catalog, error) {
	if err := ValidateImport(rows, ImportPolicy{}); err != nil {
		return nil, err
	}
	if policy.UsageClass == "" {
		return nil, fmt.Errorf("%w: usage class is required", errUsageClassMismatch)
	}
	byID := make(map[string]ImportRow, len(rows))
	ordered := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.UsageClass != policy.UsageClass {
			return nil, fmt.Errorf("%w: want %s, got %s", errUsageClassMismatch, policy.UsageClass, row.UsageClass)
		}
		if policy.SourceOrigin != "" && row.SourceOrigin != policy.SourceOrigin {
			return nil, fmt.Errorf("%w: source origin want %s, got %s", errUsageClassMismatch, policy.SourceOrigin, row.SourceOrigin)
		}
		// Keep only the immutable reference/provenance projection.  Body is
		// already rejected above, so this also documents the authority boundary.
		row.Body = ""
		byID[row.QuestionVersionID] = row
		ordered = append(ordered, row.QuestionVersionID)
	}
	sort.Strings(ordered)
	return &Catalog{policy: policy, rows: byID, ordered: ordered}, nil
}

// PreviewSelection validates and digests a proposed exact set without
// changing state.
func (c *Catalog) PreviewSelection(req SelectionRequest) (SelectionPreview, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids, err := normalizeSelection(req.QuestionVersionIDs, req.MaxBatch, c.rows)
	if err != nil {
		return SelectionPreview{}, err
	}
	return SelectionPreview{QuestionVersionIDs: ids, Count: len(ids), SelectionDigest: digestIDs(ids)}, nil
}

// CommitSelection atomically replaces the current selected set after checking
// the caller's current-set digest.  It returns the new exact set/digest.
func (c *Catalog) CommitSelection(req SelectionRequest) (SelectionReceipt, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	currentDigest := digestIDs(c.selected)
	if req.ExpectedDigest != currentDigest {
		return SelectionReceipt{}, fmt.Errorf("%w: expected %s, current %s", errSelectionCAS, req.ExpectedDigest, currentDigest)
	}
	ids, err := normalizeSelection(req.QuestionVersionIDs, req.MaxBatch, c.rows)
	if err != nil {
		return SelectionReceipt{}, err
	}
	c.selected = append([]string(nil), ids...)
	return SelectionReceipt{QuestionVersionIDs: append([]string(nil), ids...), Count: len(ids), SelectionDigest: digestIDs(ids)}, nil
}

// SelectedQuestionVersionIDs returns a defensive copy of the current exact
// selection.
func (c *Catalog) SelectedQuestionVersionIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.selected...)
}

// CurrentSelectionDigest returns the CAS token for the current selected set.
func (c *Catalog) CurrentSelectionDigest() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return digestIDs(c.selected)
}

// Rows returns the immutable lineage projection in deterministic order.
func (c *Catalog) Rows() []ImportRow {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]ImportRow, 0, len(c.ordered))
	for _, id := range c.ordered {
		result = append(result, c.rows[id])
	}
	return result
}

func normalizeSelection(ids []string, maxBatch int, rows map[string]ImportRow) ([]string, error) {
	if maxBatch <= 0 {
		maxBatch = 500
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: at least one question version is required", errInvalidSelection)
	}
	if len(ids) > maxBatch || len(ids) > 500 {
		return nil, fmt.Errorf("%w: selection exceeds 500-question batch limit", errInvalidSelection)
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("%w: question version id is empty", errInvalidSelection)
		}
		if _, ok := rows[id]; !ok {
			return nil, fmt.Errorf("%w: %s", errSelectionNotFound, id)
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("%w: duplicate %s", errInvalidSelection, id)
		}
		seen[id] = struct{}{}
	}
	// Preserve the caller's explicit order. The order is the package position
	// contract and is covered by the selection digest.
	return append([]string(nil), ids...), nil
}

func digestIDs(ids []string) string {
	h := sha256.New()
	for position, id := range ids {
		fmt.Fprintf(h, "%d\x00%s\n", position, id)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// SelectionDigest returns the stable ordered CAS digest used by the HTTP scope
// contract. Selection order is part of the released Audit package contract;
// callers must not sort or otherwise rewrite the ordered identities before
// persisting this digest.
func SelectionDigest(ids []string) string { return digestIDs(ids) }
