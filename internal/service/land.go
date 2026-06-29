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
	ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error)
	ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error)
	CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error)
	UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error)
	DeleteAcquisitionCategory(ctx context.Context, id int) error
	ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error)
	Create(ctx context.Context, req CreateLandAcquisitionRequest) (*domain.LandAcquisition, error)
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, bool, error)
	ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error)
	UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error
	SyncParcel(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) error
	GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error)
	AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy, decreeNumber string, decreeDate *time.Time) (*domain.LandAcquisition, error)
	UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error)
	ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error)
	CreateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error)
	UpdateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error)
	DeleteAsset(ctx context.Context, id uuid.UUID) error
	// Compensation
	ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.Compensation, error)
	CreateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error)
	UpdateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error)
	DeleteCompensation(ctx context.Context, id uuid.UUID) error
	ApproveCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error
	RejectCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error
	UploadCompensationReport(ctx context.Context, acqID, compID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Compensation, error)
	ListCompensationHistory(ctx context.Context, compensationID uuid.UUID) ([]*domain.CompensationHistory, error)
	// CompensationGrant
	CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error)
	UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error)
	GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error)
	DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error
	DeleteCompensationGrantByCompensationID(ctx context.Context, compensationID uuid.UUID) error
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error)
	DeleteAcquisitionDocument(ctx context.Context, documentID uuid.UUID) error
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error)
	DeleteParcelDocument(ctx context.Context, documentID uuid.UUID) error
	GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error)
	Suggest(ctx context.Context, q string) ([]*domain.LandAcquisition, error)
	// Assignee
	SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error
	GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error)
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error)
	// Parcel workflow
	ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error)
	CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error)
	DeleteParcelWorkflow(ctx context.Context, id int) error
	GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error)
	UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error
	// Parcel status history
	ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error)
	// FundingSource
	ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error)
	CreateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error)
	UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error)
	DeleteFundingSource(ctx context.Context, id uuid.UUID) error
	// AuthorizedRepresentative
	ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error)
	CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) (*domain.AuthorizedRepresentative, error)
	DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error
	// AcquisitionWorkflow
	ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error)
	CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error)
	DeleteAcquisitionWorkflow(ctx context.Context, id int) error
	// GlobalCompensation
	ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error)
	GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error)
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
func (s *landService) ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error) {
	return s.landRepo.ListConstructionTypes(ctx)
}

func (s *landService) ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error) {
	return s.landRepo.ListAcquisitionCategories(ctx, parentID)
}

func (s *landService) CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error) {
	return s.landRepo.CreateAcquisitionCategory(ctx, name, parentID, sortOrder)
}

func (s *landService) UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error) {
	return s.landRepo.UpdateAcquisitionCategory(ctx, id, name, sortOrder)
}

func (s *landService) DeleteAcquisitionCategory(ctx context.Context, id int) error {
	return s.landRepo.DeleteAcquisitionCategory(ctx, id)
}

func (s *landService) ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error) {
	return s.landRepo.ListParcelStatuses(ctx)
}

