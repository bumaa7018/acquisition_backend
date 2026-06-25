package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/repository"
)

type landAcqRepo struct{ db *pgxpool.Pool }

func NewLandAcquisitionRepo(db *pgxpool.Pool) repository.LandAcquisitionRepository {
	return &landAcqRepo{db: db}
}

func (r *landAcqRepo) ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, name, sort_order FROM construction_type ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.ConstructionType
	for rows.Next() {
		ct := &domain.ConstructionType{}
		if err := rows.Scan(&ct.ID, &ct.Code, &ct.Name, &ct.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, ct)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, name, sort_order FROM parcel_status ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.ParcelStatus
	for rows.Next() {
		status := &domain.ParcelStatus{}
		if err := rows.Scan(&status.ID, &status.Code, &status.Name, &status.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, status)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) Create(ctx context.Context, acq *domain.LandAcquisition) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO land_acquisition (
			id, plan_id, plan_code, plan_name,
			plan_geom, plan_area_m2,
			geometry, area_m2,
			status, start_date, end_date,
			created_at, created_by, updated_at, updated_by,
			acquisition_name, implementing_org, reason, responsible_org, funding_source,
			construction_type_id
			) VALUES (
				$1, $2, $3, $4,
				ST_GeomFromText(NULLIF($5, ''), 4326)::geometry(Polygon, 4326), $6,
				ST_GeomFromText(NULLIF($7, ''), 4326)::geometry(Polygon, 4326), $8,
				$9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18, $19, $20,
				$21
			)
	`,
		acq.ID, acq.PlanID, acq.PlanCode, acq.PlanName,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.Status, acq.StartDate, acq.EndDate,
		acq.CreatedAt, acq.CreatedBy, acq.UpdatedAt, acq.UpdatedBy,
		acq.AcquisitionName, acq.ImplementingOrg, acq.Reason, acq.ResponsibleOrg, acq.FundingSource,
		acq.ConstructionTypeID,
	)
	return err
}

func (r *landAcqRepo) List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error) {
	args := []any{}
	where := "1=1"
	i := 1

	if filter.PlanCode != "" {
		where += fmt.Sprintf(" AND plan_code ILIKE $%d", i)
		args = append(args, "%"+filter.PlanCode+"%")
		i++
	}
	if filter.Status != 0 {
		where += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, filter.Status)
		i++
	}
	if filter.StartDateFrom != nil {
		where += fmt.Sprintf(" AND start_date >= $%d", i)
		args = append(args, filter.StartDateFrom)
		i++
	}
	if filter.StartDateTo != nil {
		where += fmt.Sprintf(" AND start_date <= $%d", i)
		args = append(args, filter.StartDateTo)
		i++
	}
	if filter.AU3Code != "" {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM land_acquisition_au
			WHERE land_acquisition_id = land_acquisition.id AND au3_code = $%d
		)`, i)
		args = append(args, filter.AU3Code)
		i++
	}
	if len(filter.Years) > 0 {
		placeholders := make([]string, len(filter.Years))
		for j, yr := range filter.Years {
			placeholders[j] = fmt.Sprintf("$%d", i)
			args = append(args, yr)
			i++
		}
		where += fmt.Sprintf(" AND EXTRACT(YEAR FROM start_date) IN (%s)", strings.Join(placeholders, ","))
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM land_acquisition WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT la.id, la.plan_id, la.plan_code, COALESCE(la.plan_name,''),
			       COALESCE(ST_AsText(la.plan_geom),''), COALESCE(la.plan_area_m2,0),
			       COALESCE(ST_AsText(la.geometry),''), COALESCE(la.area_m2,0),
			       la.status, la.start_date, la.end_date,
			       la.created_at, COALESCE(la.created_by,''),
			       la.updated_at, COALESCE(la.updated_by,''),
			       COALESCE(la.acquisition_name,''), COALESCE(la.implementing_org,''),
			       COALESCE(la.reason,''), COALESCE(la.responsible_org,''), COALESCE(la.funding_source,''),
			       la.construction_type_id, COALESCE(ct.name,''),
			       (SELECT COUNT(*) FROM parcel WHERE acquisition_id = la.id)
		FROM land_acquisition la
		LEFT JOIN construction_type ct ON ct.id = la.construction_type_id
		WHERE %s
		ORDER BY la.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, i, i+1), append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.LandAcquisition
	for rows.Next() {
		acq := &domain.LandAcquisition{}
		if err := rows.Scan(
			&acq.ID, &acq.PlanID, &acq.PlanCode, &acq.PlanName,
			&acq.PlanGeomWKT, &acq.PlanAreaM2,
			&acq.GeometryWKT, &acq.AreaM2,
			&acq.Status, &acq.StartDate, &acq.EndDate,
			&acq.CreatedAt, &acq.CreatedBy,
			&acq.UpdatedAt, &acq.UpdatedBy,
			&acq.AcquisitionName, &acq.ImplementingOrg,
			&acq.Reason, &acq.ResponsibleOrg, &acq.FundingSource,
			&acq.ConstructionTypeID, &acq.ConstructionTypeName,
			&acq.ParcelCount,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, acq)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error) {
	acq := &domain.LandAcquisition{}
	err := r.db.QueryRow(ctx, `
			SELECT la.id, la.plan_id, la.plan_code, COALESCE(la.plan_name,''),
			       COALESCE(ST_AsText(la.plan_geom),''), COALESCE(la.plan_area_m2,0),
			       COALESCE(ST_AsText(la.geometry),''), COALESCE(la.area_m2,0),
			       la.status, la.start_date, la.end_date,
			       la.created_at, COALESCE(la.created_by,''),
			       la.updated_at, COALESCE(la.updated_by,''),
			       COALESCE(la.acquisition_name,''), COALESCE(la.implementing_org,''),
			       COALESCE(la.reason,''), COALESCE(la.responsible_org,''), COALESCE(la.funding_source,''),
			       la.construction_type_id, COALESCE(ct.name,''),
			       COALESCE(la.decree_number,''), la.decree_date
		FROM land_acquisition la
		LEFT JOIN construction_type ct ON ct.id = la.construction_type_id
		WHERE la.id = $1
	`, id).Scan(
		&acq.ID, &acq.PlanID, &acq.PlanCode, &acq.PlanName,
		&acq.PlanGeomWKT, &acq.PlanAreaM2,
		&acq.GeometryWKT, &acq.AreaM2,
		&acq.Status, &acq.StartDate, &acq.EndDate,
		&acq.CreatedAt, &acq.CreatedBy,
		&acq.UpdatedAt, &acq.UpdatedBy,
		&acq.AcquisitionName, &acq.ImplementingOrg,
		&acq.Reason, &acq.ResponsibleOrg, &acq.FundingSource,
		&acq.ConstructionTypeID, &acq.ConstructionTypeName,
		&acq.DecreeNumber, &acq.DecreeDate,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	aus, err := r.getAUs(ctx, id)
	if err != nil {
		return nil, err
	}
	acq.AUs = aus

	parcels, total, err := r.GetParcels(ctx, id, domain.ParcelFilter{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, err
	}
	acq.Parcels = parcels
	acq.ParcelCount = total

	return acq, nil
}

func (r *landAcqRepo) getAUs(ctx context.Context, acqID uuid.UUID) ([]domain.LandAcquisitionAU, error) {
	rows, err := r.db.Query(ctx, `
		SELECT land_acquisition_id,
		       COALESCE(au1_code,''), COALESCE(au1_name,''),
		       COALESCE(au2_code,''), COALESCE(au2_name,''),
		       au3_code, COALESCE(au3_name,'')
		FROM land_acquisition_au
		WHERE land_acquisition_id = $1
	`, acqID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aus []domain.LandAcquisitionAU
	for rows.Next() {
		var au domain.LandAcquisitionAU
		if err := rows.Scan(
			&au.LandAcquisitionID,
			&au.AU1Code, &au.AU1Name,
			&au.AU2Code, &au.AU2Name,
			&au.AU3Code, &au.AU3Name,
		); err != nil {
			return nil, err
		}
		aus = append(aus, au)
	}
	return aus, rows.Err()
}

// ValidateWithin uses PostGIS to check inner ⊂ outer.
func (r *landAcqRepo) ValidateWithin(ctx context.Context, innerWKT, outerWKT string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
			SELECT ST_Within(
				ST_GeomFromText($1, 4326)::geometry(Polygon, 4326),
				ST_GeomFromText($2, 4326)::geometry(Polygon, 4326)
			)
		`, innerWKT, outerWKT).Scan(&ok)
	return ok, err
}

// FindIntersectingAU3 returns bags (au3) that intersect with the given geometry.
func (r *landAcqRepo) FindIntersectingAU3(ctx context.Context, geomWKT string) ([]domain.LandAcquisitionAU, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a3.code, COALESCE(a3.name,''),
		       a3.au1_code, COALESCE(a1.name,''),
		       a3.au2_code, COALESCE(a2.name,'')
		FROM au3 a3
		JOIN au1 a1 ON a1.code = a3.au1_code
		JOIN au2 a2 ON a2.code = a3.au2_code
			WHERE a3.geometry IS NOT NULL
			  AND ST_Intersects(a3.geometry, ST_GeomFromText($1, 4326)::geometry(Polygon, 4326))
	`, geomWKT)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aus []domain.LandAcquisitionAU
	for rows.Next() {
		var au domain.LandAcquisitionAU
		if err := rows.Scan(
			&au.AU3Code, &au.AU3Name,
			&au.AU1Code, &au.AU1Name,
			&au.AU2Code, &au.AU2Name,
		); err != nil {
			return nil, err
		}
		aus = append(aus, au)
	}
	return aus, rows.Err()
}

func (r *landAcqRepo) CreateAU(ctx context.Context, au *domain.LandAcquisitionAU) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		INSERT INTO land_acquisition_au (
			land_acquisition_id,
			au1_code, au1_name, au2_code, au2_name, au3_code, au3_name,
			created_at, created_by, updated_at, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'system', $9, 'system')
		ON CONFLICT (land_acquisition_id, au3_code) DO NOTHING
	`,
		au.LandAcquisitionID,
		au.AU1Code, au.AU1Name, au.AU2Code, au.AU2Name, au.AU3Code, au.AU3Name,
		now, now,
	)
	return err
}

// BulkInsertParcels inserts source parcels that intersect the acquisition geometry.
func (r *landAcqRepo) BulkInsertParcels(ctx context.Context, acquisitionID uuid.UUID, parcels []*domain.SourceParcel, acqWKT string) error {
	now := time.Now()
	for _, sp := range parcels {
		var intersects bool
		if err := r.db.QueryRow(ctx, `
				SELECT ST_Intersects(
					ST_GeomFromText($1, 4326)::geometry(Polygon, 4326),
					ST_GeomFromText($2, 4326)::geometry(Polygon, 4326)
				)
			`, sp.GeometryWKT, acqWKT).Scan(&intersects); err != nil {
			continue // невалид geometry → skip
		}
		if !intersects {
			continue
		}

		_, err := r.db.Exec(ctx, `
				WITH geom AS (
					SELECT
						ST_GeomFromText($6, 4326)::geometry(Polygon, 4326) AS parcel_geom,
						ST_GeomFromText($7, 4326)::geometry(Polygon, 4326) AS acquisition_geom
				),
				overlap AS (
					SELECT ST_GeometryN(ST_CollectionExtract(ST_Intersection(parcel_geom, acquisition_geom), 3), 1)::geometry(Polygon, 4326) AS geom
					FROM geom
				)
				INSERT INTO parcel (
					id, parcel_id, acquisition_id,
					au1_code, au2_code, au3_code,
					geometry,
				acquisition_geom,
				acquisition_area_m2,
				right_type, landuse, area_m2,
				created_at, created_by, updated_at, updated_by
				)
				SELECT
					uuid_generate_v4(), $1, $2,
					$3, $4, $5,
					geom.parcel_geom,
					overlap.geom,
					ST_Area(overlap.geom::geography),
					$8, $9, $10,
					$11, 'system', $12, 'system'
				FROM geom, overlap
				WHERE overlap.geom IS NOT NULL
				  AND NOT ST_IsEmpty(overlap.geom)
				ON CONFLICT (parcel_id) DO NOTHING
			`,
			sp.ParcelID, acquisitionID,
			sp.AU1Code, sp.AU2Code, sp.AU3Code,
			sp.GeometryWKT, acqWKT,
			sp.RightType, sp.Landuse, sp.AreaM2,
			now, now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *landAcqRepo) GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error) {
	args := []any{acquisitionID}
	where := "acquisition_id = $1"
	i := 2

	if filter.ParcelID != "" {
		where += fmt.Sprintf(" AND parcel_id ILIKE $%d", i)
		args = append(args, "%"+filter.ParcelID+"%")
		i++
	}
	if filter.AU1Code != "" {
		where += fmt.Sprintf(" AND au1_code = $%d", i)
		args = append(args, filter.AU1Code)
		i++
	}
	if filter.AU2Code != "" {
		where += fmt.Sprintf(" AND au2_code = $%d", i)
		args = append(args, filter.AU2Code)
		i++
	}
	if filter.AU3Code != "" {
		where += fmt.Sprintf(" AND au3_code = $%d", i)
		args = append(args, filter.AU3Code)
		i++
	}
	if filter.RightType != 0 {
		where += fmt.Sprintf(" AND right_type = $%d", i)
		args = append(args, filter.RightType)
		i++
	}
	if filter.Landuse != "" {
		where += fmt.Sprintf(" AND landuse = $%d", i)
		args = append(args, filter.Landuse)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM parcel WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := r.db.Query(ctx,
		fmt.Sprintf(`
				SELECT id, COALESCE(parcel_id,''), COALESCE(old_parcel_id,''),
				       acquisition_id, valid_from, valid_till,
				       COALESCE(au1_code,''), COALESCE(au2_code,''), COALESCE(au3_code,''),
				       COALESCE(ST_AsText(geometry),''),
				       COALESCE(acquisition_area_m2,0),
				       COALESCE(right_type,0), COALESCE(landuse,''), COALESCE(area_m2,0),
				       COALESCE(compensation_paid,false),
				       created_at, COALESCE(created_by,''),
				       COALESCE(remaining_area_m2, area_m2 - COALESCE(acquisition_area_m2,0)),
				       COALESCE(db_changed,false), COALESCE(changed_parcel_id,'')
			FROM parcel
			WHERE %s
			ORDER BY parcel_id
			LIMIT $%d OFFSET $%d
		`, where, i, i+1),
		append(args, pageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.Parcel
	for rows.Next() {
		p := &domain.Parcel{}
		var remainingAreaM2 float64
		if err := rows.Scan(
			&p.ID, &p.ParcelID, &p.OldParcelID,
			&p.AcquisitionID, &p.ValidFrom, &p.ValidTill,
			&p.AU1Code, &p.AU2Code, &p.AU3Code,
			&p.GeometryWKT,
			&p.AcquisitionAreaM2,
			&p.RightType, &p.Landuse, &p.AreaM2,
			&p.CompensationPaid,
			&p.CreatedAt, &p.CreatedBy,
			&remainingAreaM2,
			&p.DbChanged, &p.ChangedParcelID,
		); err != nil {
			return nil, 0, err
		}
		p.RemainingAreaM2 = &remainingAreaM2
		result = append(result, p)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error) {
	p := &domain.Parcel{}
	var remainingAreaM2 float64
	err := r.db.QueryRow(ctx, `
			SELECT id, COALESCE(parcel_id,''), COALESCE(old_parcel_id,''),
			       acquisition_id, valid_from, valid_till,
			       COALESCE(au1_code,''), COALESCE(au2_code,''), COALESCE(au3_code,''),
			       COALESCE(ST_AsText(geometry),''),
			       COALESCE(ST_AsText(acquisition_geom),''),
			       COALESCE(acquisition_area_m2,0),
			       COALESCE(right_type,0), COALESCE(landuse,''), COALESCE(area_m2,0),
			       created_at, COALESCE(created_by,''),
			       COALESCE(remaining_area_m2, area_m2 - COALESCE(acquisition_area_m2,0)),
			       COALESCE(db_changed,false), COALESCE(changed_parcel_id,'')
		FROM parcel
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.ParcelID, &p.OldParcelID,
		&p.AcquisitionID, &p.ValidFrom, &p.ValidTill,
		&p.AU1Code, &p.AU2Code, &p.AU3Code,
		&p.GeometryWKT,
		&p.AcquisitionGeomWKT,
		&p.AcquisitionAreaM2,
		&p.RightType, &p.Landuse, &p.AreaM2,
		&p.CreatedAt, &p.CreatedBy,
		&remainingAreaM2,
		&p.DbChanged, &p.ChangedParcelID,
	)
	if err == nil {
		p.RemainingAreaM2 = &remainingAreaM2
	}
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (r *landAcqRepo) GetParcelDetailByID(ctx context.Context, id uuid.UUID) (*domain.ParcelFull, error) {
	pf := &domain.ParcelFull{}
	d := &domain.ParcelDetailInfo{}

	var pfRemainingAreaM2 float64
	err := r.db.QueryRow(ctx, `
		SELECT
				p.id, COALESCE(p.parcel_id,''), COALESCE(p.old_parcel_id,''),
				p.acquisition_id, p.valid_from, p.valid_till,
				COALESCE(p.au1_code,''), COALESCE(p.au2_code,''), COALESCE(p.au3_code,''),
				COALESCE(ST_AsText(p.geometry),''),
				COALESCE(ST_AsText(p.acquisition_geom),''),
				COALESCE(p.acquisition_area_m2,0),
				COALESCE(p.right_type,0), COALESCE(p.landuse,''), COALESCE(p.area_m2,0),
				p.created_at, COALESCE(p.created_by,''),
				COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2,0)),
				COALESCE(p.db_changed,false), COALESCE(p.changed_parcel_id,''),
			COALESCE(pd.right_type,0),
			COALESCE(pd.app_no,''), pd.app_timestamps,
			COALESCE(pd.decision_no,''), pd.decision_date,
			COALESCE(pd.contract_no,''), pd.contract_date,
			COALESCE(pd.certificate_no,''), pd.certificate_date,
			COALESCE(rh.last_name,''), COALESCE(rh.name,''),
			COALESCE(rh.register_no,''), COALESCE(rh.phone,''),
			COALESCE(rh.email,''), COALESCE(rh.holder_type,''),
			COALESCE(rh.civil_id,''),
			COALESCE(pd.valuation_zone,''), pd.base_price_per_ha, pd.auction_coeff, pd.auction_price
		FROM parcel p
		LEFT JOIN parcel_detail pd ON pd.parcel_id = p.id
		LEFT JOIN right_holder rh ON rh.id = pd.holder_id
		WHERE p.id = $1
	`, id).Scan(
		&pf.ID, &pf.ParcelID, &pf.OldParcelID,
		&pf.AcquisitionID, &pf.ValidFrom, &pf.ValidTill,
		&pf.AU1Code, &pf.AU2Code, &pf.AU3Code,
		&pf.GeometryWKT,
		&pf.AcquisitionGeomWKT,
		&pf.AcquisitionAreaM2,
		&pf.RightType, &pf.Landuse, &pf.AreaM2,
		&pf.CreatedAt, &pf.CreatedBy,
		&pfRemainingAreaM2,
		&pf.DbChanged, &pf.ChangedParcelID,
		&d.RightType,
		&d.AppNo, &d.AppTimestamps,
		&d.DecisionNo, &d.DecisionDate,
		&d.ContractNo, &d.ContractDate,
		&d.CertificateNo, &d.CertificateDate,
		&d.HolderLastName, &d.HolderName,
		&d.HolderRegisterNo, &d.HolderPhone,
		&d.HolderEmail, &d.HolderType,
		&d.HolderCivilID,
		&d.ValuationZone, &d.BasePricePerHa, &d.AuctionCoeff, &d.AuctionPrice,
	)
	if err == nil {
		pf.RemainingAreaM2 = &pfRemainingAreaM2
	}
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if d.HolderName != "" || d.RightType != 0 {
		pf.Detail = d
	}
	return pf, nil
}

func (r *landAcqRepo) UpdateParcelAcquisitionGeom(ctx context.Context, parcelID uuid.UUID, acqWKT string) error {
	_, err := r.db.Exec(ctx, `
		WITH overlap AS (
			SELECT ST_GeometryN(
				ST_CollectionExtract(
					ST_Intersection(geometry, ST_GeomFromText($2, 4326)::geometry(Polygon, 4326)),
					3
				),
				1
			)::geometry(Polygon, 4326) AS geom
			FROM parcel
			WHERE id = $1
		)
		UPDATE parcel SET
			acquisition_geom    = (SELECT geom FROM overlap),
			acquisition_area_m2 = ST_Area((SELECT geom FROM overlap)::geography),
			updated_at          = NOW()
		WHERE id = $1
		  AND geometry IS NOT NULL
		  AND (SELECT geom FROM overlap) IS NOT NULL
		  AND NOT ST_IsEmpty((SELECT geom FROM overlap))
	`, parcelID, acqWKT)
	return err
}

// UpsertHolder finds or creates a right_holder record.
func (r *landAcqRepo) UpsertHolder(ctx context.Context, src *domain.SourceParcel) (uuid.UUID, error) {
	if src.HolderRegisterNo != "" {
		var id uuid.UUID
		err := r.db.QueryRow(ctx,
			"SELECT id FROM right_holder WHERE register_no = $1",
			src.HolderRegisterNo,
		).Scan(&id)
		if err == nil {
			return id, nil
		}
	}

	var regNo *string
	if src.HolderRegisterNo != "" {
		regNo = &src.HolderRegisterNo
	}

	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO right_holder (
			id, last_name, name, register_no, phone, email, holder_type, civil_id,
			created_by, updated_by
		) VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, $6, $7, 'system', 'system')
		RETURNING id
	`,
		src.HolderLastName, src.HolderName, regNo,
		src.HolderPhone, src.HolderEmail,
		src.HolderType, src.HolderCivilID,
	).Scan(&id)
	return id, err
}

// UpsertParcelRight creates a parcel_detail record if one doesn't exist for the parcel.
func (r *landAcqRepo) UpsertParcelRight(ctx context.Context, parcelID, acquisitionID, holderID uuid.UUID, src *domain.SourceParcel) error {
	var count int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM parcel_detail WHERE parcel_id = $1", parcelID,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO parcel_detail (
			id, parcel_id, acquisition_id, right_type, holder_id,
			app_no, app_timestamps,
			decision_no, decision_date,
			contract_no, contract_date,
			certificate_no, certificate_date,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(), $1, $2, $3, $4,
			$5, NOW(),
			$6, $7,
			$8, $9,
			$10, $11,
			'system', 'system'
		)
	`,
		parcelID, acquisitionID, src.RightType, holderID,
		src.AppNo,
		src.DecisionNo, src.DecisionDate,
		src.ContractNo, src.ContractDate,
		src.CertificateNo, src.CertificateDate,
	)
	return err
}

