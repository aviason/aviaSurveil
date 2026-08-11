//go:build dynamodblocal

package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"kindred_server/internal/user"
)

func newUserA() user.User {
	now := time.Now().UTC()
	return user.User{
		ID:          "user-a",
		Email:       "a@example.com",
		DisplayName: "Alice",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestUserRepository_Create_EmailUniqueness(t *testing.T) {
	fx := setupDDB(t)
	repo := &UserRepository{client: fx.Client, table: fx.UsersTable}
	ctx := context.Background()

	a := newUserA()
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Second user, different ID, same email → must fail with ErrAlreadyExists.
	b := newUserA()
	b.ID = "user-b"
	b.DisplayName = "Bob"
	if err := repo.Create(ctx, b); !errors.Is(err, user.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUserRepository_GetByEmail_RoundTrip(t *testing.T) {
	fx := setupDDB(t)
	repo := &UserRepository{client: fx.Client, table: fx.UsersTable}
	ctx := context.Background()

	a := newUserA()
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByEmail(ctx, "A@Example.com") // case-insensitive
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.ID != a.ID {
		t.Fatalf("got %q want %q", got.ID, a.ID)
	}
}

func TestUserRepository_Delete_RemovesBothRows(t *testing.T) {
	fx := setupDDB(t)
	repo := &UserRepository{client: fx.Client, table: fx.UsersTable}
	ctx := context.Background()

	a := newUserA()
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Delete(ctx, a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, a.ID); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("user row still present: %v", err)
	}
	if _, err := repo.GetByEmail(ctx, a.Email); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("email lock still present: %v", err)
	}
	// Email is now reusable.
	b := newUserA()
	b.ID = "user-b"
	if err := repo.Create(ctx, b); err != nil {
		t.Fatalf("re-create after delete: %v", err)
	}
}
