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

func (s *landService) DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID) error {
	photoURL, err := s.landRepo.DeleteAsset(ctx, acquisitionID, id)
	if err != nil {
		return err
	}
	// Хавсаргасан зургийн PDF-ийг хадгалалтаас цэвэрлэнэ (үлдэгдэл файл үлдээхгүй).
	if photoURL != "" {
		_ = s.files.Delete(ctx, photoURL)
	}
	return nil
}

func (s *landService) UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error {
	return s.landRepo.UpsertAssetSpecs(ctx, acquisitionID, assetID, specs)
}

func (s *landService) ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error) {
	return s.landRepo.ListAssetSpecs(ctx, acquisitionID, assetID)
}

func (s *landService) UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error {
	return s.landRepo.UpsertAssetCalculations(ctx, acquisitionID, assetID, calcs)
}

func (s *landService) ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error) {
	return s.landRepo.ListAssetCalculations(ctx, acquisitionID, assetID)
}

func (s *landService) ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error) {
	return s.landRepo.ListAssetSpecTypes(ctx)
}

func (s *landService) CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	return s.landRepo.CreateAssetSpecType(ctx, t)
}

func (s *landService) UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	return s.landRepo.UpdateAssetSpecType(ctx, t)
}

func (s *landService) DeleteAssetSpecType(ctx context.Context, id int) error {
	return s.landRepo.DeleteAssetSpecType(ctx, id)
}

func (s *landService) ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error) {
	return s.landRepo.ListAssetCalcTypes(ctx)
}

func (s *landService) CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	return s.landRepo.CreateAssetCalcType(ctx, t)
}

func (s *landService) UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	return s.landRepo.UpdateAssetCalcType(ctx, t)
}

func (s *landService) DeleteAssetCalcType(ctx context.Context, id int) error {
	return s.landRepo.DeleteAssetCalcType(ctx, id)
}

func (s *landService) UpsertLandValuation(ctx context.Context, v *domain.LandValuation) (*domain.LandValuation, error) {
	if err := s.landRepo.UpsertLandValuation(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *landService) GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error) {
	return s.landRepo.GetLandValuation(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error {
	return s.landRepo.DeleteLandValuation(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error) {
	return s.landRepo.ImportValuation(ctx, acquisitionID, imp)
}

func (s *landService) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error) {
	return s.landRepo.ListCompensations(ctx, acquisitionID, parcelID, valuationType)
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
	if existing.AcquisitionID != c.AcquisitionID {
		return nil, domain.ErrNotFound
	}
	if existing.Status == "approved" {
		return nil, domain.ErrApproved
	}
	if err := s.landRepo.UpdateCompensation(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *landService) DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error {
	existing, err := s.landRepo.GetCompensation(ctx, id)
	if err != nil {
		return err
	}
	if existing.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}
	if existing.Status == "approved" {
		return domain.ErrApproved
	}
	return s.landRepo.DeleteCompensation(ctx, acquisitionID, id)
}

func (s *landService) ApproveCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, id); err != nil {
		return err
	}
	return s.landRepo.ApproveCompensation(ctx, id, note, reviewedBy)
}

func (s *landService) RejectCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, id); err != nil {
		return err
	}
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

func (s *landService) UploadAssetPhoto(ctx context.Context, acqID, assetID uuid.UUID, filename, contentType string, data []byte) (string, string, error) {
	stored, err := s.files.Save(ctx, "asset-photo", assetID.String(), filename, contentType, data)
	if err != nil {
		return "", "", err
	}
	// UpdateAssetPhoto нь асеет уг чөлөөлөлтөд хамаарахыг шалгана (WHERE id AND acquisition_id).
	if err := s.landRepo.UpdateAssetPhoto(ctx, acqID, assetID, stored.URL, stored.Name); err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return "", "", err
	}
	return stored.URL, stored.Name, nil
}

func (s *landService) ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error) {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, compensationID); err != nil {
		return nil, err
	}
	return s.landRepo.ListCompensationHistory(ctx, acquisitionID, compensationID)
}

func (s *landService) CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	if err := s.ensureCompensationInAcquisition(ctx, g.AcquisitionID, g.CompensationID); err != nil {
		return nil, err
	}
	if err := s.landRepo.CreateCompensationGrant(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *landService) UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	if err := s.ensureCompensationInAcquisition(ctx, g.AcquisitionID, g.CompensationID); err != nil {
		return nil, err
	}
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

func (s *landService) DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, compensationID); err != nil {
		return err
	}
	return s.landRepo.DeleteCompensationGrantByCompensationID(ctx, acquisitionID, compensationID)
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

func (s *landService) DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error {
	doc, err := s.landRepo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.ErrNotFound
	}
	if err := s.landRepo.DeleteAcquisitionDocument(ctx, acquisitionID, documentID); err != nil {
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

func (s *landService) DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error {
	doc, err := s.landRepo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.ErrNotFound
	}
	if err := s.landRepo.DeleteParcelDocument(ctx, parcelID, documentID); err != nil {
		return err
	}
	_ = s.files.Delete(ctx, doc.FileURL)
	return nil
}

func (s *landService) ensureCompensationInAcquisition(ctx context.Context, acquisitionID, compensationID uuid.UUID) error {
	comp, err := s.landRepo.GetCompensation(ctx, compensationID)
	if err != nil {
		return err
	}
	if comp.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}
	return nil
}
