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
		`SELECT id, code, name, sort_order, COALESCE(color, '')
		   FROM parcel_status ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ParcelStatus
	for rows.Next() {
		ps := &domain.ParcelStatus{}
		if err := rows.Scan(&ps.ID, &ps.Code, &ps.Name, &ps.SortOrder, &ps.Color); err != nil {
			return nil, err
		}
		list = append(list, ps)
	}
	return list, rows.Err()
}

func (r *ParcelStatusRepo) GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error) {
	ps := &domain.ParcelStatus{}
	err := r.db.QueryRow(ctx,
		`SELECT id, code, name, sort_order, COALESCE(color, '')
		   FROM parcel_status WHERE id = $1`, id).
		Scan(&ps.ID, &ps.Code, &ps.Name, &ps.SortOrder, &ps.Color)
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
		`INSERT INTO parcel_status (code, name, sort_order, color)
		 VALUES ($1, $2, $3, NULLIF($4, '')) RETURNING id`,
		ps.Code, ps.Name, ps.SortOrder, ps.Color).Scan(&ps.ID)
}

func (r *ParcelStatusRepo) Update(ctx context.Context, ps *domain.ParcelStatus) error {
	res, err := r.db.Exec(ctx,
		`UPDATE parcel_status SET code=$1, name=$2, sort_order=$3, color=NULLIF($4, '')
		  WHERE id=$5`,
		ps.Code, ps.Name, ps.SortOrder, ps.Color, ps.ID)
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
