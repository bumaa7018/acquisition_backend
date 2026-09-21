package service

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
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

func (s *landService) UpdateAsset(ctx context.Context, u *domain.AssetUpdate) (*domain.Asset, error) {
	return s.landRepo.UpdateAsset(ctx, u)
}

func (s *landService) GetAssetAcquisition(ctx context.Context, assetID uuid.UUID) (uuid.UUID, error) {
	return s.landRepo.GetAssetAcquisition(ctx, assetID)
}

func (s *landService) GetCompensationAcquisition(ctx context.Context, compID uuid.UUID) (uuid.UUID, error) {
	return s.landRepo.GetCompensationAcquisition(ctx, compID)
}

// DeleteAsset — allowedStreams хоосон бол хязгаарлалтгүй; эс бөгөөс зөвхөн
// дуудагчийн үнэлгээний урсгалын (asset/independent/mika) мөр устана.
func (s *landService) DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID, allowedStreams []string) error {
	photoURL, err := s.landRepo.DeleteAsset(ctx, acquisitionID, id, allowedStreams)
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
	s.auditLog(ctx, "upserted", domain.AuditResourceValuation, v.ParcelID, uuidPtr(v.AcquisitionID), nil, map[string]any{
		"parcel_id":      v.ParcelID,
		"valuation_type": v.ValuationType,
	})
	return v, nil
}

func (s *landService) GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error) {
	return s.landRepo.GetLandValuation(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error {
	if err := s.landRepo.DeleteLandValuation(ctx, acquisitionID, parcelID, valuationType); err != nil {
		return err
	}
	s.auditLog(ctx, "deleted", domain.AuditResourceValuation, parcelID, uuidPtr(acquisitionID), nil, map[string]any{
		"parcel_id":      parcelID,
		"valuation_type": valuationType,
	})
	return nil
}

// ListValuationNotes — нэгж талбарын үнэлгээний хүснэгт бүрийн тайлбар.
func (s *landService) ListValuationNotes(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.ValuationSectionNote, error) {
	return s.landRepo.ListValuationNotes(ctx, acquisitionID, parcelID, valuationType)
}

// SaveValuationNotes — тайлбаруудыг хадгалаад, шинэчилсэн бүрэн жагсаалтыг буцаана.
func (s *landService) SaveValuationNotes(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string, notes map[string]string, updatedBy string) ([]*domain.ValuationSectionNote, error) {
	if err := s.landRepo.UpsertValuationNotes(ctx, acquisitionID, parcelID, valuationType, notes, updatedBy); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(notes))
	for k := range notes {
		keys = append(keys, k)
	}
	s.auditLog(ctx, "updated", domain.AuditResourceValuation, parcelID, uuidPtr(acquisitionID), nil, map[string]any{
		"parcel_id":      parcelID,
		"valuation_type": valuationType,
		"section_keys":   keys,
	})
	return s.landRepo.ListValuationNotes(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error) {
	res, err := s.landRepo.ImportValuation(ctx, acquisitionID, imp)
	if err != nil {
		return nil, err
	}
	// Replace-ээр устгагдсан мөрүүдийн файлуудыг (зураг, тайлан) хадгалалтаас цэвэрлэнэ.
	for _, fileURL := range res.OrphanedFileURLs {
		_ = s.files.Delete(ctx, fileURL)
	}
	s.auditLog(ctx, "imported", domain.AuditResourceValuation, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"replace":             imp.Replace,
		"parcel_id":           imp.ParcelID,
		"valuation_type":      imp.ValuationType,
		"asset_count":         len(imp.Assets),
		"orphaned_file_count": len(res.OrphanedFileURLs),
		"created_assets":      res.CreatedAssets,
		"created_comps":       res.CreatedComps,
		"deleted_assets":      res.DeletedAssets,
		"deleted_comps":       res.DeletedComps,
	})
	return res, nil
}

func (s *landService) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error) {
	return s.landRepo.ListCompensations(ctx, acquisitionID, parcelID, valuationType)
}

