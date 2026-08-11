package mail

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgreSQLOutboxRetriesAndRecoversLeases(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("not run: AVIA_AUTH_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open disposable PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply auth migrations: %v", err)
	}
	clock := &outboxTestClock{now: time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC)}
	outbox, err := NewOutbox(OutboxConfig{
		Pool: pool, EncryptionKey: []byte("01234567890123456789012345678901"), Clock: clock.Now,
		LeaseTTL: 5 * time.Second, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("create PostgreSQL outbox: %v", err)
	}
	delivery := Delivery{
		Recipient: "outbox-recipient@example.invalid",
		Subject:   "AS360 verification",
		Body:      "Disposable reset material must stay encrypted at rest.",
	}
	id, err := outbox.Enqueue(ctx, delivery)
	if err != nil {
		t.Fatalf("enqueue mail: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_identity.mail_deliveries WHERE delivery_id = $1`, id)
	}()
	var recipientCiphertext, subjectCiphertext, bodyCiphertext []byte
	if err := pool.QueryRow(ctx, `SELECT recipient_ciphertext, subject_ciphertext, body_ciphertext FROM auth_identity.mail_deliveries WHERE delivery_id = $1`, id).Scan(&recipientCiphertext, &subjectCiphertext, &bodyCiphertext); err != nil {
		t.Fatalf("read encrypted outbox row: %v", err)
	}
	for _, ciphertext := range [][]byte{recipientCiphertext, subjectCiphertext, bodyCiphertext} {
		if strings.Contains(string(ciphertext), delivery.Recipient) || strings.Contains(string(ciphertext), delivery.Body) {
			t.Fatal("outbox persisted mail plaintext")
		}
	}

	failingSender := &outboxTestSender{err: errors.New("disposable SMTP unavailable")}
	if found, err := outbox.DeliverOnce(ctx, failingSender); err != nil || !found {
		t.Fatalf("first delivery attempt = found:%v err:%v", found, err)
	}
	afterFailure, err := outbox.Snapshot(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailure.State != DeliveryRetryable || afterFailure.AttemptCount != 1 || afterFailure.LastErrorClass != "transport" {
		t.Fatalf("retryable outbox state = %+v", afterFailure)
	}
	if found, err := outbox.DeliverOnce(ctx, &outboxTestSender{}); err != nil || found {
		t.Fatalf("delivery bypassed bounded retry delay: found:%v err:%v", found, err)
	}
	clock.Advance(time.Second)
	passingSender := &outboxTestSender{}
	if found, err := outbox.DeliverOnce(ctx, passingSender); err != nil || !found {
		t.Fatalf("retry delivery = found:%v err:%v", found, err)
	}
	if got := passingSender.Deliveries(); len(got) != 1 || got[0] != delivery {
		t.Fatalf("SMTP delivery = %+v", got)
	}
	completed, err := outbox.Snapshot(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != DeliveryDelivered || completed.AttemptCount != 2 || completed.DeliveredAt.IsZero() {
		t.Fatalf("completed outbox state = %+v", completed)
	}

	leaseID, err := outbox.Enqueue(ctx, Delivery{Recipient: "lease@example.invalid", Subject: "Lease recovery", Body: "lease recovery body"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_identity.mail_deliveries WHERE delivery_id = $1`, leaseID)
	}()
	claimed, found, err := outbox.Claim(ctx)
	if err != nil || !found || claimed.ID != leaseID {
		t.Fatalf("claim lease recovery delivery = %+v/%v/%v", claimed, found, err)
	}
	if err := outbox.Acknowledge(ctx, leaseID, "wrong-lease"); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("wrong lease acknowledgement = %v", err)
	}
	clock.Advance(6 * time.Second)
	recovered, found, err := outbox.Claim(ctx)
	if err != nil || !found || recovered.ID != leaseID || recovered.Attempt != 2 || recovered.LeaseToken == claimed.LeaseToken {
		t.Fatalf("recovered lease = %+v/%v/%v", recovered, found, err)
	}
	if err := outbox.Acknowledge(ctx, recovered.ID, recovered.LeaseToken); err != nil {
		t.Fatalf("acknowledge recovered lease: %v", err)
	}
}

type outboxTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *outboxTestClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *outboxTestClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
}

type outboxTestSender struct {
	mu         sync.Mutex
	deliveries []Delivery
	err        error
}

func (sender *outboxTestSender) Send(_ context.Context, recipient, subject, body string) error {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if sender.err != nil {
		return sender.err
	}
	sender.deliveries = append(sender.deliveries, Delivery{Recipient: recipient, Subject: subject, Body: body})
	return nil
}

func (sender *outboxTestSender) Deliveries() []Delivery {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return append([]Delivery(nil), sender.deliveries...)
}
