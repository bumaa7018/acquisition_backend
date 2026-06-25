package service

import (
	"context"

	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/repository"
)

type DocumentTypeService interface {
	List(ctx context.Context) ([]*domain.DocumentType, error)
	GetByID(ctx context.Context, id int) (*domain.DocumentType, error)
	Create(ctx context.Context, req CreateDocumentTypeRequest) (*domain.DocumentType, error)
	Update(ctx context.Context, id int, req UpdateDocumentTypeRequest) (*domain.DocumentType, error)
	Delete(ctx context.Context, id int) error
}

type CreateDocumentTypeRequest struct {
	Type        string  `json:"type"        validate:"required"`
	Name        string  `json:"name"        validate:"required"`
	Description *string `json:"description"`
	CreatedBy   *string `json:"created_by"`
}

type UpdateDocumentTypeRequest struct {
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	UpdatedBy   *string `json:"updated_by"`
}

type documentTypeService struct {
	repo repository.DocumentTypeRepository
}

func NewDocumentTypeService(repo repository.DocumentTypeRepository) DocumentTypeService {
	return &documentTypeService{repo: repo}
}

func (s *documentTypeService) List(ctx context.Context) ([]*domain.DocumentType, error) {
	return s.repo.List(ctx)
}

func (s *documentTypeService) GetByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *documentTypeService) Create(ctx context.Context, req CreateDocumentTypeRequest) (*domain.DocumentType, error) {
	dt := &domain.DocumentType{
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}
	if err := s.repo.Create(ctx, dt); err != nil {
		return nil, err
	}
	return dt, nil
}

func (s *documentTypeService) Update(ctx context.Context, id int, req UpdateDocumentTypeRequest) (*domain.DocumentType, error) {
	dt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Type != "" {
		dt.Type = req.Type
	}
	if req.Name != "" {
		dt.Name = req.Name
	}
	dt.Description = req.Description
	dt.UpdatedBy = req.UpdatedBy
	if err := s.repo.Update(ctx, dt); err != nil {
		return nil, err
	}
	return dt, nil
}

func (s *documentTypeService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
