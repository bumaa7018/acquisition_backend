package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/filestorage"
	"github.com/your-org/go-api-template/internal/pkg/shapefile"
	"github.com/your-org/go-api-template/internal/repository"
)

type LandService interface {
	Create(ctx context.Context, req CreateLandAcquisitionRequest) (*domain.LandAcquisition, error)
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error)
	SyncParcel(ctx context.Context, acquisitionID, parcelID uuid.UUID) error
	GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error)
	AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy string) (*domain.LandAcquisition, error)
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error)
	DeleteAcquisitionDocument(ctx context.Context, documentID uuid.UUID) error
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error)
	DeleteParcelDocument(ctx context.Context, documentID uuid.UUID) error
}

type landService struct {
	planRepo repository.PlanRepository
	landRepo repository.LandAcquisitionRepository
	files    filestorage.Storage
}

func NewLandService(
	planRepo repository.PlanRepository,
	landRepo repository.LandAcquisitionRepository,
	storages ...filestorage.Storage,
) LandService {
	var files filestorage.Storage = filestorage.NewMockStorage("http://localhost:8080/mock-files", "uploads/mock-file-server")
	if len(storages) > 0 && storages[0] != nil {
		files = storages[0]
	}
	return &landService{planRepo: planRepo, landRepo: landRepo, files: files}
}

