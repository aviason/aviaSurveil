package user_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	apperrors "kindred_server/internal/platform/errors"
	"kindred_server/internal/testutil"
	"kindred_server/internal/user"
)

func TestService_Create_Success(t *testing.T) {
	repo := testutil.NewFakeUserRepo()
	svc := user.NewService(repo)
	u, err := svc.Create(context.Background(), user.CreateRequest{
		Email:       "  Foo@Example.com  ",
		DisplayName: "  Foo  ",
		Phone:       "  +1  ",
		City:        "  NYC  ",
		BirthYear:   1990,
		Gender:      "  non_binary  ",
	}, "hash")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if u.ID == "" {
		t.Error("ID not assigned")
	}
	if u.Email != "foo@example.com" {
		t.Errorf("Email = %q, want lowercased+trimmed", u.Email)
	}
	if u.DisplayName != "Foo" || u.Phone != "+1" || u.City != "NYC" {
		t.Errorf("fields not trimmed: %+v", u)
	}
	if u.BirthYear != 1990 || u.Gender != "non_binary" {
		t.Errorf("demographics not captured: %+v", u)
	}
	if u.PasswordHash != "hash" {
		t.Errorf("PasswordHash = %q", u.PasswordHash)
	}
	if u.CreatedAt.IsZero() || u.UpdatedAt.IsZero() {
		t.Error("timestamps not set")
	}

	// Repo persisted it.
	got, err := repo.GetByID(context.Background(), u.ID)
	if err != nil || got.Email != u.Email {
		t.Errorf("not persisted: %v / %+v", err, got)
	}
}

func TestService_Create_InvalidEmail(t *testing.T) {
	svc := user.NewService(testutil.NewFakeUserRepo())
	_, err := svc.Create(context.Background(), user.CreateRequest{
		Email:       "not-an-email",
		DisplayName: "X",
	}, "h")
	var ae *apperrors.AppError
	if !errors.As(err, &ae) || ae.Status != 400 {
		t.Fatalf("err = %v, want BadRequest", err)
	}
}

func TestService_Create_MissingDisplayName(t *testing.T) {
	svc := user.NewService(testutil.NewFakeUserRepo())
	_, err := svc.Create(context.Background(), user.CreateRequest{
		Email:       "a@b.com",
		DisplayName: "   ",
	}, "h")
	var ae *apperrors.AppError
	if !errors.As(err, &ae) || ae.Status != 400 {
		t.Fatalf("err = %v, want BadRequest", err)
	}
}

func TestService_Create_DuplicateEmailConflict(t *testing.T) {
	repo := testutil.NewFakeUserRepo()
	svc := user.NewService(repo)
	req := user.CreateRequest{Email: "dup@example.com", DisplayName: "D"}
	if _, err := svc.Create(context.Background(), req, "h"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(context.Background(), req, "h")
	var ae *apperrors.AppError
	if !errors.As(err, &ae) || ae.Status != 409 {
		t.Fatalf("err = %v, want Conflict", err)
	}
}

func TestRepoFlow_GetUpdate(t *testing.T) {
	repo := testutil.NewFakeUserRepo()
	svc := user.NewService(repo)
	u, err := svc.Create(context.Background(), user.CreateRequest{
		Email: "g@example.com", DisplayName: "G",
	}, "h")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// GetByID
	got, err := repo.GetByID(context.Background(), u.ID)
	if err != nil || got.ID != u.ID {
		t.Fatalf("GetByID: %v / %+v", err, got)
	}

	// GetByEmail (case-insensitive)
	got2, err := repo.GetByEmail(context.Background(), "G@Example.com")
	if err != nil || got2.ID != u.ID {
		t.Fatalf("GetByEmail: %v / %+v", err, got2)
	}

	// Update
	got.DisplayName = "Renamed"
	if err := repo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got3, _ := repo.GetByID(context.Background(), u.ID)
	if got3.DisplayName != "Renamed" {
		t.Errorf("after update DisplayName = %q", got3.DisplayName)
	}

	// Get missing
	_, err = repo.GetByID(context.Background(), "nope")
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("missing id err = %v, want ErrNotFound", err)
	}
}

func TestNewID_UUIDv7Shape(t *testing.T) {
	// id.go is package-private; assert shape via a Created user.
	repo := testutil.NewFakeUserRepo()
	svc := user.NewService(repo)
	u, err := svc.Create(context.Background(), user.CreateRequest{
		Email: "id@example.com", DisplayName: "I",
	}, "h")
	if err != nil {
		t.Fatal(err)
	}
	// UUID format: 8-4-4-4-12 hex.
	parts := strings.Split(u.ID, "-")
	if len(parts) != 5 || len(parts[0]) != 8 || len(parts[1]) != 4 ||
		len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		t.Fatalf("id %q is not a UUID", u.ID)
	}
	// Version nibble = first hex of group 3 should be '7' for UUIDv7.
	if parts[2][0] != '7' {
		t.Errorf("version nibble = %q, want 7 (UUIDv7)", string(parts[2][0]))
	}
}