func (s *landService) CreateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error) {
	if err := s.landRepo.CreateCompensation(ctx, c); err != nil {
		return nil, err
	}
	s.notifyCompensationCreated(ctx, c)
	s.auditLog(ctx, "created", domain.AuditResourceCompensation, c.ID.String(), uuidPtr(c.AcquisitionID), nil, map[string]any{
		"parcel_id":      c.ParcelID,
		"valuation_type": c.ValuationType,
		"amount":         c.Amount,
		"status":         c.Status,
	})
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
	s.auditLog(ctx, "updated", domain.AuditResourceCompensation, c.ID.String(), uuidPtr(c.AcquisitionID), nil, map[string]any{
		"parcel_id":       c.ParcelID,
		"valuation_type":  c.ValuationType,
		"amount":          c.Amount,
		"previous_status": existing.Status,
	})
	return c, nil
}

// DeleteCompensation — allowedStreams хоосон бол хязгаарлалтгүй; эс бөгөөс
// зөвхөн дуудагчийн үнэлгээний урсгалын мөр устана.
func (s *landService) DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID, allowedStreams []string) error {
	existing, err := s.landRepo.GetCompensation(ctx, id)
	if err != nil {
		return err
	}
	if existing.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}
	if !domain.ValuationStreamAllowed(existing.ValuationType, allowedStreams) {
		return domain.ErrForbidden
	}
	if existing.Status == "approved" {
		return domain.ErrApproved
	}
	if err := s.landRepo.DeleteCompensation(ctx, acquisitionID, id); err != nil {
		return err
	}
	// Хавсаргасан үнэлгээний тайлангийн файлыг хадгалалтаас цэвэрлэнэ.
	if existing.ValuationReportURL != "" {
		_ = s.files.Delete(ctx, existing.ValuationReportURL)
	}
	s.auditLog(ctx, "deleted", domain.AuditResourceCompensation, id.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"parcel_id": existing.ParcelID,
		"amount":    existing.Amount,
		"status":    existing.Status,
	})
	return nil
}

func (s *landService) ApproveCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, id); err != nil {
		return err
	}
	if err := s.landRepo.ApproveCompensation(ctx, id, note, reviewedBy); err != nil {
		return err
	}
	s.notifyCompensationReviewed(ctx, acquisitionID, id, true, note)
	s.auditLog(ctx, "approved", domain.AuditResourceCompensation, id.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"note":        note,
		"reviewed_by": reviewedBy,
	})
	return nil
}

func (s *landService) RejectCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, id); err != nil {
		return err
	}
	if err := s.landRepo.RejectCompensation(ctx, id, note, reviewedBy); err != nil {
		return err
	}
	s.notifyCompensationReviewed(ctx, acquisitionID, id, false, note)
	s.auditLog(ctx, "rejected", domain.AuditResourceCompensation, id.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"note":        note,
		"reviewed_by": reviewedBy,
	})
	return nil
}

func (s *landService) UploadCompensationReport(ctx context.Context, acqID, compID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Compensation, error) {
	comp, err := s.landRepo.GetCompensation(ctx, compID)
	if err != nil || comp.AcquisitionID != acqID {
		return nil, domain.ErrNotFound
	}
	// Үнэлгээний тайланг зөвхөн санхүү баталгаажуулсны (олговор approved) дараа хавсаргана.
	if comp.Status != "approved" {
		return nil, domain.ErrReportBeforeApproval
	}
	stored, err := s.files.Save(ctx, "compensation-report", compID.String(), filename, contentType, data, true)
	if err != nil {
		return nil, err
	}
	if err := s.landRepo.UpdateCompensationReport(ctx, compID, stored.URL, stored.Name); err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return nil, err
	}
	// Солигдсон хуучин тайлангийн файлыг цэвэрлэнэ — орфон файл үлдээхгүй.
	if comp.ValuationReportURL != "" && comp.ValuationReportURL != stored.URL {
		_ = s.files.Delete(ctx, comp.ValuationReportURL)
	}
	comp.ValuationReportURL = stored.URL
	comp.ValuationReportName = stored.Name
	s.auditLog(ctx, "report_uploaded", domain.AuditResourceCompensation, compID.String(), uuidPtr(acqID), nil, map[string]any{
		"filename":    stored.Name,
		"uploaded_by": uploadedBy,
	})
	return comp, nil
}

