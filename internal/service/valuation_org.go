package service

import (
	"context"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

// ValuationOrgService — мэргэжлийн байгууллагын app-owned profile-ийг
// бүртгэнэ. sdplatform-ийн HR өгөгдөл энэ бүртгэлийн цонхоор өөрчлөгдөхгүй.
type ValuationOrgService interface {
	List(ctx context.Context, filter domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error)
	Create(ctx context.Context, req ValuationOrgRequest) (*domain.ValuationOrg, error)
	Update(ctx context.Context, id domain.ID, req ValuationOrgRequest) (*domain.ValuationOrg, error)
	Delete(ctx context.Context, id domain.ID) error
}

type ValuationOrgRequest struct {
	Org domain.ValuationOrg
}

type valuationOrgService struct {
	repo repository.ValuationOrgRepository
}

func NewValuationOrgService(repo repository.ValuationOrgRepository) ValuationOrgService {
	return &valuationOrgService{repo: repo}
}

func (s *valuationOrgService) List(ctx context.Context, filter domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *valuationOrgService) GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *valuationOrgService) Create(ctx context.Context, req ValuationOrgRequest) (*domain.ValuationOrg, error) {
	org, err := prepareValuationOrg(req)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, org); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, org.ID)
}

func (s *valuationOrgService) Update(ctx context.Context, id domain.ID, req ValuationOrgRequest) (*domain.ValuationOrg, error) {
	org, err := prepareValuationOrg(req)
	if err != nil {
		return nil, err
	}
	org.ID = id
	if err := s.repo.Update(ctx, org); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *valuationOrgService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.Delete(ctx, id)
}

// prepareValuationOrg — profile-ийн оролтыг цэвэрлэж шалгана.
func prepareValuationOrg(req ValuationOrgRequest) (*domain.ValuationOrg, error) {
	org := req.Org
	org.Name = strings.TrimSpace(org.Name)
	if org.Name == "" {
		return nil, domain.ErrInvalidInput
	}
	org.ShortName = strings.TrimSpace(org.ShortName)
	org.RegisterNo = strings.TrimSpace(org.RegisterNo)
	org.LicenseNo = strings.TrimSpace(org.LicenseNo)
	org.Phone = strings.TrimSpace(org.Phone)
	org.Email = strings.TrimSpace(org.Email)
	org.Address = strings.TrimSpace(org.Address)

	return &org, nil
}
