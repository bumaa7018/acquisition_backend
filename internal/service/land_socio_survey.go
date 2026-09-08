package service

// Нийгэм, эдийн засгийн судалгаа — нөлөөлөлд өртсөн иргэн, хуулийн этгээдээс
// авсан судалгааны ТООН дүн (зөвшөөрсөн/татгалзсан) + судалгааны эх баримт
// (PDF). Чөлөөлөлтөд НЭГ бүртгэл: дахин оруулахад орлогдоно.
//
// Хэрэглэгч олон PDF сонгосон бол browser дээр НЭГ файл болгож нэгтгэдэг тул
// сервер рүү үргэлж нэг PDF ирнэ (нэгтгэлийг сервер дээр дахин хийхгүй).

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// SocioSurveyInput — судалгааны бүртгэлийн орц.
//
// FileData хоосон бол ХАДГАЛАГДСАН файл хэвээр үлдэнэ (тоо/тайлбар л
// шинэчлэгдэнэ). Шинэ бүртгэлд файл ЗААВАЛ.
type SocioSurveyInput struct {
	AgreedCount   int
	RejectedCount int
	Note          string
	FileName      string
	ContentType   string
	FileData      []byte
	SavedBy       string
}

func (s *landService) GetAcquisitionSocioSurvey(
	ctx context.Context, acquisitionID uuid.UUID,
) (*domain.AcquisitionSocioSurvey, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	survey, err := s.landRepo.GetAcquisitionSocioSurvey(ctx, acquisitionID)
	if err == domain.ErrNotFound {
		// Бүртгэл ороогүй нь алдаа биш — дэлгэц хоосон хэсэг харуулна.
		return nil, nil
	}
	if err != nil {
		slog.Error("судалгааны мэдээлэл уншихад алдаа", "acquisition_id", acquisitionID, "error", err)
		return nil, domain.ErrInternalServer
	}
	return survey, nil
}

func (s *landService) SaveAcquisitionSocioSurvey(
	ctx context.Context, acquisitionID uuid.UUID, in SocioSurveyInput,
) (*domain.AcquisitionSocioSurvey, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	if in.AgreedCount < 0 || in.RejectedCount < 0 {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.landRepo.GetAcquisitionSocioSurvey(ctx, acquisitionID)
	if err != nil && err != domain.ErrNotFound {
		slog.Error("судалгааны мэдээлэл уншихад алдаа", "acquisition_id", acquisitionID, "error", err)
		return nil, domain.ErrInternalServer
	}

	record := &domain.AcquisitionSocioSurvey{
		ID:            uuid.New(),
		AcquisitionID: acquisitionID,
		AgreedCount:   in.AgreedCount,
		RejectedCount: in.RejectedCount,
		Note:          strings.TrimSpace(in.Note),
		UpdatedBy:     in.SavedBy,
		UpdatedAt:     time.Now(),
	}
	if existing != nil {
		record.ID = existing.ID
		record.CreatedAt = existing.CreatedAt
		record.CreatedBy = existing.CreatedBy
		// Шинэ файл ирээгүй бол хадгалагдсаныг хэвээр үлдээнэ.
		record.FileURL = existing.FileURL
		record.FileName = existing.FileName
		record.SizeBytes = existing.SizeBytes
	}

	if len(in.FileData) > 0 {
		stored, saveErr := s.files.Save(
			ctx, "socio-survey", acquisitionID.String(),
			in.FileName, in.ContentType, in.FileData, true,
		)
		if saveErr != nil {
			slog.Error("судалгааны файл хадгалахад алдаа", "acquisition_id", acquisitionID, "error", saveErr)
			return nil, saveErr
		}
		record.FileURL = stored.URL
		record.FileName = stored.Name
		record.SizeBytes = stored.SizeBytes
	}
	// Судалгааны файл ЗААВАЛ — файлгүй бүртгэл үүсгэхгүй.
	if record.FileURL == "" {
		return nil, domain.ErrSurveyFileRequired
	}

	oldFileURL, err := s.landRepo.UpsertAcquisitionSocioSurvey(ctx, record)
	if err != nil {
		// Мөр бичигдээгүй тул саяхан хадгалсан файлыг цэвэрлэнэ.
		if len(in.FileData) > 0 && record.FileURL != "" {
			_ = s.files.Delete(ctx, record.FileURL)
		}
		slog.Error("судалгааны мэдээлэл хадгалахад алдаа", "acquisition_id", acquisitionID, "error", err)
		return nil, domain.ErrInternalServer
	}
	// Солигдсон хуучин файлыг хадгалалтаас цэвэрлэнэ — орфон файл үлдээхгүй.
	if oldFileURL != "" && oldFileURL != record.FileURL {
		_ = s.files.Delete(ctx, oldFileURL)
	}

	s.auditLog(ctx, "saved", domain.AuditResourceAcquisition, acquisitionID.String(),
		uuidPtr(acquisitionID), nil, map[string]any{
			"socio_survey":   true,
			"agreed_count":   record.AgreedCount,
			"rejected_count": record.RejectedCount,
			"file_name":      record.FileName,
			"saved_by":       in.SavedBy,
		})
	return record, nil
}
