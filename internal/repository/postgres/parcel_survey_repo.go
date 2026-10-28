package postgres

// Чөлөөлөгдсөн газрын судалгаа (parcel_survey) — нэгж талбарт НЭГ бүртгэл.
// Нэгж талбар үүсэхэд хоосон мөр trigger-ээр үүснэ (migration 000057);
// хадгалахад parcel_id-гаар орлогдоно.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type parcelSurveyRepo struct{ db *pgxpool.Pool }

func NewParcelSurveyRepo(db *pgxpool.Pool) repository.ParcelSurveyRepository {
	return &parcelSurveyRepo{db: db}
}

const parcelSurveyColumns = `
	s.id, s.parcel_id, s.chimney_count, s.toilet_count, s.cohabitant_count,
	COALESCE(s.relocation_type,''), COALESCE(s.relocation_address,''), COALESCE(s.note,''),
	s.created_at, COALESCE(s.created_by,''), s.updated_at, COALESCE(s.updated_by,'')`

func scanParcelSurvey(row pgx.Row, s *domain.ParcelSurvey, extra ...any) error {
	var chimney, toilet, cohabitant *int16
	dest := []any{
		&s.ID, &s.ParcelID, &chimney, &toilet, &cohabitant,
		&s.RelocationType, &s.RelocationAddress, &s.Note,
		&s.CreatedAt, &s.CreatedBy, &s.UpdatedAt, &s.UpdatedBy,
	}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return err
	}
	s.ChimneyCount = int16PtrToInt(chimney)
	s.ToiletCount = int16PtrToInt(toilet)
	s.CohabitantCount = int16PtrToInt(cohabitant)
	return nil
}

func int16PtrToInt(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

func (r *parcelSurveyRepo) ParcelExists(ctx context.Context, parcelID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM parcel WHERE id = $1)`, parcelID).Scan(&exists)
	return exists, err
}

func (r *parcelSurveyRepo) GetByParcel(ctx context.Context, parcelID uuid.UUID) (*domain.ParcelSurvey, error) {
	s := &domain.ParcelSurvey{}
	err := scanParcelSurvey(r.db.QueryRow(ctx,
		`SELECT `+parcelSurveyColumns+` FROM parcel_survey s WHERE s.parcel_id = $1`, parcelID), s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Upsert — parcel_id-гаар орлуулна. created_* нь анхны бүртгэлийнх хэвээр.
func (r *parcelSurveyRepo) Upsert(ctx context.Context, s *domain.ParcelSurvey) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO parcel_survey (
			id, parcel_id, chimney_count, toilet_count, cohabitant_count,
			relocation_type, relocation_address, note, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), NULLIF($9,''))
		ON CONFLICT (parcel_id) DO UPDATE SET
			chimney_count      = EXCLUDED.chimney_count,
			toilet_count       = EXCLUDED.toilet_count,
			cohabitant_count   = EXCLUDED.cohabitant_count,
			relocation_type    = EXCLUDED.relocation_type,
			relocation_address = EXCLUDED.relocation_address,
			note               = EXCLUDED.note,
			updated_at         = NOW(),
			updated_by         = EXCLUDED.updated_by
		RETURNING id, created_at, COALESCE(created_by,''), updated_at`,
		s.ID, s.ParcelID, s.ChimneyCount, s.ToiletCount, s.CohabitantCount,
		s.RelocationType, s.RelocationAddress, s.Note, s.UpdatedBy,
	).Scan(&s.ID, &s.CreatedAt, &s.CreatedBy, &s.UpdatedAt)
}

// surveyAnsweredSQL — бүртгэл бөглөгдсөн эсэх (аль нэг хариулт эсвэл тайлбар).
const surveyAnsweredSQL = `(s.chimney_count IS NOT NULL OR s.toilet_count IS NOT NULL
	OR s.cohabitant_count IS NOT NULL OR s.relocation_type IS NOT NULL OR s.note IS NOT NULL)`