func (r *landAcqRepo) UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE parcel_detail SET
			valuation_zone    = $2,
			base_price_per_ha = $3,
			auction_coeff     = $4,
			auction_price     = $5,
			updated_by        = 'user'
		WHERE parcel_id = $1
	`, parcelID, zone, basePricePerHa, auctionCoeff, auctionPrice)
	return err
}

func (r *landAcqRepo) Update(ctx context.Context, acq *domain.LandAcquisition) error {
	_, err := r.db.Exec(ctx, `
			UPDATE land_acquisition SET
				plan_id              = $2,
				plan_code            = $3,
				plan_geom            = ST_GeomFromText(NULLIF($4, ''), 4326)::geometry(Polygon, 4326),
				plan_area_m2         = $5,
				geometry             = ST_GeomFromText(NULLIF($6, ''), 4326)::geometry(Polygon, 4326),
				area_m2              = $7,
				start_date           = $8,
				end_date             = $9,
				acquisition_name     = $10,
				implementing_org     = $11,
				reason               = $12,
				responsible_org      = $13,
				funding_source       = $14,
				construction_type_id = $15,
				updated_at           = NOW(),
				updated_by           = $16
		WHERE id = $1
	`,
		acq.ID, acq.PlanID, acq.PlanCode,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.StartDate, acq.EndDate,
		acq.AcquisitionName, acq.ImplementingOrg, acq.Reason, acq.ResponsibleOrg, acq.FundingSource,
		acq.ConstructionTypeID,
		acq.UpdatedBy,
	)
	return err
}

// DeleteParcelsAndAUs нэгж талбар болон АУ холбоосыг бүгдийг устгана.
// parcel устгахад parcel_detail CASCADE-р автоматаар устана.
func (r *landAcqRepo) DeleteParcelsAndAUs(ctx context.Context, acquisitionID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx,
		"DELETE FROM parcel WHERE acquisition_id = $1", acquisitionID,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx,
		"DELETE FROM land_acquisition_au WHERE land_acquisition_id = $1", acquisitionID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Delete газар чөлөөлөлт болон холбоотой бүх мэдээллийг устгана.
// CASCADE: land_acquisition устгахад parcel, land_acquisition_au, parcel_detail автоматаар устана.
func (r *landAcqRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM land_acquisition WHERE id = $1", id)
	return err
}

func (r *landAcqRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status int) error {
	_, err := r.db.Exec(ctx,
		"UPDATE land_acquisition SET status = $2, updated_at = NOW() WHERE id = $1",
		id, status,
	)
	return err
}

func (r *landAcqRepo) UpdateDecree(ctx context.Context, id uuid.UUID, number string, date *time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE land_acquisition SET decree_number = $2, decree_date = $3, updated_at = NOW() WHERE id = $1",
		id, number, date,
	)
	return err
}

func (r *landAcqRepo) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE parcel SET db_changed = $2, changed_parcel_id = $3 WHERE id = $1",
		parcelID, dbChanged, changedParcelID,
	)
	return err
}

func (r *landAcqRepo) ListProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, from_status, to_status,
		       COALESCE(note,''), changed_by, changed_at
		FROM acquisition_progress
		WHERE acquisition_id = $1
		ORDER BY changed_at ASC
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.AcquisitionProgress
	for rows.Next() {
		p := &domain.AcquisitionProgress{}
		if err := rows.Scan(
			&p.ID, &p.AcquisitionID,
			&p.FromStatus, &p.ToStatus,
			&p.Note, &p.ChangedBy, &p.ChangedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) CreateProgress(ctx context.Context, p *domain.AcquisitionProgress) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO acquisition_progress (id, acquisition_id, from_status, to_status, note, changed_by, changed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		p.ID, p.AcquisitionID, p.FromStatus, p.ToStatus, p.Note, p.ChangedBy, p.ChangedAt,
	)
	return err
}

func (r *landAcqRepo) SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error {
	_, err := r.db.Exec(ctx, `UPDATE parcel SET compensation_paid = $1 WHERE id = $2`, paid, parcelID)
	return err
}

func (r *landAcqRepo) ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error) {
	args := []any{}
	where := "1=1"
	i := 1

	if filter.ParcelID != "" {
		where += fmt.Sprintf(" AND p.parcel_id ILIKE $%d", i)
		args = append(args, "%"+filter.ParcelID+"%")
		i++
	}
	if filter.RightType != 0 {
		where += fmt.Sprintf(" AND p.right_type = $%d", i)
		args = append(args, filter.RightType)
		i++
	}
	if filter.Landuse != "" {
		where += fmt.Sprintf(" AND p.landuse ILIKE $%d", i)
		args = append(args, "%"+filter.Landuse+"%")
		i++
	}
	if filter.AU3Code != "" {
		where += fmt.Sprintf(" AND p.au3_code = $%d", i)
		args = append(args, filter.AU3Code)
		i++
	}
	if filter.AcquisitionName != "" {
		where += fmt.Sprintf(" AND la.acquisition_name ILIKE $%d", i)
		args = append(args, "%"+filter.AcquisitionName+"%")
		i++
	}
	if filter.AcquisitionID != "" {
		where += fmt.Sprintf(" AND la.id = $%d::uuid", i)
		args = append(args, filter.AcquisitionID)
		i++
	}
	if filter.PlanCode != "" {
		where += fmt.Sprintf(" AND la.plan_code ILIKE $%d", i)
		args = append(args, "%"+filter.PlanCode+"%")
		i++
	}
	if filter.Status != 0 {
		where += fmt.Sprintf(" AND la.status = $%d", i)
		args = append(args, filter.Status)
		i++
	}
	if len(filter.Years) > 0 {
		placeholders := make([]string, len(filter.Years))
		for j, yr := range filter.Years {
			placeholders[j] = fmt.Sprintf("$%d", i)
			args = append(args, yr)
			i++
		}
		where += fmt.Sprintf(" AND EXTRACT(YEAR FROM la.start_date) IN (%s)", strings.Join(placeholders, ","))
	}

	var total int64
	if err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM parcel p
		JOIN land_acquisition la ON la.id = p.acquisition_id
		WHERE %s
	`, where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT p.id, COALESCE(p.parcel_id,''), COALESCE(p.old_parcel_id,''),
		       p.acquisition_id, p.valid_from, p.valid_till,
		       COALESCE(p.au1_code,''), COALESCE(p.au2_code,''), COALESCE(p.au3_code,''),
		       COALESCE(ST_AsText(p.geometry),''),
		       '',
		       COALESCE(p.acquisition_area_m2,0),
		       COALESCE(p.right_type,0), COALESCE(p.landuse,''), COALESCE(p.area_m2,0),
		       COALESCE(p.compensation_paid,false),
		       p.created_at, COALESCE(p.created_by,''),
		       COALESCE(p.status,0),
		       COALESCE(la.acquisition_name,''), COALESCE(la.plan_code,''),
		       la.status, la.start_date, la.end_date
		FROM parcel p
		JOIN land_acquisition la ON la.id = p.acquisition_id
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, i, i+1), append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.GlobalParcel
	for rows.Next() {
		gp := &domain.GlobalParcel{}
		if err := rows.Scan(
			&gp.ID, &gp.ParcelID, &gp.OldParcelID,
			&gp.AcquisitionID, &gp.ValidFrom, &gp.ValidTill,
			&gp.AU1Code, &gp.AU2Code, &gp.AU3Code,
			&gp.GeometryWKT,
			&gp.AcquisitionGeomWKT,
			&gp.AcquisitionAreaM2,
			&gp.RightType, &gp.Landuse, &gp.AreaM2,
			&gp.CompensationPaid,
			&gp.CreatedAt, &gp.CreatedBy,
			&gp.Status,
			&gp.AcquisitionName, &gp.PlanCode,
			&gp.AcquisitionStatus, &gp.StartDate, &gp.EndDate,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, gp)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error) {
	args := []any{}
	where := "1=1"
	i := 1

	if filter.AcquisitionID != "" {
		where += fmt.Sprintf(" AND la.id = $%d", i)
		args = append(args, filter.AcquisitionID)
		i++
	}
	if filter.PlanCode != "" {
		where += fmt.Sprintf(" AND la.plan_code ILIKE $%d", i)
		args = append(args, "%"+filter.PlanCode+"%")
		i++
	}
	if filter.AcquisitionName != "" {
		where += fmt.Sprintf(" AND la.acquisition_name ILIKE $%d", i)
		args = append(args, "%"+filter.AcquisitionName+"%")
		i++
	}
	if filter.AU3Code != "" {
		where += fmt.Sprintf(" AND p.au3_code = $%d", i)
		args = append(args, filter.AU3Code)
		i++
	}
	if filter.RightType != 0 {
		where += fmt.Sprintf(" AND p.right_type = $%d", i)
		args = append(args, filter.RightType)
		i++
	}
	if filter.Landuse != "" {
		where += fmt.Sprintf(" AND p.landuse ILIKE $%d", i)
		args = append(args, "%"+filter.Landuse+"%")
		i++
	}
	if len(filter.Years) > 0 {
		placeholders := make([]string, len(filter.Years))
		for j, yr := range filter.Years {
			placeholders[j] = fmt.Sprintf("$%d", i)
			args = append(args, yr)
			i++
		}
		where += fmt.Sprintf(" AND EXTRACT(YEAR FROM la.start_date) IN (%s)", strings.Join(placeholders, ","))
	}
	if filter.CompensationType != "" {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM compensation c2
			WHERE c2.acquisition_id = p.acquisition_id
			  AND c2.parcel_id = p.parcel_id
			  AND c2.compensation_type = $%d
		)`, i)
		args = append(args, filter.CompensationType)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM parcel p
		JOIN land_acquisition la ON la.id = p.acquisition_id
		WHERE %s
	`, where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT
			COALESCE(p.parcel_id, ''),
			COALESCE(p.area_m2, 0),
			COALESCE(p.acquisition_area_m2, 0),
			COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2, 0), 0),
			COALESCE(p.right_type, 0),
			COALESCE(p.db_changed, false),
			COALESCE(p.changed_parcel_id, ''),
			la.id,
			COALESCE(la.acquisition_name, ''),
			COALESCE(la.plan_code, ''),
			COALESCE(ct.name, ''),
			COALESCE(la.decree_number, ''),
			la.decree_date,
			COALESCE(rh.last_name, ''),
			COALESCE(rh.name, ''),
			COALESCE(rh.register_no, ''),
			COALESCE(SUM(CASE WHEN c.target_type = 'parcel' THEN c.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN c.target_type = 'asset' AND a.asset_type = 'real_state' THEN c.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN c.target_type = 'asset' AND a.asset_type = 'property' THEN c.amount ELSE 0 END), 0)
		FROM parcel p
		JOIN land_acquisition la ON la.id = p.acquisition_id
		LEFT JOIN construction_type ct ON ct.id = la.construction_type_id
		LEFT JOIN parcel_detail pd ON pd.parcel_id = p.id
		LEFT JOIN right_holder rh ON rh.id = pd.holder_id
		LEFT JOIN compensation c ON c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
		LEFT JOIN assets a ON a.id = c.asset_id
		WHERE %s
		GROUP BY
			p.id, p.parcel_id, p.area_m2, p.acquisition_area_m2,
			p.remaining_area_m2, p.right_type, p.db_changed, p.changed_parcel_id,
			la.id, la.acquisition_name, la.plan_code, ct.name, la.decree_number, la.decree_date,
			rh.last_name, rh.name, rh.register_no
		ORDER BY la.acquisition_name, p.parcel_id
		LIMIT $%d OFFSET $%d
	`, where, i, i+1), append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.ReportParcelRow
	for rows.Next() {
		row := &domain.ReportParcelRow{}
		if err := rows.Scan(
			&row.ParcelID,
			&row.AreaM2,
			&row.AcquisitionAreaM2,
			&row.RemainingAreaM2,
			&row.RightType,
			&row.DbChanged,
			&row.ChangedParcelID,
			&row.AcquisitionID,
			&row.AcquisitionName,
			&row.PlanCode,
			&row.ConstructionTypeName,
			&row.DecreeNumber,
			&row.DecreeDate,
			&row.HolderLastName,
			&row.HolderName,
			&row.HolderRegisterNo,
			&row.LandComp,
			&row.RealStateComp,
			&row.PropertyComp,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, parcel_id, COALESCE(amount,0), COALESCE(currency,'MNT'),
		       paid_at, COALESCE(note,''), created_at, COALESCE(created_by,'')
		FROM parcel_payment
		WHERE parcel_id = $1
		ORDER BY created_at DESC
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.ParcelPayment
	for rows.Next() {
		p := &domain.ParcelPayment{}
		if err := rows.Scan(
			&p.ID, &p.ParcelID, &p.Amount, &p.Currency,
			&p.PaidAt, &p.Note, &p.CreatedAt, &p.CreatedBy,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO parcel_payment (id, parcel_id, amount, currency, paid_at, note, created_at, created_by)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, NOW(), $6)
	`, p.ParcelID, p.Amount, p.Currency, p.PaidAt, p.Note, p.CreatedBy)
	return err
}

