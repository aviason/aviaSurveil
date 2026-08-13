package mail

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDeliveryNotFound = errors.New("mail delivery not found")
	ErrLeaseLost        = errors.New("mail delivery lease is no longer held")
)

const (
	defaultLeaseTTL    = 30 * time.Second
	defaultMaxAttempts = 5
	defaultRetention   = 7 * 24 * time.Hour
	maxRetryDelay      = 15 * time.Minute
)

// Clock keeps retry and lease boundaries deterministic in focused tests.
type Clock func() time.Time

// OutboxConfig deliberately requires an application-held encryption key: an
// accidental database-only backup must not reveal verification or recovery
// material embedded in mail messages.
type OutboxConfig struct {
	Pool          *pgxpool.Pool
	EncryptionKey []byte
	Clock         Clock
	LeaseTTL      time.Duration
	MaxAttempts   int
	Retention     time.Duration
}

type Outbox struct {
	pool        *pgxpool.Pool
	key         []byte
	clock       Clock
	leaseTTL    time.Duration
	maxAttempts int
	retention   time.Duration
}

type Delivery struct {
	Recipient string
	Subject   string
	Body      string
	DedupeKey string
}

type ClaimedDelivery struct {
	ID         string
	LeaseToken string
	Delivery   Delivery
	Attempt    int
}

type DeliveryState string

const (
	DeliveryQueued          DeliveryState = "queued"
	DeliveryLeased          DeliveryState = "leased"
	DeliveryRetryable       DeliveryState = "retryable"
	DeliveryDelivered       DeliveryState = "delivered"
	DeliveryTerminalFailure DeliveryState = "terminal-failure"
)

type Snapshot struct {
	ID             string
	State          DeliveryState
	AttemptCount   int
	AvailableAt    time.Time
	LeaseUntil     time.Time
	DeliveredAt    time.Time
	LastErrorClass string
}

// DeliverySender is intentionally narrow, so the outbox can exercise the same TLS
// sender against Mailpit without acquiring broader SMTP configuration access.
type DeliverySender interface {
	Send(context.Context, string, string, string) error
}

func NewOutbox(configuration OutboxConfig) (*Outbox, error) {
	if configuration.Pool == nil || len(configuration.EncryptionKey) != 32 {
		return nil, errors.New("mail outbox requires a PostgreSQL pool and 32-byte encryption key")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	if configuration.LeaseTTL == 0 {
		configuration.LeaseTTL = defaultLeaseTTL
	}
	if configuration.LeaseTTL < time.Second || configuration.LeaseTTL > 5*time.Minute {
		return nil, errors.New("mail outbox lease duration is invalid")
	}
	if configuration.MaxAttempts == 0 {
		configuration.MaxAttempts = defaultMaxAttempts
	}
	if configuration.MaxAttempts < 1 || configuration.MaxAttempts > 12 {
		return nil, errors.New("mail outbox attempt limit is invalid")
	}
	if configuration.Retention == 0 {
		configuration.Retention = defaultRetention
	}
	if configuration.Retention < 24*time.Hour || configuration.Retention > 365*24*time.Hour {
		return nil, errors.New("mail outbox retention is invalid")
	}
	return &Outbox{
		pool: configuration.Pool, key: append([]byte(nil), configuration.EncryptionKey...),
		clock: configuration.Clock, leaseTTL: configuration.LeaseTTL, maxAttempts: configuration.MaxAttempts,
		retention: configuration.Retention,
	}, nil
}

func (outbox *Outbox) Enqueue(ctx context.Context, delivery Delivery) (string, error) {
	tx, err := outbox.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin mail enqueue: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := outbox.EnqueueTx(ctx, tx, delivery)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit mail enqueue: %w", err)
	}
	return id, nil
}

