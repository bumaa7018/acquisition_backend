package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/domain"
)

type DocumentTypeRepo struct {
	db *pgxpool.Pool
}

func NewDocumentTypeRepo(db *pgxpool.Pool) *DocumentTypeRepo {
	return &DocumentTypeRepo{db: db}
}

func (r *DocumentTypeRepo) List(ctx context.Context) ([]*domain.DocumentType, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, type, name, description, created_at, updated_at, created_by, updated_by
		 FROM document_type ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DocumentType
	for rows.Next() {
		dt := &domain.DocumentType{}
		if err := rows.Scan(
			&dt.ID, &dt.Type, &dt.Name, &dt.Description,
			&dt.CreatedAt, &dt.UpdatedAt, &dt.CreatedBy, &dt.UpdatedBy,
		); err != nil {
			return nil, err
		}
		list = append(list, dt)
	}
	return list, rows.Err()
}

func (r *DocumentTypeRepo) GetByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	dt := &domain.DocumentType{}
	err := r.db.QueryRow(ctx,
		`SELECT id, type, name, description, created_at, updated_at, created_by, updated_by
		 FROM document_type WHERE id = $1`, id).
		Scan(&dt.ID, &dt.Type, &dt.Name, &dt.Description,
			&dt.CreatedAt, &dt.UpdatedAt, &dt.CreatedBy, &dt.UpdatedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return dt, nil
}

func (r *DocumentTypeRepo) Create(ctx context.Context, dt *domain.DocumentType) error {
	now := time.Now()
	dt.CreatedAt = now
	dt.UpdatedAt = now
	return r.db.QueryRow(ctx,
		`INSERT INTO document_type (type, name, description, created_at, updated_at, created_by, updated_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		dt.Type, dt.Name, dt.Description, dt.CreatedAt, dt.UpdatedAt, dt.CreatedBy, dt.UpdatedBy,
	).Scan(&dt.ID)
}

func (r *DocumentTypeRepo) Update(ctx context.Context, dt *domain.DocumentType) error {
	dt.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx,
		`UPDATE document_type SET type=$1, name=$2, description=$3, updated_at=$4, updated_by=$5 WHERE id=$6`,
		dt.Type, dt.Name, dt.Description, dt.UpdatedAt, dt.UpdatedBy, dt.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DocumentTypeRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.Exec(ctx, `DELETE FROM document_type WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
