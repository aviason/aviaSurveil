package documents

import (
	"errors"
	"time"
)

var (
	ErrForbidden = errors.New("document forbidden")
	ErrNotFound  = errors.New("document not found")
	ErrNotReady  = errors.New("document not ready")
	ErrInvalid   = errors.New("document invalid")
	ErrConflict  = errors.New("document conflict")
)

type JobStatus string

const (
	JobPending   JobStatus = "PENDING"
	JobRunning   JobStatus = "RUNNING"
	JobSucceeded JobStatus = "SUCCEEDED"
	JobFailed    JobStatus = "FAILED"
)

type RenderSnapshot struct {
	ReportVersionID  string   `json:"reportVersionId"`
	ReportID         string   `json:"reportId"`
	Kind             string   `json:"kind"`
	OrganizationID   string   `json:"organizationId"`
	AuditID          string   `json:"auditId"`
	FindingIDs       []string `json:"findingIds"`
	ContentHash      string   `json:"contentHash"`
	Version          int64    `json:"version"`
	CreatedBySubject string   `json:"createdBySubject"`
	// Source is the complete, immutable canonical narrative consumed by the
	// renderer. Keeping it in the job snapshot prevents a later database read
	// or mutable template from changing an already-decided report.
	Source ReportRenderSource `json:"source"`
}

type Download struct {
	DocumentVersionID string
	FileName          string
	MediaType         string
	SHA256            string
	SizeBytes         int64
	URL               string
	ExpiresAt         time.Time
}