func (r *landAcqRepo) ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.name, d.file_type, d.file_url, d.size_bytes, COALESCE(d.note,''), d.uploaded_by, d.uploaded_at
		FROM document d
		JOIN acquisition_document ad ON ad.document_id = d.id
		WHERE ad.acquisition_id = $1
		ORDER BY d.uploaded_at DESC
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Document
	for rows.Next() {
		d := &domain.Document{}
		if err := rows.Scan(&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.Note, &d.UploadedBy, &d.UploadedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) AddAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, doc *domain.Document) error {
	_, err := r.db.Exec(ctx, `
		WITH ins AS (
			INSERT INTO document (id, name, file_type, file_url, size_bytes, note, uploaded_by, uploaded_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		)
		INSERT INTO acquisition_document (acquisition_id, document_id)
		SELECT $9, id FROM ins
	`, doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note, doc.UploadedBy, doc.UploadedAt, acquisitionID)
	return err
}

func (r *landAcqRepo) ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.name, d.file_type, d.file_url, d.size_bytes, COALESCE(d.note,''), d.uploaded_by, d.uploaded_at
		FROM document d
		JOIN parcel_document pd ON pd.document_id = d.id
		WHERE pd.parcel_id = $1
		ORDER BY d.uploaded_at DESC
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Document
	for rows.Next() {
		d := &domain.Document{}
		if err := rows.Scan(&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.Note, &d.UploadedBy, &d.UploadedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) AddParcelDocument(ctx context.Context, parcelID uuid.UUID, doc *domain.Document) error {
	_, err := r.db.Exec(ctx, `
		WITH ins AS (
			INSERT INTO document (id, name, file_type, file_url, size_bytes, note, uploaded_by, uploaded_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		)
		INSERT INTO parcel_document (parcel_id, document_id)
		SELECT $9, id FROM ins
	`, doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note, doc.UploadedBy, doc.UploadedAt, parcelID)
	return err
}

func (r *landAcqRepo) DeleteDocument(ctx context.Context, documentID uuid.UUID) error {
	// deleting from document cascades to mapping tables via FK
	_, err := r.db.Exec(ctx, "DELETE FROM document WHERE id = $1", documentID)
	return err
}

func (r *landAcqRepo) GetDocument(ctx context.Context, documentID uuid.UUID) (*domain.Document, error) {
	d := &domain.Document{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, file_type, file_url, size_bytes, COALESCE(note,''), uploaded_by, uploaded_at
		FROM document WHERE id = $1
	`, documentID).Scan(&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.Note, &d.UploadedBy, &d.UploadedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// ── Asset CRUD ────────────────────────────────────────────────────────────────

func (r *landAcqRepo) ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error) {
	where := "WHERE acquisition_id = $1"
	args := []any{acquisitionID}
	i := 2
	if filter.ParcelID != "" {
		where += fmt.Sprintf(" AND parcel_id = $%d", i)
		args = append(args, filter.ParcelID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, parcel_id, asset_number, asset_type, asset_name,
		       floor_count, area_m2, owner_name, address, notes, created_at, updated_at
		FROM assets `+where+` ORDER BY created_at DESC
	`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []*domain.Asset
	for rows.Next() {
		b := &domain.Asset{}
		if err := rows.Scan(&b.ID, &b.AcquisitionID, &b.ParcelID, &b.AssetNumber, &b.AssetType, &b.AssetName,
			&b.FloorCount, &b.AreaM2, &b.OwnerName, &b.Address, &b.Notes, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, b)
	}
	return list, int64(len(list)), rows.Err()
}

func (r *landAcqRepo) CreateAsset(ctx context.Context, b *domain.Asset) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO assets (acquisition_id, parcel_id, asset_number, asset_type, asset_name,
		                      floor_count, area_m2, owner_name, address, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at
	`, b.AcquisitionID, b.ParcelID, b.AssetNumber, b.AssetType, b.AssetName,
		b.FloorCount, b.AreaM2, b.OwnerName, b.Address, b.Notes,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

func (r *landAcqRepo) UpdateAsset(ctx context.Context, b *domain.Asset) error {
	return r.db.QueryRow(ctx, `
		UPDATE assets SET parcel_id=$2, asset_number=$3, asset_type=$4, asset_name=$5,
		    floor_count=$6, area_m2=$7, owner_name=$8, address=$9, notes=$10, updated_at=NOW()
		WHERE id=$1 RETURNING updated_at
	`, b.ID, b.ParcelID, b.AssetNumber, b.AssetType, b.AssetName,
		b.FloorCount, b.AreaM2, b.OwnerName, b.Address, b.Notes,
	).Scan(&b.UpdatedAt)
}

func (r *landAcqRepo) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM assets WHERE id=$1", id)
	return err
}

// ── Compensation CRUD ─────────────────────────────────────────────────────────

func (r *landAcqRepo) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.Compensation, error) {
	args := []any{acquisitionID}
	where := "c.acquisition_id = $1"
	if parcelID != "" {
		where += " AND c.parcel_id = $2"
		args = append(args, parcelID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.acquisition_id, COALESCE(c.target_type,'asset'), COALESCE(c.parcel_id,''),
		       c.asset_id, COALESCE(c.compensation_type,'cash'),
		       COALESCE(c.coverage_percent,100), COALESCE(c.amount,0),
		       c.compensation_date, COALESCE(c.note,''),
		       c.created_at, COALESCE(c.created_by,''), c.updated_at,
		       cg.id, cg.acquisition_id, cg.compensation_id, COALESCE(cg.amount,0), cg.grant_date, COALESCE(cg.note,''),
		       COALESCE(cg.land_area_m2,0), COALESCE(cg.land_price,0), COALESCE(cg.land_location,''),
		       COALESCE(cg.land_purpose,''), COALESCE(cg.land_use_type,''),
		       COALESCE(cg.parcel_number,''), COALESCE(cg.decree_number,''),
		       cg.created_at, COALESCE(cg.created_by,''), cg.updated_at
		FROM compensation c
		LEFT JOIN compensation_grant cg ON cg.compensation_id = c.id
		WHERE `+where+`
		ORDER BY c.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Compensation
	for rows.Next() {
		c := &domain.Compensation{}
		var gID, gAcqID, gCompID *uuid.UUID
		var gAmount *float64
		var gGrantDate *time.Time
		var gNote *string
		var gLandAreaM2, gLandPrice *float64
		var gLandLocation, gLandPurpose, gLandUseType, gParcelNumber, gDecreeNumber *string
		var gCreatedAt *time.Time
		var gCreatedBy *string
		var gUpdatedAt *time.Time
		if err := rows.Scan(
			&c.ID, &c.AcquisitionID, &c.TargetType, &c.ParcelID,
			&c.AssetID, &c.CompensationType,
			&c.CoveragePercent, &c.Amount,
			&c.CompensationDate, &c.Note,
			&c.CreatedAt, &c.CreatedBy, &c.UpdatedAt,
			&gID, &gAcqID, &gCompID, &gAmount, &gGrantDate, &gNote,
			&gLandAreaM2, &gLandPrice, &gLandLocation,
			&gLandPurpose, &gLandUseType,
			&gParcelNumber, &gDecreeNumber,
			&gCreatedAt, &gCreatedBy, &gUpdatedAt,
		); err != nil {
			return nil, err
		}
		if gID != nil {
			g := &domain.CompensationGrant{ID: *gID}
			if gAcqID != nil {
				g.AcquisitionID = *gAcqID
			}
			if gCompID != nil {
				g.CompensationID = *gCompID
			}
			if gAmount != nil {
				g.Amount = *gAmount
			}
			g.GrantDate = gGrantDate
			if gNote != nil {
				g.Note = *gNote
			}
			if gLandAreaM2 != nil {
				g.LandAreaM2 = *gLandAreaM2
			}
			if gLandPrice != nil {
				g.LandPrice = *gLandPrice
			}
			if gLandLocation != nil {
				g.LandLocation = *gLandLocation
			}
			if gLandPurpose != nil {
				g.LandPurpose = *gLandPurpose
			}
			if gLandUseType != nil {
				g.LandUseType = *gLandUseType
			}
			if gParcelNumber != nil {
				g.ParcelNumber = *gParcelNumber
			}
			if gDecreeNumber != nil {
				g.DecreeNumber = *gDecreeNumber
			}
			if gCreatedAt != nil {
				g.CreatedAt = *gCreatedAt
			}
			if gCreatedBy != nil {
				g.CreatedBy = *gCreatedBy
			}
			if gUpdatedAt != nil {
				g.UpdatedAt = *gUpdatedAt
			}
			c.Grant = g
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) CreateCompensation(ctx context.Context, c *domain.Compensation) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO compensation (
			acquisition_id, target_type, parcel_id, asset_id,
			compensation_type, coverage_percent, amount, compensation_date,
			note, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at
	`, c.AcquisitionID, c.TargetType, c.ParcelID, c.AssetID,
		c.CompensationType, c.CoveragePercent, c.Amount, c.CompensationDate,
		c.Note, c.CreatedBy,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *landAcqRepo) UpdateCompensation(ctx context.Context, c *domain.Compensation) error {
	return r.db.QueryRow(ctx, `
		UPDATE compensation SET
			target_type=$2, parcel_id=$3, asset_id=$4,
			compensation_type=$5, coverage_percent=$6, amount=$7,
			compensation_date=$8, note=$9,
			updated_at=NOW()
		WHERE id=$1 RETURNING updated_at
	`, c.ID, c.TargetType, c.ParcelID, c.AssetID,
		c.CompensationType, c.CoveragePercent, c.Amount,
		c.CompensationDate, c.Note,
	).Scan(&c.UpdatedAt)
}

func (r *landAcqRepo) DeleteCompensation(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM compensation WHERE id=$1", id)
	return err
}

// ── CompensationGrant CRUD ────────────────────────────────────────────────────

func (r *landAcqRepo) CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO compensation_grant (
			acquisition_id, compensation_id, amount, grant_date, note,
			land_area_m2, land_price, land_location, land_purpose, land_use_type,
			parcel_number, decree_number, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at
	`, g.AcquisitionID, g.CompensationID, g.Amount, g.GrantDate, g.Note,
		g.LandAreaM2, g.LandPrice, g.LandLocation, g.LandPurpose, g.LandUseType,
		g.ParcelNumber, g.DecreeNumber, g.CreatedBy,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (r *landAcqRepo) UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error {
	return r.db.QueryRow(ctx, `
		UPDATE compensation_grant SET
			amount=$2, grant_date=$3, note=$4,
			land_area_m2=$5, land_price=$6, land_location=$7,
			land_purpose=$8, land_use_type=$9, parcel_number=$10,
			decree_number=$11, updated_at=NOW()
		WHERE compensation_id=$1 RETURNING updated_at
	`, g.CompensationID, g.Amount, g.GrantDate, g.Note,
		g.LandAreaM2, g.LandPrice, g.LandLocation,
		g.LandPurpose, g.LandUseType, g.ParcelNumber,
		g.DecreeNumber,
	).Scan(&g.UpdatedAt)
}

func (r *landAcqRepo) GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error) {
	g := &domain.CompensationGrant{}
	err := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, compensation_id, COALESCE(amount,0), grant_date, COALESCE(note,''),
		       COALESCE(land_area_m2,0), COALESCE(land_price,0), COALESCE(land_location,''),
		       COALESCE(land_purpose,''), COALESCE(land_use_type,''),
		       COALESCE(parcel_number,''), COALESCE(decree_number,''),
		       created_at, COALESCE(created_by,''), updated_at
		FROM compensation_grant WHERE id=$1
	`, id).Scan(
		&g.ID, &g.AcquisitionID, &g.CompensationID, &g.Amount, &g.GrantDate, &g.Note,
		&g.LandAreaM2, &g.LandPrice, &g.LandLocation,
		&g.LandPurpose, &g.LandUseType,
		&g.ParcelNumber, &g.DecreeNumber,
		&g.CreatedAt, &g.CreatedBy, &g.UpdatedAt,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return g, nil
}

func (r *landAcqRepo) DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM compensation_grant WHERE id=$1", id)
	return err
}

func (r *landAcqRepo) DeleteCompensationGrantByCompensationID(ctx context.Context, compensationID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM compensation_grant WHERE compensation_id=$1", compensationID)
	return err
}

func (r *landAcqRepo) Suggest(ctx context.Context, q string) ([]*domain.LandAcquisition, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(acquisition_name,''), COALESCE(plan_code,'')
		FROM land_acquisition
		WHERE acquisition_name ILIKE $1 OR plan_code ILIKE $1
		ORDER BY acquisition_name
		LIMIT 20
	`, "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.LandAcquisition
	for rows.Next() {
		a := &domain.LandAcquisition{}
		if err := rows.Scan(&a.ID, &a.AcquisitionName, &a.PlanCode); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	// ── шүүлтүүрийн WHERE clause ──────────────────────────────────────────────
	args := []any{}
	where := "1=1"
	i := 1

	if filter.AcquisitionID != "" {
		where += fmt.Sprintf(" AND id = $%d::uuid", i)
		args = append(args, filter.AcquisitionID)
		i++
	}
	if filter.PlanCode != "" {
		where += fmt.Sprintf(" AND plan_code ILIKE $%d", i)
		args = append(args, "%"+filter.PlanCode+"%")
		i++
	}
	if len(filter.Years) > 0 {
		placeholders := make([]string, len(filter.Years))
		for j, yr := range filter.Years {
			placeholders[j] = fmt.Sprintf("$%d", i)
			args = append(args, yr)
			i++
		}
		where += fmt.Sprintf(" AND EXTRACT(YEAR FROM start_date) IN (%s)", strings.Join(placeholders, ","))
	}

	acqSubquery := fmt.Sprintf("SELECT id FROM land_acquisition WHERE %s", where)

	// ── горутинуудаар зэрэгцүүлэн асуулга ────────────────────────────────────
	type acqRes struct {
		list []*domain.LandAcquisition
		err  error
	}
	type statusRes struct {
		list []*domain.ParcelStatus
		err  error
	}
	type statsRes struct {
		totalParcels int64
		freedParcels int64
		freedAreaM2  float64
		err          error
	}
	type breakdownRes struct {
		rows []domain.ParcelStatusStat
		err  error
	}
	type timelineRes struct {
		points []domain.TimelinePoint
		err    error
	}
	type compRes struct {
		total float64
		err   error
	}

	acqCh       := make(chan acqRes, 1)
	statusCh    := make(chan statusRes, 1)
	statsCh     := make(chan statsRes, 1)
	breakdownCh := make(chan breakdownRes, 1)
	timelineCh  := make(chan timelineRes, 1)
	compCh      := make(chan compRes, 1)

	// 1. Шүүлтэд тохирох чөлөөлөлтүүд
	go func() {
		list, _, err := r.List(ctx, domain.LandAcquisitionFilter{
			PlanCode:      filter.PlanCode,
			Years:         filter.Years,
			Page:          1,
			PageSize:      1000,
		})
		// acquisition_id шүүлтүүр байвал нэг л acquisition авна
		if err == nil && filter.AcquisitionID != "" {
			filtered := list[:0]
			for _, a := range list {
				if a.ID.String() == filter.AcquisitionID {
					filtered = append(filtered, a)
					break
				}
			}
			list = filtered
		}
		acqCh <- acqRes{list, err}
	}()

	// 2. Нэгж талбарын төлөвийн lookup
	go func() {
		list, err := r.ListParcelStatuses(ctx)
		statusCh <- statusRes{list, err}
	}()

	// 3. Нэгж талбарын нийт тоо, чөлөөлөгдсөн тоо, чөлөөлөгдсөн талбай
	go func() {
		var total, freed int64
		var freedArea float64
		err := r.db.QueryRow(ctx, fmt.Sprintf(`
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE compensation_paid = true),
				COALESCE(SUM(acquisition_area_m2) FILTER (WHERE compensation_paid = true), 0)
			FROM parcel
			WHERE acquisition_id IN (%s)
		`, acqSubquery), args...).Scan(&total, &freed, &freedArea)
		statsCh <- statsRes{total, freed, freedArea, err}
	}()

	// 4. Төлөвийн задаргаа (нэгж талбар бүрийн тоо, талбай)
	go func() {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT p.status, COALESCE(ps.name, CAST(p.status AS text)),
			       COUNT(*), COALESCE(SUM(p.acquisition_area_m2), 0)
			FROM parcel p
			LEFT JOIN parcel_status ps ON ps.id = p.status
			WHERE p.acquisition_id IN (%s)
			GROUP BY p.status, ps.name
			ORDER BY p.status
		`, acqSubquery), args...)
		if err != nil {
			breakdownCh <- breakdownRes{nil, err}
			return
		}
		defer rows.Close()
		var result []domain.ParcelStatusStat
		for rows.Next() {
			var s domain.ParcelStatusStat
			if err := rows.Scan(&s.StatusID, &s.Name, &s.Count, &s.AreaM2); err != nil {
				breakdownCh <- breakdownRes{nil, err}
				return
			}
			result = append(result, s)
		}
		breakdownCh <- breakdownRes{result, rows.Err()}
	}()

	// 5. Цаг хугацааны график (чөлөөлөлтийн эхлэх огноогоор нэгж талбарын тоо)
	go func() {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT TO_CHAR(la.start_date, 'YYYY.MM.DD'), COUNT(p.id)
			FROM parcel p
			JOIN land_acquisition la ON la.id = p.acquisition_id
			WHERE p.acquisition_id IN (%s)
			  AND la.start_date IS NOT NULL
			GROUP BY la.start_date
			ORDER BY la.start_date
		`, acqSubquery), args...)
		if err != nil {
			timelineCh <- timelineRes{nil, err}
			return
		}
		defer rows.Close()
		var result []domain.TimelinePoint
		for rows.Next() {
			var pt domain.TimelinePoint
			if err := rows.Scan(&pt.Date, &pt.Count); err != nil {
				timelineCh <- timelineRes{nil, err}
				return
			}
			result = append(result, pt)
		}
		timelineCh <- timelineRes{result, rows.Err()}
	}()

	// 6. Нийт нөхөн олговорын дүн
	go func() {
		var total float64
		err := r.db.QueryRow(ctx, fmt.Sprintf(`
			SELECT COALESCE(SUM(c.amount), 0)
			FROM compensation c
			WHERE c.acquisition_id IN (%s)
		`, acqSubquery), args...).Scan(&total)
		compCh <- compRes{total, err}
	}()

	// ── үр дүн цуглуулах ─────────────────────────────────────────────────────
	acqR    := <-acqCh
	statusR := <-statusCh
	statsR  := <-statsCh
	bkR     := <-breakdownCh
	tlR     := <-timelineCh
	compR   := <-compCh

	for _, e := range []error{acqR.err, statusR.err, statsR.err, bkR.err, tlR.err, compR.err} {
		if e != nil {
			return nil, e
		}
	}

	// plan_area_m2 болон total_orders-ийг чөлөөлөлтийн жагсаалтаас тооцно
	var planAreaM2 float64
	var totalOrders int64
	for _, a := range acqR.list {
		planAreaM2 += a.AreaM2
		if a.DecreeNumber != "" {
			totalOrders++
		}
	}

	return &domain.DashboardSummary{
		Acquisitions:      acqR.list,
		ParcelStatuses:    statusR.list,
		TotalParcels:      statsR.totalParcels,
		FreedParcels:      statsR.freedParcels,
		FreedAreaM2:       statsR.freedAreaM2,
		PlanAreaM2:        planAreaM2,
		TotalOrders:       totalOrders,
		TotalCompensation: compR.total,
		StatusBreakdown:   bkR.rows,
		Timeline:          tlR.points,
	}, nil
}

var _ repository.LandAcquisitionRepository = (*landAcqRepo)(nil)