// Create — газар чөлөөлөлтын мэдээлэл үүсгэх
//
// Алхам:
//  1. auth DB-аас plan_code-р төлөвлөгөө хайна
//  2. Shapefile zip задалж polygon WKT гаргана
//  3. PostGIS-аар чөлөөлөх хилийг төлөвлөгөөний хилтэй шалгана
//  4. land_acquisition үүсгэнэ
//  5. Огтлолцох баг (au3) олж холбоно
//  6. auth DB-аас нэгж талбар татаж, давхцахыг шалгаж main DB-д хадгална
func (s *landService) Create(ctx context.Context, req CreateLandAcquisitionRequest) (*domain.LandAcquisition, error) {
	// 1. Төлөвлөгөө хайх (auth DB)
	plan, err := s.planRepo.GetByCode(ctx, req.PlanCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// 2. Shapefile задлах
	acqWKT, err := shapefile.Parse(req.ShapefileData)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	// 3. Хилийн шалгалт: чөлөөлөх хил ⊂ төлөвлөгөөний хил
	within, err := s.landRepo.ValidateWithin(ctx, acqWKT, plan.BoundaryWKT)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	if !within {
		return nil, domain.ErrInvalidInput
	}

	// 4. land_acquisition үүсгэх
	now := time.Now()
	planIDCopy := plan.ID
	acq := &domain.LandAcquisition{
		ID:              uuid.New(),
		PlanID:          &planIDCopy,
		PlanCode:        plan.PlanCode,
		PlanName:        plan.Name,
		PlanGeomWKT:     plan.BoundaryWKT,
		PlanAreaM2:      plan.AreaM2,
		GeometryWKT:     acqWKT,
		Status:          domain.LandAcquisitionStatusNew,
		StartDate:       &req.StartDate,
		EndDate:         &req.EndDate,
		AcquisitionName: req.AcquisitionName,
		ImplementingOrg: req.ImplementingOrg,
		Reason:          req.Reason,
		ResponsibleOrg:  req.ResponsibleOrg,
		FundingSource:   req.FundingSource,
		CreatedAt:       now,
		CreatedBy:       req.CreatedBy,
		UpdatedAt:       now,
		UpdatedBy:       req.CreatedBy,
	}

	if err := s.landRepo.Create(ctx, acq); err != nil {
		return nil, domain.ErrInternalServer
	}

	_ = s.landRepo.CreateProgress(ctx, &domain.AcquisitionProgress{
		ID:            uuid.New(),
		AcquisitionID: acq.ID,
		FromStatus:    0,
		ToStatus:      domain.LandAcquisitionStatusNew,
		Note:          "Үүсгэгдсэн",
		ChangedBy:     req.CreatedBy,
		ChangedAt:     now,
	})

	// 5. Огтлолцох баг олж холбох
	aus, err := s.landRepo.FindIntersectingAU3(ctx, acqWKT)
	if err == nil {
		for i := range aus {
			aus[i].LandAcquisitionID = acq.ID
			_ = s.landRepo.CreateAU(ctx, &aus[i])
		}
		acq.AUs = aus
	}

	// 6. Нэгж талбар татаж хадгалах (auth DB → main DB)
	srcParcels, err := s.planRepo.GetSourceParcels(ctx, plan.PlanCode)
	if err == nil && len(srcParcels) > 0 {
		_ = s.landRepo.BulkInsertParcels(ctx, acq.ID, srcParcels, acqWKT)
	}

	return acq, nil
}

// Update — газар чөлөөлөлтийн мэдээлэл шинэчлэх.
// Хоёрдахь буцах утга (bool) нь геометр өөрчлөгдсөн эсэхийг илэрхийлнэ
// (true үед хуучин нэгж талбарууд устгагдсан тухай анхааруулга харуулна).
func (s *landService) Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, bool, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return nil, false, domain.ErrNotFound
	}

	geometryChanged := false

	if len(req.ShapefileData) > 0 {
		// Шинэ план эсвэл шинэ хилийн файл өгсөн үед геометрийг шинэчилнэ
		targetPlanCode := acq.PlanCode
		if req.PlanCode != "" {
			targetPlanCode = req.PlanCode
		}

		plan, err := s.planRepo.GetByCode(ctx, targetPlanCode)
		if err != nil {
			return nil, false, domain.ErrNotFound
		}

		newWKT, err := shapefile.Parse(req.ShapefileData)
		if err != nil {
			return nil, false, domain.ErrInvalidInput
		}

		within, err := s.landRepo.ValidateWithin(ctx, newWKT, plan.BoundaryWKT)
		if err != nil {
			return nil, false, domain.ErrInternalServer
		}
		if !within {
			return nil, false, domain.ErrInvalidInput
		}

		// Хуучин нэгж талбар болон AU мэдээллийг устгана
		if err := s.landRepo.DeleteParcelsAndAUs(ctx, id); err != nil {
			return nil, false, domain.ErrInternalServer
		}
		geometryChanged = true

		planIDCopy := plan.ID
		acq.PlanID = &planIDCopy
		acq.PlanCode = plan.PlanCode
		acq.PlanName = plan.Name
		acq.PlanGeomWKT = plan.BoundaryWKT
		acq.PlanAreaM2 = plan.AreaM2
		acq.GeometryWKT = newWKT
	} else if req.PlanCode != "" {
		// Shapefile өгөөгүй атал план солихыг хүсвэл алдаа
		return nil, false, domain.ErrInvalidInput
	}

	if req.StartDate != nil {
		acq.StartDate = req.StartDate
	}
	if req.ClearEndDate {
		acq.EndDate = nil
	} else if req.EndDate != nil {
		acq.EndDate = req.EndDate
	}
	acq.AcquisitionName = req.AcquisitionName
	acq.ImplementingOrg = req.ImplementingOrg
	acq.Reason = req.Reason
	acq.ResponsibleOrg = req.ResponsibleOrg
	acq.FundingSource = req.FundingSource
	acq.UpdatedBy = req.UpdatedBy

	if err := s.landRepo.Update(ctx, acq); err != nil {
		return nil, false, domain.ErrInternalServer
	}

	if geometryChanged {
		// Шинэ хилтэй огтлолцох AU болон нэгж талбаруудыг дахин холбоно
		aus, err := s.landRepo.FindIntersectingAU3(ctx, acq.GeometryWKT)
		if err == nil {
			for i := range aus {
				aus[i].LandAcquisitionID = acq.ID
				_ = s.landRepo.CreateAU(ctx, &aus[i])
			}
			acq.AUs = aus
		}
		srcParcels, err := s.planRepo.GetSourceParcels(ctx, acq.PlanCode)
		if err == nil && len(srcParcels) > 0 {
			_ = s.landRepo.BulkInsertParcels(ctx, acq.ID, srcParcels, acq.GeometryWKT)
		}
	}

	return acq, geometryChanged, nil
}

