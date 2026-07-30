package service

import (
	"context"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/pkg/shapefile"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

type LandLookupService interface {
	ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error)
	ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error)
	CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error)
	UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error)
	DeleteAcquisitionCategory(ctx context.Context, id int) error
	ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error)
}

type LandAcquisitionService interface {
	Create(ctx context.Context, req CreateLandAcquisitionRequest) (*domain.LandAcquisition, error)
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	IsAcquisitionLocked(ctx context.Context, id uuid.UUID) (bool, error)
	IsParcelAcquisitionLocked(ctx context.Context, parcelID uuid.UUID) (bool, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, bool, error)
	ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error)
	// EnsureParcelInAcquisition — parcel тухайн чөлөөлөлтөд харьяалагдаж буйг
	// шалгана (IDOR-оос сэргийлж, parcel-ийн үйлдлийг зөв acquisition-д хязгаарлана).
	EnsureParcelInAcquisition(ctx context.Context, acquisitionID, parcelID uuid.UUID) error
	UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error
	SyncParcel(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) error
	GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error)
	AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy, decreeNumber string, decreeDate *time.Time) (*domain.LandAcquisition, error)
	UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error)
	GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error)
	GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error)
	Suggest(ctx context.Context, q string, status int) ([]*domain.LandAcquisition, error)
	SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *uuid.UUID) error
	SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *uuid.UUID) error
}

type LandAssetService interface {
	ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error)
	CreateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error)
	UpdateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error)
	DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID) error
	UploadAssetPhoto(ctx context.Context, acqID, assetID uuid.UUID, filename, contentType string, data []byte) (photoURL, photoName string, err error)
	UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error
	ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error)
	UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error
	ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error)
	ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error)
	CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error
	UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error
	DeleteAssetSpecType(ctx context.Context, id int) error
	ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error)
	CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error
	UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error
	DeleteAssetCalcType(ctx context.Context, id int) error
	UpsertLandValuation(ctx context.Context, v *domain.LandValuation) (*domain.LandValuation, error)
	GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error)
	DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error
	ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error)
}

type LandCompensationService interface {
	ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error)
	CreateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error)
	UpdateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error)
	DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error
	ApproveCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error
	RejectCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error
	UploadCompensationReport(ctx context.Context, acqID, compID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Compensation, error)
	ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error)
	CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error)
	UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error)
	GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error)
	DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error
	DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error
	ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error)
	GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error)
}

type LandPaymentService interface {
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
}

type LandDocumentService interface {
	// GetDocumentType — хавсралт байршуулах үед зөвшөөрөгдөх файлын
	// агуулгын төрлийг (жишээ нь "meeting_minutes" бол PDF-ээс гадна DOCX)
	// тодорхойлоход ашиглана.
	GetDocumentType(ctx context.Context, id int) (*domain.DocumentType, error)
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error)
	DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error)
	DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error
}

type LandAssigneeService interface {
	SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error
	GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error)
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error)
	IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error)
	IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error)
	IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error)
	// IsProfMemberOfParcelByCode — parcel_id (код)-оор parcel-ийг олж, тухайн хэрэглэгч
	// уг acquisition-д харьяалагдах parcel-ийн гишүүн мөн эсэхийг шалгана.
	// ImportValuation зэрэг parcel_id-г JSON body-гоор хүлээн авдаг route-уудад ашиглана
	// (URL param биш тул профParcelMemberByParam middleware-ээр шалгах боломжгүй).
	IsProfMemberOfParcelByCode(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, userID uuid.UUID) (bool, error)
}

type LandWorkflowService interface {
	ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error)
	CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error)
	DeleteParcelWorkflow(ctx context.Context, id int) error
	GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error)
	UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error
	ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error)
	GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error)
	TransitionValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType, action, note, userID string) (*domain.ValuationSubmission, error)
	ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error)
	ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error)
	CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error)
	DeleteAcquisitionWorkflow(ctx context.Context, id int) error
}

