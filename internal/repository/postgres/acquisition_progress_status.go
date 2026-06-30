package postgres

import (
	"context"
	"errors"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AcquisitionProgressStatusRepo struct {
	db *pgxpool.Pool
}

func NewAcquisitionProgressStatusRepo(db *pgxpool.Pool) *AcquisitionProgressStatusRepo {
	return &AcquisitionProgressStatusRepo{db: db}
}

func (r *AcquisitionProgressStatusRepo) List(ctx context.Context) ([]*domain.AcquisitionProgressStatus, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, sort_order FROM acquisition_progress_status ORDER BY COALESCE(sort_order, id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.AcquisitionProgressStatus
	for rows.Next() {
		aps := &domain.AcquisitionProgressStatus{}
		if err := rows.Scan(&aps.ID, &aps.Name, &aps.Description, &aps.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, aps)
	}
	return list, rows.Err()
}

func (r *AcquisitionProgressStatusRepo) GetByID(ctx context.Context, id int) (*domain.AcquisitionProgressStatus, error) {
	aps := &domain.AcquisitionProgressStatus{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, sort_order FROM acquisition_progress_status WHERE id = $1`, id).
		Scan(&aps.ID, &aps.Name, &aps.Description, &aps.SortOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return aps, nil
}

func (r *AcquisitionProgressStatusRepo) Create(ctx context.Context, aps *domain.AcquisitionProgressStatus) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO acquisition_progress_status (name, description, sort_order) VALUES ($1, $2, $3) RETURNING id`,
		aps.Name, aps.Description, aps.SortOrder).Scan(&aps.ID)
}

func (r *AcquisitionProgressStatusRepo) Update(ctx context.Context, aps *domain.AcquisitionProgressStatus) error {
	res, err := r.db.Exec(ctx,
		`UPDATE acquisition_progress_status SET name=$1, description=$2, sort_order=$3 WHERE id=$4`,
		aps.Name, aps.Description, aps.SortOrder, aps.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AcquisitionProgressStatusRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.Exec(ctx, `DELETE FROM acquisition_progress_status WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
