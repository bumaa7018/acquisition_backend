package service

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type PlanService interface {
	Search(ctx context.Context, code string) (*domain.Plan, error)
	Suggest(ctx context.Context, q string) ([]*domain.Plan, error)
}

type planService struct {
	planRepo repository.PlanRepository
}

func NewPlanService(planRepo repository.PlanRepository) PlanService {
	return &planService{planRepo: planRepo}
}

func (s *planService) Search(ctx context.Context, code string) (*domain.Plan, error) {
	return s.planRepo.GetByCode(ctx, code)
}

func (s *planService) Suggest(ctx context.Context, q string) ([]*domain.Plan, error) {
	return s.planRepo.Suggest(ctx, q)
}
