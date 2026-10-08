package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentTypeRepo struct {
	db *pgxpool.Pool
}

func NewDocumentTypeRepo(db *pgxpool.Pool) *DocumentTypeRepo {
	return &DocumentTypeRepo{db: db}
}

func (r *DocumentTypeRepo) List(ctx context.Context, target string) ([]*domain.DocumentType, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, type, name, target, description, created_at, updated_at, created_by, updated_by
		 FROM document_type
		 WHERE ($1 = '' OR target = $1 OR target = 'both')
		 ORDER BY created_at DESC NULLS LAST, id DESC`, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DocumentType
	for rows.Next() {
		dt := &domain.DocumentType{}
		if err := rows.Scan(
			&dt.ID, &dt.Type, &dt.Name, &dt.Target, &dt.Description,
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
		`SELECT id, type, name, target, description, created_at, updated_at, created_by, updated_by
		 FROM document_type WHERE id = $1`, id).
		Scan(&dt.ID, &dt.Type, &dt.Name, &dt.Target, &dt.Description,
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
	if dt.Target == "" {
		dt.Target = "both"
	}
	return r.db.QueryRow(ctx,
		`INSERT INTO document_type (type, name, target, description, created_at, updated_at, created_by, updated_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		dt.Type, dt.Name, dt.Target, dt.Description, dt.CreatedAt, dt.UpdatedAt, dt.CreatedBy, dt.UpdatedBy,
	).Scan(&dt.ID)
}

func (r *DocumentTypeRepo) Update(ctx context.Context, dt *domain.DocumentType) error {
	dt.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx,
		`UPDATE document_type SET type=$1, name=$2, target=$3, description=$4, updated_at=$5, updated_by=$6 WHERE id=$7`,
		dt.Type, dt.Name, dt.Target, dt.Description, dt.UpdatedAt, dt.UpdatedBy, dt.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete — ашиглагдаж буй (хавсралт, ажлын урсгалын шаардлага) эсвэл
// системийн кодтой төрлийг устгахгүй — *domain.InUseError.
func (r *DocumentTypeRepo) Delete(ctx context.Context, id int) error {
	var code, name string
	var docs, parcelDocs, workflowReqs int
	err := r.db.QueryRow(ctx, `
		SELECT dt.type, dt.name,
		       (SELECT COUNT(*) FROM document d WHERE d.document_type_id = dt.id),
		       (SELECT COUNT(*) FROM document d JOIN parcel_document pd ON pd.document_id = d.id
		         WHERE d.document_type_id = dt.id),
		       (SELECT COUNT(*) FROM parcel_workflow_required_document rd WHERE rd.document_type_id = dt.id)
		  FROM document_type dt WHERE dt.id = $1`, id).
		Scan(&code, &name, &docs, &parcelDocs, &workflowReqs)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if domain.SystemDocumentTypeCodes[code] {
		return &domain.InUseError{Message: fmt.Sprintf(
			"«%s» нь системийн хавсралтын төрөл (код: %s) тул устгах боломжгүй.", name, code)}
	}
	if docs > 0 || workflowReqs > 0 {
		var parts []string
		if docs > 0 {
			parts = append(parts, fmt.Sprintf("%d хавсралт (нэгж талбар %d, чөлөөлөлт %d)", docs, parcelDocs, docs-parcelDocs))
		}
		if workflowReqs > 0 {
			parts = append(parts, fmt.Sprintf("нэгж талбарын ажлын урсгалын %d шаардлага", workflowReqs))
		}
		return &domain.InUseError{Message: fmt.Sprintf(
			"«%s» хавсралтын төрлийг ашиглаж байгаа тул устгах боломжгүй: %s.", name, strings.Join(parts, ", "))}
	}
	res, err := r.db.Exec(ctx, `DELETE FROM document_type WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
