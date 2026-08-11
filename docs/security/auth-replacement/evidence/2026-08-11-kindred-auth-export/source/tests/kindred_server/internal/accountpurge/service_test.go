package accountpurge

import (
	"context"
	"errors"
	"testing"
	"time"

	"kindred_server/internal/user"
)

func TestRunPurgesOnlyDueDeletionPendingAccounts(t *testing.T) {
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	requestedAt := now.Add(-30 * 24 * time.Hour)
	repo := &fakeRepo{
		users: []user.User{
			{ID: "due", AccountStatus: user.AccountStatusDeletionPending, DeletionRequestedAt: requestedAt, ScheduledPurgeAt: now.Add(-time.Minute)},
			{ID: "future", AccountStatus: user.AccountStatusDeletionPending, ScheduledPurgeAt: now.Add(time.Minute)},
			{ID: "deactivated", AccountStatus: user.AccountStatusDeactivated, ScheduledPurgeAt: now.Add(-time.Minute)},
		},
	}
	eraser := &fakeDataEraser{}

	result, err := NewService(repo, WithDataEraser(eraser)).Run(context.Background(), now, 100)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if result.Scanned != 3 || result.Purged != 1 {
		t.Fatalf("result = %+v, want scanned=3 purged=1", result)
	}
	if len(repo.purged) != 1 || repo.purged[0] != "due" {
		t.Fatalf("purged = %#v, want due", repo.purged)
	}
	if len(eraser.deleted) != 1 || eraser.deleted[0].userID != "due" || !eraser.deleted[0].requestedAt.Equal(requestedAt) {
		t.Fatalf("data erasures = %#v", eraser.deleted)
	}
}

func TestRunStopsBeforePurgeWhenDataErasureFails(t *testing.T) {
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		users: []user.User{
			{ID: "due", AccountStatus: user.AccountStatusDeletionPending, ScheduledPurgeAt: now.Add(-time.Minute)},
		},
	}
	eraser := &fakeDataEraser{err: errors.New("data lake unavailable")}

	_, err := NewService(repo, WithDataEraser(eraser)).Run(context.Background(), now, 100)
	if err == nil {
		t.Fatal("Run error = nil, want data erasure error")
	}
	if len(repo.purged) != 0 {
		t.Fatalf("purged = %#v, want none", repo.purged)
	}
}

type fakeRepo struct {
	users  []user.User
	purged []string
}

func (r *fakeRepo) ListDeletionPending(context.Context, time.Time, int) ([]user.User, error) {
	return r.users, nil
}

func (r *fakeRepo) PurgeAccount(_ context.Context, u user.User, _ time.Time) error {
	r.purged = append(r.purged, u.ID)
	return nil
}

type fakeDataEraser struct {
	deleted []struct {
		userID      string
		requestedAt time.Time
		deletedAt   time.Time
	}
	err error
}

func (e *fakeDataEraser) RecordUserDataDeletion(_ context.Context, userID string, requestedAt, deletedAt time.Time) error {
	if e.err != nil {
		return e.err
	}
	e.deleted = append(e.deleted, struct {
		userID      string
		requestedAt time.Time
		deletedAt   time.Time
	}{userID: userID, requestedAt: requestedAt, deletedAt: deletedAt})
	return nil
}