// EnqueueTx prepares one encrypted delivery in a caller-owned transaction.
// DedupeKey is a caller-provided non-PII hash; a pending delivery wins the
// race and its ID is returned to all coalesced callers.
func (outbox *Outbox) EnqueueTx(ctx context.Context, tx pgx.Tx, delivery Delivery) (string, error) {
	if tx == nil {
		return "", errors.New("mail enqueue transaction is required")
	}
	if err := validateDelivery(delivery); err != nil {
		return "", err
	}
	id, err := newDeliveryID()
	if err != nil {
		return "", err
	}
	recipient, err := outbox.encrypt(id, "recipient", delivery.Recipient)
	if err != nil {
		return "", err
	}
	subject, err := outbox.encrypt(id, "subject", delivery.Subject)
	if err != nil {
		return "", err
	}
	body, err := outbox.encrypt(id, "body", delivery.Body)
	if err != nil {
		return "", err
	}
	now := outbox.clock().UTC()
	var storedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO auth_identity.mail_deliveries(
			delivery_id, recipient_ciphertext, subject_ciphertext, body_ciphertext,
			state, attempt_count, available_at, created_at, updated_at, dedupe_key
		) VALUES ($1, $2, $3, $4, 'queued', 0, $5, $5, $5, NULLIF($6, ''))
		ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL AND state IN ('queued', 'leased', 'retryable') DO NOTHING
		RETURNING delivery_id`, id, recipient, subject, body, now, strings.TrimSpace(delivery.DedupeKey)).Scan(&storedID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `SELECT delivery_id FROM auth_identity.mail_deliveries WHERE dedupe_key = $1 AND state IN ('queued', 'leased', 'retryable') ORDER BY created_at, delivery_id LIMIT 1`, strings.TrimSpace(delivery.DedupeKey)).Scan(&storedID); err != nil {
			return "", fmt.Errorf("load coalesced mail delivery: %w", err)
		}
		return storedID, nil
	}
	if err != nil {
		return "", fmt.Errorf("enqueue mail delivery: %w", err)
	}
	return storedID, nil
}

// Claim atomically recovers expired leases and claims one due delivery. The
// opaque lease token must be presented by the worker that finalizes the send.
func (outbox *Outbox) Claim(ctx context.Context) (ClaimedDelivery, bool, error) {
	tx, err := outbox.pool.Begin(ctx)
	if err != nil {
		return ClaimedDelivery{}, false, fmt.Errorf("begin mail claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := outbox.clock().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.mail_deliveries
		SET state = 'retryable', available_at = $1, lease_until = NULL,
			lease_token_hash = NULL, updated_at = $1, last_error_class = 'lease-expired'
		WHERE state = 'leased' AND lease_until <= $1`, now); err != nil {
		return ClaimedDelivery{}, false, fmt.Errorf("recover expired mail leases: %w", err)
	}
	row := tx.QueryRow(ctx, `
		SELECT delivery_id, recipient_ciphertext, subject_ciphertext, body_ciphertext, attempt_count
		FROM auth_identity.mail_deliveries
		WHERE state IN ('queued', 'retryable') AND available_at <= $1
		ORDER BY available_at, created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`, now)
	var id string
	var recipient, subject, body []byte
	var attempts int
	if err := row.Scan(&id, &recipient, &subject, &body, &attempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.Commit(ctx); err != nil {
				return ClaimedDelivery{}, false, fmt.Errorf("commit empty mail claim: %w", err)
			}
			return ClaimedDelivery{}, false, nil
		}
		return ClaimedDelivery{}, false, fmt.Errorf("select mail delivery: %w", err)
	}
	leaseToken, err := randomLeaseToken()
	if err != nil {
		return ClaimedDelivery{}, false, err
	}
	leaseHash := sha256.Sum256([]byte(leaseToken))
	leaseUntil := now.Add(outbox.leaseTTL)
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.mail_deliveries
		SET state = 'leased', attempt_count = attempt_count + 1, lease_until = $2,
			lease_token_hash = $3, updated_at = $1
		WHERE delivery_id = $4`, now, leaseUntil, leaseHash[:], id); err != nil {
		return ClaimedDelivery{}, false, fmt.Errorf("lease mail delivery: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ClaimedDelivery{}, false, fmt.Errorf("commit mail claim: %w", err)
	}
	decodedRecipient, err := outbox.decrypt(id, "recipient", recipient)
	if err != nil {
		return ClaimedDelivery{}, false, err
	}
	decodedSubject, err := outbox.decrypt(id, "subject", subject)
	if err != nil {
		return ClaimedDelivery{}, false, err
	}
	decodedBody, err := outbox.decrypt(id, "body", body)
	if err != nil {
		return ClaimedDelivery{}, false, err
	}
	return ClaimedDelivery{ID: id, LeaseToken: leaseToken, Delivery: Delivery{
		Recipient: decodedRecipient, Subject: decodedSubject, Body: decodedBody,
	}, Attempt: attempts + 1}, true, nil
}

func (outbox *Outbox) Acknowledge(ctx context.Context, id, leaseToken string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(leaseToken) == "" {
		return ErrLeaseLost
	}
	now := outbox.clock().UTC()
	hash := sha256.Sum256([]byte(leaseToken))
	command, err := outbox.pool.Exec(ctx, `
		UPDATE auth_identity.mail_deliveries
		SET state = 'delivered', delivered_at = $1, lease_until = NULL,
			lease_token_hash = NULL, last_error_class = NULL, updated_at = $1
		WHERE delivery_id = $2 AND state = 'leased' AND lease_until > $1 AND lease_token_hash = $3`, now, id, hash[:])
	if err != nil {
		return fmt.Errorf("acknowledge mail delivery: %w", err)
	}
	if command.RowsAffected() != 1 {
		return outbox.leaseOrDeliveryError(ctx, id)
	}
	return nil
}

// Retry finalizes a failed lease without retaining arbitrary SMTP error text.
// A bounded exponential delay keeps a dependency outage from becoming a tight
// retry loop; a delivery becomes terminal only after its configured budget.
func (outbox *Outbox) Retry(ctx context.Context, id, leaseToken, errorClass string) (DeliveryState, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(leaseToken) == "" || !validErrorClass(errorClass) {
		return "", ErrLeaseLost
	}
	tx, err := outbox.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin mail retry: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := outbox.clock().UTC()
	hash := sha256.Sum256([]byte(leaseToken))
	var attempts int
	if err := tx.QueryRow(ctx, `
		SELECT attempt_count
		FROM auth_identity.mail_deliveries
		WHERE delivery_id = $1 AND state = 'leased' AND lease_until > $2 AND lease_token_hash = $3
		FOR UPDATE`, id, now, hash[:]).Scan(&attempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", outbox.leaseOrDeliveryError(ctx, id)
		}
		return "", fmt.Errorf("load mail delivery retry: %w", err)
	}
	state := DeliveryRetryable
	availableAt := now.Add(retryDelay(attempts))
	if attempts >= outbox.maxAttempts {
		state = DeliveryTerminalFailure
		availableAt = now
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.mail_deliveries
		SET state = $1, available_at = $2, lease_until = NULL, lease_token_hash = NULL,
			last_error_class = $3, updated_at = $4
		WHERE delivery_id = $5`, state, availableAt, errorClass, now, id); err != nil {
		return "", fmt.Errorf("record mail retry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit mail retry: %w", err)
	}
	return state, nil
}

