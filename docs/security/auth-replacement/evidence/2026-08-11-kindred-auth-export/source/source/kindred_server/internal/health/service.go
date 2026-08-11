package health

import "time"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Check() Status {
	name, env := s.repo.Metadata()
	return Status{Service: name, Environment: env, Status: "ok", Time: time.Now().UTC()}
}
