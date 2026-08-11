package user

import "context"

type Repository interface {
	Create(ctx context.Context, user User) error
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context, limit int) ([]User, error)
	Update(ctx context.Context, user User) error
	Delete(ctx context.Context, id string) error
}
