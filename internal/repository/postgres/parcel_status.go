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

func (r *ParcelStatusRepo) Create(ctx context.Context, ps *domain.ParcelStatus) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO parcel_status (code, name, sort_order, color,
		        is_final, locks_progress, is_valuation_stage, is_released, requires_reason)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9) RETURNING id`,
		ps.Code, ps.Name, ps.SortOrder, ps.Color,
		ps.IsFinal, ps.LocksProgress, ps.IsValuationStage, ps.IsReleased, ps.RequiresReason).Scan(&ps.ID)
}

func (r *ParcelStatusRepo) Update(ctx context.Context, ps *domain.ParcelStatus) error {
	res, err := r.db.Exec(ctx,
		`UPDATE parcel_status SET code=$1, name=$2, sort_order=$3, color=NULLIF($4, ''),
		        is_final=$6, locks_progress=$7, is_valuation_stage=$8, is_released=$9, requires_reason=$10
		  WHERE id=$5`,
		ps.Code, ps.Name, ps.SortOrder, ps.Color, ps.ID,
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
