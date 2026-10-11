package postgres

import (
	"context"
	"errors"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ParcelStatusRepo struct {
	db *pgxpool.Pool
}

func NewParcelStatusRepo(db *pgxpool.Pool) *ParcelStatusRepo {
	return &ParcelStatusRepo{db: db}
}

func (r *ParcelStatusRepo) List(ctx context.Context) ([]*domain.ParcelStatus, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, name, sort_order, COALESCE(color, ''),
		        is_final, locks_progress, is_valuation_stage, is_released, requires_reason
		   FROM parcel_status ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ParcelStatus
	for rows.Next() {
		ps := &domain.ParcelStatus{}
		if err := rows.Scan(&ps.ID, &ps.Code, &ps.Name, &ps.SortOrder, &ps.Color,
			&ps.IsFinal, &ps.LocksProgress, &ps.IsValuationStage, &ps.IsReleased, &ps.RequiresReason); err != nil {
			return nil, err
		}
		list = append(list, ps)
	}
	return list, rows.Err()
}

func (r *ParcelStatusRepo) GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error) {
	ps := &domain.ParcelStatus{}
	err := r.db.QueryRow(ctx,
		`SELECT id, code, name, sort_order, COALESCE(color, ''),
		        is_final, locks_progress, is_valuation_stage, is_released, requires_reason
		   FROM parcel_status WHERE id = $1`, id).
		Scan(&ps.ID, &ps.Code, &ps.Name, &ps.SortOrder, &ps.Color,
			&ps.IsFinal, &ps.LocksProgress, &ps.IsValuationStage, &ps.IsReleased, &ps.RequiresReason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ps, nil
}

// Create — id нь sequence-ээс (seeder 109 дээр тавьсан тул 110…), code нь
// түүнтэй ИЖИЛ.
func (r *ParcelStatusRepo) Create(ctx context.Context, ps *domain.ParcelStatus) error {
	return r.db.QueryRow(ctx,
		`WITH n AS (SELECT nextval('parcel_status_id_seq')::int AS id)
		 INSERT INTO parcel_status (id, code, name, sort_order, color,
		        is_final, locks_progress, is_valuation_stage, is_released, requires_reason)
		 SELECT n.id, n.id::text, $1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8 FROM n
		 RETURNING id, code`,
		ps.Name, ps.SortOrder, ps.Color,
		ps.IsFinal, ps.LocksProgress, ps.IsValuationStage, ps.IsReleased, ps.RequiresReason).Scan(&ps.ID, &ps.Code)
}

// Update — code-г хөндөхгүй (id-тай ижил хэвээр).
func (r *ParcelStatusRepo) Update(ctx context.Context, ps *domain.ParcelStatus) error {
	res, err := r.db.Exec(ctx,
		`UPDATE parcel_status SET name=$1, sort_order=$2, color=NULLIF($3, ''),
		        is_final=$5, locks_progress=$6, is_valuation_stage=$7, is_released=$8, requires_reason=$9
		  WHERE id=$4`,
		ps.Name, ps.SortOrder, ps.Color, ps.ID,
		ps.IsFinal, ps.LocksProgress, ps.IsValuationStage, ps.IsReleased, ps.RequiresReason)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ParcelStatusRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.Exec(ctx, `DELETE FROM parcel_status WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
