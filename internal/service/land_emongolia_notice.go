package service

// Мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлэх (төлөвлөгөөний 2.8.5).
//
// PDF-ийг frontend загвараас үүсгэж (дэлгэцэд харуулсны дараа) энд илгээнэ:
// сервер файлыг MinIO-д хадгалж, E-Mongolia-ийн гарц руу дамжуулж, илгээлт
// бүрийг ТҮҮХЭД бүртгэнэ.
//
// Гадаад холболт одоогоор МОК тул түүхэд status="mocked" гэж бичигдэнэ —
// хүргэгдсэн эсэхийг эндүүрэхгүй байхын тулд.

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// EMongoliaNoticeInput — илгээх хүсэлтийн орц.
type EMongoliaNoticeInput struct {
	RegisterNo    string
	RecipientName string
	FileName      string
	FileData      []byte
	SentBy        string
}

const emongoliaNoticeSubject = "Урьдчилан мэдэгдэх хуудас"

func emongoliaNoticeBody(parcelCode string) string {
	return fmt.Sprintf(`Танд газар чөлөөлөлтийн нөлөөлөлд өртсөн %s дугаартай нэгж талбарын `+
		`талаарх "Урьдчилан мэдэгдэх хуудас"-ыг хүргүүлж байна. Хавсралтад заасан `+
		`хугацаанд Нийслэлийн Газар зохион байгуулалтын албаны Газар чөлөөлөх `+
		`нэгдүгээр хэлтэст өөрийн биеэр ирж уулзана уу.`, parcelCode)
}

func (s *landService) ListParcelEMongoliaNotices(
	ctx context.Context, parcelID uuid.UUID,
) ([]domain.ParcelEMongoliaNotice, error) {
	if _, err := s.landRepo.GetParcelDetailByID(ctx, parcelID); err != nil {
		return nil, domain.ErrNotFound
	}
	list, err := s.landRepo.ListParcelEMongoliaNotices(ctx, parcelID)
	if err != nil {
		slog.Error("E-Mongolia мэдэгдлийн түүх уншихад алдаа", "parcel_id", parcelID, "error", err)
		return nil, domain.ErrInternalServer
	}
	return list, nil
}

// SendParcelEMongoliaNotice — мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлнэ.
func (s *landService) SendParcelEMongoliaNotice(
	ctx context.Context, parcelID uuid.UUID, in EMongoliaNoticeInput,
) (*domain.ParcelEMongoliaNotice, error) {
	registerNo := strings.TrimSpace(in.RegisterNo)
	if registerNo == "" {
		return nil, domain.ErrRegisterNoRequired
	}
	if len(in.FileData) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if s.emongolia == nil {
		return nil, domain.ErrEMongoliaNotConfigured
	}

	parcel, err := s.landRepo.GetParcelDetailByID(ctx, parcelID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// Илгээсэн ЯГ ТЭР файлыг дараа нь дахин харах боломжтой байхаар хадгална.
	fileName := strings.TrimSpace(in.FileName)
	if fileName == "" {
		fileName = "medegdeh_huudas.pdf"
	}
	stored, err := s.files.Save(ctx, "emongolia", parcelID.String(), fileName, "application/pdf", in.FileData, true)
	if err != nil {
		slog.Error("E-Mongolia мэдэгдлийн файл хадгалахад алдаа", "parcel_id", parcelID, "error", err)
		return nil, err
	}

	record := &domain.ParcelEMongoliaNotice{
		ID:            uuid.New(),
		ParcelID:      parcelID,
		AcquisitionID: parcel.AcquisitionID,
		RegisterNo:    registerNo,
		RecipientName: strings.TrimSpace(in.RecipientName),
		Subject:       emongoliaNoticeSubject,
		FileURL:       stored.URL,
		FileName:      stored.Name,
		SizeBytes:     stored.SizeBytes,
		Status:        domain.EMongoliaStatusSent,
		SentAt:        time.Now(),
		SentBy:        in.SentBy,
	}

	res, sendErr := s.emongolia.SendNotice(ctx, EMongoliaNoticeRequest{
		RegisterNo:    registerNo,
		RecipientName: record.RecipientName,
		Subject:       emongoliaNoticeSubject,
		Message:       emongoliaNoticeBody(parcel.ParcelID),
		ParcelCode:    parcel.ParcelID,
		FileName:      fileName,
		FileData:      in.FileData,
	})
	switch {
	case sendErr != nil:
		record.Status = domain.EMongoliaStatusFailed
		record.ErrorMessage = sendErr.Error()
	case res != nil:
		record.ExternalRef = res.ExternalRef
		// Мок үед "илгээгдсэн" гэж бүртгэхгүй — хүргэгдсэн эсэх нь хуулийн
		// хугацаа тоолоход хамаардаг тул төлөвийг ил тод үлдээнэ.
		if res.Mocked {
			record.Status = domain.EMongoliaStatusMocked
		}
	}

	// Түүхийг амжилтгүй үед ч бичнэ — дараа нь шалтгаанаа харна.
	if addErr := s.landRepo.AddParcelEMongoliaNotice(ctx, record); addErr != nil {
		slog.Error("E-Mongolia мэдэгдлийн түүх бичихэд алдаа", "parcel_id", parcelID, "error", addErr)
		if sendErr == nil {
			return nil, domain.ErrInternalServer
		}
	}

	s.auditLog(ctx, "emongolia_notice", domain.AuditResourceParcel, parcelID.String(),
		uuidPtr(parcel.AcquisitionID), uuidPtr(parcelID), map[string]any{
			"register_no":  registerNo,
			"status":       record.Status,
			"external_ref": record.ExternalRef,
			"file_name":    record.FileName,
			"sent_by":      in.SentBy,
		})

	if sendErr != nil {
		slog.Error("E-Mongolia мэдэгдэл илгээхэд алдаа",
			"parcel_id", parcelID, "register_no", registerNo, "error", sendErr)
		return record, domain.ErrEMongoliaSendFailed
	}
	return record, nil
}