func (s *landService) UploadAssetPhoto(ctx context.Context, acqID, assetID uuid.UUID, filename, contentType string, data []byte) (string, string, error) {
	stored, err := s.files.Save(ctx, "asset-photo", assetID.String(), filename, contentType, data, true)
	if err != nil {
		return "", "", err
	}
	// UpdateAssetPhoto нь асеет уг чөлөөлөлтөд хамаарахыг шалгаж, солигдсон
	// хуучин зургийн URL-ийг буцаана.
	oldURL, err := s.landRepo.UpdateAssetPhoto(ctx, acqID, assetID, stored.URL, stored.Name)
	if err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return "", "", err
	}
	// Хуучин зургийг хадгалалтаас цэвэрлэнэ — replace үед орфон файл үлдээхгүй.
	if oldURL != "" && oldURL != stored.URL {
		_ = s.files.Delete(ctx, oldURL)
	}
	s.auditLog(ctx, "photo_uploaded", domain.AuditResourceParcel, assetID.String(), uuidPtr(acqID), nil, map[string]any{
		"asset_id": assetID,
		"filename": stored.Name,
	})
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
	s.auditLog(ctx, "grant_created", domain.AuditResourceCompensation, g.CompensationID.String(), uuidPtr(g.AcquisitionID), nil, map[string]any{
		"grant_id":      g.ID,
		"amount":        g.Amount,
		"grant_date":    g.GrantDate,
		"land_area_m2":  g.LandAreaM2,
		"parcel_number": g.ParcelNumber,
	})
	return g, nil
}

func (s *landService) UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	if err := s.ensureCompensationInAcquisition(ctx, g.AcquisitionID, g.CompensationID); err != nil {
		return nil, err
	}
	if err := s.landRepo.UpdateCompensationGrant(ctx, g); err != nil {
		return nil, err
	}
	s.auditLog(ctx, "grant_updated", domain.AuditResourceCompensation, g.CompensationID.String(), uuidPtr(g.AcquisitionID), nil, map[string]any{
		"grant_id":      g.ID,
		"amount":        g.Amount,
		"grant_date":    g.GrantDate,
		"land_area_m2":  g.LandAreaM2,
		"parcel_number": g.ParcelNumber,
	})
	return g, nil
}

func (s *landService) GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error) {
	return s.landRepo.GetCompensationGrant(ctx, id)
}

func (s *landService) DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error {
	if err := s.landRepo.DeleteCompensationGrant(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, "grant_deleted", domain.AuditResourceCompensation, id.String(), nil, nil, nil)
	return nil
}

func (s *landService) DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error {
	if err := s.ensureCompensationInAcquisition(ctx, acquisitionID, compensationID); err != nil {
		return err
	}
	if err := s.landRepo.DeleteCompensationGrantByCompensationID(ctx, acquisitionID, compensationID); err != nil {
		return err
	}
	s.auditLog(ctx, "grant_deleted", domain.AuditResourceCompensation, compensationID.String(), uuidPtr(acquisitionID), nil, nil)
	return nil
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
	s.auditLog(ctx, "payment_added", domain.AuditResourcePayment, p.ID.String(), nil, uuidPtr(p.ParcelID), map[string]any{
		"parcel_id": p.ParcelID,
		"amount":    p.Amount,
		"currency":  p.Currency,
		"paid_at":   p.PaidAt,
	})
	return nil
}

func (s *landService) ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error) {
	return s.landRepo.ListAcquisitionDocuments(ctx, acquisitionID)
}

// buildDocumentFileName — хавсралтын физик файлын нэр:
// <эзэмшигчийн дугаар>_<хавсралтын төрлийн код>.<өргөтгөл>.
// Ижил нэртэй файл байвал _2, _3 гэж чөлөөт дугаар олно (давхардал/дарагдахаас сэргийлнэ).
func buildDocumentFileName(ownerCode string, dt *domain.DocumentType, originalFilename string, existingDocs []*domain.Document) string {
	if dt == nil || ownerCode == "" {
		return originalFilename
	}
	taken := make(map[string]bool, len(existingDocs))
	for _, d := range existingDocs {
		segs := strings.Split(d.FileURL, "/")
		last := segs[len(segs)-1]
		if dec, err := url.PathUnescape(last); err == nil {
			last = dec
		}
		taken[last] = true
	}
	ext := filepath.Ext(originalFilename)
	base := ownerCode + "_" + dt.Type
	name := base + ext
	for i := 2; taken[name]; i++ {
		name = base + "_" + strconv.Itoa(i) + ext
	}
	return name
}

// documentDisplayName — дэлгэцэд харагдах нэр: хэрэглэгчийн өгсөн нэр,
// эс бөгөөс хавсралтын төрлийн нэр, эс бөгөөс эх файлын нэр.
func documentDisplayName(displayName string, dt *domain.DocumentType, fallback string) string {
	if n := strings.TrimSpace(displayName); n != "" {
		return n
	}
	if dt != nil {
		return dt.Name
	}
	return fallback
}

