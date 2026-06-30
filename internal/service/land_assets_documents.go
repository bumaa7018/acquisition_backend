package service

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// ListAssets — чөлөөлөлтөд хамаарах барилгын мэдээллийн жагсаалт
func (s *landService) ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, 0, domain.ErrNotFound
	}
	list, total, err := s.landRepo.ListAssets(ctx, acquisitionID, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

func (s *landService) CreateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error) {
	if err := s.landRepo.CreateAsset(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *landService) UpdateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error) {
	if err := s.landRepo.UpdateAsset(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *landService) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	return s.landRepo.DeleteAsset(ctx, id)
}

func (s *landService) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.Compensation, error) {
	return s.landRepo.ListCompensations(ctx, acquisitionID, parcelID)
}

func (s *landService) CreateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error) {
	if err := s.landRepo.CreateCompensation(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *landService) UpdateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error) {
	existing, err := s.landRepo.GetCompensation(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	if existing.Status == "approved" {
		return nil, domain.ErrApproved
	}
	if err := s.landRepo.UpdateCompensation(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *landService) DeleteCompensation(ctx context.Context, id uuid.UUID) error {
	existing, err := s.landRepo.GetCompensation(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status == "approved" {
		return domain.ErrApproved
	}
	return s.landRepo.DeleteCompensation(ctx, id)
}

func (s *landService) ApproveCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error {
	return s.landRepo.ApproveCompensation(ctx, id, note, reviewedBy)
}

func (s *landService) RejectCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error {
	return s.landRepo.RejectCompensation(ctx, id, note, reviewedBy)
}

func (s *landService) UploadCompensationReport(ctx context.Context, acqID, compID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Compensation, error) {
	comp, err := s.landRepo.GetCompensation(ctx, compID)
	if err != nil || comp.AcquisitionID != acqID {
		return nil, domain.ErrNotFound
	}
	stored, err := s.files.Save(ctx, "compensation-report", compID.String(), filename, contentType, data)
	if err != nil {
		return nil, err
	}
	if err := s.landRepo.UpdateCompensationReport(ctx, compID, stored.URL, stored.Name); err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return nil, err
	}
	comp.ValuationReportURL = stored.URL
	comp.ValuationReportName = stored.Name
	return comp, nil
}

func (s *landService) ListCompensationHistory(ctx context.Context, compensationID uuid.UUID) ([]*domain.CompensationHistory, error) {
	return s.landRepo.ListCompensationHistory(ctx, compensationID)
}

func (s *landService) CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	if err := s.landRepo.CreateCompensationGrant(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *landService) UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	if err := s.landRepo.UpdateCompensationGrant(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *landService) GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error) {
	return s.landRepo.GetCompensationGrant(ctx, id)
}

func (s *landService) DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error {
	return s.landRepo.DeleteCompensationGrant(ctx, id)
}

func (s *landService) DeleteCompensationGrantByCompensationID(ctx context.Context, compensationID uuid.UUID) error {
	return s.landRepo.DeleteCompensationGrantByCompensationID(ctx, compensationID)
}

// ListParcelPayments — нэгж талбарын төлбөрийн бичлэгүүд
func (s *landService) ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error) {
	list, err := s.landRepo.ListParcelPayments(ctx, parcelID)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return list, nil
}

// AddParcelPayment — нэгж талбарт шинэ төлбөрийн бичлэг нэмэх
func (s *landService) AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error {
	if err := s.landRepo.AddParcelPayment(ctx, p); err != nil {
		return domain.ErrInternalServer
	}
	return nil
}

func (s *landService) ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error) {
	return s.landRepo.ListAcquisitionDocuments(ctx, acquisitionID)
}

func (s *landService) UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error) {
	stored, err := s.files.Save(ctx, "acquisition", acquisitionID.String(), filename, contentType, data)
	if err != nil {
		return nil, err
	}
	doc := &domain.Document{
		ID:             uuid.New(),
		Name:           stored.Name,
		FileType:       stored.Type,
		FileURL:        stored.URL,
		SizeBytes:      stored.SizeBytes,
		UploadedBy:     uploadedBy,
		UploadedAt:     time.Now(),
		DocumentTypeID: documentTypeID,
	}
	if err := s.landRepo.AddAcquisitionDocument(ctx, acquisitionID, doc); err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return nil, err
	}
	return doc, nil
}

func (s *landService) DeleteAcquisitionDocument(ctx context.Context, documentID uuid.UUID) error {
	doc, err := s.landRepo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.ErrNotFound
	}
	if err := s.landRepo.DeleteDocument(ctx, documentID); err != nil {
		return err
	}
	_ = s.files.Delete(ctx, doc.FileURL)
	return nil
}

func (s *landService) ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error) {
	return s.landRepo.ListParcelDocuments(ctx, parcelID)
}

func (s *landService) UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error) {
	stored, err := s.files.Save(ctx, "parcel", parcelID.String(), filename, contentType, data)
	if err != nil {
		return nil, err
	}
	doc := &domain.Document{
		ID:         uuid.New(),
		Name:       stored.Name,
		FileType:   stored.Type,
		FileURL:    stored.URL,
		SizeBytes:  stored.SizeBytes,
		UploadedBy: uploadedBy,
		UploadedAt: time.Now(),
	}
	if err := s.landRepo.AddParcelDocument(ctx, parcelID, doc); err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return nil, err
	}
	return doc, nil
}

func (s *landService) DeleteParcelDocument(ctx context.Context, documentID uuid.UUID) error {
	doc, err := s.landRepo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.ErrNotFound
	}
	if err := s.landRepo.DeleteDocument(ctx, documentID); err != nil {
		return err
	}
	_ = s.files.Delete(ctx, doc.FileURL)
	return nil
}