func (outbox *Outbox) DeliverOnce(ctx context.Context, sender DeliverySender) (bool, error) {
	if sender == nil {
		return false, errors.New("mail sender is required")
	}
	claimed, found, err := outbox.Claim(ctx)
	if err != nil || !found {
		return found, err
	}
	if err := sender.Send(ctx, claimed.Delivery.Recipient, claimed.Delivery.Subject, claimed.Delivery.Body); err != nil {
		_, retryErr := outbox.Retry(ctx, claimed.ID, claimed.LeaseToken, "transport")
		if retryErr != nil {
			return true, fmt.Errorf("deliver mail and record retry: %w", retryErr)
		}
		return true, nil
	}
	if err := outbox.Acknowledge(ctx, claimed.ID, claimed.LeaseToken); err != nil {
		return true, err
	}
	return true, nil
}

func (outbox *Outbox) Snapshot(ctx context.Context, id string) (Snapshot, error) {
	var snapshot Snapshot
	var leaseUntil, deliveredAt *time.Time
	err := outbox.pool.QueryRow(ctx, `
		SELECT delivery_id, state, attempt_count, available_at, lease_until, delivered_at, COALESCE(last_error_class, '')
		FROM auth_identity.mail_deliveries WHERE delivery_id = $1`, id).Scan(
		&snapshot.ID, &snapshot.State, &snapshot.AttemptCount, &snapshot.AvailableAt,
		&leaseUntil, &deliveredAt, &snapshot.LastErrorClass,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrDeliveryNotFound
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("load mail delivery: %w", err)
	}
	if leaseUntil != nil {
		snapshot.LeaseUntil = *leaseUntil
	}
	if deliveredAt != nil {
		snapshot.DeliveredAt = *deliveredAt
	}
	return snapshot, nil
}

