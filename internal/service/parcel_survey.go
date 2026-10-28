package service

// Чөлөөлөгдсөн газрын судалгаа — нөхөх олговор олгосны дараа нэгж талбарын
// үлдэгдэл иргэнээс авах 4 асуулт (яндан, нойл, хамтран амьдрагч, шилжин
// суурьшсан байршил). Нэгж талбарт НЭГ бүртгэл: нэгж талбар үүсэхэд хоосон
// бүртгэл автоматаар (DB trigger) үүсэж, хадгалахад орлогдоно.

import (
	"context"
	"log/slog"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

// ParcelSurveyInput — судалгааны орц. Тоон хариулт 1..domain.SurveyCountMax эсвэл nil.
type ParcelSurveyInput struct {
	ChimneyCount      *int
	ToiletCount       *int
	CohabitantCount   *int
	RelocationType    string
	RelocationAddress string
	Note              string
	SavedBy           string
}

type ParcelSurveyService interface {
	// GetParcelSurvey — бүртгэл ороогүй бол (nil, nil).
	GetParcelSurvey(ctx context.Context, parcelID uuid.UUID) (*domain.ParcelSurvey, error)
	SaveParcelSurvey(ctx context.Context, parcelID uuid.UUID, in ParcelSurveyInput) (*domain.ParcelSurvey, error)
	// ClearParcelSurvey — хариултыг хоосолно (бүртгэл нэгж талбартайгаа үлдэнэ).
	ClearParcelSurvey(ctx context.Context, parcelID uuid.UUID, clearedBy string) error
	ListParcelSurveys(ctx context.Context, filter domain.ParcelSurveyFilter) ([]*domain.ParcelSurveyRow, int64, error)
	ParcelSurveySummary(ctx context.Context, filter domain.ParcelSurveyFilter) (*domain.ParcelSurveySummary, error)
}

type parcelSurveyService struct {
	repo  repository.ParcelSurveyRepository
	audit AuditService
}

func NewParcelSurveyService(repo repository.ParcelSurveyRepository, audit AuditService) ParcelSurveyService {
	return &parcelSurveyService{repo: repo, audit: audit}
}

func (s *parcelSurveyService) ensureParcel(ctx context.Context, parcelID uuid.UUID) error {
	exists, err := s.repo.ParcelExists(ctx, parcelID)
	if err != nil {
		slog.Error("нэгж талбар шалгахад алдаа", "parcel_id", parcelID, "error", err)
		return domain.ErrInternalServer
	}
	if !exists {
		return domain.ErrNotFound
	}
	return nil
}

func (s *parcelSurveyService) GetParcelSurvey(ctx context.Context, parcelID uuid.UUID) (*domain.ParcelSurvey, error) {
	if err := s.ensureParcel(ctx, parcelID); err != nil {
		return nil, err
	}
	survey, err := s.repo.GetByParcel(ctx, parcelID)
	if err == domain.ErrNotFound {
		// Trigger-ээс өмнө үүссэн, backfill-д ороогүй нэгж талбар — алдаа биш,
		// таб хоосон маягт харуулж, хадгалахад үүснэ.
		return nil, nil
	}
	if err != nil {
		slog.Error("судалгаа уншихад алдаа", "parcel_id", parcelID, "error", err)
		return nil, domain.ErrInternalServer
	}
	return survey, nil
}

func validSurveyCount(v *int) bool {
	return v == nil || (*v >= 1 && *v <= domain.SurveyCountMax)
}

func (s *parcelSurveyService) SaveParcelSurvey(
	ctx context.Context, parcelID uuid.UUID, in ParcelSurveyInput,
) (*domain.ParcelSurvey, error) {
	if !validSurveyCount(in.ChimneyCount) || !validSurveyCount(in.ToiletCount) || !validSurveyCount(in.CohabitantCount) {
		return nil, domain.ErrInvalidInput
	}
	in.RelocationType = strings.TrimSpace(in.RelocationType)
	// «Хаяг» сонголт хасагдсан — хаягийг хадгалахгүй.
	in.RelocationAddress = ""
	switch in.RelocationType {
	case "", domain.RelocationGerDistrict, domain.RelocationApartment:
	default:
		return nil, domain.ErrInvalidInput
	}
	if err := s.ensureParcel(ctx, parcelID); err != nil {
		return nil, err
	}

	record := &domain.ParcelSurvey{
		ID:                uuid.New(),
		ParcelID:          parcelID,
		ChimneyCount:      in.ChimneyCount,
		ToiletCount:       in.ToiletCount,
		CohabitantCount:   in.CohabitantCount,
		RelocationType:    in.RelocationType,
		RelocationAddress: in.RelocationAddress,
		Note:              strings.TrimSpace(in.Note),
		UpdatedBy:         in.SavedBy,
	}
	if err := s.repo.Upsert(ctx, record); err != nil {
		slog.Error("судалгаа хадгалахад алдаа", "parcel_id", parcelID, "error", err)
		return nil, domain.ErrInternalServer
	}

	if s.audit != nil {
		s.audit.Log(ctx, AuditEntry{
			Action:       "saved",
			ResourceType: domain.AuditResourceParcel,
			ResourceID:   parcelID.String(),
			ParcelID:     uuidPtr(parcelID),
			Details: map[string]any{
				"parcel_survey":    true,
				"chimney_count":    record.ChimneyCount,
				"toilet_count":     record.ToiletCount,
				"cohabitant_count": record.CohabitantCount,
				"relocation_type":  record.RelocationType,
				"saved_by":         in.SavedBy,
			},
		})
	}
	return record, nil
}

func (s *parcelSurveyService) ClearParcelSurvey(ctx context.Context, parcelID uuid.UUID, clearedBy string) error {
	err := s.repo.ClearByParcel(ctx, parcelID, clearedBy)
	if err == domain.ErrNotFound {
		return err
	}
	if err != nil {
		slog.Error("судалгаа цэвэрлэхэд алдаа", "parcel_id", parcelID, "error", err)
		return domain.ErrInternalServer
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditEntry{
			Action:       "deleted",
			ResourceType: domain.AuditResourceParcel,
			ResourceID:   parcelID.String(),
			ParcelID:     uuidPtr(parcelID),
			Details:      map[string]any{"parcel_survey": true, "cleared_by": clearedBy},
		})
	}
	return nil
}

func (s *parcelSurveyService) ListParcelSurveys(
	ctx context.Context, filter domain.ParcelSurveyFilter,
) ([]*domain.ParcelSurveyRow, int64, error) {
	list, total, err := s.repo.List(ctx, filter)
	if err != nil {
		slog.Error("судалгааны жагсаалт уншихад алдаа", "error", err)
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

func (s *parcelSurveyService) ParcelSurveySummary(
	ctx context.Context, filter domain.ParcelSurveyFilter,
) (*domain.ParcelSurveySummary, error) {
	sum, err := s.repo.Summary(ctx, filter)
	if err != nil {
		slog.Error("судалгааны дүн уншихад алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	return sum, nil
}
