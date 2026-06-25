package service

import (
	"context"

	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/repository"
)

type AcquisitionProgressStatusService interface {
	List(ctx context.Context) ([]*domain.AcquisitionProgressStatus, error)
	GetByID(ctx context.Context, id int) (*domain.AcquisitionProgressStatus, error)
	Create(ctx context.Context, req CreateAcquisitionProgressStatusRequest) (*domain.AcquisitionProgressStatus, error)
	Update(ctx context.Context, id int, req UpdateAcquisitionProgressStatusRequest) (*domain.AcquisitionProgressStatus, error)
	Delete(ctx context.Context, id int) error
}

type CreateAcquisitionProgressStatusRequest struct {
	Name        string  `json:"name"        validate:"required"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type UpdateAcquisitionProgressStatusRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type acquisitionProgressStatusService struct {
	repo repository.AcquisitionProgressStatusRepository
}

func NewAcquisitionProgressStatusService(repo repository.AcquisitionProgressStatusRepository) AcquisitionProgressStatusService {
	return &acquisitionProgressStatusService{repo: repo}
}

func (s *acquisitionProgressStatusService) List(ctx context.Context) ([]*domain.AcquisitionProgressStatus, error) {
	return s.repo.List(ctx)
}

func (s *acquisitionProgressStatusService) GetByID(ctx context.Context, id int) (*domain.AcquisitionProgressStatus, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *acquisitionProgressStatusService) Create(ctx context.Context, req CreateAcquisitionProgressStatusRequest) (*domain.AcquisitionProgressStatus, error) {
	aps := &domain.AcquisitionProgressStatus{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}
	if err := s.repo.Create(ctx, aps); err != nil {
		return nil, err
	}
	return aps, nil
}

func (s *acquisitionProgressStatusService) Update(ctx context.Context, id int, req UpdateAcquisitionProgressStatusRequest) (*domain.AcquisitionProgressStatus, error) {
	aps, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		aps.Name = req.Name
	}
	aps.Description = req.Description
	aps.SortOrder = req.SortOrder
	if err := s.repo.Update(ctx, aps); err != nil {
		return nil, err
	}
	return aps, nil
}

func (s *acquisitionProgressStatusService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
