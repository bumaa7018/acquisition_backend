package service

import (
	"context"

	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/repository"
)

type ProgressTypeService interface {
	List(ctx context.Context) ([]*domain.ProgressType, error)
	GetByID(ctx context.Context, id int) (*domain.ProgressType, error)
	Create(ctx context.Context, req CreateProgressTypeRequest) (*domain.ProgressType, error)
	Update(ctx context.Context, id int, req UpdateProgressTypeRequest) (*domain.ProgressType, error)
	Delete(ctx context.Context, id int) error
}

type CreateProgressTypeRequest struct {
	Name        string  `json:"name"        validate:"required"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type UpdateProgressTypeRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type progressTypeService struct {
	repo repository.ProgressTypeRepository
}

func NewProgressTypeService(repo repository.ProgressTypeRepository) ProgressTypeService {
	return &progressTypeService{repo: repo}
}

func (s *progressTypeService) List(ctx context.Context) ([]*domain.ProgressType, error) {
	return s.repo.List(ctx)
}

func (s *progressTypeService) GetByID(ctx context.Context, id int) (*domain.ProgressType, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *progressTypeService) Create(ctx context.Context, req CreateProgressTypeRequest) (*domain.ProgressType, error) {
	pt := &domain.ProgressType{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}
	if err := s.repo.Create(ctx, pt); err != nil {
		return nil, err
	}
	return pt, nil
}

func (s *progressTypeService) Update(ctx context.Context, id int, req UpdateProgressTypeRequest) (*domain.ProgressType, error) {
	pt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		pt.Name = req.Name
	}
	pt.Description = req.Description
	pt.SortOrder = req.SortOrder
	if err := s.repo.Update(ctx, pt); err != nil {
		return nil, err
	}
	return pt, nil
}

func (s *progressTypeService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
