package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/repository"
)

type planRepo struct{ db *pgxpool.Pool }

func NewPlanRepo(db *pgxpool.Pool) repository.PlanRepository {
	return &planRepo{db: db}
}

func (r *planRepo) GetByCode(ctx context.Context, planCode string) (*domain.Plan, error) {
	var p domain.Plan
	err := r.db.QueryRow(ctx, `
		SELECT id, plan_code, name, ST_AsText(boundary_wkt), COALESCE(area_m2, 0), status, created_at, updated_at
		FROM plan
		WHERE plan_code = $1
	`, planCode).Scan(
		&p.ID, &p.PlanCode, &p.Name, &p.BoundaryWKT,
		&p.AreaM2, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *planRepo) GetSourceParcels(ctx context.Context, planCode string) ([]*domain.SourceParcel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, parcel_id, plan_code, ST_AsText(geometry_wkt),
		       COALESCE(au1_code,''), COALESCE(au2_code,''), COALESCE(au3_code,''),
		       COALESCE(right_type,0), COALESCE(landuse,''), COALESCE(area_m2,0),
		       COALESCE(holder_last_name,''), COALESCE(holder_name,''),
		       COALESCE(holder_register_no,''), COALESCE(holder_phone,''),
		       COALESCE(holder_email,''), COALESCE(holder_type,''), COALESCE(holder_civil_id,''),
		       COALESCE(app_no,''), COALESCE(decision_no,''), decision_date,
		       COALESCE(contract_no,''), contract_date,
		       COALESCE(certificate_no,''), certificate_date,
		       created_at
		FROM source_parcel
		WHERE plan_code = $1
	`, planCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.SourceParcel
	for rows.Next() {
		sp := &domain.SourceParcel{}
		err := rows.Scan(
			&sp.ID, &sp.ParcelID, &sp.PlanCode, &sp.GeometryWKT,
			&sp.AU1Code, &sp.AU2Code, &sp.AU3Code,
			&sp.RightType, &sp.Landuse, &sp.AreaM2,
			&sp.HolderLastName, &sp.HolderName,
			&sp.HolderRegisterNo, &sp.HolderPhone,
			&sp.HolderEmail, &sp.HolderType, &sp.HolderCivilID,
			&sp.AppNo, &sp.DecisionNo, &sp.DecisionDate,
			&sp.ContractNo, &sp.ContractDate,
			&sp.CertificateNo, &sp.CertificateDate,
			&sp.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, sp)
	}
	return result, rows.Err()
}

func (r *planRepo) GetSourceParcelByID(ctx context.Context, parcelID string) (*domain.SourceParcel, error) {
	sp := &domain.SourceParcel{}
	err := r.db.QueryRow(ctx, `
		SELECT id, parcel_id, plan_code, ST_AsText(geometry_wkt),
		       COALESCE(au1_code,''), COALESCE(au2_code,''), COALESCE(au3_code,''),
		       COALESCE(right_type,0), COALESCE(landuse,''), COALESCE(area_m2,0),
		       COALESCE(holder_last_name,''), COALESCE(holder_name,''),
		       COALESCE(holder_register_no,''), COALESCE(holder_phone,''),
		       COALESCE(holder_email,''), COALESCE(holder_type,''), COALESCE(holder_civil_id,''),
		       COALESCE(app_no,''), COALESCE(decision_no,''), decision_date,
		       COALESCE(contract_no,''), contract_date,
		       COALESCE(certificate_no,''), certificate_date,
		       created_at
		FROM source_parcel
		WHERE parcel_id = $1
	`, parcelID).Scan(
		&sp.ID, &sp.ParcelID, &sp.PlanCode, &sp.GeometryWKT,
		&sp.AU1Code, &sp.AU2Code, &sp.AU3Code,
		&sp.RightType, &sp.Landuse, &sp.AreaM2,
		&sp.HolderLastName, &sp.HolderName,
		&sp.HolderRegisterNo, &sp.HolderPhone,
		&sp.HolderEmail, &sp.HolderType, &sp.HolderCivilID,
		&sp.AppNo, &sp.DecisionNo, &sp.DecisionDate,
		&sp.ContractNo, &sp.ContractDate,
		&sp.CertificateNo, &sp.CertificateDate,
		&sp.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return sp, nil
}

// compile-time check
var _ repository.PlanRepository = (*planRepo)(nil)

// helper — used only in this file
func ptrTime(t time.Time) *time.Time  { return &t }
func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
