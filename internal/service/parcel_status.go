package service

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type ParcelStatusService interface {
	List(ctx context.Context) ([]*domain.ParcelStatus, error)
	GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error)
	Create(ctx context.Context, req CreateParcelStatusRequest) (*domain.ParcelStatus, error)
	Update(ctx context.Context, id int, req UpdateParcelStatusRequest) (*domain.ParcelStatus, error)
	Delete(ctx context.Context, id int) error
}

type CreateParcelStatusRequest struct {
	Code      string `json:"code"       validate:"required"`
	Name      string `json:"name"       validate:"required"`
	SortOrder int    `json:"sort_order"`
}

type UpdateParcelStatusRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type parcelStatusService struct {
	repo repository.ParcelStatusRepository
}

func NewParcelStatusService(repo repository.ParcelStatusRepository) ParcelStatusService {
	return &parcelStatusService{repo: repo}
}

func (s *parcelStatusService) List(ctx context.Context) ([]*domain.ParcelStatus, error) {
	return s.repo.List(ctx)
}

func (s *parcelStatusService) GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *parcelStatusService) Create(ctx context.Context, req CreateParcelStatusRequest) (*domain.ParcelStatus, error) {
	ps := &domain.ParcelStatus{
		Code:      req.Code,
		Name:      req.Name,
		SortOrder: req.SortOrder,
	}
	if err := s.repo.Create(ctx, ps); err != nil {
		return nil, err
	}
	return ps, nil
}

func (s *parcelStatusService) Update(ctx context.Context, id int, req UpdateParcelStatusRequest) (*domain.ParcelStatus, error) {
	ps, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Code != "" {
		ps.Code = req.Code
	}
	if req.Name != "" {
		ps.Name = req.Name
	}
	ps.SortOrder = req.SortOrder
	if err := s.repo.Update(ctx, ps); err != nil {
		return nil, err
	}
	return ps, nil
}

func (s *parcelStatusService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
