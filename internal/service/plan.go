package service

import (
	"context"

	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/repository"
)

type PlanService interface {
	Search(ctx context.Context, code string) (*domain.Plan, error)
}

type planService struct {
	planRepo repository.PlanRepository
}

func NewPlanService(planRepo repository.PlanRepository) PlanService {
	return &planService{planRepo: planRepo}
}

func (s *planService) Search(ctx context.Context, code string) (*domain.Plan, error) {
	p, err := s.planRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}
