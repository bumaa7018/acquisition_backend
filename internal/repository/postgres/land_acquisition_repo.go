package postgres

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (r *landAcqRepo) ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error) {
	var rows pgx.Rows
	var err error
	if parentID == nil {
		rows, err = r.db.Query(ctx, `SELECT id, name, parent_id, sort_order FROM acquisition_category WHERE parent_id IS NULL ORDER BY sort_order`)
	} else {
		rows, err = r.db.Query(ctx, `SELECT id, name, parent_id, sort_order FROM acquisition_category WHERE parent_id = $1 ORDER BY sort_order`, *parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AcquisitionCategory
	for rows.Next() {
		c := &domain.AcquisitionCategory{}
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error) {
	c := &domain.AcquisitionCategory{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO acquisition_category (name, parent_id, sort_order) VALUES ($1, $2, $3) RETURNING id, name, parent_id, sort_order`,
		name, parentID, sortOrder,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder)
	return c, err
}

func (r *landAcqRepo) UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error) {
	c := &domain.AcquisitionCategory{}
	err := r.db.QueryRow(ctx,
		`UPDATE acquisition_category SET name=$1, sort_order=$2 WHERE id=$3 RETURNING id, name, parent_id, sort_order`,
		name, sortOrder, id,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder)
	return c, err
}

func (r *landAcqRepo) DeleteAcquisitionCategory(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM acquisition_category WHERE id=$1`, id)
	return err
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
			general_category_id, sub_category_id
			) VALUES (
				$1, $2, $3, $4,
				ST_GeomFromText(NULLIF($5, ''), 4326)::geometry(Polygon, 4326), $6,
				ST_GeomFromText(NULLIF($7, ''), 4326)::geometry(Polygon, 4326), $8,
				$9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18, $19, $20,
				$21, $22
			)
	`,
		acq.ID, acq.PlanID, acq.PlanCode, acq.PlanName,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.Status, acq.StartDate, acq.EndDate,
		acq.CreatedAt, acq.CreatedBy, acq.UpdatedAt, acq.UpdatedBy,
		acq.AcquisitionName, acq.ImplementingOrg, acq.Reason, acq.ResponsibleOrg, acq.FundingSource,
		acq.GeneralCategoryID, acq.SubCategoryID,
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
	if filter.AcquisitionName != "" {
		where += fmt.Sprintf(" AND acquisition_name ILIKE $%d", i)
		args = append(args, "%"+filter.AcquisitionName+"%")
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
			WHERE land_acquisition_id = la.id AND au3_code = $%d
		)`, i)
		args = append(args, filter.AU3Code)
		i++
	}
	if filter.GeneralCategoryID != 0 {
		where += fmt.Sprintf(" AND general_category_id = $%d", i)
		args = append(args, filter.GeneralCategoryID)
		i++
	}
	if filter.SubCategoryID != 0 {
		where += fmt.Sprintf(" AND sub_category_id = $%d", i)
		args = append(args, filter.SubCategoryID)
		i++
	}
	if len(filter.Years) > 0 {
		placeholders := make([]string, len(filter.Years))
		for j, yr := range filter.Years {
			placeholders[j] = fmt.Sprintf("$%d", i)
			args = append(args, yr)
			i++
		}
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM parcel p_yr
			WHERE p_yr.acquisition_id = la.id
			AND EXTRACT(YEAR FROM (
				SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p_yr.parcel_id
			)) IN (%s)
		)`, strings.Join(placeholders, ","))
	}
	if filter.AssignedUserID != "" {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM land_acquisition_assignee
			WHERE acquisition_id = la.id AND user_id = $%d::uuid
		)`, i)
		args = append(args, filter.AssignedUserID)
		i++
	}
	if filter.ProfessionalOrgUserID != "" {
		where += fmt.Sprintf(` AND (
			la.professional_org_id = $%d::uuid
			OR EXISTS (
				SELECT 1 FROM parcel p_org
				WHERE p_org.acquisition_id = la.id AND p_org.independent_org_id = $%d::uuid
			)
		)`, i, i)
		args = append(args, filter.ProfessionalOrgUserID)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM land_acquisition la WHERE "+where, args...,
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

	// Геометрийг жагсаалтад буцаахгүй — ST_AsText нь олон мянган мөрөнд маш удаан
	// бөгөөд том payload үүсгэдэг. Геометр зөвхөн GetByID хүсэлтэд буцна.
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT la.id, la.plan_id, la.plan_code, COALESCE(la.plan_name,''),
			       '', COALESCE(la.plan_area_m2,0),
			       '', COALESCE(la.area_m2,0),
			       la.status, la.start_date, la.end_date,
			       la.created_at, COALESCE(la.created_by,''),
			       la.updated_at, COALESCE(la.updated_by,''),
			       COALESCE(la.acquisition_name,''), COALESCE(la.implementing_org,''),
			       COALESCE(la.reason,''), COALESCE(la.responsible_org,''), COALESCE(la.funding_source,''),
			       la.general_category_id, COALESCE(gc.name,''),
			       la.sub_category_id, COALESCE(sc.name,''),
			       COALESCE(la.decree_number,''),
			       (SELECT COUNT(*) FROM parcel WHERE acquisition_id = la.id),
			       la.professional_org_id,
			       ''
		FROM land_acquisition la
		LEFT JOIN acquisition_category gc ON gc.id = la.general_category_id
		LEFT JOIN acquisition_category sc ON sc.id = la.sub_category_id
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
			&acq.GeneralCategoryID, &acq.GeneralCategoryName,
			&acq.SubCategoryID, &acq.SubCategoryName,
			&acq.DecreeNumber,
			&acq.ParcelCount,
			&acq.ProfessionalOrgID, &acq.ProfessionalOrgName,
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
			       la.general_category_id, COALESCE(gc.name,''),
			       la.sub_category_id, COALESCE(sc.name,''),
			       COALESCE(la.decree_number,''), la.decree_date,
			       (SELECT COUNT(*) FROM parcel WHERE acquisition_id = la.id),
			       la.professional_org_id,
			       ''
		FROM land_acquisition la
		LEFT JOIN acquisition_category gc ON gc.id = la.general_category_id
		LEFT JOIN acquisition_category sc ON sc.id = la.sub_category_id
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
		&acq.GeneralCategoryID, &acq.GeneralCategoryName,
		&acq.SubCategoryID, &acq.SubCategoryName,
		&acq.DecreeNumber, &acq.DecreeDate,
		&acq.ParcelCount,
		&acq.ProfessionalOrgID, &acq.ProfessionalOrgName,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	aus, err := r.getAUs(ctx, id)
	if err != nil {
		return nil, err
	}
	acq.AUs = aus

	assignees, err := r.GetAssignees(ctx, id)
	if err != nil {
		return nil, err
	}
	acq.AssignedUsers = assignees

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

// ValidateOverlap checks that the acquisition geometry has a real area overlap with the plan.
func (r *landAcqRepo) ValidateOverlap(ctx context.Context, geomWKT, planWKT string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
			WITH g AS (
				SELECT
					ST_MakeValid(ST_GeomFromText($1, 4326))::geometry AS acquisition_geom,
					ST_MakeValid(ST_GeomFromText($2, 4326))::geometry AS plan_geom
			)
			SELECT ST_Area(ST_CollectionExtract(ST_Intersection(acquisition_geom, plan_geom), 3)) > 0
			FROM g
		`, geomWKT, planWKT).Scan(&ok)
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
		// Insert initial status history
		var insertedID uuid.UUID
		if err := r.db.QueryRow(ctx,
			`SELECT id FROM parcel WHERE parcel_id = $1 AND acquisition_id = $2`,
			sp.ParcelID, acquisitionID,
		).Scan(&insertedID); err == nil {
			_ = r.InsertParcelStatusHistory(ctx, insertedID, acquisitionID, 0, "system")
		}
	}
	return nil
}

func (r *landAcqRepo) CreateBoundaryHistory(ctx context.Context, history *domain.LandAcquisitionBoundaryHistory) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO land_acquisition_boundary_history (
			id, land_acquisition_id, old_geometry, new_geometry, changed_by, changed_at
		) VALUES (
			$1, $2,
			ST_GeomFromText(NULLIF($3, ''), 4326)::geometry(Polygon, 4326),
			ST_GeomFromText(NULLIF($4, ''), 4326)::geometry(Polygon, 4326),
			$5, $6
		)
	`,
		history.ID,
		history.LandAcquisitionID,
		history.OldGeometryWKT,
		history.NewGeometryWKT,
		history.ChangedBy,
		history.ChangedAt,
	)
	return err
}

