package postgres

// Мэдэгдэх хуудсыг имэйлээр илгээсэн түүх (parcel_notice_email).
// Мөр бүр НЭГ илгээлт — хуучин мөр хөндөгдөхгүй, зөвхөн нэмэгдэнэ.

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

func (r *landAcqRepo) AddParcelNoticeEmail(ctx context.Context, m *domain.ParcelNoticeEmail) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO parcel_notice_email (
			id, parcel_id, acquisition_id, to_email, subject,
			file_url, file_name, size_bytes, status, error_message, sent_at, sent_by
		) VALUES (
			$1, $2, NULLIF($3, '00000000-0000-0000-0000-000000000000'::uuid), $4, NULLIF($5,''),
			NULLIF($6,''), NULLIF($7,''), NULLIF($8, 0), $9, NULLIF($10,''), $11, NULLIF($12,'')
		)
	`, m.ID, m.ParcelID, m.AcquisitionID, m.ToEmail, m.Subject,
		m.FileURL, m.FileName, m.SizeBytes, m.Status, m.ErrorMessage, m.SentAt, m.SentBy)
	return err
}

// ListParcelNoticeEmails — түүх, ШИНЭ илгээлт эхэнд.
func (r *landAcqRepo) ListParcelNoticeEmails(
	ctx context.Context, parcelID uuid.UUID,
) ([]domain.ParcelNoticeEmail, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, parcel_id, COALESCE(acquisition_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       to_email, COALESCE(subject,''), COALESCE(file_url,''), COALESCE(file_name,''),
		       COALESCE(size_bytes,0), status, COALESCE(error_message,''), sent_at, COALESCE(sent_by,'')
		FROM parcel_notice_email
		WHERE parcel_id = $1
		ORDER BY sent_at DESC
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelNoticeEmail
	for rows.Next() {
		m := domain.ParcelNoticeEmail{}
		if err := rows.Scan(&m.ID, &m.ParcelID, &m.AcquisitionID, &m.ToEmail, &m.Subject,
			&m.FileURL, &m.FileName, &m.SizeBytes, &m.Status, &m.ErrorMessage,
			&m.SentAt, &m.SentBy); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
