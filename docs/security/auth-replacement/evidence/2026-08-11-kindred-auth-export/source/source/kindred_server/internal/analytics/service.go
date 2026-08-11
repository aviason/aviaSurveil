package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "kindred_server/internal/platform/errors"
)

const defaultMaxBatchSize = 100

type Config struct {
	Enabled       bool
	SchemaVersion int
	Environment   string
	RecordSource  string
	MaxBatchSize  int
}

type Service struct {
	repo      ConsentRepository
	publisher Publisher
	cfg       Config
	now       func() time.Time
}

func NewService(repo ConsentRepository, publisher Publisher, cfg Config) *Service {
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = 1
	}
	if cfg.Environment == "" {
		cfg.Environment = "dev"
	}
	if cfg.RecordSource == "" {
		cfg.RecordSource = "api"
	}
	if cfg.MaxBatchSize == 0 {
		cfg.MaxBatchSize = defaultMaxBatchSize
	}
	if publisher == nil {
		publisher = NoopPublisher{}
	}
	return &Service{
		repo:      repo,
		publisher: publisher,
		cfg:       cfg,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Ingest(ctx context.Context, userID string, req BatchRequest) (BatchResponse, error) {
	if len(req.Events) > s.cfg.MaxBatchSize {
		return BatchResponse{}, apperrors.BadRequest("analytics batch exceeds max size")
	}
	if !s.cfg.Enabled {
		return BatchResponse{Rejected: len(req.Events)}, nil
	}
	current, err := s.repo.CurrentConsents(ctx, userID)
	if err != nil {
		return BatchResponse{}, apperrors.Internal(err)
	}

	accepted := make([]EnrichedEvent, 0, len(req.Events))
	rejected := 0
	receivedAt := s.now()
	for _, event := range req.Events {
		enriched, ok := s.enrich(event, userID, current, receivedAt)
		if !ok {
			rejected++
			continue
		}
		accepted = append(accepted, enriched)
	}
	if len(accepted) > 0 {
		if err := s.publisher.Publish(ctx, accepted); err != nil {
			return BatchResponse{}, apperrors.Internal(err)
		}
	}
	return BatchResponse{Accepted: len(accepted), Rejected: rejected}, nil
}

func (s *Service) CurrentConsents(ctx context.Context, userID string) (ConsentStateResponse, error) {
	current, err := s.repo.CurrentConsents(ctx, userID)
	if err != nil {
		return ConsentStateResponse{}, apperrors.Internal(err)
	}
	return consentResponse(current), nil
}

func (s *Service) UpdateConsents(ctx context.Context, userID string, req ConsentUpdateRequest) (ConsentStateResponse, error) {
	if len(req.Consents) == 0 {
		return ConsentStateResponse{}, apperrors.BadRequest("consents is required")
	}
	for rawPurpose := range req.Consents {
		purpose := Purpose(rawPurpose)
		if !purpose.Valid() {
			return ConsentStateResponse{}, apperrors.BadRequest("unsupported consent purpose")
		}
	}
	current, err := s.repo.CurrentConsents(ctx, userID)
	if err != nil {
		return ConsentStateResponse{}, apperrors.Internal(err)
	}
	now := s.now()
	for rawPurpose, granted := range req.Consents {
		purpose := Purpose(rawPurpose)
		version := current[purpose].Version + 1
		record := ConsentRecord{
			UserID:    userID,
			Purpose:   purpose,
			Granted:   granted,
			Version:   version,
			UpdatedAt: now,
			Source:    "api",
		}
		if err := s.repo.PutConsent(ctx, record); err != nil {
			return ConsentStateResponse{}, apperrors.Internal(err)
		}
		current[purpose] = record
	}
	return consentResponse(current), nil
}

func (s *Service) SetInitialConsents(ctx context.Context, userID string, consents map[string]bool) error {
	if len(consents) == 0 {
		return nil
	}
	_, err := s.UpdateConsents(ctx, userID, ConsentUpdateRequest{Consents: consents})
	return err
}

func (s *Service) RecordUserDataDeletion(ctx context.Context, userID string, requestedAt, deletedAt time.Time) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return apperrors.BadRequest("userID is required")
	}
	if !s.cfg.Enabled {
		return nil
	}
	if requestedAt.IsZero() {
		requestedAt = deletedAt
	}
	if deletedAt.IsZero() {
		deletedAt = s.now()
	}
	// TODO(data-pipeline): consume this marker in a compaction/purge job that
	// physically removes this user's historical rows from analytics datasets.
	record := DataDeletionRecord{
		RecordType:     "user_data_deletion",
		RecordID:       fmt.Sprintf("user_data_deletion:%s:%d", userID, requestedAt.Unix()),
		UserID:         userID,
		RequestedAt:    requestedAt,
		DeletedAt:      deletedAt,
		RecordSource:   s.cfg.RecordSource,
		Environment:    s.cfg.Environment,
		DeleteCategory: "account_delete",
	}
	if err := s.publisher.PublishDataDeletions(ctx, []DataDeletionRecord{record}); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) enrich(event EventEnvelope, userID string, current map[Purpose]ConsentRecord, receivedAt time.Time) (EnrichedEvent, bool) {
	if !s.validEvent(event) {
		return EnrichedEvent{}, false
	}
	consentVersion := 0
	for _, purpose := range event.Purposes {
		if !purpose.Valid() {
			return EnrichedEvent{}, false
		}
		record, ok := current[purpose]
		if !ok || !record.Granted {
			return EnrichedEvent{}, false
		}
		if record.Version > consentVersion {
			consentVersion = record.Version
		}
	}
	if containsForbiddenMessageProperty(event.Properties) {
		return EnrichedEvent{}, false
	}
	deviceIDHash := hashDeviceID(event.AnonymousID)
	event.AnonymousID = ""
	return EnrichedEvent{
		EventEnvelope:  event,
		UserID:         userID,
		DeviceIDHash:   deviceIDHash,
		ReceivedAt:     receivedAt,
		RecordSource:   s.cfg.RecordSource,
		Environment:    s.cfg.Environment,
		ConsentVersion: consentVersion,
	}, true
}

