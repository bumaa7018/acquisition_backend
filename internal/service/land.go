package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
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
}

type landService struct {
	planRepo repository.PlanRepository
	landRepo repository.LandAcquisitionRepository
}

func NewLandService(
	planRepo repository.PlanRepository,
	landRepo repository.LandAcquisitionRepository,
) LandService {
	return &landService{planRepo: planRepo, landRepo: landRepo}
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
		ID:          uuid.New(),
		PlanID:      &planIDCopy,
		PlanCode:    plan.PlanCode,
		PlanGeomWKT: plan.BoundaryWKT,
		PlanAreaM2:  plan.AreaM2,
		GeometryWKT: acqWKT,
		Status:      domain.LandAcquisitionStatusActive,
		StartDate:   &req.StartDate,
		EndDate:     &req.EndDate,
		CreatedAt:   now,
		CreatedBy:   req.CreatedBy,
		UpdatedAt:   now,
		UpdatedBy:   req.CreatedBy,
	}

	if err := s.landRepo.Create(ctx, acq); err != nil {
		return nil, domain.ErrInternalServer
	}

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
	if req.EndDate != nil {
		acq.EndDate = req.EndDate
	}
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