// LandDroneService — дроны ортофото.
//
// Зураг нь ЗӨВХӨН файлын системд (MinIO) хадгалагдана: browser тийш шууд
// байршуулж, GeoServer тэндээс HTTP Range хүсэлтээр шууд уншина (COG). Backend
// растрыг өөр дээрээ хадгалахгүй — зөвхөн зам, хүрээ, давхаргыг зохицуулна.
type LandDroneService interface {
	ListDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error)
	// CreateDroneUploadURL — browser файлыг ШУУД файлын систем руу тавих
	// зөвшөөрөл. Байт API-аар дамжихгүй тул файлын хэмжээ хязгаарлагдахгүй.
	CreateDroneUploadURL(ctx context.Context, acquisitionID uuid.UUID, originalName string) (*filestorage.PresignedUpload, error)
	// RegisterDroneImage — шууд байршуулсан объектыг шалгаж бүртгэнэ.
	RegisterDroneImage(ctx context.Context, acquisitionID uuid.UUID, storedName, originalName string, uploadedBy *uuid.UUID) (*domain.DroneImage, error)
	RefreshDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error)
	DeleteDroneImage(ctx context.Context, acquisitionID, imageID uuid.UUID) error
	// DroneLayerName — frontend WMS дуудахад хэрэглэх давхаргын нэр ("land:drone")
	// DroneLayerName — зураг тус бүр өөрийн давхаргатай тул файлын нэрээс тооцно
	DroneLayerName(fileName string) string
}

type LandFundingService interface {
	ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error)
	CreateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error)
	UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error)
	DeleteFundingSource(ctx context.Context, id uuid.UUID) error
}

type LandAuthorizedRepresentativeService interface {
	ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error)
	CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) (*domain.AuthorizedRepresentative, error)
	DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error
}

type LandService interface {
	LandLookupService
	LandAcquisitionService
	LandAssetService
	LandCompensationService
	LandPaymentService
	LandDocumentService
	LandAssigneeService
	LandWorkflowService
	LandFundingService
	LandDroneService
	LandAuthorizedRepresentativeService
}

type landService struct {
	planRepo repository.PlanRepository
	landRepo repository.LandAcquisitionRepository
	files    filestorage.Storage
	// userRepo — auth DB-ийн хэрэглэгчид (мэрг. байгууллагын нэр харуулахад);
	// nil байж болно (тест) — тэр үед нэр хоосон үлдэнэ.
	userRepo repository.UserRepository
	// notifier — төлөв өөрчлөгдөх бүрд холбогдох хэрэглэгчдэд мэдэгдэл илгээнэ;
	// nil байж болно (тест) — тэр үед мэдэгдэл илгээгдэхгүй.
	notifier NotificationService
	// audit — ажилтны бизнес үйлдлийн лог. nil байж болно (тест).
	audit AuditService
	// drone — дроны ортофотог хадгалах хавтас + GeoServer-ийн мозайк.
	// Тохируулаагүй үед дроны зургийн үйлдэл ErrDroneNotConfigured буцаана.
	drone DroneStorage
}

func NewLandService(
	planRepo repository.PlanRepository,
	landRepo repository.LandAcquisitionRepository,
	files filestorage.Storage,
	userRepo repository.UserRepository,
	// extras — заавал бус dependency-үүд
	// (NotificationService, AuditService, DroneStorage).
	extras ...any,
) LandService {
	if files == nil {
		files = filestorage.DisabledStorage{}
	}
	s := &landService{planRepo: planRepo, landRepo: landRepo, files: files, userRepo: userRepo}
	for _, extra := range extras {
		switch v := extra.(type) {
		case NotificationService:
			s.notifier = v
		case AuditService:
			s.audit = v
		case DroneStorage:
			s.drone = v
		}
	}
	return s
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

	s.auditLog(ctx, "created", domain.AuditResourceAcquisition, acq.ID.String(), uuidPtr(acq.ID), nil, map[string]any{
		"name":      acq.AcquisitionName,
		"plan_code": acq.PlanCode,
		"status":    acq.Status,
	})
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

		// BulkInsertParcels шинэ нэгж талбар нэмсэн байж болзошгүй тул acq.ParcelCount
		// (шинэ хилтэй давхцах parcel-үүд нэмэгдэхээс ӨМНӨ уншсан хуучин утга) хуучирсан
		// байна — хариултад зөв тоо буцаахын тулд DB-ээс дахин уншина.
		if fresh, ferr := s.landRepo.GetByID(ctx, acq.ID); ferr == nil {
			acq.ParcelCount = fresh.ParcelCount
		}
	}

	s.auditLog(ctx, "updated", domain.AuditResourceAcquisition, acq.ID.String(), uuidPtr(acq.ID), nil, map[string]any{
		"name":             acq.AcquisitionName,
		"plan_code":        acq.PlanCode,
		"geometry_changed": geometryChanged,
	})
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
	if err := s.landRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, "deleted", domain.AuditResourceAcquisition, id.String(), uuidPtr(id), nil, map[string]any{
		"name":      acq.AcquisitionName,
		"plan_code": acq.PlanCode,
		"status":    acq.Status,
	})
	return nil
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
	// Мэрг. байгууллага нь auth DB-ийн хэрэглэгч тул нэрийг эндээс нөхөж бөглөнө
	// (main DB-ийн query JOIN хийж чадахгүй '' буцаадаг).
	if acq.ProfessionalOrgID != nil && s.userRepo != nil {
		if u, err := s.userRepo.GetByID(ctx, *acq.ProfessionalOrgID); err == nil && u != nil {
			name := strings.TrimSpace(u.FirstName + " " + u.LastName)
			if name == "" {
				name = u.Email
			}
			acq.ProfessionalOrgName = name
		}
	}
	return acq, nil
}