// 3. PostGIS-аар чөлөөлөх хил төлөвлөгөөний хилтэй давхцаж байгааг шалгана
// 4. land_acquisition үүсгэнэ
// 5. Огтлолцох баг (au3) олж холбоно
// 6. auth DB-аас нэгж талбар татаж, давхцахыг шалгаж main DB-д хадгална
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

	// 3. Хилийн шалгалт: төлөвлөгөөний хилээс илүү гарч болно, гэхдээ бодит талбайгаар давхцах ёстой
	overlaps, err := s.landRepo.ValidateOverlap(ctx, acqWKT, plan.BoundaryWKT)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	if !overlaps {
		return nil, domain.ErrInvalidInput
	}

	// 4. land_acquisition үүсгэх
	now := time.Now()
	planIDCopy := plan.ID
	acq := &domain.LandAcquisition{
		ID:                uuid.New(),
		PlanID:            &planIDCopy,
		PlanCode:          plan.PlanCode,
		PlanName:          plan.Name,
		PlanGeomWKT:       plan.BoundaryWKT,
		PlanAreaM2:        plan.AreaM2,
		GeometryWKT:       acqWKT,
		Status:            domain.LandAcquisitionStatusNew,
		StartDate:         &req.StartDate,
		EndDate:           req.EndDate,
		AcquisitionName:   req.AcquisitionName,
		ImplementingOrg:   req.ImplementingOrg,
		Reason:            req.Reason,
		ResponsibleOrg:    req.ResponsibleOrg,
		FundingSource:     req.FundingSource,
		GeneralCategoryID: req.GeneralCategoryID,
		SubCategoryID:     req.SubCategoryID,
		CreatedAt:         now,
		CreatedBy:         req.CreatedBy,
		UpdatedAt:         now,
		UpdatedBy:         req.CreatedBy,
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
// (true үед шинэ хилээр дутуу нэгж талбар нэмэгдсэн тухай анхааруулга харуулна).
func (s *landService) Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, bool, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return nil, false, domain.ErrNotFound
	}

	geometryChanged := false
	oldGeometryWKT := acq.GeometryWKT

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

		overlaps, err := s.landRepo.ValidateOverlap(ctx, newWKT, plan.BoundaryWKT)
		if err != nil {
			return nil, false, domain.ErrInternalServer
		}
		if !overlaps {
			return nil, false, domain.ErrInvalidInput
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
	acq.GeneralCategoryID = req.GeneralCategoryID
	acq.SubCategoryID = req.SubCategoryID
	acq.UpdatedBy = req.UpdatedBy
	// Shapefile байхгүй үед гараас оруулсан талбайн утгыг хэрэглэнэ
	if len(req.ShapefileData) == 0 && req.AreaM2 != nil {
		acq.AreaM2 = *req.AreaM2
	}

	if err := s.landRepo.Update(ctx, acq); err != nil {
		return nil, false, domain.ErrInternalServer
	}

	if geometryChanged {
		if err := s.landRepo.CreateBoundaryHistory(ctx, &domain.LandAcquisitionBoundaryHistory{
			ID:                uuid.New(),
			LandAcquisitionID: acq.ID,
			OldGeometryWKT:    oldGeometryWKT,
			NewGeometryWKT:    acq.GeometryWKT,
			ChangedBy:         req.UpdatedBy,
			ChangedAt:         time.Now(),
		}); err != nil {
			return nil, false, domain.ErrInternalServer
		}

		// Шинэ хилтэй огтлолцох AU-г шинэчилж, зөвхөн шинээр нэмэгдэх нэгж талбаруудыг холбоно.
		if err := s.landRepo.DeleteAUs(ctx, acq.ID); err != nil {
			return nil, false, domain.ErrInternalServer
		}
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

func (s *landService) ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error) {
	return s.landRepo.ListBoundaryHistory(ctx, acquisitionID)
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

func (s *landService) UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error {
	if err := s.landRepo.UpdateParcelValuation(ctx, parcelID, zone, basePricePerHa, auctionCoeff, auctionPrice); err != nil {
		return domain.ErrInternalServer
	}
	return nil
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

// GetAvailableStatuses — одоогийн статусаас шилжих боломжтой статусуудыг буцаана.
// DB-д ажлын урсгал тохируулсан бол тэрийг ашиглана, эс бол hard-coded эрэмбийг ашиглана.
func (s *landService) GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	rules, _ := s.landRepo.ListAcquisitionWorkflows(ctx)
	if len(rules) > 0 {
		var opts []domain.StatusOption
		for _, r := range rules {
			if r.FromStatusID != nil && *r.FromStatusID == acq.Status {
				opts = append(opts, domain.StatusOption{
					ID:    r.ToStatusID,
					Label: domain.AcquisitionStatusLabels[r.ToStatusID],
				})
			}
		}
		return opts, nil
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

// AdvanceStatus — сонгосон статус руу шилжүүлнэ.
// DB-д ажлын урсгал тохируулсан бол тэрийг шалгана, эс бол hard-coded дүрмийг ашиглана.
func (s *landService) AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy, decreeNumber string, decreeDate *time.Time) (*domain.LandAcquisition, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	rules, _ := s.landRepo.ListAcquisitionWorkflows(ctx)
	if len(rules) > 0 {
		allowed := false
		for _, r := range rules {
			if r.FromStatusID != nil && *r.FromStatusID == acq.Status && r.ToStatusID == toStatus {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, domain.ErrInvalidInput
		}
	} else {
		if toStatus <= acq.Status {
			return nil, domain.ErrInvalidInput
		}
		if _, ok := domain.AcquisitionStatusLabels[toStatus]; !ok {
			return nil, domain.ErrInvalidInput
		}
	}

	// status=3 (Баталгаажсан) болоход НЗД-ын захирамжийн мэдээлэл заавал шаардагдана
	if toStatus == 3 && (decreeNumber == "" || decreeDate == nil) {
		return nil, domain.ErrDecreeRequired
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

	if decreeNumber != "" && decreeDate != nil {
		if err := s.landRepo.UpdateDecree(ctx, acquisitionID, decreeNumber, decreeDate); err != nil {
			return nil, domain.ErrInternalServer
		}
		acq.DecreeNumber = decreeNumber
		acq.DecreeDate = decreeDate
	}

	acq.Status = toStatus
	return acq, nil
}

// UpdateParcelMeta — нэгж талбарын мэдээллийн сангийн өөрчлөлтийн мэдээллийг шинэчлэнэ
func (s *landService) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
	return s.landRepo.UpdateParcelMeta(ctx, parcelID, dbChanged, changedParcelID, acquisitionAreaM2, acquisitionGeomWKT)
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

// ListReportParcels — тайлангийн нэгж талбарын жагсаалт (нөхөн төлбөр, эзэмшигч нэгтгэсэн)
func (s *landService) ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error) {
	list, total, err := s.landRepo.ListReportParcels(ctx, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

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

// SyncParcel — нэг нэгж талбарын дэлгэрэнгүй мэдээллийг auth DB-аас татаж хадгална.
// Чөлөөлөлтийн бүстэй давхцах геометрийг PostGIS-р тооцоолж парцел дээр хадгална.
func (s *landService) SyncParcel(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) error {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return domain.ErrNotFound
	}

	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return domain.ErrNotFound
	}

	if err := s.landRepo.UpdateParcelAcquisitionGeom(ctx, p.ID, acq.GeometryWKT); err != nil {
		return domain.ErrInternalServer
	}

	src, err := s.planRepo.GetSourceParcelByID(ctx, parcelCode)
	if err != nil {
		return domain.ErrNotFound
	}

	holderID, err := s.landRepo.UpsertHolder(ctx, src)
	if err != nil {
		return domain.ErrInternalServer
	}

	return s.landRepo.UpsertParcelRight(ctx, p.ID, acq.ID, holderID, src)
}

func (s *landService) GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	return s.landRepo.GetDashboard(ctx, filter)
}

func (s *landService) Suggest(ctx context.Context, q string) ([]*domain.LandAcquisition, error) {
	return s.landRepo.Suggest(ctx, q)
}

func (s *landService) SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return domain.ErrNotFound
	}
	return s.landRepo.SetAssignees(ctx, acquisitionID, assignees)
}

func (s *landService) GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error) {
	return s.landRepo.GetAssignees(ctx, acquisitionID)
}

