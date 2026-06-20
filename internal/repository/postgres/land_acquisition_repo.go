package postgres

import (
	"context"
	"fmt"
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

func (r *landAcqRepo) Create(ctx context.Context, acq *domain.LandAcquisition) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO land_acquisition (
			id, plan_id, plan_code,
			plan_geom, plan_area_m2,
			geometry, area_m2,
			status, start_date, end_date,
			created_at, created_by, updated_at, updated_by
			) VALUES (
				$1, $2, $3,
				ST_GeomFromText(NULLIF($4, ''), 4326)::geometry(Polygon, 4326), $5,
				ST_GeomFromText(NULLIF($6, ''), 4326)::geometry(Polygon, 4326), $7,
				$8, $9, $10,
				$11, $12, $13, $14
			)
	`,
		acq.ID, acq.PlanID, acq.PlanCode,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.Status, acq.StartDate, acq.EndDate,
		acq.CreatedAt, acq.CreatedBy, acq.UpdatedAt, acq.UpdatedBy,
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
			SELECT id, plan_id, plan_code,
			       COALESCE(ST_AsText(plan_geom),''), COALESCE(plan_area_m2,0),
			       COALESCE(ST_AsText(geometry),''), COALESCE(area_m2,0),
			       status, start_date, end_date,
			       created_at, COALESCE(created_by,''),
			       updated_at, COALESCE(updated_by,'')
		FROM land_acquisition
		WHERE %s
		ORDER BY created_at DESC
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
			&acq.ID, &acq.PlanID, &acq.PlanCode,
			&acq.PlanGeomWKT, &acq.PlanAreaM2,
			&acq.GeometryWKT, &acq.AreaM2,
			&acq.Status, &acq.StartDate, &acq.EndDate,
			&acq.CreatedAt, &acq.CreatedBy,
			&acq.UpdatedAt, &acq.UpdatedBy,
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
			SELECT id, plan_id, plan_code,
			       COALESCE(ST_AsText(plan_geom),''), COALESCE(plan_area_m2,0),
			       COALESCE(ST_AsText(geometry),''), COALESCE(area_m2,0),
			       status, start_date, end_date,
			       created_at, COALESCE(created_by,''),
			       updated_at, COALESCE(updated_by,'')
		FROM land_acquisition
		WHERE id = $1
	`, id).Scan(
		&acq.ID, &acq.PlanID, &acq.PlanCode,
		&acq.PlanGeomWKT, &acq.PlanAreaM2,
		&acq.GeometryWKT, &acq.AreaM2,
		&acq.Status, &acq.StartDate, &acq.EndDate,
		&acq.CreatedAt, &acq.CreatedBy,
		&acq.UpdatedAt, &acq.UpdatedBy,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	aus, err := r.getAUs(ctx, id)
	if err != nil {
		return nil, err
	}
	acq.AUs = aus

	parcels, _, err := r.GetParcels(ctx, id, domain.ParcelFilter{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, err
	}
	acq.Parcels = parcels

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
	if filter.RightType != "" {
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
				       COALESCE(right_type,''), COALESCE(landuse,''), COALESCE(area_m2,0),
				       created_at, COALESCE(created_by,'')
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
		if err := rows.Scan(
			&p.ID, &p.ParcelID, &p.OldParcelID,
			&p.AcquisitionID, &p.ValidFrom, &p.ValidTill,
			&p.AU1Code, &p.AU2Code, &p.AU3Code,
			&p.GeometryWKT,
			&p.AcquisitionAreaM2,
			&p.RightType, &p.Landuse, &p.AreaM2,
			&p.CreatedAt, &p.CreatedBy,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, p)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error) {
	p := &domain.Parcel{}
	err := r.db.QueryRow(ctx, `
			SELECT id, COALESCE(parcel_id,''), COALESCE(old_parcel_id,''),
			       acquisition_id, valid_from, valid_till,
			       COALESCE(au1_code,''), COALESCE(au2_code,''), COALESCE(au3_code,''),
			       COALESCE(ST_AsText(geometry),''),
			       COALESCE(ST_AsText(acquisition_geom),''),
			       COALESCE(acquisition_area_m2,0),
			       COALESCE(right_type,''), COALESCE(landuse,''), COALESCE(area_m2,0),
			       created_at, COALESCE(created_by,'')
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
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (r *landAcqRepo) GetParcelDetailByID(ctx context.Context, id uuid.UUID) (*domain.ParcelFull, error) {
	pf := &domain.ParcelFull{}
	d := &domain.ParcelDetailInfo{}

	err := r.db.QueryRow(ctx, `
		SELECT
				p.id, COALESCE(p.parcel_id,''), COALESCE(p.old_parcel_id,''),
				p.acquisition_id, p.valid_from, p.valid_till,
				COALESCE(p.au1_code,''), COALESCE(p.au2_code,''), COALESCE(p.au3_code,''),
				COALESCE(ST_AsText(p.geometry),''),
				COALESCE(ST_AsText(p.acquisition_geom),''),
				COALESCE(p.acquisition_area_m2,0),
				COALESCE(p.right_type,''), COALESCE(p.landuse,''), COALESCE(p.area_m2,0),
				p.created_at, COALESCE(p.created_by,''),
			COALESCE(pd.right_type,''),
			COALESCE(pd.app_no,''), pd.app_timestamps,
			COALESCE(pd.decision_no,''), pd.decision_date,
			COALESCE(pd.contract_no,''), pd.contract_date,
			COALESCE(pd.certificate_no,''), pd.certificate_date,
			COALESCE(rh.last_name,''), COALESCE(rh.name,''),
			COALESCE(rh.register_no,''), COALESCE(rh.phone,''),
			COALESCE(rh.email,''), COALESCE(rh.holder_type,''),
			COALESCE(rh.civil_id,'')
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
		&d.RightType,
		&d.AppNo, &d.AppTimestamps,
		&d.DecisionNo, &d.DecisionDate,
		&d.ContractNo, &d.ContractDate,
		&d.CertificateNo, &d.CertificateDate,
		&d.HolderLastName, &d.HolderName,
		&d.HolderRegisterNo, &d.HolderPhone,
		&d.HolderEmail, &d.HolderType,
		&d.HolderCivilID,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if d.HolderName != "" || d.RightType != "" {
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

func (r *landAcqRepo) Update(ctx context.Context, acq *domain.LandAcquisition) error {
	_, err := r.db.Exec(ctx, `
			UPDATE land_acquisition SET
				plan_id       = $2,
				plan_code     = $3,
				plan_geom     = ST_GeomFromText(NULLIF($4, ''), 4326)::geometry(Polygon, 4326),
				plan_area_m2  = $5,
				geometry      = ST_GeomFromText(NULLIF($6, ''), 4326)::geometry(Polygon, 4326),
				area_m2       = $7,
			start_date    = $8,
			end_date      = $9,
			updated_at    = NOW(),
			updated_by    = $10
		WHERE id = $1
	`,
		acq.ID, acq.PlanID, acq.PlanCode,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.StartDate, acq.EndDate,
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
	defer tx.Rollback(ctx)

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

var _ repository.LandAcquisitionRepository = (*landAcqRepo)(nil)