func (s *landService) IsAcquisitionLocked(ctx context.Context, id uuid.UUID) (bool, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return false, domain.ErrNotFound
	}
	return acq.Status == domain.LandAcquisitionStatusVerified, nil
}

func (s *landService) IsParcelAcquisitionLocked(ctx context.Context, parcelID uuid.UUID) (bool, error) {
	parcel, err := s.landRepo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return false, domain.ErrNotFound
	}
	return s.IsAcquisitionLocked(ctx, parcel.AcquisitionID)
}

// EnsureParcelInAcquisition — parcel тухайн чөлөөлөлтөд харьяалагдаж буйг шалгана.
// Харьяалагдахгүй бол ErrNotFound буцаана (IDOR-оос сэргийлнэ).
func (s *landService) EnsureParcelInAcquisition(ctx context.Context, acquisitionID, parcelID uuid.UUID) error {
	parcel, err := s.landRepo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return domain.ErrNotFound
	}
	if parcel.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}
	return nil
}

func (s *landService) GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	// parcel тухайн чөлөөлөлтөд харьяалагдаж буйг баталгаажуулна (IDOR).
	if err := s.EnsureParcelInAcquisition(ctx, acquisitionID, parcelID); err != nil {
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
	if toStatus == domain.LandAcquisitionStatusVerified {
		hasNonFinalParcels, err := s.landRepo.HasNonFinalParcels(ctx, acquisitionID)
		if err != nil {
			return nil, domain.ErrInternalServer
		}
		if hasNonFinalParcels {
			return nil, domain.ErrNonFinalParcels
		}
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

	// Мэдэгдэл: статус амжилттай шилжсэний дараа (энэ мөчид acq.Status
	// хуучин утгаараа тул from→to хоёуланг нь дамжуулж чадна).
	s.notifyAcquisitionStatusChanged(ctx, acq, toStatus)
	s.auditLog(ctx, "status_changed", domain.AuditResourceAcquisition, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"from_status":   acq.Status,
		"to_status":     toStatus,
		"note":          note,
		"decree_number": decreeNumber,
	})

	acq.Status = toStatus
	return acq, nil
}

// UpdateParcelMeta — нэгж талбарын мэдээллийн сангийн өөрчлөлтийн мэдээллийг шинэчлэнэ
func (s *landService) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
	parcel, _ := s.landRepo.GetParcelByID(ctx, parcelID)
	if err := s.landRepo.UpdateParcelMeta(ctx, parcelID, dbChanged, changedParcelID, acquisitionAreaM2, acquisitionGeomWKT); err != nil {
		return err
	}
	acqID := uuid.Nil
	if parcel != nil {
		acqID = parcel.AcquisitionID
	}
	s.auditLog(ctx, "meta_updated", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acqID), uuidPtr(parcelID), map[string]any{
		"db_changed":          dbChanged,
		"changed_parcel_id":   changedParcelID,
		"acquisition_area_m2": acquisitionAreaM2,
		"geometry_changed":    acquisitionGeomWKT != nil,
	})
	return nil
}