func (s *landService) UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error) {
	// Физик нэр: <төлөвлөгөөний код>_<төрлийн код>.<өргөтгөл>
	var dt *domain.DocumentType
	storedFilename := filename
	if documentTypeID > 0 {
		t, err := s.landRepo.GetDocumentType(ctx, documentTypeID)
		if err != nil {
			if err == domain.ErrNotFound {
				return nil, domain.ErrInvalidInput
			}
			return nil, err
		}
		dt = t
		acq, err := s.landRepo.GetByID(ctx, acquisitionID)
		if err != nil {
			return nil, err
		}
		docs, err := s.landRepo.ListAcquisitionDocuments(ctx, acquisitionID)
		if err != nil {
			return nil, err
		}
		storedFilename = buildDocumentFileName(acq.PlanCode, dt, filename, docs)
	}
	stored, err := s.files.Save(ctx, "acquisition", acquisitionID.String(), storedFilename, contentType, data, dt == nil)
	if err != nil {
		return nil, err
	}
	doc := &domain.Document{
		ID:             uuid.New(),
		Name:           documentDisplayName(displayName, dt, stored.Name),
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
	s.auditLog(ctx, "uploaded", domain.AuditResourceDocument, doc.ID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"name":             doc.Name,
		"document_type_id": documentTypeID,
		"uploaded_by":      uploadedBy,
	})
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
	s.auditLog(ctx, "deleted", domain.AuditResourceDocument, documentID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"name": doc.Name,
	})
	return nil
}

func (s *landService) ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error) {
	return s.landRepo.ListParcelDocuments(ctx, parcelID)
}

// GetDocumentType — handler давхаргад хавсралтын төрлөөр (жишээ нь "meeting_minutes")
// зөвшөөрөгдөх файлын агуулгын төрлийг (PDF/DOCX) шийдэхэд ашиглана.
func (s *landService) GetDocumentType(ctx context.Context, id int) (*domain.DocumentType, error) {
	return s.landRepo.GetDocumentType(ctx, id)
}

func (s *landService) UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error) {
	// Физик нэр: <нэгж талбарын дугаар>_<төрлийн код>.<өргөтгөл>
	var dt *domain.DocumentType
	storedFilename := filename
	if documentTypeID > 0 {
		t, err := s.landRepo.GetDocumentType(ctx, documentTypeID)
		if err != nil {
			if err == domain.ErrNotFound {
				return nil, domain.ErrInvalidInput
			}
			return nil, err
		}
		dt = t
		parcel, err := s.landRepo.GetParcelByID(ctx, parcelID)
		if err != nil {
			return nil, err
		}
		docs, err := s.landRepo.ListParcelDocuments(ctx, parcelID)
		if err != nil {
			return nil, err
		}
		storedFilename = buildDocumentFileName(parcel.ParcelID, dt, filename, docs)
	}
	// Archived reports retain their object URL after the live document is removed.
	unique := dt == nil || dt.Type == "valuation_report"
	stored, err := s.files.Save(ctx, "parcel", parcelID.String(), storedFilename, contentType, data, unique)
	if err != nil {
		return nil, err
	}
	doc := &domain.Document{
		ID:             uuid.New(),
		Name:           documentDisplayName(displayName, dt, stored.Name),
		FileType:       stored.Type,
		FileURL:        stored.URL,
		SizeBytes:      stored.SizeBytes,
		UploadedBy:     uploadedBy,
		UploadedAt:     time.Now(),
		DocumentTypeID: documentTypeID,
	}
	if err := s.landRepo.AddParcelDocument(ctx, parcelID, doc); err != nil {
		_ = s.files.Delete(ctx, stored.URL)
		return nil, err
	}
	s.auditLog(ctx, "uploaded", domain.AuditResourceDocument, doc.ID.String(), nil, uuidPtr(parcelID), map[string]any{
		"name":             doc.Name,
		"document_type_id": documentTypeID,
		"uploaded_by":      uploadedBy,
	})
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
	s.auditLog(ctx, "deleted", domain.AuditResourceDocument, documentID.String(), nil, uuidPtr(parcelID), map[string]any{
		"name": doc.Name,
	})
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
