package postgres

// Нийгэм, эдийн засгийн судалгаа (acquisition_socio_survey) — чөлөөлөлтөд НЭГ
// бүртгэл. Дахин оруулахад мөр орлогдож, солигдсон файл дуудагч талд
// цэвэрлэгдэнэ.

import (
	"context"
	"errors"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *landAcqRepo) GetAcquisitionSocioSurvey(
	ctx context.Context, acquisitionID uuid.UUID,
) (*domain.AcquisitionSocioSurvey, error) {
	s := &domain.AcquisitionSocioSurvey{}
	err := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, agreed_count, rejected_count, COALESCE(note,''),
		       file_url, COALESCE(file_name,''), COALESCE(size_bytes,0),
		       created_at, COALESCE(created_by,''), updated_at, COALESCE(updated_by,'')
		FROM acquisition_socio_survey
		WHERE acquisition_id = $1
	`, acquisitionID).Scan(
		&s.ID, &s.AcquisitionID, &s.AgreedCount, &s.RejectedCount, &s.Note,
		&s.FileURL, &s.FileName, &s.SizeBytes,
		&s.CreatedAt, &s.CreatedBy, &s.UpdatedAt, &s.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// UpsertAcquisitionSocioSurvey — acquisition_id-гаар орлуулна.
//
// Солигдсон хуучин файлын URL-ийг буцаана: шинэ файл өөр бол дуудагч тал
// хадгалалтаас хуучныг устгаж, орфон файл үлдэхээс сэргийлнэ.
func (r *landAcqRepo) UpsertAcquisitionSocioSurvey(
	ctx context.Context, s *domain.AcquisitionSocioSurvey,
) (string, error) {
	// Хуучин файлыг ТУСДАА уншина — INSERT ... RETURNING дотор CTE-ээс уншвал
	// бүртгэл байхгүй үед NULL болж Scan унана.
	var oldFileURL string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(file_url,'') FROM acquisition_socio_survey WHERE acquisition_id = $1`,
		s.AcquisitionID,
	).Scan(&oldFileURL)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	if _, err := r.db.Exec(ctx, `
		INSERT INTO acquisition_socio_survey (
			id, acquisition_id, agreed_count, rejected_count, note,
			file_url, file_name, size_bytes, created_by, updated_by
		) VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, NULLIF($7,''), $8, NULLIF($9,''), NULLIF($9,''))
		ON CONFLICT (acquisition_id) DO UPDATE SET
			agreed_count   = EXCLUDED.agreed_count,
			rejected_count = EXCLUDED.rejected_count,
			note           = EXCLUDED.note,
			file_url       = EXCLUDED.file_url,
			file_name      = EXCLUDED.file_name,
			size_bytes     = EXCLUDED.size_bytes,
			updated_at     = NOW(),
			updated_by     = EXCLUDED.updated_by
	`, s.ID, s.AcquisitionID, s.AgreedCount, s.RejectedCount, s.Note,
		s.FileURL, s.FileName, s.SizeBytes, s.UpdatedBy,
	); err != nil {
		return "", err
	}
	return oldFileURL, nil
}
