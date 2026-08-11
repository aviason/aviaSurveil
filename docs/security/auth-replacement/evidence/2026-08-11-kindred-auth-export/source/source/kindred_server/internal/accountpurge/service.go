package accountpurge

import (
	"context"
	"time"

	"kindred_server/internal/user"
)

type Repository interface {
	ListDeletionPending(ctx context.Context, now time.Time, limit int) ([]user.User, error)
	PurgeAccount(ctx context.Context, u user.User, now time.Time) error
}

type DataEraser interface {
	RecordUserDataDeletion(ctx context.Context, userID string, requestedAt, deletedAt time.Time) error
}

type Service struct {
	repo       Repository
	dataEraser DataEraser
}

type Result struct {
	Scanned int `json:"scanned"`
	Purged  int `json:"purged"`
}

type Option func(*Service)

func WithDataEraser(dataEraser DataEraser) Option {
	return func(s *Service) {
		s.dataEraser = dataEraser
	}
}

func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Run(ctx context.Context, now time.Time, limit int) (Result, error) {
	if limit <= 0 {
		limit = 100
	}
	due, err := s.repo.ListDeletionPending(ctx, now, limit)
	if err != nil {
		return Result{}, err
	}
	result := Result{Scanned: len(due)}
	for _, u := range due {
		if u.Status() != user.AccountStatusDeletionPending || u.ScheduledPurgeAt.IsZero() || now.Before(u.ScheduledPurgeAt) {
			continue
		}
		if s.dataEraser != nil {
			if err := s.dataEraser.RecordUserDataDeletion(ctx, u.ID, u.DeletionRequestedAt, now); err != nil {
				return result, err
			}
		}
		if err := s.repo.PurgeAccount(ctx, u, now); err != nil {
			return result, err
		}
		result.Purged++
	}
	return result, nil
}