func (s *Service) validEvent(event EventEnvelope) bool {
	if event.EventID == "" || event.SessionID == "" || event.Source != "ios" {
		return false
	}
	if _, err := uuid.Parse(event.EventID); err != nil {
		return false
	}
	if _, err := uuid.Parse(event.SessionID); err != nil {
		return false
	}
	if !event.EventName.Valid() || event.SchemaVersion != s.cfg.SchemaVersion || event.EventTime.IsZero() {
		return false
	}
	return len(event.Purposes) > 0
}

func consentResponse(current map[Purpose]ConsentRecord) ConsentStateResponse {
	out := ConsentStateResponse{Consents: map[Purpose]ConsentStatus{}}
	for _, purpose := range AllPurposes() {
		record, ok := current[purpose]
		if !ok {
			out.Consents[purpose] = ConsentStatus{}
			continue
		}
		updated := record.UpdatedAt
		out.Consents[purpose] = ConsentStatus{
			Granted:   record.Granted,
			Version:   record.Version,
			UpdatedAt: &updated,
		}
	}
	return out
}

func hashDeviceID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func containsForbiddenMessageProperty(properties map[string]any) bool {
	for key, value := range properties {
		if forbiddenMessageProperty(key) {
			return true
		}
		if nested, ok := value.(map[string]any); ok && containsForbiddenMessageProperty(nested) {
			return true
		}
	}
	return false
}

func forbiddenMessageProperty(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "plaintext", "ciphertext", "ratchetheader", "ratchet_header", "reportproof", "report_proof":
		return true
	default:
		return false
	}
}
