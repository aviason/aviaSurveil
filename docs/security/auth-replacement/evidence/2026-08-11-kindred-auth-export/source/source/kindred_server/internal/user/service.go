package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	apperrors "kindred_server/internal/platform/errors"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateRequest, passwordHash string) (User, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return User{}, apperrors.BadRequest("valid email is required")
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		return User{}, apperrors.BadRequest("displayName is required")
	}
	now := time.Now().UTC()
	u := User{
		ID:            newID(),
		Email:         email,
		DisplayName:   strings.TrimSpace(req.DisplayName),
		Phone:         strings.TrimSpace(req.Phone),
		City:          strings.TrimSpace(req.City),
		BirthYear:     req.BirthYear,
		Gender:        strings.TrimSpace(req.Gender),
		PasswordHash:  passwordHash,
		CreatedAt:     now,
		UpdatedAt:     now,
		AccountStatus: AccountStatusActive,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return User{}, apperrors.Conflict("email already registered")
		}
		var cond *types.ConditionalCheckFailedException
		if errors.As(err, &cond) {
			return User{}, apperrors.Conflict("email already registered")
		}
		return User{}, apperrors.Internal(err)
	}
	return u, nil
}
