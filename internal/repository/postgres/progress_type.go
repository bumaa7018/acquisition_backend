package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/domain"
)

type ProgressTypeRepo struct {
	db *pgxpool.Pool
}

func NewProgressTypeRepo(db *pgxpool.Pool) *ProgressTypeRepo {
	return &ProgressTypeRepo{db: db}
}

func (r *ProgressTypeRepo) List(ctx context.Context) ([]*domain.ProgressType, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, sort_order FROM progress_type ORDER BY COALESCE(sort_order, id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ProgressType
	for rows.Next() {
		pt := &domain.ProgressType{}
		if err := rows.Scan(&pt.ID, &pt.Name, &pt.Description, &pt.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, pt)
	}
	return list, rows.Err()
}

func (r *ProgressTypeRepo) GetByID(ctx context.Context, id int) (*domain.ProgressType, error) {
	pt := &domain.ProgressType{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, sort_order FROM progress_type WHERE id = $1`, id).
		Scan(&pt.ID, &pt.Name, &pt.Description, &pt.SortOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return pt, nil
}

func (r *ProgressTypeRepo) Create(ctx context.Context, pt *domain.ProgressType) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO progress_type (name, description, sort_order) VALUES ($1, $2, $3) RETURNING id`,
		pt.Name, pt.Description, pt.SortOrder).Scan(&pt.ID)
}

func (r *ProgressTypeRepo) Update(ctx context.Context, pt *domain.ProgressType) error {
	res, err := r.db.Exec(ctx,
		`UPDATE progress_type SET name=$1, description=$2, sort_order=$3 WHERE id=$4`,
		pt.Name, pt.Description, pt.SortOrder, pt.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProgressTypeRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.Exec(ctx, `DELETE FROM progress_type WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
