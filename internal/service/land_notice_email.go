package service

// Урьдчилан мэдэгдэх хуудсыг ИМЭЙЛЭЭР хүргүүлэх (төлөвлөгөөний 2.4.2.2).
//
// PDF-ийг frontend үүсгээд (загварын ижил үг хэллэгээр) энд илгээнэ: сервер
// хавсралтыг MinIO-д хадгалж, SMTP-ээр илгээж, илгээлт бүрийг ТҮҮХЭД бүртгэнэ.
//
// Алдаатай оролдлого ч бүртгэгдэнэ ("failed"): мэдэгдэл хүргэгдсэн эсэх нь
// Захиргааны Ерөнхий хуулийн хугацаа тоолоход хамаардаг тул "илгээсэн гэж
// бодох" эндүүрэл гарах ёсгүй.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/email"

	"github.com/google/uuid"
)

// NoticeEmailInput — илгээх хүсэлтийн орц.
type NoticeEmailInput struct {
	ToEmail  string
	FileName string
	FileData []byte
	SentBy   string
}

const noticeEmailSubject = "Урьдчилан мэдэгдэх хуудас"

func noticeEmailBody(parcelCode string) string {
	return fmt.Sprintf(`Сайн байна уу,

Танд газар чөлөөлөлтийн нөлөөлөлд өртсөн %s дугаартай нэгж талбарын талаарх
"Урьдчилан мэдэгдэх хуудас"-ыг хавсралтаар хүргүүлж байна.

Хавсралтад заасан хугацаанд Нийслэлийн Газар зохион байгуулалтын албаны Газар
чөлөөлөх нэгдүгээр хэлтэст өөрийн биеэр ирж уулзана уу.

Хүндэтгэсэн,
Нийслэлийн Газар зохион байгуулалтын алба`, parcelCode)
}

// ListParcelNoticeEmails — илгээсэн мэдэгдлүүдийн түүх (шинэ нь эхэнд).
func (s *landService) ListParcelNoticeEmails(
	ctx context.Context, parcelID uuid.UUID,
) ([]domain.ParcelNoticeEmail, error) {
	if _, err := s.landRepo.GetParcelDetailByID(ctx, parcelID); err != nil {
		return nil, domain.ErrNotFound
	}
	list, err := s.landRepo.ListParcelNoticeEmails(ctx, parcelID)
	if err != nil {
		slog.Error("мэдэгдлийн түүх уншихад алдаа", "parcel_id", parcelID, "error", err)
		return nil, domain.ErrInternalServer
	}
	return list, nil
}

// SendParcelNoticeEmail — мэдэгдэх хуудсыг хавсралттайгаар имэйлээр илгээнэ.
func (s *landService) SendParcelNoticeEmail(
	ctx context.Context, parcelID uuid.UUID, in NoticeEmailInput,
) (*domain.ParcelNoticeEmail, error) {
	to := strings.TrimSpace(in.ToEmail)
	if _, err := mail.ParseAddress(to); err != nil {
		return nil, domain.ErrInvalidEmail
	}
	if len(in.FileData) == 0 {
		return nil, domain.ErrInvalidInput
	}
	// SMTP тохируулаагүй бол ЭНД зогсоно: файл хадгалахгүй, түүхэд ч
	// "амжилтгүй" мөр бичихгүй (тохиргооны алдааг илгээлтийн түүх бохирдуулах
	// шаардлагагүй). NoopSender нь тохируулаагүй үеийн хэрэгжүүлэлт.
	if s.emailSender == nil {
		return nil, domain.ErrEmailNotConfigured
	}
	if _, notConfigured := s.emailSender.(email.NoopSender); notConfigured {
		return nil, domain.ErrEmailNotConfigured
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
	stored, err := s.files.Save(ctx, "notice", parcelID.String(), fileName, "application/pdf", in.FileData, true)
	if err != nil {
		slog.Error("мэдэгдлийн файл хадгалахад алдаа", "parcel_id", parcelID, "error", err)
		return nil, err
	}

	record := &domain.ParcelNoticeEmail{
		ID:            uuid.New(),
		ParcelID:      parcelID,
		AcquisitionID: parcel.AcquisitionID,
		ToEmail:       to,
		Subject:       noticeEmailSubject,
		FileURL:       stored.URL,
		FileName:      stored.Name,
		SizeBytes:     stored.SizeBytes,
		Status:        domain.NoticeEmailStatusSent,
		SentAt:        time.Now(),
		SentBy:        in.SentBy,
	}

	sendErr := s.emailSender.SendWithAttachment(
		to, noticeEmailSubject, noticeEmailBody(parcel.ParcelID), fileName, in.FileData,
	)
	if sendErr != nil {
		record.Status = domain.NoticeEmailStatusFailed
		record.ErrorMessage = sendErr.Error()
	}

	// Түүхийг АМЖИЛТГҮЙ үед ч бичнэ — дараа нь шалтгаанаа харна.
	if addErr := s.landRepo.AddParcelNoticeEmail(ctx, record); addErr != nil {
		slog.Error("мэдэгдлийн түүх бичихэд алдаа", "parcel_id", parcelID, "error", addErr)
		if sendErr == nil {
			// Имэйл илгээгдсэн ч түүх бичигдээгүй — файлыг үлдээнэ (илгээсэн
			// баримт), гэхдээ дуудагчид алдаа мэдэгдэнэ.
			return nil, domain.ErrInternalServer
		}
	}

	s.auditLog(ctx, "notice_email", domain.AuditResourceParcel, parcelID.String(),
		uuidPtr(parcel.AcquisitionID), uuidPtr(parcelID), map[string]any{
			"to":        to,
			"status":    record.Status,
			"file_name": record.FileName,
			"sent_by":   in.SentBy,
		})

	if sendErr != nil {
		slog.Error("мэдэгдэх хуудас илгээхэд алдаа", "parcel_id", parcelID, "to", to, "error", sendErr)
		if errors.Is(sendErr, email.ErrNotConfigured) {
			return record, domain.ErrEmailNotConfigured
		}
		return record, domain.ErrEmailSendFailed
	}
	return record, nil
}
