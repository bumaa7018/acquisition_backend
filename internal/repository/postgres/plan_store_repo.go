package postgres

import (
	"context"
	"errors"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// planStoreRepo — MAIN DB-ийн plan хүснэгт. Дундын сервисээс татсан
// төлөвлөгөөг хадгалж, хайлт/сонголтод үүнээс уншина.
type planStoreRepo struct{ db *pgxpool.Pool }

func NewPlanStoreRepo(db *pgxpool.Pool) repository.PlanStoreRepository {
	return &planStoreRepo{db: db}
}

const planStoreColumns = `
	id, plan_code, COALESCE(project_id, ''), COALESCE(name, ''), COALESCE(plan_type_name, ''),
	COALESCE(ST_AsText(boundary_wkt), ''), COALESCE(area_m2, 0), status,
	start_data, end_data, approved_date, created_at, updated_at`

func scanPlan(row pgx.Row) (*domain.Plan, error) {
	var p domain.Plan
	err := row.Scan(
		&p.ID, &p.PlanCode, &p.ProjectID, &p.Name, &p.PlanTypeName,
		&p.BoundaryWKT, &p.AreaM2, &p.Status,
		&p.StartData, &p.EndData, &p.ApprovedDate, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	p.Code = p.PlanCode
	return &p, nil
}

// Upsert — plan_code-оор нэгтгэнэ (идемпотент).
//
// Хоосон утга ХУУЧНЫГ дарж бичихгүй: /plan/project хил (boundary) буцаадаггүй
// тул шинэ дуудалт хуучин хилийг NULL болгох ёсгүй. Мөн ID нь хуучин мөрийнх
// хэвээр үлдэнэ — land_acquisition.plan_id түүнд холбогддог.
func (r *planStoreRepo) Upsert(ctx context.Context, p *domain.Plan) (*domain.Plan, error) {
	if p == nil || p.PlanCode == "" {
		return nil, domain.ErrInvalidInput
	}

	id := p.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	var boundary *string
	if p.BoundaryWKT != "" {
		boundary = &p.BoundaryWKT
	}
	var area *float64
	if p.AreaM2 != 0 {
		area = &p.AreaM2
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO plan (
			id, plan_code, project_id, name, plan_type_name,
			boundary_wkt, area_m2, status, start_data, end_data, approved_date
		) VALUES (
			$1, $2, NULLIF($3, ''), $4, NULLIF($5, ''),
			CASE WHEN $6::text IS NULL THEN NULL ELSE ST_GeomFromText($6, 4326) END,
			$7, $8, $9, $10, $11
		)
		ON CONFLICT (plan_code) DO UPDATE SET
			project_id     = COALESCE(NULLIF(EXCLUDED.project_id, ''), plan.project_id),
			name           = COALESCE(NULLIF(EXCLUDED.name, ''), plan.name),
			plan_type_name = COALESCE(NULLIF(EXCLUDED.plan_type_name, ''), plan.plan_type_name),
			boundary_wkt   = COALESCE(EXCLUDED.boundary_wkt, plan.boundary_wkt),
			area_m2        = COALESCE(EXCLUDED.area_m2, plan.area_m2),
			status         = CASE WHEN EXCLUDED.status = 0 THEN plan.status ELSE EXCLUDED.status END,
			start_data     = COALESCE(EXCLUDED.start_data, plan.start_data),
			end_data       = COALESCE(EXCLUDED.end_data, plan.end_data),
			approved_date  = COALESCE(EXCLUDED.approved_date, plan.approved_date),
			updated_at     = NOW()
		RETURNING `+planStoreColumns,
		id, p.PlanCode, p.ProjectID, p.Name, p.PlanTypeName,
		boundary, area, p.Status, p.StartData, p.EndData, p.ApprovedDate,
	)
	return scanPlan(row)
}

func (r *planStoreRepo) GetByCode(ctx context.Context, planCode string) (*domain.Plan, error) {
	return scanPlan(r.db.QueryRow(ctx,
		`SELECT `+planStoreColumns+` FROM plan WHERE plan_code = $1`, planCode))
}

// Suggest — дугаар/нэрээр хайх (жагсаалт, шүүлтүүрийн сонголтод).
func (r *planStoreRepo) Suggest(ctx context.Context, q string) ([]*domain.Plan, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+planStoreColumns+`
		FROM plan
		WHERE plan_code ILIKE $1 OR name ILIKE $1
		ORDER BY plan_code
		LIMIT 20
	`, "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// compile-time check
var _ repository.PlanStoreRepository = (*planStoreRepo)(nil)