// SetParcelCompensation — нэгж талбарын нөхөн төлбөрийн төлөв шинэчлэнэ
func (s *landService) SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error {
	parcel, _ := s.landRepo.GetParcelByID(ctx, parcelID)
	if err := s.landRepo.SetParcelCompensation(ctx, parcelID, paid); err != nil {
		return err
	}
	acqID := uuid.Nil
	parcelCode := ""
	if parcel != nil {
		acqID = parcel.AcquisitionID
		parcelCode = parcel.ParcelID
	}
	s.auditLog(ctx, "compensation_flag_set", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acqID), uuidPtr(parcelID), map[string]any{
		"paid":        paid,
		"parcel_code": parcelCode,
	})
	return nil
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

// GetReportSummary — тайлангийн жагсаалтын дээд хэсгийн статистикийн карт
// (нийт чөлөөлөлт/нэгж талбар, статус болон оны задаргаа, нийт талбай/олговор)
func (s *landService) GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error) {
	summary, err := s.landRepo.GetReportSummary(ctx, filter)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return summary, nil
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

	if err := s.landRepo.UpsertParcelRight(ctx, p.ID, acq.ID, holderID, src); err != nil {
		return err
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(), uuidPtr(acq.ID), uuidPtr(p.ID), map[string]any{
		"parcel_code": parcelCode,
	})
	return nil
}

func (s *landService) GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	return s.landRepo.GetDashboard(ctx, filter)
}

func (s *landService) Suggest(ctx context.Context, q string, status int) ([]*domain.LandAcquisition, error) {
	return s.landRepo.Suggest(ctx, q, status)
}

func (s *landService) SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return domain.ErrNotFound
	}
	// Мэдэгдэлд зөвхөн ШИНЭЭР нэмэгдсэн хүмүүсийг ялгахын тулд өмнөх
	// жагсаалтыг сольхоос нь өмнө авч үлдэнэ.
	previous, _ := s.landRepo.GetAssignees(ctx, acquisitionID)
	if err := s.landRepo.SetAssignees(ctx, acquisitionID, assignees); err != nil {
		return err
	}
	s.notifyAssigneesSet(ctx, acquisitionID, previous, assignees)
	s.auditLog(ctx, "assignees_set", domain.AuditResourceAcquisition, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"previous_count": len(previous),
		"current_count":  len(assignees),
		"assignees":      assignees,
	})
	return nil
}

func (s *landService) GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error) {
	return s.landRepo.GetAssignees(ctx, acquisitionID)
}

func (s *landService) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.landRepo.IsAssigned(ctx, acquisitionID, userID)
}

func (s *landService) IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.landRepo.IsAssignedToParcel(ctx, parcelID, userID)
}

func (s *landService) IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.landRepo.IsProfMemberOfAcquisition(ctx, acquisitionID, userID)
}

func (s *landService) IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.landRepo.IsProfMemberOfParcel(ctx, parcelID, userID)
}

func (s *landService) IsProfMemberOfParcelByCode(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, userID uuid.UUID) (bool, error) {
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return false, err
	}
	if p.AcquisitionID != acquisitionID {
		return false, domain.ErrNotFound
	}
	return s.landRepo.IsProfMemberOfParcel(ctx, p.ID, userID)
}

func (s *landService) SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *uuid.UUID) error {
	if err := s.landRepo.SetAcquisitionProfessionalOrg(ctx, acquisitionID, orgUserID); err != nil {
		return err
	}
	s.notifyOrgAssigned(ctx, acquisitionID, nil, orgUserID, false)
	s.auditLog(ctx, "professional_org_set", domain.AuditResourceAcquisition, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"org_user_id": orgUserID,
	})
	return nil
}

func (s *landService) SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *uuid.UUID) error {
	if err := s.landRepo.SetParcelIndependentOrg(ctx, acqID, parcelID, orgUserID); err != nil {
		return err
	}
	s.notifyOrgAssigned(ctx, acqID, &parcelID, orgUserID, true)
	s.auditLog(ctx, "independent_org_set", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acqID), uuidPtr(parcelID), map[string]any{
		"org_user_id": orgUserID,
	})
	return nil
}