func (s *landService) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.landRepo.IsAssigned(ctx, acquisitionID, userID)
}

func (s *landService) ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error) {
	return s.landRepo.ListParcelWorkflows(ctx)
}

func (s *landService) CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error) {
	return s.landRepo.CreateParcelWorkflow(ctx, fromStatusID, toStatusID, sortOrder)
}

func (s *landService) DeleteParcelWorkflow(ctx context.Context, id int) error {
	return s.landRepo.DeleteParcelWorkflow(ctx, id)
}

func (s *landService) GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error) {
	return s.landRepo.GetAvailableParcelStatuses(ctx, currentStatusID)
}

func (s *landService) UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	return s.landRepo.UpdateParcelStatus(ctx, parcelID, acquisitionID, statusID, createdBy)
}

func (s *landService) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	return s.landRepo.ListParcelStatusHistory(ctx, parcelID)
}

func (s *landService) ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error) {
	return s.landRepo.ListFundingSources(ctx, acquisitionID)
}

func (s *landService) CreateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	if err := s.landRepo.CreateFundingSource(ctx, fs); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *landService) UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	if err := s.landRepo.UpdateFundingSource(ctx, fs); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *landService) DeleteFundingSource(ctx context.Context, id uuid.UUID) error {
	return s.landRepo.DeleteFundingSource(ctx, id)
}

func (s *landService) ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error) {
	return s.landRepo.ListAuthorizedRepresentatives(ctx, acquisitionID, parcelID)
}

func (s *landService) CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) (*domain.AuthorizedRepresentative, error) {
	if err := s.landRepo.CreateAuthorizedRepresentative(ctx, rep); err != nil {
		return nil, err
	}
	return rep, nil
}

func (s *landService) DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error {
	return s.landRepo.DeleteAuthorizedRepresentative(ctx, id)
}

func (s *landService) ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error) {
	return s.landRepo.ListAcquisitionWorkflows(ctx)
}

func (s *landService) CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error) {
	return s.landRepo.CreateAcquisitionWorkflow(ctx, fromStatusID, toStatusID, sortOrder)
}

func (s *landService) DeleteAcquisitionWorkflow(ctx context.Context, id int) error {
	return s.landRepo.DeleteAcquisitionWorkflow(ctx, id)
}

func (s *landService) ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error) {
	return s.landRepo.ListAllCompensations(ctx, filter)
}

func (s *landService) GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error) {
	return s.landRepo.GetGlobalCompensation(ctx, id)
}