func (r *landAcqRepo) ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, land_acquisition_id,
		       COALESCE(ST_AsText(old_geometry), ''),
		       COALESCE(ST_AsText(new_geometry), ''),
		       COALESCE(changed_by, ''), changed_at
		FROM land_acquisition_boundary_history
		WHERE land_acquisition_id = $1
		ORDER BY changed_at DESC
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.LandAcquisitionBoundaryHistory
	for rows.Next() {
		h := &domain.LandAcquisitionBoundaryHistory{}
		if err := rows.Scan(
			&h.ID,
			&h.LandAcquisitionID,
			&h.OldGeometryWKT,
			&h.NewGeometryWKT,
			&h.ChangedBy,
			&h.ChangedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error) {
	args := []any{acquisitionID}
	where := "p.acquisition_id = $1"
	i := 2

	if filter.ParcelID != "" {
		where += fmt.Sprintf(" AND p.parcel_id ILIKE $%d", i)
		args = append(args, "%"+filter.ParcelID+"%")
		i++
	}
	if filter.AU1Code != "" {
		where += fmt.Sprintf(" AND p.au1_code = $%d", i)
		args = append(args, filter.AU1Code)
		i++
	}
	if filter.AU2Code != "" {
		where += fmt.Sprintf(" AND p.au2_code = $%d", i)
		args = append(args, filter.AU2Code)
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
		where += fmt.Sprintf(" AND p.landuse = $%d", i)
		args = append(args, filter.Landuse)
		i++
	}
	if filter.StatusID != 0 {
		where += fmt.Sprintf(" AND p.status = $%d", i)
		args = append(args, filter.StatusID)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM parcel p WHERE "+where, args...,
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
				SELECT p.id, COALESCE(p.parcel_id,''), COALESCE(p.old_parcel_id,''),
				       p.acquisition_id, p.valid_from, p.valid_till,
				       COALESCE(p.au1_code,''), COALESCE(p.au2_code,''), COALESCE(p.au3_code,''),
				       COALESCE(ST_AsText(p.geometry),''),
				       COALESCE(p.acquisition_area_m2,0),
				       COALESCE(p.right_type,0), COALESCE(p.landuse,''), COALESCE(p.area_m2,0),
				       COALESCE(p.compensation_paid,false),
				       p.created_at, COALESCE(p.created_by,''),
				       COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2,0)),
				       COALESCE(p.db_changed,false), COALESCE(p.changed_parcel_id,''),
				       COALESCE(p.status,0), COALESCE(ps.name,''),
				       p.independent_org_id,
				       '',
				       COALESCE((SELECT SUM(c.amount) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'cash' AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')), 0),
				       COALESCE((SELECT SUM(c.amount) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'land_grant' AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')), 0),
				       COALESCE((SELECT COUNT(*) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'land_grant'), 0),
				       COALESCE((SELECT jsonb_object_agg(vs.valuation_type, vs.status) FROM parcel_valuation_submission vs WHERE vs.acquisition_id = p.acquisition_id AND vs.parcel_id = p.parcel_id), '{}'::jsonb)
				FROM parcel p
				LEFT JOIN parcel_status ps ON ps.id = p.status
				WHERE %s
			ORDER BY p.parcel_id
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
			&p.StatusID, &p.StatusName,
			&p.IndependentOrgID, &p.IndependentOrgName,
			&p.CashAmount, &p.LandGrantAmount, &p.LandGrantCount,
			&p.ValuationStatuses,
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

func (r *landAcqRepo) GetParcelByCode(ctx context.Context, parcelCode string) (*domain.Parcel, error) {
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
		WHERE parcel_id = $1
	`, parcelCode).Scan(
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
				COALESCE(p.status,0), COALESCE(ps.name,''),
				p.independent_org_id, p.selected_valuation_type,
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
		LEFT JOIN parcel_status ps ON ps.id = p.status
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
		&pf.StatusID, &pf.StatusName,
		&pf.IndependentOrgID, &pf.SelectedValuationType,
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
				general_category_id  = $15,
				sub_category_id      = $16,
				updated_at           = NOW(),
				updated_by           = $17
		WHERE id = $1
	`,
		acq.ID, acq.PlanID, acq.PlanCode,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.StartDate, acq.EndDate,
		acq.AcquisitionName, acq.ImplementingOrg, acq.Reason, acq.ResponsibleOrg, acq.FundingSource,
		acq.GeneralCategoryID, acq.SubCategoryID,
		acq.UpdatedBy,
	)
	return err
}

// DeleteAUs removes only administrative unit links for an acquisition.
func (r *landAcqRepo) DeleteAUs(ctx context.Context, acquisitionID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM land_acquisition_au WHERE land_acquisition_id = $1", acquisitionID,
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

func (r *landAcqRepo) HasNonFinalParcels(ctx context.Context, acquisitionID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM parcel
			WHERE acquisition_id = $1
			  AND COALESCE(status, 0) NOT IN ($2, $3, $4)
		)`,
		acquisitionID,
		domain.ParcelStatusAffectedOut,
		domain.ParcelStatusRejected,
		domain.ParcelStatusReleased,
	).Scan(&exists)
	return exists, err
}

func (r *landAcqRepo) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE parcel SET
			db_changed = $2,
			changed_parcel_id = $3,
			acquisition_area_m2 = CASE
				WHEN $4::numeric IS NULL THEN acquisition_area_m2
				ELSE $4
			END,
			remaining_area_m2 = CASE
				WHEN $4::numeric IS NULL THEN remaining_area_m2
				ELSE area_m2 - $4
			END,
			acquisition_geom = CASE
				WHEN $5::text IS NULL THEN acquisition_geom
				ELSE ST_GeomFromText(NULLIF($5, ''), 4326)::geometry(Polygon, 4326)
			END,
			updated_at = NOW(),
			updated_by = 'user'
		WHERE id = $1
	`, parcelID, dbChanged, changedParcelID, acquisitionAreaM2, acquisitionGeomWKT)
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
		where += fmt.Sprintf(` AND EXTRACT(YEAR FROM (
			SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id
		)) IN (%s)`, strings.Join(placeholders, ","))
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
		       COALESCE(p.status,0), COALESCE(ps.name,''),
		       COALESCE(la.acquisition_name,''), COALESCE(la.plan_code,''),
		       la.status, la.start_date, la.end_date,
		       COALESCE((SELECT SUM(c.amount) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'cash' AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')), 0),
		       COALESCE((SELECT SUM(c.amount) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'land_grant' AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')), 0),
		       COALESCE((SELECT COUNT(*) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'land_grant'), 0)
		FROM parcel p
		JOIN land_acquisition la ON la.id = p.acquisition_id
		LEFT JOIN parcel_status ps ON ps.id = p.status
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
			&gp.StatusID, &gp.StatusName,
			&gp.AcquisitionName, &gp.PlanCode,
			&gp.AcquisitionStatus, &gp.StartDate, &gp.EndDate,
			&gp.CashAmount, &gp.LandGrantAmount, &gp.LandGrantCount,
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
		where += fmt.Sprintf(` AND EXTRACT(YEAR FROM (
			SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id
		)) IN (%s)`, strings.Join(placeholders, ","))
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
	if filter.GeneralCategoryID != 0 {
		where += fmt.Sprintf(" AND la.general_category_id = $%d", i)
		args = append(args, filter.GeneralCategoryID)
		i++
	}
	if filter.SubCategoryID != 0 {
		where += fmt.Sprintf(" AND la.sub_category_id = $%d", i)
		args = append(args, filter.SubCategoryID)
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
			COALESCE(gc.name, ''),
			COALESCE(sc.name, ''),
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
		LEFT JOIN acquisition_category gc ON gc.id = la.general_category_id
		LEFT JOIN acquisition_category sc ON sc.id = la.sub_category_id
		LEFT JOIN parcel_detail pd ON pd.parcel_id = p.id
		LEFT JOIN right_holder rh ON rh.id = pd.holder_id
		LEFT JOIN compensation c ON c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
			AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
		LEFT JOIN assets a ON a.id = c.asset_id
		WHERE %s
		GROUP BY
			p.id, p.parcel_id, p.area_m2, p.acquisition_area_m2,
			p.remaining_area_m2, p.right_type, p.db_changed, p.changed_parcel_id,
			la.id, la.acquisition_name, la.plan_code, gc.name, sc.name, la.decree_number, la.decree_date,
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
			&row.GeneralCategoryName,
			&row.SubCategoryName,
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
		SELECT d.id, d.name, d.file_type, d.file_url, d.size_bytes, COALESCE(d.note,''), d.uploaded_by, d.uploaded_at, lad.file_type
		FROM document d
		JOIN land_acquisition_document lad ON lad.document_id = d.id
		WHERE lad.land_acquisition_id = $1
		ORDER BY d.uploaded_at DESC
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Document
	for rows.Next() {
		d := &domain.Document{}
		if err := rows.Scan(&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.Note, &d.UploadedBy, &d.UploadedAt, &d.DocumentTypeID); err != nil {
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
		INSERT INTO land_acquisition_document (land_acquisition_id, document_id, file_type)
		SELECT $9, id, $10 FROM ins
	`, doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note, doc.UploadedBy, doc.UploadedAt, acquisitionID, doc.DocumentTypeID)
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

func (r *landAcqRepo) DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `
		DELETE FROM document d
		WHERE d.id = $1
		  AND EXISTS (
		      SELECT 1 FROM land_acquisition_document lad
		      WHERE lad.document_id = d.id AND lad.land_acquisition_id = $2
		  )
	`, documentID, acquisitionID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `
		DELETE FROM document d
		WHERE d.id = $1
		  AND EXISTS (
		      SELECT 1 FROM parcel_document pd
		      WHERE pd.document_id = d.id AND pd.parcel_id = $2
		  )
	`, documentID, parcelID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
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
		i++
	}
	if filter.ValuationType != "" {
		where += fmt.Sprintf(" AND valuation_type = $%d", i)
		args = append(args, filter.ValuationType)
		i++
	}
	_ = i // сүүлийн нөхцөлийн дараа ашиглагдахгүй байгаад анхааруулга гаргахаас сэргийлнэ
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, parcel_id, asset_number, asset_type, valuation_type, asset_name,
		       floor_count, area_m2, owner_name, address, notes,
		       unit, capacity, description, unit_price, photo_pdf_url, photo_pdf_name, created_at, updated_at
		FROM assets `+where+` ORDER BY created_at DESC
	`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []*domain.Asset
	for rows.Next() {
		b := &domain.Asset{}
		if err := rows.Scan(
			&b.ID, &b.AcquisitionID, &b.ParcelID, &b.AssetNumber, &b.AssetType, &b.ValuationType, &b.AssetName,
			&b.FloorCount, &b.AreaM2, &b.OwnerName, &b.Address, &b.Notes,
			&b.Unit, &b.Capacity, &b.Description, &b.UnitPrice, &b.PhotoPdfURL, &b.PhotoPdfName, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, b)
	}
	return list, int64(len(list)), rows.Err()
}

func (r *landAcqRepo) CreateAsset(ctx context.Context, b *domain.Asset) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO assets (acquisition_id, parcel_id, asset_number, asset_type, asset_name,
		                    floor_count, area_m2, owner_name, address, notes, unit, capacity, description, unit_price, valuation_type)
		SELECT $1::uuid,$2::varchar,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,COALESCE(NULLIF($15,''),'asset')
		WHERE EXISTS (
			SELECT 1 FROM parcel WHERE acquisition_id=$1 AND parcel_id=$2
		)
		RETURNING id, created_at, updated_at
	`,
		b.AcquisitionID, b.ParcelID, b.AssetNumber, b.AssetType, b.AssetName,
		b.FloorCount, b.AreaM2, b.OwnerName, b.Address, b.Notes,
		b.Unit, b.Capacity, b.Description, b.UnitPrice, b.ValuationType,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	return err
}

func (r *landAcqRepo) UpdateAsset(ctx context.Context, b *domain.Asset) error {
	err := r.db.QueryRow(ctx, `
		UPDATE assets SET parcel_id=$2, asset_number=$3, asset_type=$4, asset_name=$5,
		    floor_count=$6, area_m2=$7, owner_name=$8, address=$9, notes=$10,
		    unit=$11, capacity=$12, description=$13, unit_price=$14, updated_at=NOW()
		WHERE id=$1 AND acquisition_id=$15
		  AND EXISTS (
		      SELECT 1 FROM parcel WHERE acquisition_id=$15 AND parcel_id=$2
		  )
		RETURNING updated_at
	`, b.ID, b.ParcelID, b.AssetNumber, b.AssetType, b.AssetName,
		b.FloorCount, b.AreaM2, b.OwnerName, b.Address, b.Notes,
		b.Unit, b.Capacity, b.Description, b.UnitPrice, b.AcquisitionID,
	).Scan(&b.UpdatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	return err
}

// DeleteAsset — хөрөнгийг устгаад, хавсаргасан зургийн PDF-ийн URL-ыг буцаана
// (service нь тухайн файлыг хадгалалтаас устгахад ашиглана).
func (r *landAcqRepo) DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID) (string, error) {
	var photoURL string
	err := r.db.QueryRow(ctx,
		"DELETE FROM assets WHERE id=$1 AND acquisition_id=$2 RETURNING COALESCE(photo_pdf_url,'')",
		id, acquisitionID).Scan(&photoURL)
	if err == pgx.ErrNoRows {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return photoURL, nil
}

func (r *landAcqRepo) UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error {
	if err := r.ensureAssetInAcquisition(ctx, acquisitionID, assetID); err != nil {
		return err
	}
	for _, s := range specs {
		_, err := r.db.Exec(ctx, `
			INSERT INTO asset_spec (asset_id, spec_type_id, value)
			VALUES ($1, $2, $3)
			ON CONFLICT (asset_id, spec_type_id) DO UPDATE SET value=$3
		`, assetID, s.SpecTypeID, s.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *landAcqRepo) ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.asset_id, s.spec_type_id, t.code, t.name, s.value
		FROM asset_spec s
		JOIN asset_spec_type t ON t.id = s.spec_type_id
		JOIN assets a ON a.id = s.asset_id
		WHERE s.asset_id = $1 AND a.acquisition_id = $2
		ORDER BY t.sort_order
	`, assetID, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AssetSpec
	for rows.Next() {
		s := &domain.AssetSpec{}
		if err := rows.Scan(&s.ID, &s.AssetID, &s.SpecTypeID, &s.SpecCode, &s.SpecName, &s.Value); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error {
	if err := r.ensureAssetInAcquisition(ctx, acquisitionID, assetID); err != nil {
		return err
	}
	for _, c := range calcs {
		_, err := r.db.Exec(ctx, `
			INSERT INTO asset_calculation (asset_id, calc_type_id, unit, value)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (asset_id, calc_type_id) DO UPDATE SET unit=$3, value=$4
		`, assetID, c.CalcTypeID, c.Unit, c.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *landAcqRepo) ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.asset_id, c.calc_type_id, t.code, t.name, COALESCE(t.grp,''), c.unit, c.value
		FROM asset_calculation c
		JOIN asset_calc_type t ON t.id = c.calc_type_id
		JOIN assets a ON a.id = c.asset_id
		WHERE c.asset_id = $1 AND a.acquisition_id = $2
		ORDER BY c.sort_index, t.sort_order
	`, assetID, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AssetCalculation
	for rows.Next() {
		c := &domain.AssetCalculation{}
		if err := rows.Scan(&c.ID, &c.AssetID, &c.CalcTypeID, &c.CalcCode, &c.CalcName, &c.CalcGroup, &c.Unit, &c.Value); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) UpsertLandValuation(ctx context.Context, v *domain.LandValuation) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	vType := v.ValuationType
	if vType == "" {
		vType = domain.ValuationTypeAsset
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO land_valuation (acquisition_id, parcel_id, valuation_type, land_area_m2, base_price_per_m2,
			ownership_cert_no, appraiser_org_name, appraiser_reg_no, appraiser_state_reg_no,
			appraiser_director, appraiser_license, appraiser_address, appraiser_contact,
			source_file_name, source_file_hash)
		SELECT $1::uuid,$2::varchar,$15,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		WHERE EXISTS (
			SELECT 1 FROM parcel WHERE acquisition_id=$1 AND parcel_id=$2
		)
		ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE
		SET land_area_m2=$3, base_price_per_m2=$4,
			ownership_cert_no=$5, appraiser_org_name=$6, appraiser_reg_no=$7, appraiser_state_reg_no=$8,
			appraiser_director=$9, appraiser_license=$10, appraiser_address=$11, appraiser_contact=$12,
			source_file_name=$13, source_file_hash=$14, updated_at=NOW()
		RETURNING id, created_at, updated_at
	`, v.AcquisitionID, v.ParcelID, v.LandAreaM2, v.BasePricePerM2,
		v.OwnershipCertNo, v.AppraiserOrgName, v.AppraiserRegNo, v.AppraiserStateRegNo,
		v.AppraiserDirector, v.AppraiserLicense, v.AppraiserAddress, v.AppraiserContact,
		v.SourceFileName, v.SourceFileHash, vType,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	// Газрын үнэлгээг нөхөх олговрын нэг хэсэг болгож compensation-д тусгана (тайлан/нэгтгэлд орно).
	if err = syncLandCompensation(ctx, tx, v.AcquisitionID, v.ParcelID, vType, v.LandAreaM2*v.BasePricePerM2, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *landAcqRepo) GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error) {
	if valuationType == "" {
		valuationType = domain.ValuationTypeAsset
	}
	v := &domain.LandValuation{}
	err := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, parcel_id, valuation_type, land_area_m2, base_price_per_m2,
			ownership_cert_no, appraiser_org_name, appraiser_reg_no, appraiser_state_reg_no,
			appraiser_director, appraiser_license, appraiser_address, appraiser_contact,
			source_file_name, source_file_hash, created_at, updated_at
		FROM land_valuation WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3
	`, acquisitionID, parcelID, valuationType).Scan(
		&v.ID, &v.AcquisitionID, &v.ParcelID, &v.ValuationType, &v.LandAreaM2, &v.BasePricePerM2,
		&v.OwnershipCertNo, &v.AppraiserOrgName, &v.AppraiserRegNo, &v.AppraiserStateRegNo,
		&v.AppraiserDirector, &v.AppraiserLicense, &v.AppraiserAddress, &v.AppraiserContact,
		&v.SourceFileName, &v.SourceFileHash, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// DeleteLandValuation — нэгж талбарын тухайн урсгалын газрын үнэлгээг устгана.
func (r *landAcqRepo) DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error {
	if valuationType == "" {
		valuationType = domain.ValuationTypeAsset
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx,
		"DELETE FROM land_valuation WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3",
		acquisitionID, parcelID, valuationType)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	// Уг урсгалын авто-үүсгэсэн газрын олговрыг мөн цэвэрлэнэ.
	if err = syncLandCompensation(ctx, tx, acquisitionID, parcelID, valuationType, 0, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// upsertCalcTypeByName — тооцооллын төрлийг нэрээр нь олж, байхгүй бол шинээр үүсгэж id буцаана.
// (Excel-ийн барилгын өртгийн хандлагын шинэ мөр/итгэлцүүрийг бүрэн хадгалахад ашиглана.)
func upsertCalcTypeByName(ctx context.Context, tx pgx.Tx, name, unit, grp string) (int, error) {
	var id int
	err := tx.QueryRow(ctx, "SELECT id FROM asset_calc_type WHERE name=$1 LIMIT 1", name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return 0, err
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	code := fmt.Sprintf("imp_%08x", h.Sum32())
	if unit == "" {
		unit = "коэф"
	}
	// sort_order 7 — итгэлцүүрүүдийн дунд (нөхөн орлуулах өртөг=10-аас өмнө) байрлуулна
	err = tx.QueryRow(ctx, `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order, grp) VALUES ($1,$2,$3,7,$4)
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, grp=EXCLUDED.grp
		RETURNING id
	`, code, name, unit, grp).Scan(&id)
	return id, err
}

// landCompNote — газрын үнэлгээнээс авто-үүсгэсэн parcel-түвшний нөхөх олговрын таних тэмдэг.
const landCompNote = "Газрын үнэлгээ"

// syncLandCompensation — газрын үнэлгээг (талбай×суурь үнэ) нэгж талбарын түвшний cash нөхөх
// олговор болгож compensation-д ижилтгэнэ. Хуучин авто-мөрийг устгаж, шинийг бичнэ (давхардуулахгүй).
func syncLandCompensation(ctx context.Context, tx pgx.Tx, acqID uuid.UUID, parcelID, valuationType string, amount float64, createdBy string) error {
	if valuationType == "" {
		valuationType = domain.ValuationTypeAsset
	}
	// ЗӨВХӨН тухайн урсгалын авто-мөрийг устгана (бусад урсгалын газрын олговрыг хадгална).
	if _, err := tx.Exec(ctx,
		`DELETE FROM compensation WHERE acquisition_id=$1 AND parcel_id=$2 AND target_type='parcel' AND note=$3 AND valuation_type=$4`,
		acqID, parcelID, landCompNote, valuationType); err != nil {
		return err
	}
	if amount <= 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO compensation (acquisition_id, target_type, parcel_id, compensation_type,
			coverage_percent, amount, note, created_by, valuation_type)
		VALUES ($1,'parcel',$2,'cash',100,$3,$4,$5,$6)
	`, acqID, parcelID, amount, landCompNote, createdBy, valuationType)
	return err
}

// ImportValuation — бүхэл нэгж талбарын үнэлгээг НЭГ ТРАНЗАКЦААР оруулна.
// Replace=true бол хуучин олговор/хөрөнгийг эхлээд устгаж, дараа нь газрын үнэлгээ,
// хөрөнгө, spec, calc, нөхөн олговрыг бүгдийг нэг дор бичнэ. Аль нэг алдвал бүгд rollback.
func (r *landAcqRepo) ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	res := &domain.ValuationImportResult{}

	// Нэгж талбар уг чөлөөлөлтөд хамаарах эсэхийг шалгах
	var exists bool
	if err = tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM parcel WHERE acquisition_id=$1 AND parcel_id=$2)",
		acquisitionID, imp.ParcelID,
	).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}

	vType := imp.ValuationType
	if vType == "" {
		vType = domain.ValuationTypeAsset
	}

	// 0. Хуучин мэдээлэл цэвэрлэх — ЗӨВХӨН тухайн урсгалынхыг (бусад урсгал хэвээр үлдэнэ)
	if imp.Replace {
		ct, e := tx.Exec(ctx, "DELETE FROM compensation WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3", acquisitionID, imp.ParcelID, vType)
		if e != nil {
			return nil, e
		}
		res.DeletedComps = int(ct.RowsAffected())
		ct, e = tx.Exec(ctx, "DELETE FROM assets WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3", acquisitionID, imp.ParcelID, vType)
		if e != nil {
			return nil, e
		}
		res.DeletedAssets = int(ct.RowsAffected())
	}

	// 1. Газрын үнэлгээ (утга байвал upsert)
	l := imp.Land
	if l.LandAreaM2 != 0 || l.BasePricePerM2 != 0 || l.OwnershipCertNo != "" || l.AppraiserOrgName != "" {
		if _, err = tx.Exec(ctx, `
			INSERT INTO land_valuation (acquisition_id, parcel_id, valuation_type, land_area_m2, base_price_per_m2,
				ownership_cert_no, appraiser_org_name, appraiser_reg_no, appraiser_state_reg_no,
				appraiser_director, appraiser_license, appraiser_address, appraiser_contact,
				source_file_name, source_file_hash)
			VALUES ($1,$2,$15,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE
			SET land_area_m2=$3, base_price_per_m2=$4,
				ownership_cert_no=$5, appraiser_org_name=$6, appraiser_reg_no=$7, appraiser_state_reg_no=$8,
				appraiser_director=$9, appraiser_license=$10, appraiser_address=$11, appraiser_contact=$12,
				source_file_name=$13, source_file_hash=$14, updated_at=NOW()
		`, acquisitionID, imp.ParcelID, l.LandAreaM2, l.BasePricePerM2,
			l.OwnershipCertNo, l.AppraiserOrgName, l.AppraiserRegNo, l.AppraiserStateRegNo,
			l.AppraiserDirector, l.AppraiserLicense, l.AppraiserAddress, l.AppraiserContact,
			l.SourceFileName, l.SourceFileHash, vType); err != nil {
			return nil, err
		}
	}
	// Газрын үнэлгээг нөхөх олговрын нэг хэсэг болгож (parcel-түвшний cash) compensation-д тусгана.
	landAmount := l.LandAreaM2 * l.BasePricePerM2
	if err = syncLandCompensation(ctx, tx, acquisitionID, imp.ParcelID, vType, landAmount, imp.CreatedBy); err != nil {
		return nil, err
	}
	if landAmount > 0 {
		res.CreatedComps++
	}

	// 2. Хөрөнгө бүр → asset + spec + calc + нөхөн олговор
	for _, a := range imp.Assets {
		var assetID uuid.UUID
		if err = tx.QueryRow(ctx, `
			INSERT INTO assets (acquisition_id, parcel_id, asset_number, asset_type, asset_name,
				floor_count, area_m2, owner_name, address, notes, unit, capacity, description, unit_price, valuation_type)
			VALUES ($1,$2,$3,$4,$5,0,$6,$7,'',$8,$9,$10,$11,$12,$13)
			RETURNING id
		`, acquisitionID, imp.ParcelID, a.AssetNumber, a.AssetType, a.AssetName,
			a.AreaM2, a.OwnerName, a.Notes, a.Unit, a.Capacity, a.Description, a.UnitPrice, vType).Scan(&assetID); err != nil {
			return nil, err
		}
		res.CreatedAssets++

		for _, s := range a.Specs {
			if _, err = tx.Exec(ctx, `
				INSERT INTO asset_spec (asset_id, spec_type_id, value) VALUES ($1,$2,$3)
				ON CONFLICT (asset_id, spec_type_id) DO UPDATE SET value=$3
			`, assetID, s.SpecTypeID, s.Value); err != nil {
				return nil, err
			}
		}
		for idx, c := range a.Calculations {
			ctid := c.CalcTypeID
			// calc_type_id байхгүй бол нэрээр нь олж, байхгүй бол шинээр үүсгэнэ
			// (загварт шинэ үзүүлэлт/итгэлцүүр нэмэгдсэн ч бүрэн хадгалагдана).
			if ctid == 0 && c.CalcName != "" {
				if ctid, err = upsertCalcTypeByName(ctx, tx, c.CalcName, c.Unit, c.CalcGroup); err != nil {
					return nil, err
				}
			}
			if ctid == 0 {
				continue
			}
			// sort_index = файл дахь мөрийн дараалал (харуулахдаа энэ дарааллыг сэргээнэ)
			if _, err = tx.Exec(ctx, `
				INSERT INTO asset_calculation (asset_id, calc_type_id, unit, value, sort_index) VALUES ($1,$2,$3,$4,$5)
				ON CONFLICT (asset_id, calc_type_id) DO UPDATE SET unit=$3, value=$4, sort_index=$5
			`, assetID, ctid, c.Unit, c.Value, idx); err != nil {
				return nil, err
			}
		}
		if a.CompensationAmount > 0 {
			if _, err = tx.Exec(ctx, `
				INSERT INTO compensation (acquisition_id, target_type, parcel_id, asset_id,
					compensation_type, coverage_percent, amount, note, created_by, valuation_type)
				VALUES ($1,'asset',$2,$3,'cash',100,$4,'Excel-ээс импортолсон үнэлгээ',$5,$6)
			`, acquisitionID, imp.ParcelID, assetID, a.CompensationAmount, imp.CreatedBy, vType); err != nil {
				return nil, err
			}
			res.CreatedComps++
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

// ── Compensation CRUD ─────────────────────────────────────────────────────────

func (r *landAcqRepo) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error) {
	args := []any{acquisitionID}
	where := "c.acquisition_id = $1"
	if parcelID != "" {
		args = append(args, parcelID)
		where += fmt.Sprintf(" AND c.parcel_id = $%d", len(args))
	}
	if valuationType != "" {
		args = append(args, valuationType)
		where += fmt.Sprintf(" AND c.valuation_type = $%d", len(args))
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.acquisition_id, COALESCE(c.target_type,'asset'), COALESCE(c.valuation_type,'asset'), COALESCE(c.parcel_id,''),
		       c.asset_id, COALESCE(c.compensation_type,'cash'),
		       COALESCE(c.coverage_percent,100), COALESCE(c.amount,0),
		       c.compensation_date, COALESCE(c.note,''),
		       COALESCE(c.status,'pending'), COALESCE(c.review_note,''),
		       COALESCE(c.reviewed_by,''), c.reviewed_at,
		       COALESCE(c.valuation_report_url,''), COALESCE(c.valuation_report_name,''),
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
			&c.ID, &c.AcquisitionID, &c.TargetType, &c.ValuationType, &c.ParcelID,
			&c.AssetID, &c.CompensationType,
			&c.CoveragePercent, &c.Amount,
			&c.CompensationDate, &c.Note,
			&c.Status, &c.ReviewNote, &c.ReviewedBy, &c.ReviewedAt,
			&c.ValuationReportURL, &c.ValuationReportName,
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
	err := r.db.QueryRow(ctx, `
		INSERT INTO compensation (
			acquisition_id, target_type, parcel_id, asset_id,
			compensation_type, coverage_percent, amount, compensation_date,
			note, created_by, valuation_type
		)
		SELECT $1::uuid,$2,$3::varchar,$4::uuid,$5,$6,$7,$8,$9,$10,COALESCE(NULLIF($11,''),'asset')
		WHERE ($3 = '' OR EXISTS (
			SELECT 1 FROM parcel WHERE acquisition_id=$1 AND parcel_id=$3
		))
		AND ($4::uuid IS NULL OR EXISTS (
			SELECT 1 FROM assets WHERE acquisition_id=$1 AND id=$4
		))
		RETURNING id, created_at, updated_at
	`, c.AcquisitionID, c.TargetType, c.ParcelID, c.AssetID,
		c.CompensationType, c.CoveragePercent, c.Amount, c.CompensationDate,
		c.Note, c.CreatedBy, c.ValuationType,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	return err
}

func (r *landAcqRepo) UpdateCompensation(ctx context.Context, c *domain.Compensation) error {
	// Татгалзсан үнэлгээг засварлахад автоматаар 'pending' болгоно
	err := r.db.QueryRow(ctx, `
		UPDATE compensation SET
			target_type=$2, parcel_id=$3, asset_id=$4,
			compensation_type=$5, coverage_percent=$6, amount=$7,
			compensation_date=$8, note=$9,
			status = CASE WHEN status='rejected' THEN 'pending' ELSE status END,
			review_note = CASE WHEN status='rejected' THEN '' ELSE review_note END,
			reviewed_by = CASE WHEN status='rejected' THEN '' ELSE reviewed_by END,
			reviewed_at = CASE WHEN status='rejected' THEN NULL ELSE reviewed_at END,
			updated_at=NOW()
		WHERE id=$1 AND acquisition_id=$10
		  AND ($3 = '' OR EXISTS (
		      SELECT 1 FROM parcel WHERE acquisition_id=$10 AND parcel_id=$3
		  ))
		  AND ($4::uuid IS NULL OR EXISTS (
		      SELECT 1 FROM assets WHERE acquisition_id=$10 AND id=$4
		  ))
		RETURNING updated_at
	`, c.ID, c.TargetType, c.ParcelID, c.AssetID,
		c.CompensationType, c.CoveragePercent, c.Amount,
		c.CompensationDate, c.Note, c.AcquisitionID,
	).Scan(&c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	return err
}

func (r *landAcqRepo) DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error {
	ct, err := r.db.Exec(ctx, "DELETE FROM compensation WHERE id=$1 AND acquisition_id=$2", id, acquisitionID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
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
	err := r.db.QueryRow(ctx, `
		UPDATE compensation_grant SET
			amount=$2, grant_date=$3, note=$4,
			land_area_m2=$5, land_price=$6, land_location=$7,
			land_purpose=$8, land_use_type=$9, parcel_number=$10,
			decree_number=$11, updated_at=NOW()
		WHERE compensation_id=$1 AND acquisition_id=$12
		RETURNING updated_at
	`, g.CompensationID, g.Amount, g.GrantDate, g.Note,
		g.LandAreaM2, g.LandPrice, g.LandLocation,
		g.LandPurpose, g.LandUseType, g.ParcelNumber,
		g.DecreeNumber, g.AcquisitionID,
	).Scan(&g.UpdatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	return err
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

func (r *landAcqRepo) DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, "DELETE FROM compensation_grant WHERE compensation_id=$1 AND acquisition_id=$2", compensationID, acquisitionID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
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
	var yearPlaceholders []string
	if len(filter.Years) > 0 {
		yearPlaceholders = make([]string, len(filter.Years))
		for j, yr := range filter.Years {
			yearPlaceholders[j] = fmt.Sprintf("$%d", i)
			args = append(args, yr)
			i++
		}
		where += fmt.Sprintf(` AND id IN (
			SELECT DISTINCT p.acquisition_id FROM parcel p
			WHERE EXTRACT(YEAR FROM (
				SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id
			)) IN (%s)
		)`, strings.Join(yearPlaceholders, ","))
	}
	if filter.GeneralCategoryID != 0 {
		where += fmt.Sprintf(" AND general_category_id = $%d", i)
		args = append(args, filter.GeneralCategoryID)
		i++
	}
	if filter.SubCategoryID != 0 {
		where += fmt.Sprintf(" AND sub_category_id = $%d", i)
		args = append(args, filter.SubCategoryID)
		i++
	}
	if filter.AssignedUserID != "" {
		where += fmt.Sprintf(` AND id IN (
			SELECT acquisition_id FROM land_acquisition_assignee WHERE user_id = $%d::uuid
		)`, i)
		args = append(args, filter.AssignedUserID)
		i++
	}
	_ = i // suppress unused warning after last condition

	acqSubquery := fmt.Sprintf("SELECT id FROM land_acquisition WHERE %s", where)

	// нэгж талбарын түвшний он шүүлтүүр — $N параметруудыг acqSubquery-тэй хуваалцана
	parcelYearCond := ""
	if len(yearPlaceholders) > 0 {
		parcelYearCond = fmt.Sprintf(` AND EXTRACT(YEAR FROM (
			SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id
		)) IN (%s)`, strings.Join(yearPlaceholders, ","))
	}

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

	acqCh := make(chan acqRes, 1)
	statusCh := make(chan statusRes, 1)
	statsCh := make(chan statsRes, 1)
	breakdownCh := make(chan breakdownRes, 1)
	timelineCh := make(chan timelineRes, 1)
	compCh := make(chan compRes, 1)

	// 1. Шүүлтэд тохирох чөлөөлөлтүүд
	go func() {
		list, _, err := r.List(ctx, domain.LandAcquisitionFilter{
			PlanCode:          filter.PlanCode,
			Years:             filter.Years,
			GeneralCategoryID: filter.GeneralCategoryID,
			SubCategoryID:     filter.SubCategoryID,
			AssignedUserID:    filter.AssignedUserID,
			Page:              1,
			PageSize:          1000,
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
			FROM parcel p
			WHERE p.acquisition_id IN (%s)%s
		`, acqSubquery, parcelYearCond), args...).Scan(&total, &freed, &freedArea)
		statsCh <- statsRes{total, freed, freedArea, err}
	}()

	// 4. Төлөвийн задаргаа (нэгж талбар бүрийн тоо, талбай)
	go func() {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT p.status, COALESCE(ps.name, CAST(p.status AS text)),
			       COUNT(*), COALESCE(SUM(p.acquisition_area_m2), 0)
			FROM parcel p
			LEFT JOIN parcel_status ps ON ps.id = p.status
			WHERE p.acquisition_id IN (%s)%s
			GROUP BY p.status, ps.name
			ORDER BY p.status
		`, acqSubquery, parcelYearCond), args...)
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
			  AND la.start_date IS NOT NULL%s
			GROUP BY la.start_date
			ORDER BY la.start_date
		`, acqSubquery, parcelYearCond), args...)
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
		var compSQL string
		if parcelYearCond != "" {
			// Он шүүлтүүртэй үед: parcel-аас (цөөн мөр) хөтөлж, олговрыг scalar subquery-ээр
			// нийлүүлнэ. compensation (олон мөр)-оос хөтлөвөл nested correlated subquery
			// сая дахин ажиллаж (~11 сая), асуулга минутаар удааширдаг тул parcel-аас хөтөлнө.
			compSQL = fmt.Sprintf(`
				SELECT COALESCE(SUM(x.amt), 0) FROM (
					SELECT (
						SELECT COALESCE(SUM(c.amount), 0) FROM compensation c
						WHERE c.parcel_id = p.parcel_id AND c.acquisition_id = p.acquisition_id
						  AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
					) AS amt
					FROM parcel p
					WHERE p.acquisition_id IN (%s)%s
				) x
			`, acqSubquery, parcelYearCond)
		} else {
			compSQL = fmt.Sprintf(`
				SELECT COALESCE(SUM(c.amount), 0)
				FROM compensation c
				JOIN parcel p ON p.parcel_id = c.parcel_id
				WHERE c.acquisition_id IN (%s)
				  AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
			`, acqSubquery)
		}
		err := r.db.QueryRow(ctx, compSQL, args...).Scan(&total)
		compCh <- compRes{total, err}
	}()

	// 7. Шүүлтүүрт тохирох захиргааны нэгжийн код жагсаалт (au1/au2/au3 давхаргад ашиглана)
	type auCodesRes struct {
		au1 []string
		au2 []string
		au3 []string
		err error
	}
	auCodesCh := make(chan auCodesRes, 1)
	go func() {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT DISTINCT
				COALESCE(au1_code, ''), COALESCE(au2_code, ''), au3_code
			FROM land_acquisition_au
			WHERE land_acquisition_id IN (%s)
		`, acqSubquery), args...)
		if err != nil {
			auCodesCh <- auCodesRes{nil, nil, nil, err}
			return
		}
		defer rows.Close()
		au1Set, au2Set, au3Set := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
		for rows.Next() {
			var c1, c2, c3 string
			if err := rows.Scan(&c1, &c2, &c3); err != nil {
				auCodesCh <- auCodesRes{nil, nil, nil, err}
				return
			}
			if c1 != "" {
				au1Set[c1] = struct{}{}
			}
			if c2 != "" {
				au2Set[c2] = struct{}{}
			}
			if c3 != "" {
				au3Set[c3] = struct{}{}
			}
		}
		if err := rows.Err(); err != nil {
			auCodesCh <- auCodesRes{nil, nil, nil, err}
			return
		}
		toSlice := func(m map[string]struct{}) []string {
			s := make([]string, 0, len(m))
			for k := range m {
				s = append(s, k)
			}
			return s
		}
		auCodesCh <- auCodesRes{toSlice(au1Set), toSlice(au2Set), toSlice(au3Set), nil}
	}()

	// ── үр дүн цуглуулах ─────────────────────────────────────────────────────
	acqR := <-acqCh
	statusR := <-statusCh
	statsR := <-statsCh
	bkR := <-breakdownCh
	tlR := <-timelineCh
	compR := <-compCh
	auCodesR := <-auCodesCh

	for _, e := range []error{acqR.err, statusR.err, statsR.err, bkR.err, tlR.err, compR.err, auCodesR.err} {
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
		FilteredAU1Codes:  auCodesR.au1,
		FilteredAU2Codes:  auCodesR.au2,
		FilteredAU3Codes:  auCodesR.au3,
	}, nil
}

// ── Assignee methods ──────────────────────────────────────────────────────────

func (r *landAcqRepo) SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM land_acquisition_assignee WHERE acquisition_id = $1`, acquisitionID); err != nil {
		return err
	}
	for _, a := range assignees {
		if _, err := tx.Exec(ctx, `
			INSERT INTO land_acquisition_assignee (acquisition_id, user_id, user_name, user_position, assigned_by, assigned_by_name)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, acquisitionID, a.UserID, a.UserName, a.UserPosition, a.AssignedBy, a.AssignedByName); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *landAcqRepo) GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, COALESCE(user_name,''), COALESCE(user_position,''), assigned_by, COALESCE(assigned_by_name,''), assigned_at
		FROM land_acquisition_assignee
		WHERE acquisition_id = $1
		ORDER BY assigned_at
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.AcquisitionAssignee
	for rows.Next() {
		var a domain.AcquisitionAssignee
		if err := rows.Scan(&a.UserID, &a.UserName, &a.UserPosition, &a.AssignedBy, &a.AssignedByName, &a.AssignedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	if result == nil {
		result = []domain.AcquisitionAssignee{}
	}
	return result, rows.Err()
}

func (r *landAcqRepo) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM land_acquisition_assignee
			WHERE acquisition_id = $1 AND user_id = $2
		)
	`, acquisitionID, userID).Scan(&exists)
	return exists, err
}

// IsProfMemberOfAcquisition — professional_org хэрэглэгч тухайн чөлөөлөлтийн
// үндсэн байгуулга (professional_org_id) эсвэл аль нэгэн нэгж талбарын
// хөндлөнгийн байгуулга (independent_org_id) мөн эсэхийг шалгана.
func (r *landAcqRepo) IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM land_acquisition WHERE id = $1 AND professional_org_id = $2
			UNION
			SELECT 1 FROM parcel WHERE acquisition_id = $1 AND independent_org_id = $2
		)
	`, acquisitionID, userID).Scan(&exists)
	return exists, err
}

// IsProfMemberOfParcel — professional_org хэрэглэгч parcel-ийн чөлөөлөлтийн
// үндсэн байгуулга эсвэл тухайн parcel-ийн хөндлөнгийн байгуулга мөн эсэхийг шалгана.
func (r *landAcqRepo) IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM parcel p
			JOIN land_acquisition la ON la.id = p.acquisition_id
			WHERE p.id = $1
			  AND (la.professional_org_id = $2 OR p.independent_org_id = $2)
		)
	`, parcelID, userID).Scan(&exists)
	return exists, err
}

func (r *landAcqRepo) ensureAssetInAcquisition(ctx context.Context, acquisitionID, assetID uuid.UUID) error {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM assets WHERE acquisition_id = $1 AND id = $2
		)
	`, acquisitionID, assetID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}
	return nil
}

// ── Parcel Workflow ────────────────────────────────────────────────────────────

func (r *landAcqRepo) ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pw.id, pw.from_status_id, pw.to_status_id,
		       COALESCE(fs.name,''), COALESCE(ts.name,''), pw.sort_order
		FROM parcel_workflow pw
		LEFT JOIN parcel_status fs ON fs.id = pw.from_status_id
		LEFT JOIN parcel_status ts ON ts.id = pw.to_status_id
		ORDER BY pw.sort_order, pw.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.ParcelWorkflow
	for rows.Next() {
		w := &domain.ParcelWorkflow{}
		if err := rows.Scan(&w.ID, &w.FromStatusID, &w.ToStatusID, &w.FromStatusName, &w.ToStatusName, &w.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error) {
	w := &domain.ParcelWorkflow{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO parcel_workflow (from_status_id, to_status_id, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, from_status_id, to_status_id, sort_order
	`, fromStatusID, toStatusID, sortOrder).Scan(&w.ID, &w.FromStatusID, &w.ToStatusID, &w.SortOrder)
	if err != nil {
		return nil, err
	}
	// fetch names
	if w.FromStatusID != nil {
		_ = r.db.QueryRow(ctx, `SELECT name FROM parcel_status WHERE id = $1`, *w.FromStatusID).Scan(&w.FromStatusName)
	}
	_ = r.db.QueryRow(ctx, `SELECT name FROM parcel_status WHERE id = $1`, w.ToStatusID).Scan(&w.ToStatusName)
	return w, nil
}

func (r *landAcqRepo) DeleteParcelWorkflow(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM parcel_workflow WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ps.id, ps.code, ps.name, ps.sort_order
		FROM parcel_workflow pw
		JOIN parcel_status ps ON ps.id = pw.to_status_id
		WHERE pw.from_status_id = $1
		ORDER BY pw.sort_order, ps.sort_order
	`, currentStatusID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.ParcelStatus
	for rows.Next() {
		ps := &domain.ParcelStatus{}
		if err := rows.Scan(&ps.ID, &ps.Code, &ps.Name, &ps.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, ps)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	// "Чөлөөлсөн" болгохын өмнө үнэлгээний тайлан байгаа эсэхийг шалгана
	var statusName string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(name,'') FROM parcel_status WHERE id=$1`, statusID).Scan(&statusName)
	if statusName == "Чөлөөлсөн" {
		var parcelCode string
		if err := r.db.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&parcelCode); err != nil {
			return domain.ErrNotFound
		}
		var hasReport bool
		// Зөвхөн сонгосон (үндсэн) урсгалын зөвшөөрөгдсөн + тайлантай олговрыг шаардана.
		_ = r.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM compensation c
				JOIN parcel p ON p.parcel_id = c.parcel_id AND p.acquisition_id = c.acquisition_id
				WHERE c.acquisition_id=$1 AND c.parcel_id=$2
				AND c.status='approved' AND c.valuation_report_url<>''
				AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
			)`, acquisitionID, parcelCode).Scan(&hasReport)
		if !hasReport {
			return domain.ErrValuationReportRequired
		}
	}
	ct, err := r.db.Exec(ctx, `UPDATE parcel SET status = $1 WHERE id = $2`, statusID, parcelID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return r.InsertParcelStatusHistory(ctx, parcelID, acquisitionID, statusID, createdBy)
}

func (r *landAcqRepo) InsertParcelStatusHistory(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, created_by)
		SELECT parcel_id, acquisition_id, $3, $4
		FROM parcel
		WHERE id = $1 AND acquisition_id = $2`,
		parcelID, acquisitionID, statusID, createdBy,
	)
	return err
}

func (r *landAcqRepo) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT h.id, h.parcel_id, h.acquisition_id, h.status_id, ps.name, h.status_date, h.created_by
		FROM parcel_status_history h
		JOIN parcel_status ps ON ps.id = h.status_id
		JOIN parcel p ON p.parcel_id = h.parcel_id
		WHERE p.id = $1
		ORDER BY h.status_date DESC
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.ParcelStatusHistory
	for rows.Next() {
		h := &domain.ParcelStatusHistory{}
		if err := rows.Scan(&h.ID, &h.ParcelID, &h.AcquisitionID, &h.StatusID, &h.StatusName, &h.StatusDate, &h.CreatedBy); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

// ── Нөхөх олговрын үнэлгээний илгээх/зөвшөөрөх төлөв ────────────────────────────

// parcelCodeByID — route дахь parcel UUID-аас parcel-ийн код (VARCHAR)-ыг олох.
func (r *landAcqRepo) parcelCodeByID(ctx context.Context, acquisitionID, parcelID uuid.UUID) (string, error) {
	var code string
	err := r.db.QueryRow(ctx,
		`SELECT parcel_id FROM parcel WHERE id=$1 AND acquisition_id=$2`, parcelID, acquisitionID).Scan(&code)
	if err != nil {
		return "", domain.ErrNotFound
	}
	return code, nil
}

func scanValuationSubmission(row pgx.Row, acqID uuid.UUID, code, vType string) (*domain.ValuationSubmission, error) {
	s := &domain.ValuationSubmission{}
	err := row.Scan(&s.ID, &s.AcquisitionID, &s.ParcelID, &s.ValuationType, &s.Status,
		&s.SubmittedBy, &s.SubmittedAt, &s.ReviewedBy, &s.ReviewedAt,
		&s.LastNote, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		// Мөр байхгүй бол анхдагч "Ноорог" төлөвөөр буцаана
		return &domain.ValuationSubmission{AcquisitionID: acqID, ParcelID: code, ValuationType: vType, Status: domain.ValuationStatusDraft}, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func normalizeValuationType(vType string) string {
	if vType == "" {
		return domain.ValuationTypeAsset
	}
	return vType
}

func (r *landAcqRepo) GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error) {
	code, err := r.parcelCodeByID(ctx, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	vType := normalizeValuationType(valuationType)
	row := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, parcel_id, valuation_type, status, submitted_by, submitted_at,
		       reviewed_by, reviewed_at, last_note, created_at, updated_at
		FROM parcel_valuation_submission
		WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`, acquisitionID, code, vType)
	return scanValuationSubmission(row, acquisitionID, code, vType)
}

// TransitionValuationSubmission — төлөв шилжүүлэх (submit|approve|return), түүх хадгална.
// approve үед тухайн урсгал нэгж талбарын "үндсэн" (selected_valuation_type) болно.
func (r *landAcqRepo) TransitionValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType, action, note, userID string) (*domain.ValuationSubmission, error) {
	code, err := r.parcelCodeByID(ctx, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	vType := normalizeValuationType(valuationType)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Одоогийн төлөв (мөр байхгүй бол Ноорог)
	from := domain.ValuationStatusDraft
	_ = tx.QueryRow(ctx,
		`SELECT status FROM parcel_valuation_submission WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`,
		acquisitionID, code, vType).Scan(&from)

	// Шилжилтийн дүрэм
	var to string
	switch action {
	case "submit":
		if from != domain.ValuationStatusDraft && from != domain.ValuationStatusReturned {
			return nil, domain.ErrInvalidTransition
		}
		to = domain.ValuationStatusSubmitted
	case "approve":
		if from != domain.ValuationStatusSubmitted {
			return nil, domain.ErrInvalidTransition
		}
		to = domain.ValuationStatusApproved
	case "return":
		if from != domain.ValuationStatusSubmitted {
			return nil, domain.ErrInvalidTransition
		}
		to = domain.ValuationStatusReturned
	default:
		return nil, domain.ErrInvalidInput
	}

	// Upsert төлөв. submit → submitted_by/at, approve/return → reviewed_by/at.
	if action == "submit" {
		_, err = tx.Exec(ctx, `
			INSERT INTO parcel_valuation_submission (acquisition_id, parcel_id, valuation_type, status, submitted_by, submitted_at, last_note)
			VALUES ($1,$2,$6,$3,$4,NOW(),$5)
			ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE
			SET status=$3, submitted_by=$4, submitted_at=NOW(), last_note=$5, updated_at=NOW()`,
			acquisitionID, code, to, userID, note, vType)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO parcel_valuation_submission (acquisition_id, parcel_id, valuation_type, status, reviewed_by, reviewed_at, last_note)
			VALUES ($1,$2,$6,$3,$4,NOW(),$5)
			ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE
			SET status=$3, reviewed_by=$4, reviewed_at=NOW(), last_note=$5, updated_at=NOW()`,
			acquisitionID, code, to, userID, note, vType)
	}
	if err != nil {
		return nil, err
	}

	// Зөвшөөрсөн урсгал нь нэгж талбарын үндсэн (selected) үнэлгээ болно.
	if action == "approve" {
		if _, err = tx.Exec(ctx,
			`UPDATE parcel SET selected_valuation_type=$1 WHERE acquisition_id=$2 AND parcel_id=$3`,
			vType, acquisitionID, code); err != nil {
			return nil, err
		}

		// Нэгж талбарт зөвхөн НЭГ баталгаажсан үнэлгээ байна — бусад урсгалын
		// илгээх төлөвийг ХҮЧИНГҮЙ болгож, урсгал тус бүрийн түүхэд бичнэ.
		voidNote := "Өөр үнэлгээний урсгал (" + vType + ") баталгаажсан тул автоматаар хүчингүй болов"
		voidRows, verr := tx.Query(ctx, `
			UPDATE parcel_valuation_submission s
			SET status=$4, last_note=$5, updated_at=NOW()
			FROM parcel_valuation_submission old
			WHERE old.id = s.id
			  AND s.acquisition_id=$1 AND s.parcel_id=$2
			  AND s.valuation_type<>$3 AND s.status<>$4
			RETURNING s.valuation_type, old.status`,
			acquisitionID, code, vType, domain.ValuationStatusVoided, voidNote)
		if verr != nil {
			return nil, verr
		}
		type voided struct{ vt, fromStatus string }
		var voidedList []voided
		for voidRows.Next() {
			var v voided
			if err := voidRows.Scan(&v.vt, &v.fromStatus); err != nil {
				voidRows.Close()
				return nil, err
			}
			voidedList = append(voidedList, v)
		}
		voidRows.Close()
		if err := voidRows.Err(); err != nil {
			return nil, err
		}
		for _, v := range voidedList {
			if _, err = tx.Exec(ctx, `
				INSERT INTO parcel_valuation_submission_history (acquisition_id, parcel_id, valuation_type, action, from_status, to_status, note, created_by)
				VALUES ($1,$2,$3,'void',$4,$5,$6,$7)`,
				acquisitionID, code, v.vt, v.fromStatus, domain.ValuationStatusVoided, voidNote, userID); err != nil {
				return nil, err
			}
		}
	}

	// Түүх
	if _, err = tx.Exec(ctx, `
		INSERT INTO parcel_valuation_submission_history (acquisition_id, parcel_id, valuation_type, action, from_status, to_status, note, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		acquisitionID, code, vType, action, from, to, note, userID); err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	row := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, parcel_id, valuation_type, status, submitted_by, submitted_at,
		       reviewed_by, reviewed_at, last_note, created_at, updated_at
		FROM parcel_valuation_submission
		WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`, acquisitionID, code, vType)
	return scanValuationSubmission(row, acquisitionID, code, vType)
}

func (r *landAcqRepo) ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error) {
	code, err := r.parcelCodeByID(ctx, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	vType := normalizeValuationType(valuationType)
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, parcel_id, valuation_type, action, from_status, to_status, note, created_by, created_at
		FROM parcel_valuation_submission_history
		WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3
		ORDER BY created_at DESC, id DESC`, acquisitionID, code, vType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.ValuationSubmissionHistory
	for rows.Next() {
		h := &domain.ValuationSubmissionHistory{}
		if err := rows.Scan(&h.ID, &h.AcquisitionID, &h.ParcelID, &h.ValuationType, &h.Action,
			&h.FromStatus, &h.ToStatus, &h.Note, &h.CreatedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) GetCompensation(ctx context.Context, id uuid.UUID) (*domain.Compensation, error) {
	c := &domain.Compensation{}
	err := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, COALESCE(target_type,'asset'), COALESCE(parcel_id,''),
		       asset_id, COALESCE(compensation_type,'cash'),
		       COALESCE(coverage_percent,100), COALESCE(amount,0),
		       compensation_date, COALESCE(note,''),
		       COALESCE(status,'pending'), COALESCE(review_note,''),
		       COALESCE(reviewed_by,''), reviewed_at,
		       COALESCE(valuation_report_url,''), COALESCE(valuation_report_name,''),
		       created_at, COALESCE(created_by,''), updated_at
		FROM compensation WHERE id=$1
	`, id).Scan(
		&c.ID, &c.AcquisitionID, &c.TargetType, &c.ParcelID,
		&c.AssetID, &c.CompensationType,
		&c.CoveragePercent, &c.Amount,
		&c.CompensationDate, &c.Note,
		&c.Status, &c.ReviewNote, &c.ReviewedBy, &c.ReviewedAt,
		&c.ValuationReportURL, &c.ValuationReportName,
		&c.CreatedAt, &c.CreatedBy, &c.UpdatedAt,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (r *landAcqRepo) UpdateCompensationReport(ctx context.Context, id uuid.UUID, reportURL, reportName string) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE compensation SET valuation_report_url=$1, valuation_report_name=$2, updated_at=NOW() WHERE id=$3`,
		reportURL, reportName, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateAssetPhoto — хөрөнгийн зургийн PDF-ийн URL-ийг хадгална (асеет уг чөлөөлөлтөд хамаарна).
func (r *landAcqRepo) UpdateAssetPhoto(ctx context.Context, acquisitionID, assetID uuid.UUID, photoURL, photoName string) error {
	ct, err := r.db.Exec(ctx,
		`UPDATE assets SET photo_pdf_url=$1, photo_pdf_name=$2, updated_at=NOW() WHERE id=$3 AND acquisition_id=$4`,
		photoURL, photoName, assetID, acquisitionID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) ApproveCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE compensation
		SET status='approved', reviewed_by=$2, reviewed_at=NOW(), review_note=$3, updated_at=NOW()
		WHERE id=$1
	`, id, reviewedBy, note)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) RejectCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error {
	c, err := r.GetCompensation(ctx, id)
	if err != nil {
		return err
	}
	// Татгалзахын өмнө одоогийн байдлыг түүхэнд хадгална
	_, err = r.db.Exec(ctx, `
		INSERT INTO compensation_history (
			compensation_id, acquisition_id, target_type, parcel_id, asset_id,
			compensation_type, coverage_percent, amount, compensation_date, note,
			status, review_note, reviewed_by, reviewed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'rejected',$11,$12,NOW())
	`, c.ID, c.AcquisitionID, c.TargetType, c.ParcelID, c.AssetID,
		c.CompensationType, c.CoveragePercent, c.Amount, c.CompensationDate, c.Note,
		note, reviewedBy,
	)
	if err != nil {
		return err
	}
	ct, err := r.db.Exec(ctx, `
		UPDATE compensation
		SET status='rejected', review_note=$2, reviewed_by=$3, reviewed_at=NOW(), updated_at=NOW()
		WHERE id=$1
	`, id, note, reviewedBy)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, compensation_id, acquisition_id, COALESCE(target_type,'asset'), COALESCE(parcel_id,''),
		       asset_id, COALESCE(compensation_type,'cash'),
		       COALESCE(coverage_percent,0), COALESCE(amount,0),
		       compensation_date, COALESCE(note,''),
		       COALESCE(status,'rejected'), COALESCE(review_note,''),
		       COALESCE(reviewed_by,''), reviewed_at, archived_at
		FROM compensation_history
		WHERE compensation_id=$1 AND acquisition_id=$2
		ORDER BY archived_at DESC
	`, compensationID, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.CompensationHistory
	for rows.Next() {
		h := &domain.CompensationHistory{}
		if err := rows.Scan(
			&h.ID, &h.CompensationID, &h.AcquisitionID, &h.TargetType, &h.ParcelID,
			&h.AssetID, &h.CompensationType,
			&h.CoveragePercent, &h.Amount,
			&h.CompensationDate, &h.Note,
			&h.Status, &h.ReviewNote, &h.ReviewedBy, &h.ReviewedAt, &h.ArchivedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, organization_name, source_type, amount, currency, note, created_at, created_by
		FROM funding_sources
		WHERE acquisition_id = $1
		ORDER BY created_at ASC
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.FundingSource
	for rows.Next() {
		fs := &domain.FundingSource{}
		if err := rows.Scan(&fs.ID, &fs.AcquisitionID, &fs.OrganizationName, &fs.SourceType, &fs.Amount, &fs.Currency, &fs.Note, &fs.CreatedAt, &fs.CreatedBy); err != nil {
			return nil, err
		}
		list = append(list, fs)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) GetFundingSource(ctx context.Context, id uuid.UUID) (*domain.FundingSource, error) {
	fs := &domain.FundingSource{}
	err := r.db.QueryRow(ctx, `
		SELECT id, acquisition_id, organization_name, source_type, amount, currency, note, created_at, created_by
		FROM funding_sources WHERE id = $1
	`, id).Scan(&fs.ID, &fs.AcquisitionID, &fs.OrganizationName, &fs.SourceType, &fs.Amount, &fs.Currency, &fs.Note, &fs.CreatedAt, &fs.CreatedBy)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return fs, nil
}

func (r *landAcqRepo) CreateFundingSource(ctx context.Context, fs *domain.FundingSource) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO funding_sources (acquisition_id, organization_name, source_type, amount, currency, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, fs.AcquisitionID, fs.OrganizationName, fs.SourceType, fs.Amount, fs.Currency, fs.Note, fs.CreatedBy).
		Scan(&fs.ID, &fs.CreatedAt)
}

func (r *landAcqRepo) UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE funding_sources
		SET organization_name = $1, source_type = $2, amount = $3, currency = $4, note = $5
		WHERE id = $6
	`, fs.OrganizationName, fs.SourceType, fs.Amount, fs.Currency, fs.Note, fs.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) DeleteFundingSource(ctx context.Context, id uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM funding_sources WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, parcel_id, last_name, first_name, register_no, phone, email, address, note, created_at, created_by
		FROM authorized_representatives
		WHERE acquisition_id = $1 AND parcel_id = $2
		ORDER BY created_at DESC
	`, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reps []*domain.AuthorizedRepresentative
	for rows.Next() {
		rep := &domain.AuthorizedRepresentative{}
		if err := rows.Scan(&rep.ID, &rep.AcquisitionID, &rep.ParcelID, &rep.LastName, &rep.FirstName, &rep.RegisterNo, &rep.Phone, &rep.Email, &rep.Address, &rep.Note, &rep.CreatedAt, &rep.CreatedBy); err != nil {
			return nil, err
		}
		reps = append(reps, rep)
	}
	return reps, nil
}

func (r *landAcqRepo) CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO authorized_representatives (acquisition_id, parcel_id, last_name, first_name, register_no, phone, email, address, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`, rep.AcquisitionID, rep.ParcelID, rep.LastName, rep.FirstName, rep.RegisterNo, rep.Phone, rep.Email, rep.Address, rep.Note, rep.CreatedBy).
		Scan(&rep.ID, &rep.CreatedAt)
}

func (r *landAcqRepo) DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM authorized_representatives WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, from_status, to_status, sort_order
		FROM acquisition_workflow
		ORDER BY sort_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.AcquisitionWorkflow
	for rows.Next() {
		w := &domain.AcquisitionWorkflow{}
		if err := rows.Scan(&w.ID, &w.FromStatusID, &w.ToStatusID, &w.SortOrder); err != nil {
			return nil, err
		}
		if w.FromStatusID != nil {
			w.FromStatusName = domain.AcquisitionStatusLabels[*w.FromStatusID]
		}
		w.ToStatusName = domain.AcquisitionStatusLabels[w.ToStatusID]
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error) {
	w := &domain.AcquisitionWorkflow{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO acquisition_workflow (from_status, to_status, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, from_status, to_status, sort_order
	`, fromStatusID, toStatusID, sortOrder).Scan(&w.ID, &w.FromStatusID, &w.ToStatusID, &w.SortOrder)
	if err != nil {
		return nil, err
	}
	if w.FromStatusID != nil {
		w.FromStatusName = domain.AcquisitionStatusLabels[*w.FromStatusID]
	}
	w.ToStatusName = domain.AcquisitionStatusLabels[w.ToStatusID]
	return w, nil
}

func (r *landAcqRepo) DeleteAcquisitionWorkflow(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM acquisition_workflow WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error) {
	args := []any{}
	conditions := []string{}
	i := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", i))
		args = append(args, filter.Status)
		i++
	}
	if filter.AcquisitionID != "" {
		conditions = append(conditions, fmt.Sprintf("c.acquisition_id = $%d", i))
		args = append(args, filter.AcquisitionID)
		i++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(c.parcel_id ILIKE $%d OR COALESCE(rh.name,'') || ' ' || COALESCE(rh.last_name,'') ILIKE $%d)", i, i))
		args = append(args, "%"+filter.Search+"%")
		i++
	}

	where := "1=1"
	if len(conditions) > 0 {
		where = strings.Join(conditions, " AND ")
	}

	joinSQL := `
		FROM compensation c
		JOIN land_acquisition a ON a.id = c.acquisition_id
		LEFT JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
		LEFT JOIN parcel_detail pd ON pd.parcel_id = p.id
		LEFT JOIN right_holder rh ON rh.id = pd.holder_id
		WHERE ` + where

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+joinSQL, args...).Scan(&total); err != nil {
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
		SELECT c.id, c.acquisition_id, COALESCE(c.target_type,'asset'), COALESCE(c.parcel_id,''),
		       c.asset_id, COALESCE(c.compensation_type,'cash'),
		       COALESCE(c.coverage_percent,100), COALESCE(c.amount,0),
		       c.compensation_date, COALESCE(c.note,''),
		       COALESCE(c.status,'pending'), COALESCE(c.review_note,''),
		       COALESCE(c.reviewed_by,''), c.reviewed_at,
		       COALESCE(c.valuation_report_url,''), COALESCE(c.valuation_report_name,''),
		       c.created_at, COALESCE(c.created_by,''), c.updated_at,
		       COALESCE(a.acquisition_name,''),
		       COALESCE(rh.name,''), COALESCE(rh.last_name,''),
		       COALESCE(rh.register_no,''), COALESCE(rh.phone,'')
		%s
		ORDER BY c.created_at DESC
		LIMIT $%d OFFSET $%d
	`, joinSQL, i, i+1), append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.GlobalCompensation
	for rows.Next() {
		gc := &domain.GlobalCompensation{}
		if err := rows.Scan(
			&gc.ID, &gc.AcquisitionID, &gc.TargetType, &gc.ParcelID,
			&gc.AssetID, &gc.CompensationType,
			&gc.CoveragePercent, &gc.Amount,
			&gc.CompensationDate, &gc.Note,
			&gc.Status, &gc.ReviewNote, &gc.ReviewedBy, &gc.ReviewedAt,
			&gc.ValuationReportURL, &gc.ValuationReportName,
			&gc.CreatedAt, &gc.CreatedBy, &gc.UpdatedAt,
			&gc.AcquisitionName,
			&gc.HolderName, &gc.HolderLastName,
			&gc.HolderRegisterNo, &gc.HolderPhone,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, gc)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error) {
	gc := &domain.GlobalCompensation{}
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.acquisition_id, COALESCE(c.target_type,'asset'), COALESCE(c.parcel_id,''),
		       c.asset_id, COALESCE(c.compensation_type,'cash'),
		       COALESCE(c.coverage_percent,100), COALESCE(c.amount,0),
		       c.compensation_date, COALESCE(c.note,''),
		       COALESCE(c.status,'pending'), COALESCE(c.review_note,''),
		       COALESCE(c.reviewed_by,''), c.reviewed_at,
		       COALESCE(c.valuation_report_url,''), COALESCE(c.valuation_report_name,''),
		       c.created_at, COALESCE(c.created_by,''), c.updated_at,
		       COALESCE(a.acquisition_name,''),
		       COALESCE(rh.name,''), COALESCE(rh.last_name,''),
		       COALESCE(rh.register_no,''), COALESCE(rh.phone,'')
		FROM compensation c
		JOIN land_acquisition a ON a.id = c.acquisition_id
		LEFT JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
		LEFT JOIN parcel_detail pd ON pd.parcel_id = p.id
		LEFT JOIN right_holder rh ON rh.id = pd.holder_id
		WHERE c.id = $1
	`, id).Scan(
		&gc.ID, &gc.AcquisitionID, &gc.TargetType, &gc.ParcelID,
		&gc.AssetID, &gc.CompensationType,
		&gc.CoveragePercent, &gc.Amount,
		&gc.CompensationDate, &gc.Note,
		&gc.Status, &gc.ReviewNote, &gc.ReviewedBy, &gc.ReviewedAt,
		&gc.ValuationReportURL, &gc.ValuationReportName,
		&gc.CreatedAt, &gc.CreatedBy, &gc.UpdatedAt,
		&gc.AcquisitionName,
		&gc.HolderName, &gc.HolderLastName,
		&gc.HolderRegisterNo, &gc.HolderPhone,
	)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return gc, nil
}

func (r *landAcqRepo) ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, name, sort_order FROM asset_spec_type ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AssetSpecType
	for rows.Next() {
		t := &domain.AssetSpecType{}
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO asset_spec_type (code, name, sort_order) VALUES ($1,$2,$3) RETURNING id
	`, t.Code, t.Name, t.SortOrder).Scan(&t.ID)
}

func (r *landAcqRepo) UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	_, err := r.db.Exec(ctx, `
		UPDATE asset_spec_type SET code=$1, name=$2, sort_order=$3 WHERE id=$4
	`, t.Code, t.Name, t.SortOrder, t.ID)
	return err
}

func (r *landAcqRepo) DeleteAssetSpecType(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM asset_spec_type WHERE id=$1`, id)
	return err
}

func (r *landAcqRepo) ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, name, default_unit, sort_order FROM asset_calc_type ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AssetCalcType
	for rows.Next() {
		t := &domain.AssetCalcType{}
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.DefaultUnit, &t.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order) VALUES ($1,$2,$3,$4) RETURNING id
	`, t.Code, t.Name, t.DefaultUnit, t.SortOrder).Scan(&t.ID)
}

func (r *landAcqRepo) UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	_, err := r.db.Exec(ctx, `
		UPDATE asset_calc_type SET code=$1, name=$2, default_unit=$3, sort_order=$4 WHERE id=$5
	`, t.Code, t.Name, t.DefaultUnit, t.SortOrder, t.ID)
	return err
}

func (r *landAcqRepo) DeleteAssetCalcType(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM asset_calc_type WHERE id=$1`, id)
	return err
}

func (r *landAcqRepo) SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE land_acquisition SET professional_org_id = $2, updated_at = NOW() WHERE id = $1`,
		acquisitionID, orgUserID,
	)
	return err
}

func (r *landAcqRepo) SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE parcel SET independent_org_id = $3 WHERE id = $2 AND acquisition_id = $1`,
		acqID, parcelID, orgUserID,
	)
	return err
}

var _ repository.LandAcquisitionRepository = (*landAcqRepo)(nil)
