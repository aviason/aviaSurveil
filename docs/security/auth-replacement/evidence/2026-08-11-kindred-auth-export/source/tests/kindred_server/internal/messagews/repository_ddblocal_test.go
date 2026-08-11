//go:build dynamodblocal

package messagews

import (
	"context"
	"testing"
	"time"

	"kindred_server/internal/testutil/ddblocal"
)

func TestRepositoryDynamoDBLocalSubscriptions(t *testing.T) {
	client := ddblocal.Connect(t)
	usersTable, appTable, rateTable := ddblocal.UniqueTableNames(t)
	ctx := context.Background()
	if err := ddblocal.EnsureTables(ctx, client, usersTable, appTable, rateTable); err != nil {
		t.Fatalf("ensure tables: %v", err)
	}
	t.Cleanup(func() {
		_ = ddblocal.Truncate(context.Background(), client, appTable)
	})

	repo := NewRepository(client, appTable)
	now := time.Now().UTC()
	conn := Connection{
		ID:        "conn-1",
		UserID:    "u1",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour).Unix(),
	}
	if err := repo.SaveConnection(ctx, conn); err != nil {
		t.Fatalf("SaveConnection: %v", err)
	}
	got, err := repo.GetConnection(ctx, "conn-1")
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.ID != conn.ID || got.UserID != conn.UserID {
		t.Fatalf("connection = %#v", got)
	}

	sub := Subscription{ConnectionID: "conn-1", RequestID: "r1", UserID: "u1", ExpiresAt: now.Add(time.Hour).Unix()}
	if err := repo.SaveSubscription(ctx, sub); err != nil {
		t.Fatalf("SaveSubscription: %v", err)
	}
	subs, err := repo.ListSubscriptionsByRequest(ctx, "r1", 10)
	if err != nil {
		t.Fatalf("ListSubscriptionsByRequest: %v", err)
	}
	if len(subs) != 1 || subs[0].ConnectionID != "conn-1" || subs[0].RequestID != "r1" {
		t.Fatalf("subscriptions = %#v", subs)
	}

	if err := repo.DeleteConnection(ctx, "conn-1"); err != nil {
		t.Fatalf("DeleteConnection: %v", err)
	}
	if _, err := repo.GetConnection(ctx, "conn-1"); err != ErrConnectionNotFound {
		t.Fatalf("GetConnection after delete error = %v", err)
	}
	subs, err = repo.ListSubscriptionsByRequest(ctx, "r1", 10)
	if err != nil {
		t.Fatalf("ListSubscriptionsByRequest after delete: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("subscriptions after delete = %#v", subs)
	}
}
