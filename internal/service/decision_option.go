package service

import (
	"context"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

// DecisionOptionService — Ажлын төрөл / Төсөв сонголтын CRUD (Тохиргоо цэс).
type DecisionOptionService interface {
	List(ctx context.Context) ([]*domain.DecisionOption, error)
	GetByID(ctx context.Context, id int) (*domain.DecisionOption, error)
	Create(ctx context.Context, req CreateDecisionOptionRequest) (*domain.DecisionOption, error)
	Update(ctx context.Context, id int, req UpdateDecisionOptionRequest) (*domain.DecisionOption, error)
	Delete(ctx context.Context, id int) error
}

type CreateDecisionOptionRequest struct {
	Code      string `json:"code"       validate:"required"`
	Name      string `json:"name"       validate:"required"`
	SortOrder int    `json:"sort_order"`
}

type UpdateDecisionOptionRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type decisionOptionService struct {
	repo repository.DecisionOptionRepository
}

func NewDecisionOptionService(repo repository.DecisionOptionRepository) DecisionOptionService {
	return &decisionOptionService{repo: repo}
}

func (s *decisionOptionService) List(ctx context.Context) ([]*domain.DecisionOption, error) {
	return s.repo.List(ctx)
}

func (s *decisionOptionService) GetByID(ctx context.Context, id int) (*domain.DecisionOption, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *decisionOptionService) Create(ctx context.Context, req CreateDecisionOptionRequest) (*domain.DecisionOption, error) {
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	o := &domain.DecisionOption{
		Code:      strings.TrimSpace(req.Code),
		Name:      strings.TrimSpace(req.Name),
		SortOrder: req.SortOrder,
	}
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *decisionOptionService) Update(ctx context.Context, id int, req UpdateDecisionOptionRequest) (*domain.DecisionOption, error) {
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Code) != "" {
		o.Code = strings.TrimSpace(req.Code)
	}
	if strings.TrimSpace(req.Name) != "" {
		o.Name = strings.TrimSpace(req.Name)
	}
	o.SortOrder = req.SortOrder
	if err := s.repo.Update(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *decisionOptionService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
