package postgres

// Мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлсэн түүх (parcel_emongolia_notice).
// Мөр бүр НЭГ илгээлт — зөвхөн нэмэгдэнэ.

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

func (r *landAcqRepo) AddParcelEMongoliaNotice(ctx context.Context, m *domain.ParcelEMongoliaNotice) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO parcel_emongolia_notice (
			id, parcel_id, acquisition_id, register_no, recipient_name, subject,
			file_url, file_name, size_bytes, status, external_ref, error_message, sent_at, sent_by
		) VALUES (
			$1, $2, NULLIF($3, '00000000-0000-0000-0000-000000000000'::uuid), $4, NULLIF($5,''), NULLIF($6,''),
			NULLIF($7,''), NULLIF($8,''), NULLIF($9, 0), $10, NULLIF($11,''), NULLIF($12,''), $13, NULLIF($14,'')
		)
	`, m.ID, m.ParcelID, m.AcquisitionID, m.RegisterNo, m.RecipientName, m.Subject,
		m.FileURL, m.FileName, m.SizeBytes, m.Status, m.ExternalRef, m.ErrorMessage, m.SentAt, m.SentBy)
	return err
}

// ListParcelEMongoliaNotices — түүх, ШИНЭ илгээлт эхэнд.
func (r *landAcqRepo) ListParcelEMongoliaNotices(
	ctx context.Context, parcelID uuid.UUID,
) ([]domain.ParcelEMongoliaNotice, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, parcel_id, COALESCE(acquisition_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       register_no, COALESCE(recipient_name,''), COALESCE(subject,''),
		       COALESCE(file_url,''), COALESCE(file_name,''), COALESCE(size_bytes,0),
		       status, COALESCE(external_ref,''), COALESCE(error_message,''), sent_at, COALESCE(sent_by,'')
		FROM parcel_emongolia_notice
		WHERE parcel_id = $1
		ORDER BY sent_at DESC
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelEMongoliaNotice
	for rows.Next() {
		m := domain.ParcelEMongoliaNotice{}
		if err := rows.Scan(&m.ID, &m.ParcelID, &m.AcquisitionID,
			&m.RegisterNo, &m.RecipientName, &m.Subject,
			&m.FileURL, &m.FileName, &m.SizeBytes,
			&m.Status, &m.ExternalRef, &m.ErrorMessage, &m.SentAt, &m.SentBy); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