func (r *parcelSurveyRepo) ClearByParcel(ctx context.Context, parcelID uuid.UUID, clearedBy string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE parcel_survey SET
			chimney_count = NULL, toilet_count = NULL, cohabitant_count = NULL,
			relocation_type = NULL, relocation_address = NULL, note = NULL,
			updated_at = NOW(), updated_by = NULLIF($2,'')
		WHERE parcel_id = $1`, parcelID, clearedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// surveyFilterWhere — List/Summary-ийн нийтлэг шүүлт (s = parcel_survey, p = parcel).
//
// Устгасан (soft delete) нэгж талбарын судалгаа тооцогдохгүй.
func surveyFilterWhere(filter domain.ParcelSurveyFilter) (string, []any) {
	conds := []string{"p.deleted_at IS NULL"}
	var args []any
	if len(filter.AcquisitionIDs) > 0 {
		args = append(args, filter.AcquisitionIDs)
		conds = append(conds, "p.acquisition_id = ANY($"+strconv.Itoa(len(args))+")")
	}
	if code := strings.TrimSpace(filter.ParcelCode); code != "" {
		args = append(args, "%"+code+"%")
		conds = append(conds, "p.parcel_id ILIKE $"+strconv.Itoa(len(args)))
	}
	if filter.Answered != nil {
		if *filter.Answered {
			conds = append(conds, surveyAnsweredSQL)
		} else {
			conds = append(conds, "NOT "+surveyAnsweredSQL)
		}
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (r *parcelSurveyRepo) List(ctx context.Context, filter domain.ParcelSurveyFilter) ([]*domain.ParcelSurveyRow, int64, error) {
	where, args := surveyFilterWhere(filter)
	const from = ` FROM parcel_survey s
		JOIN parcel p ON p.id = s.parcel_id
		JOIN land_acquisition la ON la.id = p.acquisition_id`

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*)`+from+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := filter.PageSize, (filter.Page-1)*filter.PageSize
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT `+parcelSurveyColumns+`,
		       p.parcel_id, p.acquisition_id, la.acquisition_name, COALESCE(p.au3_code,'')`+
		from+where+`
		ORDER BY s.updated_at DESC, s.id
		LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]*domain.ParcelSurveyRow, 0)
	for rows.Next() {
		row := &domain.ParcelSurveyRow{}
		if err := scanParcelSurvey(rows, &row.ParcelSurvey,
			&row.ParcelCode, &row.AcquisitionID, &row.AcquisitionName, &row.AU3Code); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// Summary — асуулт бүрийн хариултын тоо. Нэг query-гээр (UNION ALL) нэгтгэнэ.
func (r *parcelSurveyRepo) Summary(ctx context.Context, filter domain.ParcelSurveyFilter) (*domain.ParcelSurveySummary, error) {
	where, args := surveyFilterWhere(filter)
	// Тоон хариултыг 1, 2, 3+ гэж ангилна (3-аас дээшх нь "3" түлхүүрт).
	bucketMax := strconv.Itoa(domain.SurveyCountBucketMax)
	rows, err := r.db.Query(ctx, `
		WITH f AS (
			SELECT s.*, `+surveyAnsweredSQL+` AS answered FROM parcel_survey s
			JOIN parcel p ON p.id = s.parcel_id`+where+`
		)
		SELECT 'total', '', COUNT(*) FROM f
		UNION ALL
		SELECT 'answered', '', COUNT(*) FROM f WHERE answered
		UNION ALL
		SELECT 'chimney', LEAST(chimney_count, `+bucketMax+`)::text, COUNT(*) FROM f WHERE chimney_count IS NOT NULL GROUP BY 2
		UNION ALL
		SELECT 'toilet', LEAST(toilet_count, `+bucketMax+`)::text, COUNT(*) FROM f WHERE toilet_count IS NOT NULL GROUP BY 2
		UNION ALL
		SELECT 'cohabitant', LEAST(cohabitant_count, `+bucketMax+`)::text, COUNT(*) FROM f WHERE cohabitant_count IS NOT NULL GROUP BY 2
		UNION ALL
		SELECT 'relocation', relocation_type, COUNT(*) FROM f WHERE relocation_type IS NOT NULL GROUP BY relocation_type`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sum := &domain.ParcelSurveySummary{
		ChimneyCounts:    map[string]int64{},
		ToiletCounts:     map[string]int64{},
		CohabitantCounts: map[string]int64{},
		RelocationCounts: map[string]int64{},
	}
	for rows.Next() {
		var kind, key string
		var n int64
		if err := rows.Scan(&kind, &key, &n); err != nil {
			return nil, err
		}
		switch kind {
		case "total":
			sum.Total = n
		case "answered":
			sum.Answered = n
		case "chimney":
			sum.ChimneyCounts[key] = n
		case "toilet":
			sum.ToiletCounts[key] = n
		case "cohabitant":
			sum.CohabitantCounts[key] = n
		case "relocation":
			sum.RelocationCounts[key] = n
		}
	}
	return sum, rows.Err()
}