// Delete — зөвхөн "NEW" статустай чөлөөлөлтийг устгана.
func (s *landService) Delete(ctx context.Context, id uuid.UUID) error {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrNotFound
	}
	if acq.Status != domain.LandAcquisitionStatusNew {
		return domain.ErrForbidden
	}
	return s.landRepo.Delete(ctx, id)
}

func (s *landService) List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error) {
	list, total, err := s.landRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

func (s *landService) GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, 0, domain.ErrNotFound
	}
	parcels, total, err := s.landRepo.GetParcels(ctx, acquisitionID, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return parcels, total, nil
}

func (s *landService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return acq, nil
}

func (s *landService) GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	pf, err := s.landRepo.GetParcelDetailByID(ctx, parcelID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return pf, nil
}

// GetProgress — статус өөрчлөлтийн түүхийг буцаана
func (s *landService) GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	list, err := s.landRepo.ListProgress(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return list, nil
}

// GetAvailableStatuses — одоогийн статусаас шилжих боломжтой статусуудыг буцаана
func (s *landService) GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var opts []domain.StatusOption
	for _, st := range domain.AcquisitionStatusOrder {
		if st > acq.Status {
			opts = append(opts, domain.StatusOption{
				ID:    st,
				Label: domain.AcquisitionStatusLabels[st],
			})
		}
	}
	return opts, nil
}

// AdvanceStatus — сонгосон статус руу шилжүүлнэ
func (s *landService) AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy string) (*domain.LandAcquisition, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	if toStatus <= acq.Status {
		return nil, domain.ErrInvalidInput
	}
	if _, ok := domain.AcquisitionStatusLabels[toStatus]; !ok {
		return nil, domain.ErrInvalidInput
	}

	progress := &domain.AcquisitionProgress{
		ID:            uuid.New(),
		AcquisitionID: acquisitionID,
		FromStatus:    acq.Status,
		ToStatus:      toStatus,
		Note:          note,
		ChangedBy:     changedBy,
		ChangedAt:     time.Now(),
	}
	if err := s.landRepo.CreateProgress(ctx, progress); err != nil {
		return nil, domain.ErrInternalServer
	}

	if err := s.landRepo.UpdateStatus(ctx, acquisitionID, toStatus); err != nil {
		return nil, domain.ErrInternalServer
	}

	acq.Status = toStatus
	return acq, nil
}

// SetParcelCompensation — нэгж талбарын нөхөн төлбөрийн төлөв шинэчлэнэ
func (s *landService) SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error {
	return s.landRepo.SetParcelCompensation(ctx, parcelID, paid)
}

// ListGlobalParcels — бүх чөлөөлөлтийн нэгж талбарын нэгдсэн жагсаалт
func (s *landService) ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error) {
	list, total, err := s.landRepo.ListGlobalParcels(ctx, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
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

func (s *landService) UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error) {
	stored, err := s.files.Save(ctx, "acquisition", acquisitionID.String(), filename, contentType, data)
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

// SyncParcel — нэг нэгж талбарын дэлгэрэнгүй мэдээллийг auth DB-аас татаж хадгална.
// Чөлөөлөлтийн бүстэй давхцах геометрийг PostGIS-р тооцоолж парцел дээр хадгална.
func (s *landService) SyncParcel(ctx context.Context, acquisitionID, parcelID uuid.UUID) error {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return domain.ErrNotFound
	}

	p, err := s.landRepo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return domain.ErrNotFound
	}

	// Чөлөөлөлтийн бүстэй давхцах геометр тооцоолж хадгална
	if err := s.landRepo.UpdateParcelAcquisitionGeom(ctx, p.ID, acq.GeometryWKT); err != nil {
		return domain.ErrInternalServer
	}

	src, err := s.planRepo.GetSourceParcelByID(ctx, p.ParcelID)
	if err != nil {
		return domain.ErrNotFound
	}

	holderID, err := s.landRepo.UpsertHolder(ctx, src)
	if err != nil {
		return domain.ErrInternalServer
	}

	return s.landRepo.UpsertParcelRight(ctx, p.ID, acq.ID, holderID, src)
}
