package auth

import (
	"context"

	"kindred_server/internal/user"
)

type Repository interface {
	Create(ctx context.Context, user user.User) error
	GetByEmail(ctx context.Context, email string) (user.User, error)
}