// CleanupRetention removes only delivered and terminal rows older than the
// configured retention period. The bounded delete is safe to repeat after a
// restart and never touches queued, leased, or retryable deliveries.
func (outbox *Outbox) CleanupRetention(ctx context.Context, at time.Time, limit int) (int64, error) {
	if limit < 1 || limit > 1000 {
		return 0, errors.New("mail retention cleanup limit is invalid")
	}
	cutoff := at.UTC().Add(-outbox.retention)
	command, err := outbox.pool.Exec(ctx, `
		WITH retained AS (
			SELECT delivery_id FROM auth_identity.mail_deliveries
			WHERE state IN ('delivered', 'terminal-failure')
			  AND updated_at <= $1
			ORDER BY updated_at, delivery_id
			LIMIT $2
		)
		DELETE FROM auth_identity.mail_deliveries delivery
		USING retained
		WHERE delivery.delivery_id = retained.delivery_id`, cutoff, limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup retained mail deliveries: %w", err)
	}
	return command.RowsAffected(), nil
}

func (outbox *Outbox) leaseOrDeliveryError(ctx context.Context, id string) error {
	var exists bool
	if err := outbox.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth_identity.mail_deliveries WHERE delivery_id = $1)`, id).Scan(&exists); err != nil {
		return fmt.Errorf("check mail delivery: %w", err)
	}
	if !exists {
		return ErrDeliveryNotFound
	}
	return ErrLeaseLost
}

func (outbox *Outbox) encrypt(id, field, plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(outbox.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), outboxAAD(id, field)), nil
}

func (outbox *Outbox) decrypt(id, field string, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(outbox.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("mail delivery ciphertext is unavailable")
	}
	plaintext, err := gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], outboxAAD(id, field))
	if err != nil {
		return "", errors.New("mail delivery ciphertext cannot be decrypted")
	}
	return string(plaintext), nil
}

func outboxAAD(id, field string) []byte {
	return []byte("as360-mail-outbox-v1\x00" + id + "\x00" + field)
}

func validateDelivery(delivery Delivery) error {
	if strings.TrimSpace(delivery.Recipient) == "" || strings.TrimSpace(delivery.Subject) == "" || strings.ContainsAny(delivery.Recipient, "\r\n") || strings.ContainsAny(delivery.Subject, "\r\n") || strings.Contains(delivery.Body, "\x00") || len(strings.TrimSpace(delivery.DedupeKey)) > 160 {
		return errors.New("mail delivery is invalid")
	}
	return nil
}

func validErrorClass(value string) bool {
	return value == "transport" || value == "temporary" || value == "permanent" || value == "lease-expired"
}

func newDeliveryID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "dly_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func randomLeaseToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Second * time.Duration(1<<(attempt-1))
	if delay > maxRetryDelay {
		return maxRetryDelay
	}
	return delay
}
