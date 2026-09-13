package postgres

import (
	"context"
	"errors"
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
		rows, err = r.db.Query(ctx, `SELECT id, name, parent_id, sort_order, department_id FROM acquisition_category WHERE parent_id IS NULL ORDER BY sort_order`)
	} else {
		rows, err = r.db.Query(ctx, `SELECT id, name, parent_id, sort_order, department_id FROM acquisition_category WHERE parent_id = $1 ORDER BY sort_order`, *parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AcquisitionCategory
	for rows.Next() {
		c := &domain.AcquisitionCategory{}
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder, &c.DepartmentID); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error) {
	c := &domain.AcquisitionCategory{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO acquisition_category (name, parent_id, sort_order, department_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, parent_id, sort_order, department_id`,
		name, parentID, sortOrder, departmentID,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder, &c.DepartmentID)
	return c, err
}

// UpdateAcquisitionCategory — нэр, эрэмбэ, хариуцсан алба.
//
// department_id нь NULL болгож БУЦААХ боломжтой байх ёстой (алба салгах) тул
// COALESCE-оор "хоосон бол хуучныг үлдээх" гэж бичихгүй — ирсэн утгыг ШУУД
// бичнэ. Дуудагч талбарыг заавал бүрэн илгээнэ.
func (r *landAcqRepo) UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error) {
	c := &domain.AcquisitionCategory{}
	err := r.db.QueryRow(ctx,
		`UPDATE acquisition_category
		    SET name=$1, sort_order=$2, department_id=$3
		  WHERE id=$4
		  RETURNING id, name, parent_id, sort_order, department_id`,
		name, sortOrder, departmentID, id,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder, &c.DepartmentID)
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
			general_category_id, sub_category_id,
			nith_decree_number, group_list_number
			) VALUES (
				$1, $2, $3, $4,
				ST_GeomFromText(NULLIF($5, ''), 4326)::geometry(Polygon, 4326), $6,
				ST_GeomFromText(NULLIF($7, ''), 4326)::geometry(Polygon, 4326), $8,
				$9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18, $19, $20,
				$21, $22,
				$23, $24
			)
	`,
		acq.ID, acq.PlanID, acq.PlanCode, acq.PlanName,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.Status, acq.StartDate, acq.EndDate,
		acq.CreatedAt, acq.CreatedBy, acq.UpdatedAt, acq.UpdatedBy,
		acq.AcquisitionName, acq.ImplementingOrg, acq.Reason, acq.ResponsibleOrg, acq.FundingSource,
		acq.GeneralCategoryID, acq.SubCategoryID,
		acq.NithDecreeNumber, acq.GroupListNumber,
	)
	return err
}

func (r *landAcqRepo) List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error) {
	args := []any{}
	where := "la.deleted_at IS NULL" // soft delete: устгасан чөлөөлөлт жагсаалтад орохгүй
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
	// Сум/дүүрэг — чөлөөлөлтийн хилээр тодорхойлогдож land_acquisition_au-д
	// хадгалагдсан кодоор шүүнэ (нэг чөлөөлөлт олон багтай байж болно).
	if filter.AU2Code != "" {
		where += fmt.Sprintf(` AND (
			EXISTS (
				SELECT 1 FROM land_acquisition_au
				WHERE land_acquisition_id = la.id AND au2_code = $%d
			)
			OR EXISTS (
				SELECT 1 FROM parcel p_au
				WHERE p_au.acquisition_id = la.id AND p_au.deleted_at IS NULL
				  AND p_au.au2_code = $%d
			)
		)`, i, i)
		args = append(args, filter.AU2Code)
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
			WHERE acquisition_id = la.id AND user_id = $%d::int4
		)`, i)
		args = append(args, filter.AssignedUserID)
		i++
	}
	if filter.ProfessionalOrgID != "" {
		where += fmt.Sprintf(` AND (
			la.professional_org_id = $%d::int4
			OR EXISTS (
				SELECT 1 FROM parcel p_org
				WHERE p_org.acquisition_id = la.id AND p_org.independent_org_id = $%d::int4
			)
		)`, i, i)
		args = append(args, filter.ProfessionalOrgID)
		i++
	}
	if filter.HasReviewableValuation {
		where += ` AND EXISTS (
			SELECT 1 FROM parcel_valuation_submission vs
			WHERE vs.acquisition_id = la.id AND vs.status IN ('submitted', 'approved')
		)`
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
			       COALESCE(la.nith_decree_number,''), COALESCE(la.group_list_number,''),
			       (SELECT COUNT(*) FROM parcel WHERE acquisition_id = la.id),
			       -- ЯВЦ: эцсийн төлөвт (3=Нөлөөлөгдсөн гарсан, 4=Татгалзсан,
			       -- 5=Чөлөөлсөн) шилжсэн талбар. HasNonFinalParcels-тай ижил шалгуур.
			       (SELECT COUNT(*) FROM parcel
			         WHERE acquisition_id = la.id AND COALESCE(status,0) IN (3,4,5)),
			       la.professional_org_id,
			       COALESCE(po."name", ''),
			       -- БАЙРШЛААР давхардсан нэгж талбарын тоо. 0-ээс их бол
			       -- чөлөөлөлт "Хээрийн судалгаа" төлөв рүү шилжихгүй тул
			       -- жагсаалт дээр анхааруулга харуулна.
			       (SELECT COUNT(DISTINCT po2.parcel_id)
			          FROM parcel_overlap po2 WHERE po2.acquisition_id = la.id)
		FROM land_acquisition la
		LEFT JOIN acquisition_category gc ON gc.id = la.general_category_id
		LEFT JOIN acquisition_category sc ON sc.id = la.sub_category_id
		LEFT JOIN valuation_org_profile po ON po.id = la.professional_org_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where,
		orderClause(acquisitionSortColumns, filter.SortBy, filter.SortDir,
			"la.created_at DESC", "la.id"),
		i, i+1), append(args, pageSize, offset)...)
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
			&acq.NithDecreeNumber, &acq.GroupListNumber,
			&acq.ParcelCount, &acq.FinalParcelCount,
			&acq.ProfessionalOrgID, &acq.ProfessionalOrgName,
			&acq.OverlappingParcelCount,
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
			       COALESCE(ST_AsText(la.plan_geom),''),
			       -- ГУС төлөвлөгөөний талбайг буцаадаггүй (0 ирдэг) тул
			       -- хадгалагдсан хилээс нь ГУС-тай ИЖИЛ аргаар тооцоолно.
			       COALESCE(NULLIF(la.plan_area_m2,0), NULLIF(pl.area_m2,0),
			                public.calculate_area_utm(la.plan_geom), 0),
			       COALESCE(ST_AsText(la.geometry),''), COALESCE(la.area_m2,0),
			       la.status, la.start_date, la.end_date,
			       la.created_at, COALESCE(la.created_by,''),
			       la.updated_at, COALESCE(la.updated_by,''),
			       COALESCE(la.acquisition_name,''), COALESCE(la.implementing_org,''),
			       COALESCE(la.reason,''), COALESCE(la.responsible_org,''), COALESCE(la.funding_source,''),
			       la.general_category_id, COALESCE(gc.name,''),
			       la.sub_category_id, COALESCE(sc.name,''),
			       COALESCE(la.decree_number,''), la.decree_date,
			       COALESCE(la.nith_decree_number,''), COALESCE(la.group_list_number,''),
			       (SELECT COUNT(*) FROM parcel WHERE acquisition_id = la.id),
			       la.professional_org_id,
			       COALESCE(po."name", ''),
			       -- Төлөвлөгөөний ДЭЛГЭРЭНГҮЙ: ГУС-аас татагдаад plan хүснэгтэд
			       -- хадгалагдсан утгууд (дахин ГУС руу хандах шаардлагагүй).
			       COALESCE(pl.parcel_id,''), COALESCE(pl.plan_type_name,''), COALESCE(pl.gazner,''),
			       pl.start_data, pl.end_data, pl.approved_date
		FROM land_acquisition la
		LEFT JOIN acquisition_category gc ON gc.id = la.general_category_id
		LEFT JOIN acquisition_category sc ON sc.id = la.sub_category_id
			-- Мэргэжлийн байгууллагын нэр нь app-owned profile-оос уншигдана.
		LEFT JOIN valuation_org_profile po ON po.id = la.professional_org_id
			-- plan_id хоосон хуучин бүртгэлд plan_code-оор ч холбогдоно.
		LEFT JOIN plan pl
		       ON pl.id = la.plan_id
		       OR (la.plan_id IS NULL AND pl.plan_code = la.plan_code)
		WHERE la.id = $1 AND la.deleted_at IS NULL
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
		&acq.NithDecreeNumber, &acq.GroupListNumber,
		&acq.ParcelCount,
		&acq.ProfessionalOrgID, &acq.ProfessionalOrgName,
		&acq.PlanParcelID, &acq.PlanTypeName, &acq.PlanGazner,
		&acq.PlanStartDate, &acq.PlanEndDate, &acq.PlanApprovedDate,
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

// FindIntersectingAU3 — чөлөөлөлтийн геометртэй огтлолцох БАГ/ХОРОО (au3)-г олно.
//
// Аймаг/сум (au1, au2)-ыг тусад нь орон зайгаар хайхгүй — au3 мөр дээрх
// au1_code/au2_code-оос авч, зөвхөн НЭРийг эх хүснэгтээс уншина.
//
//   - LEFT JOIN: аймаг/сумын нэр олдоогүй ч тухайн au3 хаягдахгүй (өмнө нь
//     INNER JOIN байсан тул au2 нь тааралдаагүй au3 бүрэн алга болдог байв).
//   - Геометрийг Polygon-оор хатуу cast хийхгүй: MultiPolygon хил ирэхэд
//     cast нь алдаа болж, чөлөөлөлт баггүй үлдэж байв. ST_MakeValid нь
//     эвдэрсэн (self-intersect) хилийг ч уншиж чадна.
func (r *landAcqRepo) FindIntersectingAU3(ctx context.Context, geomWKT string) ([]domain.LandAcquisitionAU, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a3.code, COALESCE(a3.name,''),
		       COALESCE(a3.au1_code,''), COALESCE(a1.name,''),
		       COALESCE(a3.au2_code,''), COALESCE(a2.name,'')
		FROM au3 a3
		LEFT JOIN au1 a1 ON a1.code = a3.au1_code
		LEFT JOIN au2 a2 ON a2.code = a3.au2_code
			WHERE a3.geometry IS NOT NULL
			  AND ST_Intersects(a3.geometry, ST_MakeValid(ST_GeomFromText($1, 4326)))
			ORDER BY a3.code
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

// FindAU2Coverage — чөлөөлөх хил ямар сум/дүүрэгтэй (au2) давхцаж, тэдгээрийн
// АЛЬ НЬ хилийг БҮРЭН бүрхэж байгааг тооцно.
//
// Гурван зүйлийг НЭГ дуудалтаар буцаана:
//   - ReferenceRows: геометртэй au2 мөрийн тоо. Давхцал огт олдоогүй үед
//     "хил нь сумын гадна" гэдгийг "лавлах өгөгдөл ачаалагдаагүй"-гээс
//     ялгаж чадахын тулд шаардлагатай (тиймээс LEFT JOIN).
//   - Covered: ST_CoveredBy — хил тухайн сум/дүүрэгт бүрэн багтсан эсэх.
//     Дамналтын шалгалт ҮҮГЭЭР хийгдэнэ (хувиар бус, топологиор).
//   - AreaM2: давхцлын талбай — татгалзсан шалтгааныг логлоход.
//
// ST_MakeValid — эвдэрсэн (self-intersect) хил ирэхэд ч ажиллана.
func (r *landAcqRepo) FindAU2Coverage(ctx context.Context, geomWKT string) (*domain.AU2Coverage, error) {
	if strings.TrimSpace(geomWKT) == "" {
		return &domain.AU2Coverage{}, nil
	}
	rows, err := r.db.Query(ctx, `
		WITH g AS (
			SELECT ST_MakeValid(ST_GeomFromText($1, 4326)) AS geom
		), ref AS (
			SELECT count(*)::int AS n FROM au2 WHERE geometry IS NOT NULL
		), hit AS (
			SELECT a2.code AS code,
			       COALESCE(a2.name, '') AS name,
			       ST_CoveredBy(g.geom, ST_MakeValid(a2.geometry)) AS covered,
			       COALESCE(public.calculate_area_utm(ST_Intersection(ST_MakeValid(a2.geometry), g.geom)), 0) AS area_m2
			FROM au2 a2, g
			WHERE a2.geometry IS NOT NULL
			  AND ST_Intersects(a2.geometry, g.geom)
		)
		SELECT ref.n, COALESCE(hit.code, ''), COALESCE(hit.name, ''),
		       COALESCE(hit.covered, false), COALESCE(hit.area_m2, 0)
		FROM ref LEFT JOIN hit ON TRUE
		ORDER BY 5 DESC
	`, geomWKT)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cov := &domain.AU2Coverage{}
	for rows.Next() {
		var (
			refRows int
			o       domain.AU2Overlap
		)
		if err := rows.Scan(&refRows, &o.Code, &o.Name, &o.Covered, &o.AreaM2); err != nil {
			return nil, err
		}
		cov.ReferenceRows = refRows
		// LEFT JOIN-ий хоосон мөр (давхцал олдоогүй) — зөвхөн тоог авна.
		if o.Code == "" {
			continue
		}
		cov.Overlaps = append(cov.Overlaps, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cov, nil
}

// CheckContainment — geomWKT нь containerWKT дотор багтсан эсэх (ST_CoveredBy),
// гадна гарсан хэсгийн талбай ба container-ийн бүтэн талбайг (м², geography)
// нэг дуудалтаар тооцно.
//
// Covered нь ТОПОЛОГИЙН шалгалт: хилийн шугам дээр давхцаж таарсан хилийг
// багтсанд тооцдог тул "хувиар хүлцэл" тавих шаардлагагүй.
// Хоёрын аль нэг хоосон бол хоосон дүн буцаана.
func (r *landAcqRepo) CheckContainment(ctx context.Context, geomWKT, containerWKT string) (*domain.GeomContainment, error) {
	if strings.TrimSpace(geomWKT) == "" || strings.TrimSpace(containerWKT) == "" {
		return &domain.GeomContainment{}, nil
	}
	res := &domain.GeomContainment{}
	err := r.db.QueryRow(ctx, `
		WITH g AS (
			SELECT ST_MakeValid(ST_GeomFromText($1, 4326)) AS geom,
			       ST_MakeValid(ST_GeomFromText($2, 4326)) AS container
		)
		SELECT ST_CoveredBy(g.geom, g.container),
		       COALESCE(public.calculate_area_utm(ST_Difference(g.geom, g.container)), 0),
		       COALESCE(public.calculate_area_utm(g.container), 0)
		FROM g
	`, geomWKT, containerWKT).Scan(&res.Covered, &res.OutsideM2, &res.ContainerM2)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ComputeAreaM2 — WKT хилийн талбайг м²-ээр тооцно (geography → бөмбөрцөг дээрх
// бодит талбай). Хил хоосон бол 0 буцаана.
func (r *landAcqRepo) ComputeAreaM2(ctx context.Context, geomWKT string) (float64, error) {
	if strings.TrimSpace(geomWKT) == "" {
		return 0, nil
	}
	var areaM2 float64
	if err := r.db.QueryRow(ctx,
		`SELECT COALESCE(public.calculate_area_utm(ST_GeomFromText($1, 4326)), 0)`,
		geomWKT,
	).Scan(&areaM2); err != nil {
		return 0, err
	}
	return areaM2, nil
}

// ListParcelRefs — чөлөөлөлтийн нэгж талбаруудын ХӨНГӨН лавлагаа (id, дугаар,
// төлөв). Хил солиход ГУС-аас шинээр олдсон дугаартай ЗӨРҮҮЛЖ, хасагдах
// талбаруудыг тодорхойлоход хэрэглэгдэнэ.
func (r *landAcqRepo) ListParcelRefs(ctx context.Context, acquisitionID uuid.UUID) ([]domain.ParcelRef, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(parcel_id, ''), COALESCE(status, 0)
		FROM parcel
		WHERE acquisition_id = $1 AND deleted_at IS NULL
		ORDER BY parcel_id
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []domain.ParcelRef
	for rows.Next() {
		var ref domain.ParcelRef
		if err := rows.Scan(&ref.ID, &ref.ParcelID, &ref.Status); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// PurgeParcels — нэгж талбаруудыг өгөгдлийн сангаас БҮРМӨСӨН устгана (шинэ
// чөлөөлөх хилд ороогүй болсон талбарууд).
//
// Хүү хүснэгтүүдийг CASCADE-д НАЙДАЛГҮЙ, нэг бүрчлэн ДАРААЛУУЛЖ устгана:
//
//   - CASCADE нь хамаагүй өргөн (23 хүснэгт) бөгөөд аль мөр устсаныг харуулдаггүй;
//   - compensation / compensation_* / parcel_valuation_submission_history дээр
//     parcel рүү FK БАЙХГҮЙ тул CASCADE тэднийг ХЭЗЭЭ Ч цэвэрлэхгүй — гараар
//     устгахгүй бол ганцаардсан мөр (болон "хий" олговор) үлдэнэ.
//
// Устгахын ӨМНӨ хадгалалт (MinIO) дээрх файлын URL-уудыг цуглуулж буцаана —
// дуудагч тэднийг фонд устгана (мөр устсаны дараа URL-ыг олох боломжгүй).
//
// БҮГД нэг транзакцид: хагас устгал нь FK-гүй хүснэгтүүд дээр ганцаардсан мөр
// үлдээх тул хамгийн бохир төлөв болно.
func (r *landAcqRepo) PurgeParcels(
	ctx context.Context, acquisitionID uuid.UUID, parcelIDs []uuid.UUID, parcelCodes []string,
) (int, []string, error) {
	if len(parcelIDs) == 0 {
		return 0, nil, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ── Файлын URL-ууд (устгахын ӨМНӨ) ──────────────────────────────────────
	var fileURLs []string
	collect := func(query string, args ...any) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var url string
			if err := rows.Scan(&url); err != nil {
				return err
			}
			if url = strings.TrimSpace(url); url != "" {
				fileURLs = append(fileURLs, url)
			}
		}
		return rows.Err()
	}
	// Баримт нь ХЭД ХЭДЭН нэгж талбарт холбогдож болно (parcel_document-ийн PK
	// нь (parcel_id, document_id)). Тиймээс ЗӨВХӨН устгагдах талбаруудаас
	// гадна хаана ч холбогдоогүй баримтын файлыг устгана — эс бөгөөс үлдэх
	// талбарын баримт нь хадгалалт дээр байхаа болино.
	const orphanDocuments = `
		SELECT d.id, COALESCE(d.file_url, '')
		FROM document d
		WHERE d.id IN (SELECT document_id FROM parcel_document WHERE parcel_id = ANY($1))
		  AND NOT EXISTS (
			SELECT 1 FROM parcel_document pd2
			WHERE pd2.document_id = d.id AND NOT (pd2.parcel_id = ANY($1))
		  )
	`
	var orphanDocIDs []uuid.UUID
	docRows, err := tx.Query(ctx, orphanDocuments, parcelIDs)
	if err != nil {
		return 0, nil, err
	}
	for docRows.Next() {
		var (
			docID uuid.UUID
			url   string
		)
		if err := docRows.Scan(&docID, &url); err != nil {
			docRows.Close()
			return 0, nil, err
		}
		orphanDocIDs = append(orphanDocIDs, docID)
		if url = strings.TrimSpace(url); url != "" {
			fileURLs = append(fileURLs, url)
		}
	}
	docRows.Close()
	if err := docRows.Err(); err != nil {
		return 0, nil, err
	}
	if err := collect(`
		SELECT COALESCE(valuation_report_url, '')
		FROM compensation WHERE acquisition_id = $1 AND parcel_id = ANY($2)
	`, acquisitionID, parcelCodes); err != nil {
		return 0, nil, err
	}
	if err := collect(`
		SELECT COALESCE(photo_pdf_url, '')
		FROM assets WHERE parcel_id = ANY($1)
	`, parcelCodes); err != nil {
		return 0, nil, err
	}

	// ── Хүү мөрүүд: ХАМГИЙН ГҮНЭЭС дээш ────────────────────────────────────
	// 1) assets-ийн хүүхдүүд (assets устахаас ӨМНӨ)
	for _, q := range []string{
		`DELETE FROM asset_calculation WHERE asset_id IN (SELECT id FROM assets WHERE parcel_id = ANY($1))`,
		`DELETE FROM asset_spec WHERE asset_id IN (SELECT id FROM assets WHERE parcel_id = ANY($1))`,
	} {
		if _, err := tx.Exec(ctx, q, parcelCodes); err != nil {
			return 0, nil, err
		}
	}
	// 2) compensation-ийн хүүхдүүд (compensation устахаас ӨМНӨ)
	for _, q := range []string{
		`DELETE FROM compensation_grant WHERE compensation_id IN (
			SELECT id FROM compensation WHERE acquisition_id = $1 AND parcel_id = ANY($2))`,
		`DELETE FROM compensation_history WHERE compensation_id IN (
			SELECT id FROM compensation WHERE acquisition_id = $1 AND parcel_id = ANY($2))`,
	} {
		if _, err := tx.Exec(ctx, q, acquisitionID, parcelCodes); err != nil {
			return 0, nil, err
		}
	}
	// 3) Нөхөх олговор + хөрөнгө (FK-гүй: parcel устахад автоматаар устахгүй)
	if _, err := tx.Exec(ctx,
		`DELETE FROM compensation WHERE acquisition_id = $1 AND parcel_id = ANY($2)`,
		acquisitionID, parcelCodes,
	); err != nil {
		return 0, nil, err
	}
	// 4) Нэгж талбарын ДУГААРААР холбогдсон мөрүүд
	for _, q := range []string{
		`DELETE FROM assets WHERE parcel_id = ANY($1)`,
		`DELETE FROM land_valuation WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_valuation_submission WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_valuation_submission_history WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_status_history WHERE parcel_id = ANY($1)`,
	} {
		if _, err := tx.Exec(ctx, q, parcelCodes); err != nil {
			return 0, nil, err
		}
	}
	// 5) Ганцаардах баримтууд: document устахад parcel_document нь CASCADE-ээр
	// дагаж устна. Бусад талбартай ХУВААЛЦСАН баримт үлдэх ба доорх
	// parcel_document DELETE зөвхөн холбоосыг л тасална.
	if len(orphanDocIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM document WHERE id = ANY($1)`, orphanDocIDs); err != nil {
			return 0, nil, err
		}
	}
	// 6) Нэгж талбарын UUID-аар холбогдсон мөрүүд
	for _, q := range []string{
		`DELETE FROM parcel_detail WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_holder WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_document WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_payment WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_invoice WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_mortgage WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_court_decision WHERE parcel_id = ANY($1)`,
		`DELETE FROM parcel_monitoring WHERE parcel_id = ANY($1)`,
		`DELETE FROM decision_draft_parcel WHERE parcel_id = ANY($1)`,
	} {
		if _, err := tx.Exec(ctx, q, parcelIDs); err != nil {
			return 0, nil, err
		}
	}
	// 7) Нэгж талбар өөрөө
	ct, err := tx.Exec(ctx, `DELETE FROM parcel WHERE id = ANY($1)`, parcelIDs)
	if err != nil {
		return 0, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, err
	}
	return int(ct.RowsAffected()), fileURLs, nil
}

// InsertParcelCodes — дундын сервисийн /parcels/by/acquisition-аас ирсэн нэгж
// талбаруудыг бүртгэнэ: дугаар + БҮРТГЭЛИЙН баганууд (геометр, талбай,
// зориулалт, хаяг, эрхийн хугацаа). Эрх зүйн ДЭЛГЭРЭНГҮЙ (хүсэлт, шийдвэр,
// гэрээ, улсын бүртгэл) нь ЭНД БИЧИГДЭХГҮЙ — түүнийг дараа нь дугаар тус
// бүрээр SyncParcel татна.
//
// insertParcelRow нь ON CONFLICT (parcel_id) DO NOTHING хийдэг тул дахин
// дуудахад давхардал үүсэхгүй: өмнө нь ЭНЭ чөлөөлөлтөд бүртгэгдсэн бол
// existing, ӨӨР АКТИВ чөлөөлөлтөд бүртгэгдсэн бол (uuid.Nil) skipped болно.
//
// БАЙГАА мөрийн бүртгэлийн баганыг ГУС-аас ирсэн утгаар нэг бүрчлэн ШИНЭЧИЛНЭ
// (refreshParcelRow): дахин татах нь эх системийн өөрчлөлтийг тусгана, хуучин
// хувилбараар зөвхөн дугаараар бүртгэгдсэн мөрүүд нөхөгдөнө. Хэрэглэгчийн
// оруулсан өгөгдөл (үнэлгээ, олговор, төлөв, баримт) болон эрх зүйн
// дэлгэрэнгүй (parcel_detail) хөндөгдөхгүй.
//
// Нэг дугаар дээрх алдаа бусдыг зогсоохгүй — эхний алдааг л буцаана.
func (r *landAcqRepo) InsertParcelCodes(
	ctx context.Context,
	acquisitionID uuid.UUID,
	parcels []*domain.SourceParcel,
	acqWKT string,
) (*domain.ParcelDiscoveryResult, error) {
	now := time.Now()
	res := &domain.ParcelDiscoveryResult{ParcelIDs: []string{}}
	var firstErr error

	seen := make(map[string]struct{}, len(parcels))
	for _, sp := range parcels {
		if sp == nil {
			continue
		}
		code := strings.TrimSpace(sp.ParcelID)
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		res.Total++

		sp.ParcelID = code
		parcelID, created, err := r.insertParcelRow(ctx, acquisitionID, sp, sp.GeometryWKT, acqWKT, now)
		if err != nil && strings.TrimSpace(sp.GeometryWKT) != "" {
			// Геометр невалид эсвэл polygon болгож хөрвөхгүй төрөл байж болно.
			// Нэгж талбар дугаараараа бүртгэгдэх ЁСТОЙ тул геометргүйгээр
			// дахин оролдоно (BulkInsertParcels-ийн батлагдсан хэв маяг).
			parcelID, created, err = r.insertParcelRow(ctx, acquisitionID, sp, "", acqWKT, now)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if parcelID == uuid.Nil {
			// parcel_id глобал unique — өөр чөлөөлөлтөд бүртгэгдсэн байна.
			res.Skipped++
			continue
		}
		if created {
			res.Created++
			_ = r.InsertParcelStatusHistory(ctx, parcelID, acquisitionID, 0, "system", "")
		} else {
			res.Existing++
		}
		res.ParcelIDs = append(res.ParcelIDs, code)
	}
	return res, firstErr
}

// insertParcelRow — нэгж талбарын мөрийг үүсгэнэ. parcelWKT хоосон бол
// геометргүйгээр бүртгэнэ. created=false нь мөр өмнө нь байсныг (ON CONFLICT)
// илэрхийлнэ — тэр үед refreshParcelRow түүнийг ГУС-аас ирсэн мэдээллээр
// шинэчилж ID-г буцаана (өөр АКТИВ чөлөөлөлтийнх бол uuid.Nil).
func (r *landAcqRepo) insertParcelRow(
	ctx context.Context,
	acquisitionID uuid.UUID,
	sp *domain.SourceParcel,
	parcelWKT, acqWKT string,
	now time.Time,
) (parcelID uuid.UUID, created bool, err error) {
	err = r.db.QueryRow(ctx, `
			WITH raw AS (
				SELECT
					ST_GeomFromText(NULLIF($9, ''), 4326)                           AS parcel_geom,
					ST_GeomFromText(NULLIF($10, ''), 4326)::geometry(Polygon, 4326) AS acquisition_geom
			),
			geom AS (
				-- parcel.geometry нь geometry(Polygon,4326): эх системээс ирсэн
				-- ӨӨР төрлийг (нэг хэсэгтэй MULTIPOLYGON) шууд cast-лахад
				-- query УНАНА. Нэг хэсэгтэйг гаргаж авна; бусад тохиолдолд
				-- NULL (мөр нь дугаараараа бүртгэгдэнэ).
				SELECT
					CASE
						WHEN parcel_geom IS NULL THEN NULL
						WHEN GeometryType(parcel_geom) = 'POLYGON' THEN parcel_geom
						ELSE ST_GeometryN(ST_CollectionExtract(parcel_geom, 3), 1)
					END::geometry(Polygon, 4326) AS parcel_geom,
					acquisition_geom
				FROM raw
			),
			overlap AS (
				SELECT ST_GeometryN(ST_CollectionExtract(ST_Intersection(parcel_geom, acquisition_geom), 3), 1)::geometry(Polygon, 4326) AS geom
				FROM geom
			)
			INSERT INTO parcel (
				id, parcel_id, old_parcel_id, acquisition_id, valid_from, valid_till,
				au1_code, au2_code, au3_code,
				geometry,
				acquisition_geom,
				acquisition_area_m2,
				right_type, landuse, landuse_name, area_m2,
				address_khashaa, address_streetname, property_no,
				created_at, created_by, updated_at, updated_by
			)
			SELECT
				uuid_generate_v4(), $1, $2, $3, $4, $5,
				$6, $7, $8,
				geom.parcel_geom,
				CASE
					WHEN overlap.geom IS NOT NULL AND NOT ST_IsEmpty(overlap.geom)
					THEN overlap.geom
				END,
				CASE
					WHEN overlap.geom IS NOT NULL AND NOT ST_IsEmpty(overlap.geom)
					THEN public.calculate_area_utm(overlap.geom)
				END,
				$11, $12, $13, $14,
				$15, $16, $17,
				$18, 'system', $19, 'system'
			FROM geom, overlap
			ON CONFLICT (parcel_id) DO NOTHING
			RETURNING id
		`,
		sp.ParcelID, sp.OldParcelID, acquisitionID, sp.ValidFrom, sp.ValidTill,
		sp.AU1Code, sp.AU2Code, sp.AU3Code,
		parcelWKT, acqWKT,
		sp.RightType, sp.Landuse, sp.LanduseName, sp.AreaM2,
		sp.AddressKhashaa, sp.AddressStreet, sp.PropertyNo,
		now, now,
	).Scan(&parcelID)
	if err == nil {
		return parcelID, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}

	// ON CONFLICT DO NOTHING — мөр аль хэдийн байна. ГУС-аас ИРСЭН бүртгэлийн
	// баганыг нэг бүрчлэн ШИНЭЧИЛНЭ (устгасан чөлөөлөлтийн мөрийг зэрэг
	// эргүүлэн эзэмшинэ).
	refreshedID, reclaimed, refreshErr := r.refreshParcelRow(ctx, acquisitionID, sp, parcelWKT, acqWKT, now)
	if refreshErr != nil {
		return uuid.Nil, false, refreshErr
	}
	if refreshedID == uuid.Nil {
		// Дугаарыг ӨӨР, АКТИВ чөлөөлөлт эзэмшиж байна.
		return uuid.Nil, false, nil
	}
	// Эргүүлэн эзэмшсэн мөр нь шинэ чөлөөлөлтөд ШИНЭЭР нэмэгдсэнтэй адил
	// (created=true) — дуудагч статусын түүх үүсгэнэ.
	return refreshedID, reclaimed, nil
}

// refreshParcelRow — ГУС-аас (/parcels/by/acquisition) ИРСЭН бүртгэлийн
// баганыг БАЙГАА мөр дээр нэг бүрчлэн шинэчилнэ. insertParcelRow-ийн
// ON CONFLICT DO NOTHING-ийн ДАРАА ажиллана.
//
// ХОЁР тохиолдлыг НЭГ query-гээр гүйцэтгэнэ:
//
//  1. Мөр нь ЭНЭ чөлөөлөлтийнх — зөвхөн бүртгэлийн багана шинэчлэгдэнэ
//     (reclaimed=false).
//  2. Мөр нь УСТГАСАН (soft delete) өөр чөлөөлөлтийнх — эргүүлэн эзэмшинэ
//     (acquisition_id, deleted_at, status) + бүртгэлийн багана
//     шинэчлэгдэнэ (reclaimed=true).
//
// Эргүүлэн эзэмших шалтгаан: parcel.parcel_id глобал unique бөгөөд FK-ийн бай
// тул устгасан чөлөөлөлтийн нэгж талбар дугаараа эзэлсээр үлддэг. Эргүүлэн
// эзэмшихгүй бол буруу бүртгэсэн чөлөөлөлтийг устгаад ДАХИН бүртгэхэд нэгж
// талбарууд "өөрд бүртгэлтэй" гэж алгасагдаж, жагсаалт хоосон үлдэнэ. Хуучин
// (устгасан) чөлөөлөлтийн хүү мөрүүд (эзэмшигч, үнэлгээ) нь ХУУЧИН
// acquisition_id-тайгаа хэвээр үлдэнэ — тэдгээр нь acquisition_id-гаар
// шүүгддэг тул шинэ чөлөөлөлтөд харагдахгүй, өгөгдөл ч алдагдахгүй.
//
// Дугаарыг ӨӨР, АКТИВ чөлөөлөлт эзэмшиж байвал юу ч бичихгүй — (uuid.Nil, false, nil).
//
// ИРСЭН утга л дарж бичнэ: хоосон/0 ирвэл хадгалагдсан утга үлдэнэ. Шалтгаан:
//   - au1_code/au3_code-ыг эх систем БУЦААХГҮЙ — тэднийг геометрээс
//     FillParcelAUCodesFromGeometry нөхдөг тул хоослож болохгүй.
//   - Геометр невалид үед дуудагч parcelWKT="" болгож ДАХИН оролддог тул
//     хадгалагдсан геометрийг хоослох ёсгүй.
//
// Геометр шинэчлэгдэж байгаа тул чөлөөлөгдөх хил/талбай, үлдэх талбайг мөн ЭНД
// дахин бодно — эс бөгөөс хил сольсны дараа хуучин давхцал хоцорно.
//
// Хэрэглэгчийн оруулсан өгөгдөл (үнэлгээ, олговор, төлөв, баримт) БОЛОН
// эрх зүйн дэлгэрэнгүй (parcel_detail) хөндөгдөхгүй.
func (r *landAcqRepo) refreshParcelRow(
	ctx context.Context,
	acquisitionID uuid.UUID,
	sp *domain.SourceParcel,
	parcelWKT, acqWKT string,
	now time.Time,
) (parcelID uuid.UUID, reclaimed bool, err error) {
	err = r.db.QueryRow(ctx, `
			WITH src AS (
				SELECT
					-- parcel.geometry нь geometry(Polygon,4326): нэг хэсэгтэй
					-- MULTIPOLYGON-ыг гаргаж авна, бусад төрөл NULL болно
					-- (тэр үед хадгалагдсан геометр хэвээр үлдэнэ).
					CASE
						WHEN raw IS NULL THEN NULL
						WHEN GeometryType(raw) = 'POLYGON' THEN raw
						ELSE ST_GeometryN(ST_CollectionExtract(raw, 3), 1)
					END::geometry(Polygon, 4326) AS parcel_geom,
					acq_geom
				FROM (
					SELECT ST_GeomFromText(NULLIF($3, ''), 4326)                          AS raw,
					       ST_GeomFromText(NULLIF($4, ''), 4326)::geometry(Polygon, 4326) AS acq_geom
				) t
			),
			target AS (
				-- Дугаараар (глобал unique) — эзэмшлийг доорх WHERE шалгана.
				SELECT id, acquisition_id, geometry FROM parcel WHERE parcel_id = $1
			),
			calc AS (
				SELECT
					t.id,
					t.acquisition_id                        AS old_acquisition_id,
					COALESCE(s.parcel_geom, t.geometry)     AS parcel_geom,
					ST_GeometryN(
						ST_CollectionExtract(
							ST_Intersection(
								ST_MakeValid(COALESCE(s.parcel_geom, t.geometry)),
								ST_MakeValid(s.acq_geom)
							), 3
						), 1
					)::geometry(Polygon, 4326)              AS overlap_geom
				FROM target t CROSS JOIN src s
			)
			UPDATE parcel p SET
				old_parcel_id      = COALESCE(NULLIF($5, ''), p.old_parcel_id),
				valid_from         = COALESCE($6, p.valid_from),
				valid_till         = COALESCE($7, p.valid_till),
				au1_code           = COALESCE(NULLIF($8, ''), p.au1_code),
				au2_code           = COALESCE(NULLIF($9, ''), p.au2_code),
				au3_code           = COALESCE(NULLIF($10, ''), p.au3_code),
				right_type         = COALESCE(NULLIF($11, 0), p.right_type),
				landuse            = COALESCE(NULLIF($12, ''), p.landuse),
				landuse_name       = COALESCE(NULLIF($13, ''), p.landuse_name),
				area_m2            = COALESCE(NULLIF($14, 0), p.area_m2),
				address_khashaa    = COALESCE(NULLIF($15, ''), p.address_khashaa),
				address_streetname = COALESCE(NULLIF($16, ''), p.address_streetname),
				property_no        = COALESCE(NULLIF($17, ''), p.property_no),
				geometry           = c.parcel_geom,
				acquisition_geom = CASE
					WHEN c.overlap_geom IS NOT NULL AND NOT ST_IsEmpty(c.overlap_geom)
					THEN c.overlap_geom ELSE p.acquisition_geom
				END,
				acquisition_area_m2 = CASE
					WHEN c.overlap_geom IS NOT NULL AND NOT ST_IsEmpty(c.overlap_geom)
					THEN public.calculate_area_utm(c.overlap_geom) ELSE p.acquisition_area_m2
				END,
				remaining_area_m2 = CASE
					WHEN c.overlap_geom IS NOT NULL AND NOT ST_IsEmpty(c.overlap_geom)
					THEN GREATEST(
						COALESCE(NULLIF($14, 0), NULLIF(p.area_m2, 0), public.calculate_area_utm(c.parcel_geom))
						- public.calculate_area_utm(c.overlap_geom), 0)
					ELSE p.remaining_area_m2
				END,
				-- Эргүүлэн эзэмших (хуучин мөр устгасан чөлөөлөлтийнх байсан)
				-- үед л acquisition_id/deleted_at/status хөдөлнө.
				acquisition_id = $2,
				deleted_at = CASE WHEN p.acquisition_id = $2 THEN p.deleted_at ELSE NULL END,
				status     = CASE WHEN p.acquisition_id = $2 THEN p.status ELSE $18 END,
				updated_at = $19,
				updated_by = 'system'
			FROM calc c
			WHERE p.id = c.id
			  AND (
			      p.acquisition_id = $2
			      OR EXISTS (
			          SELECT 1 FROM land_acquisition la
			          WHERE la.id = p.acquisition_id AND la.deleted_at IS NOT NULL
			      )
			  )
			RETURNING p.id, (c.old_acquisition_id <> $2)
		`,
		sp.ParcelID, acquisitionID, parcelWKT, acqWKT,
		sp.OldParcelID, sp.ValidFrom, sp.ValidTill,
		sp.AU1Code, sp.AU2Code, sp.AU3Code,
		sp.RightType, sp.Landuse, sp.LanduseName, sp.AreaM2,
		sp.AddressKhashaa, sp.AddressStreet, sp.PropertyNo,
		domain.ParcelStatusPending, now,
	).Scan(&parcelID, &reclaimed)
	if err == nil {
		return parcelID, reclaimed, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		// Мөр нь ӨӨР, АКТИВ чөлөөлөлтийнх — хөндөхгүй.
		return uuid.Nil, false, nil
	}
	return uuid.Nil, false, err
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
	// Талбайг ХАДГАЛАХГҮЙ, геометрээс нь УНШИХ үедээ бодно: талбайн цорын ганц
	// эх сурвалж нь public.calculate_area_utm (ГУС-тай ижил UTM арга) байх
	// ёстой — хадгалсан тоо нь функц засагдахад хоцрогдоно.
	rows, err := r.db.Query(ctx, `
		SELECT id, land_acquisition_id,
		       COALESCE(ST_AsText(old_geometry), ''),
		       COALESCE(ST_AsText(new_geometry), ''),
		       COALESCE(public.calculate_area_utm(old_geometry), 0),
		       COALESCE(public.calculate_area_utm(new_geometry), 0),
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
			&h.OldAreaM2,
			&h.NewAreaM2,
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
	// БАЙРШЛЫН давхардлаар шүүх. parcel_overlap нь хоёр чиглэлээр бичигддэг
	// тул EXISTS нь давхардалтай ХОЁУЛАНГ нь олно.
	if filter.HasOverlap != nil {
		cond := "EXISTS"
		if !*filter.HasOverlap {
			cond = "NOT EXISTS"
		}
		where += " AND " + cond + " (SELECT 1 FROM parcel_overlap po WHERE po.parcel_id = p.id)"
	}
	// Үнэлгээний байгууллагын хүрээ. Дашбоардын parcelScopeCond-той ИЖИЛ
	// дүрэм: үндсэн байгууллага бол чөлөөлөлтийн бүх нэгж талбар, эс бөгөөс
	// зөвхөн өөрт хөндлөнгийн үнэлгээгээр холбогдсон нэгж талбар.
	if filter.ProfessionalOrgID != "" {
		where += fmt.Sprintf(` AND (
			p.independent_org_id = $%d::int4
			OR EXISTS (
				SELECT 1 FROM land_acquisition la_scope
				WHERE la_scope.id = p.acquisition_id
				  AND la_scope.deleted_at IS NULL
				  AND la_scope.professional_org_id = $%d::int4
			)
		)`, i, i)
		args = append(args, filter.ProfessionalOrgID)
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
				       COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2,0), 0),
				       COALESCE(p.db_changed,''), COALESCE(p.changed_parcel_id,''),
				       COALESCE(p.status,0), COALESCE(ps.name,''),
				       p.independent_org_id,
				       -- Хараат бус үнэлгээний байгууллагын нэрийг profile-оос авна.
				       COALESCE(io.name, ''),
				       COALESCE((SELECT SUM(c.amount) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'cash' AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')), 0),
				       COALESCE((SELECT SUM(c.amount) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'land_grant' AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')), 0),
				       COALESCE((SELECT COUNT(*) FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.compensation_type = 'land_grant'), 0),
				       COALESCE((SELECT jsonb_object_agg(vs.valuation_type, vs.status) FROM parcel_valuation_submission vs WHERE vs.acquisition_id = p.acquisition_id AND vs.parcel_id = p.parcel_id), '{}'::jsonb),
				       -- ГАЗАР ӨМЧЛӨГЧ/ЭЗЭМШИГЧ. Нэг нэгж талбар дээр ОЛОН
				       -- эзэмшигч байж болох тул ҮНДСЭН хүсэлт гаргагчийг
				       -- (main_applicant) авна; тэр тэмдэглэгээгүй бол хамгийн
				       -- эрт бүртгэгдсэнийг. Хуулийн этгээд үед last_name хоосон
				       -- ирдэг тул CONCAT_WS-ээр залгаж, давхар зай гаргахгүй.
				       COALESCE((
				         SELECT TRIM(CONCAT_WS(' ', NULLIF(rh.last_name,''), rh.name))
				           FROM parcel_holder ph
				           JOIN right_holder rh ON rh.id = ph.holder_id
				          WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder'
				          ORDER BY ph.main_applicant DESC, ph.created_at
				          LIMIT 1
				       ), ''),
				       COALESCE((
				         SELECT COUNT(*) FROM parcel_holder ph
				          WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder'
				       ), 0),
				       -- БАЙРШЛЫН давхардал (parcel_overlap). Давхардалтай нэгж
				       -- талбарыг жагсаалт дээр УЛААНААР анхааруулна.
				       COALESCE((
				         SELECT COUNT(*) FROM parcel_overlap po WHERE po.parcel_id = p.id
				       ), 0)
				FROM parcel p
				LEFT JOIN parcel_status ps ON ps.id = p.status
				LEFT JOIN valuation_org_profile io ON io.id = p.independent_org_id
				WHERE %s
			ORDER BY %s
			LIMIT $%d OFFSET $%d
		`, where,
			orderClause(parcelSortColumns, filter.SortBy, filter.SortDir,
				"p.parcel_id", "p.id"),
			i, i+1),
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
			&p.HolderName, &p.HolderCount,
			&p.OverlapCount,
		); err != nil {
			return nil, 0, err
		}
		p.HasOverlap = p.OverlapCount > 0
		p.RemainingAreaM2 = &remainingAreaM2
		result = append(result, p)
	}
	return result, total, rows.Err()
}

func (r *landAcqRepo) GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error) {
	p := &domain.Parcel{}
	var remainingAreaM2 float64
	err := r.db.QueryRow(ctx, `
			SELECT p.id, COALESCE(p.parcel_id,''), COALESCE(p.old_parcel_id,''),
			       p.acquisition_id, p.valid_from, p.valid_till,
			       COALESCE(p.au1_code,''), COALESCE(p.au2_code,''), COALESCE(p.au3_code,''),
			       COALESCE(ST_AsText(p.geometry),''),
			       COALESCE(ST_AsText(p.acquisition_geom),''),
			       COALESCE(p.acquisition_area_m2,0),
			       COALESCE(p.right_type,0), COALESCE(p.landuse,''), COALESCE(p.area_m2,0),
			       p.created_at, COALESCE(p.created_by,''),
			       COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2,0), 0),
			       COALESCE(p.db_changed,''), COALESCE(p.changed_parcel_id,''),
			       COALESCE(p.status,0), COALESCE(ps.name,''),
			       p.independent_org_id
		FROM parcel p
		LEFT JOIN parcel_status ps ON ps.id = p.status
		WHERE p.id = $1
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
		&p.StatusID, &p.StatusName,
		&p.IndependentOrgID,
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
		       COALESCE(remaining_area_m2, area_m2 - COALESCE(acquisition_area_m2,0), 0),
		       COALESCE(db_changed,''), COALESCE(changed_parcel_id,'')
		FROM parcel
		WHERE parcel_id = $1 AND deleted_at IS NULL
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
				COALESCE(p.right_type,0), COALESCE(p.landuse,''), COALESCE(p.landuse_name,''), COALESCE(p.area_m2,0),
				COALESCE(p.address_khashaa,''), COALESCE(p.address_streetname,''), COALESCE(p.property_no,''),
				p.created_at, COALESCE(p.created_by,''),
				COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2,0), 0),
				COALESCE(p.db_changed,''), COALESCE(p.changed_parcel_id,''),
				COALESCE(p.status,0), COALESCE(ps.name,''),
				p.independent_org_id, p.selected_valuation_type,
				p.estimated_value, p.estimated_value_at, COALESCE(p.estimated_value_by,''), p.estimated_confidence_percent, p.estimated_base_fee_per_m2,
			COALESCE(pd.right_type,0),
			COALESCE(pd.app_no,''), pd.app_timestamps, COALESCE(pd.app_type,''), COALESCE(pd.app_status,''),
			COALESCE(pd.decision_no,''), pd.decision_date,
			COALESCE(pd.contract_no,''), pd.contract_date, pd.contract_begin, pd.contract_end,
			COALESCE(pd.contract_property_no,''), COALESCE(pd.contract_status,''),
			COALESCE(pd.certificate_no,''), pd.certificate_date,
			COALESCE(pd.record_no,''), pd.record_date, COALESCE(pd.record_certificate_no,''), COALESCE(pd.record_status,''),
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
		&pf.RightType, &pf.Landuse, &pf.LanduseName, &pf.AreaM2,
		&pf.AddressKhashaa, &pf.AddressStreet, &pf.PropertyNo,
		&pf.CreatedAt, &pf.CreatedBy,
		&pfRemainingAreaM2,
		&pf.DbChanged, &pf.ChangedParcelID,
		&pf.StatusID, &pf.StatusName,
		&pf.IndependentOrgID, &pf.SelectedValuationType,
		&pf.EstimatedValue, &pf.EstimatedValueAt, &pf.EstimatedValueBy, &pf.EstimatedConfidencePercent, &pf.EstimatedBaseFeePerM2,
		&d.RightType,
		&d.AppNo, &d.AppTimestamps, &d.AppType, &d.AppStatus,
		&d.DecisionNo, &d.DecisionDate,
		&d.ContractNo, &d.ContractDate, &d.ContractBegin, &d.ContractEnd,
		&d.ContractPropertyNo, &d.ContractStatus,
		&d.CertificateNo, &d.CertificateDate,
		&d.RecordNo, &d.RecordDate, &d.RecordCertificateNo, &d.RecordStatus,
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
	// parcel_detail нь LEFT JOIN тул мөр байхгүй үед бүх талбар хоосон гарна.
	// Үнэлгээний талбарууд (суурь үнэ, бүс) нь эзэмшигч/эрхийн төрлөөс ХАМААРАЛГҮЙ
	// бөглөгддөг тул тэднийг ч шалгана — эс бөгөөс татсан суурь үнэ дэлгэцэд
	// хүрэхгүй хаягдана.
	if d.HolderName != "" || d.RightType != 0 || d.AppNo != "" ||
		d.BasePricePerHa != nil || d.ValuationZone != "" ||
		d.AuctionCoeff != nil || d.AuctionPrice != nil {
		pf.Detail = d
	}
	// Эзэмшигчид (олон байж болно). Алдаа гарсан ч дэлгэрэнгүйг буцаана —
	// эзэмшигчийн жагсаалт хоосон үлдэнэ.
	if holders, hErr := r.ListParcelHolders(ctx, id); hErr == nil {
		pf.Holders = holders
	}
	// ГУС-аас татагдсан санхүү/эрх зүйн бүртгэлүүд. Алдаа гарсан ч
	// дэлгэрэнгүйг буцаана — тухайн жагсаалт хоосон үлдэнэ.
	if items, mErr := r.ListParcelMortgages(ctx, id); mErr == nil {
		pf.Mortgages = items
	}
	if items, cErr := r.ListParcelCourtDecisions(ctx, id); cErr == nil {
		pf.CourtDecisions = items
	}
	if items, iErr := r.ListParcelInvoices(ctx, id); iErr == nil {
		pf.Invoices = items
	}
	if items, fErr := r.ListParcelFeeItems(ctx, id); fErr == nil {
		pf.Fees = items
	}
	if items, bErr := r.ListParcelBasePriceFactors(ctx, id); bErr == nil {
		pf.BasePriceFactors = items
	}
	if items, monErr := r.ListParcelMonitorings(ctx, id); monErr == nil {
		pf.Monitorings = items
	}
	// УБЕГ-ийн өмчлөлийн бүртгэл (ГУС-аас БИШ, ХУР/XYP-ээс татагдсан).
	if items, oErr := r.ListParcelOwnerships(ctx, id); oErr == nil {
		pf.Ownerships = items
	}
	return pf, nil
}

// SetParcelDetailHolder — parcel_detail.holder_id-г ҮНДСЭН эзэмшигчид холбоно
// (нэг эзэмшигч харуулдаг хуучин дэлгэц/тайлангуудад хэрэглэгдэнэ).
func (r *landAcqRepo) SetParcelDetailHolder(ctx context.Context, parcelID, holderID uuid.UUID) error {
	if holderID == uuid.Nil {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE parcel_detail SET
			holder_id  = $2,
			updated_at = NOW(),
			updated_by = 'system'
		WHERE parcel_id = $1
	`, parcelID, holderID)
	return err
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
			acquisition_area_m2 = public.calculate_area_utm((SELECT geom FROM overlap)),
			-- Геометр нь discovery-д хадгалагдах болсон тул sync нь ЭНЭ замаар
			-- л давхцлыг тооцно — үлдэх талбайг мөн ЭНД бодно (өмнө нь зөвхөн
			-- ApplyParcelGeometry бичдэг байсан).
			remaining_area_m2   = GREATEST(
				COALESCE(NULLIF(area_m2, 0), public.calculate_area_utm(geometry))
				- public.calculate_area_utm((SELECT geom FROM overlap)), 0),
			updated_at          = NOW()
		WHERE id = $1
		  AND geometry IS NOT NULL
		  AND (SELECT geom FROM overlap) IS NOT NULL
		  AND NOT ST_IsEmpty((SELECT geom FROM overlap))
	`, parcelID, acqWKT)
	return err
}

// ApplyParcelGeometry — дундын сервисээс ИРСЭН геометрээр нэгж талбарын
// геометрийг хадгалж, чөлөөлөлтийн бүстэй давхцах хэсгийг PostGIS-аар тооцно:
//
//   - geometry            — нэгж талбарын хил (эх системийн WKT)
//   - acquisition_geom    — ЧӨЛӨӨЛӨГДӨХ талбайн хил (давхцах хэсэг)
//   - acquisition_area_m2 — ЧӨЛӨӨЛӨГДӨХ талбайн хэмжээ (м², geography)
//   - remaining_area_m2   — үлдэх талбай (нэгж талбарын талбай − чөлөөлөгдөх)
//
// Давхцахгүй бол чөлөөлөгдөх хил/хэмжээ NULL болж, үлдэх талбай нь бүтэн
// талбайтай тэнцэнэ. Үлдэх талбайг тооцохдоо эх системийн area_m2-ыг эрхэмлэж,
// байхгүй үед геометрээс тооцно.
//
// Эх системээс нэг хэсэгтэй MULTIPOLYGON ирж болох тул түүнийг polygon болгож
// хөрвүүлнэ (insertParcelRow-той ижил хэв маяг) — эс бөгөөс шууд cast нь
// query-г УНАГААЖ, геометр нөхөгдөхгүй үлдэнэ.
func (r *landAcqRepo) ApplyParcelGeometry(
	ctx context.Context,
	parcelID uuid.UUID,
	parcelWKT, acqWKT string,
) (*domain.ParcelGeometryResult, error) {
	if strings.TrimSpace(parcelWKT) == "" {
		return nil, domain.ErrInvalidInput
	}

	result := &domain.ParcelGeometryResult{}
	err := r.db.QueryRow(ctx, `
		WITH raw AS (
			SELECT ST_GeomFromText($2, 4326)                                       AS parcel_geom,
			       ST_GeomFromText(NULLIF($3, ''), 4326)::geometry(Polygon, 4326)  AS acq_geom
		),
		src AS (
			SELECT
				CASE
					WHEN parcel_geom IS NULL THEN NULL
					WHEN GeometryType(parcel_geom) = 'POLYGON' THEN parcel_geom
					ELSE ST_GeometryN(ST_CollectionExtract(parcel_geom, 3), 1)
				END::geometry(Polygon, 4326) AS parcel_geom,
				acq_geom
			FROM raw
		),
		calc AS (
			SELECT parcel_geom,
			       ST_GeometryN(
			           ST_CollectionExtract(
			               ST_Intersection(ST_MakeValid(parcel_geom), ST_MakeValid(acq_geom)), 3
			           ), 1
			       )::geometry(Polygon, 4326) AS overlap_geom
			FROM src
		),
		final AS (
			SELECT parcel_geom,
			       CASE WHEN overlap_geom IS NOT NULL AND NOT ST_IsEmpty(overlap_geom)
			            THEN overlap_geom END AS overlap_geom
			FROM calc
		)
		UPDATE parcel p SET
			-- Хөрвүүлэлт NULL болвол (олон хэсэгтэй MULTIPOLYGON) ХАДГАЛАГДСАН
			-- геометр/давхцал хэвээр үлдэнэ — нөхөх үйлдэл байгааг хоослохгүй.
			geometry            = COALESCE(f.parcel_geom, p.geometry),
			acquisition_geom    = COALESCE(f.overlap_geom, p.acquisition_geom),
			acquisition_area_m2 = CASE WHEN f.overlap_geom IS NULL
			                           THEN p.acquisition_area_m2
			                           ELSE public.calculate_area_utm(f.overlap_geom) END,
			remaining_area_m2   = CASE WHEN f.parcel_geom IS NULL
			                           THEN p.remaining_area_m2
			                           ELSE GREATEST(
				COALESCE(NULLIF(p.area_m2, 0), public.calculate_area_utm(f.parcel_geom))
				- COALESCE(CASE WHEN f.overlap_geom IS NULL
				                THEN 0 ELSE public.calculate_area_utm(f.overlap_geom) END, 0),
				0) END,
			updated_at = NOW(),
			updated_by = 'system'
		FROM final f
		WHERE p.id = $1
		RETURNING COALESCE(public.calculate_area_utm(p.geometry), 0),
		          COALESCE(p.acquisition_area_m2, 0),
		          COALESCE(p.remaining_area_m2, 0),
		          (p.acquisition_geom IS NOT NULL)
	`, parcelID, parcelWKT, acqWKT).Scan(
		&result.ParcelAreaM2,
		&result.AcquisitionAreaM2,
		&result.RemainingAreaM2,
		&result.HasOverlap,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

// FillParcelAUCodesFromGeometry — нэгж талбарын ГЕОМЕТРЭЭР харьяалагдах
// баг/хороог (au3) олж, аймаг/сум-дүүрэг/багийн кодыг хадгална. Эх систем нь
// зөвхөн au2-ыг буцаадаг тул au1/au3 үүнээс гарна.
//
// Хамгийн их давхцалтай au3-ыг авна (нэгж талбар хилийн зааг дээр байж болно).
// Олдоогүй бол одоогийн кодуудыг хөндөхгүй.
func (r *landAcqRepo) FillParcelAUCodesFromGeometry(ctx context.Context, parcelID uuid.UUID) (domain.LandAcquisitionAU, error) {
	var au domain.LandAcquisitionAU
	err := r.db.QueryRow(ctx, `
		WITH best AS (
			SELECT a3.code AS au3_code, a3.au2_code, a3.au1_code
			FROM au3 a3, parcel p
			WHERE p.id = $1
			  AND p.geometry IS NOT NULL
			  AND a3.geometry IS NOT NULL
			  AND ST_Intersects(a3.geometry, p.geometry)
			ORDER BY ST_Area(ST_Intersection(a3.geometry, p.geometry)) DESC
			LIMIT 1
		)
		UPDATE parcel p SET
			au1_code   = b.au1_code,
			au2_code   = COALESCE(NULLIF(b.au2_code, ''), p.au2_code),
			au3_code   = b.au3_code,
			updated_at = NOW(),
			updated_by = 'system'
		FROM best b
		WHERE p.id = $1
		RETURNING COALESCE(p.au1_code, ''), COALESCE(p.au2_code, ''), COALESCE(p.au3_code, '')
	`, parcelID).Scan(&au.AU1Code, &au.AU2Code, &au.AU3Code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return au, domain.ErrNotFound
		}
		return au, err
	}
	return au, nil
}

// UpsertHolder finds or creates a right_holder record.
func (r *landAcqRepo) UpsertHolder(ctx context.Context, src *domain.SourceParcel) (uuid.UUID, error) {
	if src.HolderRegisterNo == "" && src.HolderName == "" && src.HolderLastName == "" {
		return uuid.Nil, nil
	}
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

// ReplaceParcelHolders — нэгж талбарын эзэмшигчдийг дундын сервисээс татсан
// жагсаалтаар БҮРЭН шинэчилнэ (sync = тухайн үеийн бодит байдал).
//
// Эзэмшигч тус бүр right_holder-т upsert хийгдэж, parcel_holder-оор нэгж
// талбартай холбогдоно. Хуучин холбоос устгагдана (right_holder мөр өөрөө
// хэвээр — өөр нэгж талбар түүнийг заасан байж болно).
//
// Буцаах утга нь ҮНДСЭН хүсэлт гаргагчийн right_holder.id (parcel_detail.holder_id-д
// холбогдоно). Үндсэн нь тэмдэглэгдээгүй бол эхний эзэмшигчийг авна.
func (r *landAcqRepo) ReplaceParcelHolders(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	appNo string,
	holders []*domain.ParcelHolder,
) (uuid.UUID, error) {
	// Жагсаалт хоосон байсан ч хуучин эзэмшигчид УСТГАГДАНА (эх системд
	// эзэмшигч үлдээгүй гэсэн үг). Дуудалт бүтэлгүйтсэн үед энэ функц
	// дуудагдахгүй — service тэр үед хуучныг хөндөхгүй.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Нөхөн төлбөр авагчийн сонголт нь ГУС-аас ирдэггүй, ГАРААР хийгддэг тул
	// sync түүнийг устгаж болохгүй — устгахын өмнө сэргээхээр хадгална.
	var prevPaymentHolderID *uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT holder_id FROM parcel_holder
		WHERE parcel_id = $1 AND payment_recipient
		LIMIT 1
	`, parcelID).Scan(&prevPaymentHolderID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	// ЗӨВХӨН ГУС-аас татагдсан эзэмшигчийг устгана — гараар бүртгэсэн
	// итгэмжлэгдсэн төлөөлөгч sync дээр хадгалагдаж үлдэнэ.
	if _, err := tx.Exec(ctx,
		"DELETE FROM parcel_holder WHERE parcel_id = $1 AND holder_role = $2",
		parcelID, domain.ParcelHolderRoleHolder,
	); err != nil {
		return uuid.Nil, err
	}

	var mainHolderID uuid.UUID
	var foundMain bool
	for _, h := range holders {
		if h == nil {
			continue
		}
		holderID, err := upsertHolderTx(ctx, tx, h)
		if err != nil {
			return uuid.Nil, err
		}
		if holderID == uuid.Nil {
			continue
		}

		// ON CONFLICT дээр holder_role/payment_recipient-ыг ЗОРИУД хөндөхгүй:
		// эзэмшигч нь итгэмжлэгдсэн төлөөлөгчөөр бүртгэгдсэн байж болох ба
		// гараар хийсэн тэмдэглэгээг sync дарж бичих ёсгүй.
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_holder (
				id, parcel_id, acquisition_id, holder_id, app_no, main_applicant,
				holder_role, created_by, updated_by
			) VALUES (uuid_generate_v4(), $1, $2, $3, NULLIF($4, ''), $5, $6, 'system', 'system')
			ON CONFLICT (parcel_id, holder_id) DO UPDATE SET
				acquisition_id = EXCLUDED.acquisition_id,
				app_no         = EXCLUDED.app_no,
				main_applicant = parcel_holder.main_applicant OR EXCLUDED.main_applicant,
				updated_at     = NOW(),
				updated_by     = 'system'
		`, parcelID, acquisitionID, holderID, appNo, h.MainApplicant,
			domain.ParcelHolderRoleHolder,
		); err != nil {
			return uuid.Nil, err
		}

		switch {
		case h.MainApplicant && !foundMain:
			mainHolderID, foundMain = holderID, true
		case mainHolderID == uuid.Nil:
			mainHolderID = holderID
		}
	}

	// Гараар сонгосон төлбөр авагчийг сэргээнэ. Тухайн хүн шинэ жагсаалтад
	// байхгүй бол (эх системээс хасагдсан) сонголт сэргэхгүй — дэлгэц дээр
	// дахин сонгуулна. Итгэмжлэгдсэн төлөөлөгч мөр устаагүй тул түүний
	// payment_recipient=TRUE хэвээр үлдэж, доорх UPDATE зүгээр дахин баталгаажна.
	if prevPaymentHolderID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE parcel_holder SET payment_recipient = TRUE, updated_at = NOW()
			WHERE parcel_id = $1 AND holder_id = $2
		`, parcelID, *prevPaymentHolderID); err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return mainHolderID, nil
}

// upsertHolderTx — right_holder-ийг регистрийн дугаараар (байвал), эсвэл
// нэр+утсаар тааруулж олно; олдоогүй бол шинээр үүсгэнэ. Олдсон үед
// холбоо барих мэдээллийг шинэчилнэ (эх сурвалж нь дундын сервис).
func upsertHolderTx(ctx context.Context, tx pgx.Tx, h *domain.ParcelHolder) (uuid.UUID, error) {
	if strings.TrimSpace(h.RegisterNo) == "" &&
		strings.TrimSpace(h.Name) == "" &&
		strings.TrimSpace(h.LastName) == "" {
		return uuid.Nil, nil
	}

	var id uuid.UUID
	var err error
	if h.RegisterNo != "" {
		err = tx.QueryRow(ctx,
			"SELECT id FROM right_holder WHERE register_no = $1", h.RegisterNo,
		).Scan(&id)
	} else {
		// Регистргүй хүнийг нэр + утсаар тааруулна (давхардал үүсгэхээс сэргийлж).
		err = tx.QueryRow(ctx, `
			SELECT id FROM right_holder
			WHERE register_no IS NULL
			  AND COALESCE(last_name,'') = $1
			  AND COALESCE(name,'') = $2
			  AND COALESCE(phone,'') = $3
			LIMIT 1
		`, h.LastName, h.Name, h.Phone).Scan(&id)
	}
	if err == nil {
		if _, err := tx.Exec(ctx, `
			UPDATE right_holder SET
				last_name   = NULLIF($2, ''),
				name        = $3,
				phone       = NULLIF($4, ''),
				email       = NULLIF($5, ''),
				holder_type = COALESCE(NULLIF($6, ''), holder_type),
				updated_at  = NOW(),
				updated_by  = 'system'
			WHERE id = $1
		`, id, h.LastName, h.Name, h.Phone, h.Email, h.PersonType); err != nil {
			return uuid.Nil, err
		}
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	var regNo *string
	if h.RegisterNo != "" {
		regNo = &h.RegisterNo
	}
	// right_holder.name нь NOT NULL — өөрийн нэр хоосон бол овгийг тавина.
	name := h.Name
	if name == "" {
		name = h.LastName
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO right_holder (
			id, last_name, name, register_no, phone, email, holder_type,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(), NULLIF($1, ''), $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
			'system', 'system'
		)
		RETURNING id
	`, h.LastName, name, regNo, h.Phone, h.Email, h.PersonType).Scan(&id)
	return id, err
}

// ListParcelHolders — нэгж талбарын эзэмшигчид (үндсэн хүсэлт гаргагч эхэнд).
func (r *landAcqRepo) ListParcelHolders(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelHolder, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ph.id, ph.main_applicant,
		       COALESCE(rh.last_name,''), COALESCE(rh.name,''),
		       COALESCE(rh.register_no,''), COALESCE(rh.phone,''),
		       COALESCE(rh.email,''), COALESCE(rh.holder_type,''),
		       COALESCE(ph.app_no,''),
		       ph.holder_role, ph.payment_recipient,
		       COALESCE(ph.note,''), COALESCE(rh.address,'')
		FROM parcel_holder ph
		JOIN right_holder rh ON rh.id = ph.holder_id
		WHERE ph.parcel_id = $1
		ORDER BY ph.payment_recipient DESC, ph.main_applicant DESC,
		         ph.holder_role, rh.last_name, rh.name
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelHolder
	for rows.Next() {
		h := domain.ParcelHolder{}
		if err := rows.Scan(
			&h.ID, &h.MainApplicant,
			&h.LastName, &h.Name,
			&h.RegisterNo, &h.Phone,
			&h.Email, &h.PersonType, &h.AppNo,
			&h.HolderRole, &h.PaymentRecipient,
			&h.Note, &h.Address,
		); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

// SetParcelPaymentRecipient — нөхөн төлбөр хүлээн авах эзэмшигчийг сонгоно.
// Нэгж талбар тутамд ЗӨВХӨН нэг байх тул эхлээд бусдыг цэвэрлэнэ (DB-ийн
// partial unique index-тэй зөрчилдөхөөс сэргийлж НЭГ гүйлгээнд хийнэ).
func (r *landAcqRepo) SetParcelPaymentRecipient(ctx context.Context, parcelID, parcelHolderID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE parcel_holder SET payment_recipient = FALSE, updated_at = NOW()
		WHERE parcel_id = $1 AND payment_recipient AND id <> $2
	`, parcelID, parcelHolderID); err != nil {
		return err
	}

	ct, err := tx.Exec(ctx, `
		UPDATE parcel_holder SET payment_recipient = TRUE, updated_at = NOW()
		WHERE id = $1 AND parcel_id = $2
	`, parcelHolderID, parcelID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit(ctx)
}

// AddParcelRepresentative — итгэмжлэгдсэн төлөөлөгчийг эзэмшигчийн нэг төрөл
// болгож бүртгэнэ. Бүртгэмэгц ШУУД нөхөн төлбөр хүлээн авагч болно (өмнөх
// сонголт автоматаар цуцлагдана).
func (r *landAcqRepo) AddParcelRepresentative(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	holder *domain.ParcelHolder,
) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	holderID, err := upsertHolderTx(ctx, tx, &domain.ParcelHolder{
		LastName:   holder.LastName,
		Name:       holder.Name,
		RegisterNo: holder.RegisterNo,
		Phone:      holder.Phone,
		Email:      holder.Email,
		PersonType: holder.PersonType,
	})
	if err != nil {
		return uuid.Nil, err
	}
	if holderID == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidInput
	}

	// Хаяг нь зөвхөн төлөөлөгчид бөглөгддөг тул тусад нь бичнэ (upsertHolderTx
	// нь ГУС-аас ирдэг талбаруудыг л мэднэ).
	if strings.TrimSpace(holder.Address) != "" {
		if _, err := tx.Exec(ctx,
			"UPDATE right_holder SET address = $2, updated_at = NOW() WHERE id = $1",
			holderID, holder.Address,
		); err != nil {
			return uuid.Nil, err
		}
	}

	// Шинэ төлөөлөгч төлбөр авагч болох тул өмнөх сонголтыг цуцална.
	if _, err := tx.Exec(ctx, `
		UPDATE parcel_holder SET payment_recipient = FALSE, updated_at = NOW()
		WHERE parcel_id = $1 AND payment_recipient
	`, parcelID); err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO parcel_holder (
			id, parcel_id, acquisition_id, holder_id, main_applicant,
			holder_role, payment_recipient, note, created_by, updated_by
		) VALUES (uuid_generate_v4(), $1, $2, $3, FALSE, $4, TRUE, $5, 'user', 'user')
		ON CONFLICT (parcel_id, holder_id) DO UPDATE SET
			acquisition_id    = EXCLUDED.acquisition_id,
			holder_role       = EXCLUDED.holder_role,
			payment_recipient = TRUE,
			note              = EXCLUDED.note,
			updated_at        = NOW(),
			updated_by        = 'user'
		RETURNING id
	`, parcelID, acquisitionID, holderID,
		domain.ParcelHolderRoleRepresentative, holder.Note,
	).Scan(&id); err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// DeleteParcelHolder — гараар бүртгэсэн итгэмжлэгдсэн төлөөлөгчийг устгана.
// ГУС-аас татагдсан эзэмшигчийг (holder_role='holder') устгахыг зөвшөөрөхгүй —
// тэр нь дараагийн sync дээр ямар ч байсан дахин үүснэ.
func (r *landAcqRepo) DeleteParcelHolder(ctx context.Context, parcelID, parcelHolderID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `
		DELETE FROM parcel_holder
		WHERE id = $1 AND parcel_id = $2 AND holder_role = $3
	`, parcelHolderID, parcelID, domain.ParcelHolderRoleRepresentative)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ReplaceParcelDetail — нэгж талбарын эрх зүйн дэлгэрэнгүйг дундын сервисийн
// мэдээллээр БҮРЭН шинэчилнэ: хуучин parcel_detail мөрүүдийг УСТГАЖ, шинийг
// үүсгэнэ (sync = тухайн үеийн бодит байдал). Ингэснээр өмнөх хүсэлт/гэрээний
// хоцрогдсон мөр, олон давхар мөр үлдэхгүй.
//
// ОНЦГОЙ: хэрэглэгч гараас оруулсан газрын үнэлгээний талбарууд
// (valuation_zone, base_price_per_ha, auction_coeff, auction_price) нь дундын
// сервисээс ирдэггүй тул шинэ мөрөнд ХУУЛАГДАНА — sync тэднийг устгахгүй.
func (r *landAcqRepo) ReplaceParcelDetail(ctx context.Context, parcelID, acquisitionID, holderID uuid.UUID, src *domain.SourceParcel) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Нэгж талбарын БҮРТГЭЛИЙН баганууд (old_parcel_id, valid_*, au2_code,
	// right_type, landuse*, area_m2, address_*, property_no) нь /parcel/info-д
	// БАС ирдэг тул ирсэн утгаар нэг бүрчлэн шинэчилнэ — багцаар бүртгэгдээгүй
	// (эсвэл хуучин хувилбараар зөвхөн дугаараар бүртгэгдсэн) нэгж талбарын
	// ДУТУУ мэдээлэл ингэж НӨХӨГДӨНӨ.
	//
	// ИРСЭН утга л дарж бичнэ (COALESCE/NULLIF): хоосон/0 ирвэл хадгалагдсан
	// утга үлдэнэ. Эс бөгөөс дутуу хариулт бүр бүртгэлийг хоослох болно.
	// au1_code/au3_code нь эх системээс ИРДЭГГҮЙ (геометрээс
	// FillParcelAUCodesFromGeometry нөхдөг) тул ЭНД огт хөндөхгүй.
	//
	// ГЕОМЕТР нь ЭНД бичигдэхгүй — түүнийг ApplyParcelGeometry давхцалтай
	// хамт тооцож бичнэ (SyncParcel-ийн дараагийн алхам).
	if _, err := tx.Exec(ctx, `
		UPDATE parcel SET
			old_parcel_id      = COALESCE(NULLIF($2, ''), old_parcel_id),
			valid_from         = COALESCE($3, valid_from),
			valid_till         = COALESCE($4, valid_till),
			au2_code           = COALESCE(NULLIF($5, ''), au2_code),
			right_type         = COALESCE(NULLIF($6, 0), right_type),
			landuse            = COALESCE(NULLIF($7, ''), landuse),
			landuse_name       = COALESCE(NULLIF($8, ''), landuse_name),
			area_m2            = COALESCE(NULLIF($9, 0), area_m2),
			address_khashaa    = COALESCE(NULLIF($10, ''), address_khashaa),
			address_streetname = COALESCE(NULLIF($11, ''), address_streetname),
			property_no        = COALESCE(NULLIF($12, ''), property_no),
			updated_at         = NOW(),
			updated_by         = 'system'
		WHERE id = $1
	`,
		parcelID,
		src.OldParcelID, src.ValidFrom, src.ValidTill, src.AU2Code,
		src.RightType, src.Landuse, src.LanduseName, src.AreaM2,
		src.AddressKhashaa, src.AddressStreet, src.PropertyNo,
	); err != nil {
		return err
	}

	// right_type-ыг ШИНЭЧИЛСНИЙ ДАРАА мөрөөс уншина: эх систем 0 буцаасан
	// (эсвэл багана хоосон ирсэн) үед хадгалагдсан утга хэрэглэгдэнэ.
	var rightType int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(right_type, 0) FROM parcel WHERE id = $1`, parcelID,
	).Scan(&rightType); err != nil {
		return err
	}

	// Хэрэглэгчийн оруулсан үнэлгээний талбаруудыг устгахаас хамгаалж хуулна.
	var (
		valuationZone  *string
		basePricePerHa *float64
		auctionCoeff   *float64
		auctionPrice   *float64
	)
	if err := tx.QueryRow(ctx, `
		SELECT valuation_zone, base_price_per_ha, auction_coeff, auction_price
		FROM parcel_detail
		WHERE parcel_id = $1
		  AND (valuation_zone IS NOT NULL OR base_price_per_ha IS NOT NULL
		       OR auction_coeff IS NOT NULL OR auction_price IS NOT NULL)
		ORDER BY updated_at DESC
		LIMIT 1
	`, parcelID).Scan(&valuationZone, &basePricePerHa, &auctionCoeff, &auctionPrice); err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// Хуучин дэлгэрэнгүйг БҮГДИЙГ устгана.
	if _, err := tx.Exec(ctx, "DELETE FROM parcel_detail WHERE parcel_id = $1", parcelID); err != nil {
		return err
	}

	var holderIDParam any
	if holderID != uuid.Nil {
		holderIDParam = holderID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO parcel_detail (
			id, parcel_id, acquisition_id, right_type, holder_id,
			app_no, app_timestamps, app_type, app_status,
			decision_no, decision_date,
			contract_no, contract_date, contract_begin, contract_end,
			contract_property_no, contract_status,
			certificate_no, certificate_date,
			record_no, record_date, record_certificate_no, record_status,
			valuation_zone, base_price_per_ha, auction_coeff, auction_price,
			created_by, updated_by
		) VALUES (
			uuid_generate_v4(), $1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10,
			$11, $12, $13, $14,
			$15, $16,
			$17, $18,
			$19, $20, $21, $22,
			$23, $24, $25, $26,
			'system', 'system'
		)
	`,
		parcelID, acquisitionID, rightType, holderIDParam,
		src.AppNo, src.AppTimestamp, src.AppType, src.AppStatus,
		src.DecisionNo, src.DecisionDate,
		src.ContractNo, src.ContractDate, src.ContractBegin, src.ContractEnd,
		src.ContractPropertyNo, src.ContractStatus,
		src.CertificateNo, src.CertificateDate,
		src.RecordNo, src.RecordDate, src.RecordCertificateNo, src.RecordStatus,
		valuationZone, basePricePerHa, auctionCoeff, auctionPrice,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *landAcqRepo) UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE parcel_detail SET
			valuation_zone    = $2,
			base_price_per_ha = $3,
			auction_coeff     = $4,
			auction_price     = $5,
			updated_by        = 'user'
		WHERE parcel_id = $1
	`, parcelID, zone, basePricePerHa, auctionCoeff, auctionPrice)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	// parcel_detail мөр байхгүй бол ЗӨВХӨН үнэлгээний талбартай мөр үүсгэнэ.
	// Эс бөгөөс ГУС-аас татсан суурь үнэ чимээгүй хаягдана (дэлгэрэнгүй нь
	// хараахан татагдаагүй нэгж талбар дээр тохиолддог).
	_, err = r.db.Exec(ctx, `
		INSERT INTO parcel_detail (
			parcel_id, acquisition_id, valuation_zone,
			base_price_per_ha, auction_coeff, auction_price,
			created_by, updated_by
		)
		SELECT p.id, p.acquisition_id, $2, $3, $4, $5, 'user', 'user'
		FROM parcel p WHERE p.id = $1
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
				nith_decree_number   = $17,
				group_list_number    = $18,
				updated_at           = NOW(),
				updated_by           = $19
		WHERE id = $1
	`,
		acq.ID, acq.PlanID, acq.PlanCode,
		acq.PlanGeomWKT, acq.PlanAreaM2,
		acq.GeometryWKT, acq.AreaM2,
		acq.StartDate, acq.EndDate,
		acq.AcquisitionName, acq.ImplementingOrg, acq.Reason, acq.ResponsibleOrg, acq.FundingSource,
		acq.GeneralCategoryID, acq.SubCategoryID,
		acq.NithDecreeNumber, acq.GroupListNumber,
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

// CheckAcquisitionDeletable — устгах хязгаарлалтад шаардлагатай тоонуудыг
// цуглуулна:
//
//   - нэгж талбарын төлөв бүрийн тоо (Чөлөөлсөн байгаа эсэхийг шалгахад),
//   - баталгаажсан захирамжтай холбоо: чөлөөлөлт дээр ШУУД (acquisition_id)
//     эсвэл НЭГЖ ТАЛБАРААР (decision_draft_parcel) холбогдсон эсэх.
//
// Захирамжийн холбоос нь чөлөөлөлт устгахад CASCADE-ээр (decision_draft_parcel)
// чимээгүй устдаг тул баталгаажсан захирамжийг эвдэхээс сэргийлнэ.
func (r *landAcqRepo) CheckAcquisitionDeletable(ctx context.Context, id uuid.UUID) (*domain.AcquisitionDeleteCheck, error) {
	res := &domain.AcquisitionDeleteCheck{ParcelStatusCounts: map[int]int{}}

	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(status, 0) AS status, count(*)::int
		FROM parcel
		WHERE acquisition_id = $1
		GROUP BY 1
	`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status, n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return nil, err
		}
		res.ParcelStatusCounts[status] = n
		res.ParcelCount += n
		if status == domain.ParcelStatusReleased {
			res.ReleasedParcels += n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	drafts, err := r.db.Query(ctx, `
		SELECT DISTINCT COALESCE(NULLIF(d.decree_number, ''), d.id::text)
		FROM decision_draft d
		WHERE d.status = $2
		  AND (
		        d.acquisition_id = $1
		     OR EXISTS (
		          SELECT 1
		          FROM decision_draft_parcel dp
		          JOIN parcel p ON p.id = dp.parcel_id
		          WHERE dp.decision_draft_id = d.id
		            AND dp.removed_at IS NULL
		            AND p.acquisition_id = $1
		        )
		  )
	`, id, domain.DecisionDraftStatusConfirmed)
	if err != nil {
		return nil, err
	}
	defer drafts.Close()
	for drafts.Next() {
		var number string
		if err := drafts.Scan(&number); err != nil {
			return nil, err
		}
		res.DecisionNumbers = append(res.DecisionNumbers, number)
		res.ConfirmedDecisions++
	}
	return res, drafts.Err()
}

// ListAcquisitionFileURLs — чөлөөлөлтөд хамаарах хадгалалт дээрх бүх файлын
// замыг цуглуулна. Хоосон/NULL зам орохгүй, давхардлыг DISTINCT-ээр хаяна.
func (r *landAcqRepo) ListAcquisitionFileURLs(ctx context.Context, id uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		-- чөлөөлөлтийн хавсралт (хоёр холбоос хүснэгт хоёулаа хэрэглэгддэг)
		SELECT DISTINCT doc.file_url
		FROM document doc
		JOIN acquisition_document ad ON ad.document_id = doc.id
		WHERE ad.acquisition_id = $1 AND COALESCE(doc.file_url, '') <> ''
		UNION
		SELECT DISTINCT doc.file_url
		FROM document doc
		JOIN land_acquisition_document lad ON lad.document_id = doc.id
		WHERE lad.land_acquisition_id = $1 AND COALESCE(doc.file_url, '') <> ''
		UNION
		-- нэгж талбарын хавсралт
		SELECT DISTINCT doc.file_url
		FROM document doc
		JOIN parcel_document pd ON pd.document_id = doc.id
		JOIN parcel p ON p.id = pd.parcel_id
		WHERE p.acquisition_id = $1 AND COALESCE(doc.file_url, '') <> ''
		UNION
		-- дроны ортофото
		SELECT DISTINCT di.file_url
		FROM drone_image di
		WHERE di.acquisition_id = $1 AND COALESCE(di.file_url, '') <> ''
		UNION
		-- хөрөнгийн зураг (PDF)
		SELECT DISTINCT a.photo_pdf_url
		FROM assets a
		WHERE a.acquisition_id = $1 AND COALESCE(a.photo_pdf_url, '') <> ''
		UNION
		-- үнэлгээний тайлан
		SELECT DISTINCT c.valuation_report_url
		FROM compensation c
		WHERE c.acquisition_id = $1 AND COALESCE(c.valuation_report_url, '') <> ''
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

// Delete — SOFT delete: мөрийг УСТГАХГҮЙ, deleted_at/deleted_by-г тэмдэглэнэ.
//
// Хатуу DELETE нь 23 хүснэгтийг ON DELETE CASCADE-ээр цэвэрлэж, буцаах
// боломжгүй болгодог (мөн decision_draft-ын нэгж талбарын холбоосыг чимээгүй
// устгадаг) тул хэрэглэхээ больсон. Уншилтууд `deleted_at IS NULL`-ээр шүүнэ.
//
// Аль хэдийн устгасан мөрийг дахин устгах гэвэл domain.ErrNotFound.
// Нэгж талбарууд нь МӨН тэмдэглэгдэнэ: parcel.parcel_id глобал unique тул
// тэмдэглэхгүй бол устгасан чөлөөлөлт дугаарыг эзэлсээр байж, шинэ чөлөөлөлт
// тэднийг бүртгэж чадахгүй болно (uq_parcel_parcel_id_active — хэсэгчилсэн
// unique индекс нь устгасан мөрийг хамардаггүй).
//
// Хоёулаа НЭГ гүйлгээнд — хагас устгагдсан төлөв үүсэхгүй.
func (r *landAcqRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE land_acquisition
		SET deleted_at = NOW(),
		    deleted_by = NULLIF($2, ''),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, deletedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if _, err = tx.Exec(ctx, `
		UPDATE parcel
		SET deleted_at = NOW()
		WHERE acquisition_id = $1 AND deleted_at IS NULL
	`, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
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

// SetParcelEstimatedValue — ТӨСӨӨЛЛИЙН үнэлгээг хадгална (Value=nil → арилгана).
//
// Нэгж талбар тухайн чөлөөлөлтөд харьяалагдахыг ЭНД шалгана (IDOR): дуудагч
// зөвхөн өөрийн чөлөөлөлтийн нэгж талбарыг засах ёстой.
//
// ӨӨРЧЛӨЛТ БҮР `parcel_estimated_value_history`-д мөн бичигдэнэ. Хоёр бичилт
// НЭГ tranзакцад явна: түүх бичигдээгүй байхад parcel шинэчлэгдвэл (эсвэл
// эсрэгээр) бүртгэл зөрж, дараа нь хянах боломжгүй болно.
func (r *landAcqRepo) SetParcelEstimatedValue(ctx context.Context, ch domain.ParcelEstimatedValueChange) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE parcel SET
			estimated_value    = $3,
			estimated_confidence_percent = $5,
			estimated_base_fee_per_m2 = $6,
			-- Утга арилгасан бол хэн/хэзээ гэдгийг ч цэвэрлэнэ.
			estimated_value_at = CASE WHEN $3::numeric IS NULL THEN NULL ELSE NOW() END,
			estimated_value_by = CASE WHEN $3::numeric IS NULL THEN '' ELSE $4 END,
			updated_at         = NOW(),
			updated_by         = $4
		WHERE id = $2 AND acquisition_id = $1 AND deleted_at IS NULL`,
		ch.AcquisitionID, ch.ParcelID, ch.Value, ch.UserID, ch.ConfidencePercent, ch.BaseFeePerM2)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO parcel_estimated_value_history (
			parcel_id, acquisition_id, value, confidence_percent, base_fee_per_m2,
			area_m2, attachment_url, attachment_name, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		ch.ParcelID, ch.AcquisitionID, ch.Value, ch.ConfidencePercent, ch.BaseFeePerM2,
		ch.AreaM2, ch.AttachmentURL, ch.AttachmentName, ch.UserID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListParcelEstimatedValueHistory — өөрчлөлтийн түүх (шинэ → хуучин).
func (r *landAcqRepo) ListParcelEstimatedValueHistory(
	ctx context.Context, acquisitionID, parcelID uuid.UUID,
) ([]*domain.ParcelEstimatedValueHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, acquisition_id, parcel_id, value, confidence_percent, base_fee_per_m2,
		       COALESCE(area_m2,0), COALESCE(attachment_url,''), COALESCE(attachment_name,''),
		       COALESCE(created_by,''), created_at
		FROM parcel_estimated_value_history
		WHERE parcel_id = $1 AND acquisition_id = $2
		ORDER BY created_at DESC, id DESC`,
		parcelID, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*domain.ParcelEstimatedValueHistory, 0)
	for rows.Next() {
		h := &domain.ParcelEstimatedValueHistory{}
		if err := rows.Scan(
			&h.ID, &h.AcquisitionID, &h.ParcelID, &h.Value, &h.ConfidencePercent, &h.BaseFeePerM2,
			&h.AreaM2, &h.AttachmentURL, &h.AttachmentName, &h.CreatedBy, &h.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) CountParcels(ctx context.Context, acquisitionID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM parcel WHERE acquisition_id = $1 AND deleted_at IS NULL`,
		acquisitionID,
	).Scan(&n)
	return n, err
}

// overlapMinAreaM2 — давхардлыг ХҮЛЭЭН ЗӨВШӨӨРӨХ доод хязгаар (м²).
//
// Хөрш нэгж талбарууд нийтлэг хилээрээ шүргэлцдэг ба хэмжилтийн бөөрөнхийлөлт
// (топологи бүрэн цэвэр биш) нь хэдхээн см²-ийн "давхцал" гаргадаг. Түүнийг
// давхардал гэж бүртгэвэл БҮХ хөрш талбар давхардалтай болж, шалгалт утгаа
// алдана. 1 м² нь хэмжилтийн шуугианаас дээгүүр, бодит давхардлаас доогуур.
const overlapMinAreaM2 = 1.0

// DetectParcelOverlaps — чөлөөлөлтийн нэгж талбаруудын БАЙРШЛЫН давхардлыг
// дахин тооцно.
//
// ЯАГААД ST_Overlaps БИШ: ST_Overlaps нь бүрэн АГУУЛАГДАХ тохиолдлыг (нэг
// талбар нөгөөгөө бүтэн дотроо агуулах) false гэж үнэлдэг. Гэтэл ГУС дээр яг
// тэр тохиолдол хамгийн эрсдэлтэй — нэг газарт хоёр дугаар. Иймд ST_Intersects
// + давхцах хэсгийн ТАЛБАЙгаар шалгана.
//
// Талбайг public.calculate_area_utm-ээр бодно (ГУС-тай ижил UTM арга) —
// ST_Area(::geography) ХЭРЭГЛЭХГҮЙ.
//
// ХОЁР чиглэлээр бичнэ (a→b, b→a) тул нэг талбарын давхардлыг parcel_id-гаар
// шүүхэд бүрэн жагсаалт гарна.
func (r *landAcqRepo) DetectParcelOverlaps(ctx context.Context, acquisitionID uuid.UUID) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM parcel_overlap WHERE acquisition_id = $1`, acquisitionID,
	); err != nil {
		return 0, err
	}

	if _, err := tx.Exec(ctx, `
		WITH pairs AS (
			SELECT a.id AS a_id, b.id AS b_id,
			       COALESCE(a.parcel_id,'') AS a_code, COALESCE(b.parcel_id,'') AS b_code,
			       public.calculate_area_utm(
			           ST_Intersection(ST_MakeValid(a.geometry), ST_MakeValid(b.geometry))
			       ) AS area_m2
			  FROM parcel a
			  JOIN parcel b
			    ON b.acquisition_id = a.acquisition_id
			   AND b.id > a.id                  -- хос тус бүрийг НЭГ л удаа
			   AND b.deleted_at IS NULL
			   AND b.geometry IS NOT NULL
			   AND ST_Intersects(a.geometry, b.geometry)
			 WHERE a.acquisition_id = $1
			   AND a.deleted_at IS NULL
			   AND a.geometry IS NOT NULL
		),
		real_pairs AS (
			SELECT * FROM pairs WHERE area_m2 IS NOT NULL AND area_m2 >= $2
		)
		INSERT INTO parcel_overlap (
			parcel_id, other_parcel_id, acquisition_id, other_parcel_code,
			overlap_area_m2, detected_at
		)
		SELECT a_id, b_id, $1, b_code, area_m2, NOW() FROM real_pairs
		UNION ALL
		SELECT b_id, a_id, $1, a_code, area_m2, NOW() FROM real_pairs
		ON CONFLICT (parcel_id, other_parcel_id) DO UPDATE SET
			overlap_area_m2   = EXCLUDED.overlap_area_m2,
			other_parcel_code = EXCLUDED.other_parcel_code,
			detected_at       = EXCLUDED.detected_at
	`, acquisitionID, overlapMinAreaM2); err != nil {
		return 0, err
	}

	var affected int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(DISTINCT parcel_id) FROM parcel_overlap WHERE acquisition_id = $1`,
		acquisitionID,
	).Scan(&affected); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *landAcqRepo) ListParcelOverlaps(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelOverlap, error) {
	rows, err := r.db.Query(ctx, `
		SELECT po.other_parcel_id, COALESCE(po.other_parcel_code,''),
		       COALESCE(po.overlap_area_m2, 0),
		       COALESCE(op.status, 0), COALESCE(ps.name,''),
		       po.detected_at
		  FROM parcel_overlap po
		  LEFT JOIN parcel op ON op.id = po.other_parcel_id
		  LEFT JOIN parcel_status ps ON ps.id = op.status
		 WHERE po.parcel_id = $1
		 ORDER BY po.overlap_area_m2 DESC`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ParcelOverlap, 0)
	for rows.Next() {
		var o domain.ParcelOverlap
		if err := rows.Scan(
			&o.OtherParcelID, &o.OtherParcelCode, &o.OverlapAreaM2,
			&o.OtherStatusID, &o.OtherStatusName, &o.DetectedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// DeleteParcelsOutsideBoundary — чөлөөлөх хилтэй ОГТ огтлолцохгүй нэгж
// талбаруудыг устгана.
//
// Зөвхөн "Шинэ" төлөвт (нэгж талбартай ажиллаж эхлээгүй байхад) автоматаар
// дуудагдана — тэр үед устгах нь ямар ч хэрэглэгчийн оруулсан өгөгдлийг
// хөндөхгүй. Геометргүй мөрийг ХАДГАЛНА: тэр нь хилээс гарсан гэдгийн
// БАТАЛГАА биш (ГУС-аас геометр ирээгүй байж болно).
func (r *landAcqRepo) DeleteParcelsOutsideBoundary(ctx context.Context, acquisitionID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		DELETE FROM parcel p
		 WHERE p.acquisition_id = $1
		   AND p.geometry IS NOT NULL
		   AND NOT EXISTS (
		       SELECT 1 FROM land_acquisition la
		        WHERE la.id = p.acquisition_id
		          AND la.geometry IS NOT NULL
		          AND ST_Intersects(ST_MakeValid(la.geometry), ST_MakeValid(p.geometry))
		   )
		RETURNING COALESCE(p.parcel_id,'')`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var removed []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		if code != "" {
			removed = append(removed, code)
		}
	}
	return removed, rows.Err()
}

func (r *landAcqRepo) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged string, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
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
	where := "la.deleted_at IS NULL" // soft delete: устгасан чөлөөлөлтийн нэгж талбар харагдахгүй
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
	if filter.StatusID != 0 {
		where += fmt.Sprintf(" AND p.status = $%d", i)
		args = append(args, filter.StatusID)
		i++
	}
	if filter.UnlinkedOnly {
		where += ` AND NOT EXISTS (
			SELECT 1
			  FROM decision_draft_parcel ddp
			 WHERE ddp.parcel_id = p.id
			   AND ddp.removed_at IS NULL
		)`
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
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where,
		orderClause(globalParcelSortColumns, filter.SortBy, filter.SortDir,
			"p.created_at DESC", "p.id"),
		i, i+1), append(args, pageSize, offset)...)
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

// buildReportParcelWhere — ListReportParcels болон GetReportSummary хоёулаа
// адилхан шүүлтээр ажилладаг тул WHERE өгүүлбэрийг нэг дор бүтээнэ.
func buildReportParcelWhere(filter domain.ReportParcelFilter) (string, []any) {
	args := []any{}
	where := "la.deleted_at IS NULL" // soft delete: тайланд устгасан чөлөөлөлт орохгүй
	i := 1

	if filter.ParcelID != "" {
		where += fmt.Sprintf(" AND p.parcel_id = $%d", i)
		args = append(args, filter.ParcelID)
		i++
	}
	// Төлөвлөгөөний нэгж талбарын дугаар — чөлөөлөлт нь plan_id-гаар
	// төлөвлөгөөтэй холбогддог тул дэд query-гээр шүүнэ (эдгээр query-ийн
	// FROM хэсэгт plan хүснэгт байхгүй).
	if filter.PlanParcelID != "" {
		where += fmt.Sprintf(" AND la.plan_id IN (SELECT pl.id FROM plan pl WHERE pl.parcel_id = $%d)", i)
		args = append(args, filter.PlanParcelID)
		i++
	}
	if filter.PlanCodeExact != "" {
		where += fmt.Sprintf(" AND la.plan_code = $%d", i)
		args = append(args, filter.PlanCodeExact)
		i++
	}
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
	}

	return where, args
}

// GetReportSummary — тайлангийн шүүлтэд тохирох БҮХ (хуудаслаагүй) мэдээллийн
// нэгтгэсэн тоо/дүнг нэг дуудлагаар буцаана — жагсаалтын дээд хэсгийн
// статистикийн картуудад зориулагдсан; бүх мөрийг клиент рүү дамжуулахгүй.
func (r *landAcqRepo) GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error) {
	where, args := buildReportParcelWhere(filter)

	cte := fmt.Sprintf(`
		WITH parcel_agg AS (
			SELECT
				p.id AS parcel_pk,
				COALESCE(p.status, 0) AS status,
				COALESCE(ps.name, '') AS status_name,
				COALESCE(p.acquisition_area_m2, 0) AS acquisition_area_m2,
				COALESCE(p.area_m2, 0) AS area_m2,
				la.id AS acquisition_id,
				(SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id) AS status_date,
				COALESCE(SUM(c.amount), 0) AS total_comp,
				COALESCE(SUM(CASE WHEN c.target_type = 'parcel' THEN c.amount ELSE 0 END), 0) AS land_comp,
				COALESCE(SUM(CASE WHEN c.target_type = 'asset' AND a.asset_type = 'real_state' THEN c.amount ELSE 0 END), 0) AS real_state_comp,
				COALESCE(SUM(CASE WHEN c.target_type = 'asset' AND a.asset_type = 'property' THEN c.amount ELSE 0 END), 0) AS property_comp
			FROM parcel p
			JOIN land_acquisition la ON la.id = p.acquisition_id
			LEFT JOIN parcel_status ps ON ps.id = p.status
			LEFT JOIN compensation c ON c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
				AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
			LEFT JOIN assets a ON a.id = c.asset_id
			WHERE %s
			GROUP BY p.id, p.status, ps.name, p.acquisition_area_m2, p.area_m2, la.id, p.parcel_id
		)
	`, where)

	summary := &domain.ReportSummary{}
	if err := r.db.QueryRow(ctx, cte+`
		SELECT
			COUNT(DISTINCT acquisition_id),
			COUNT(*),
			COALESCE(SUM(acquisition_area_m2), 0),
			COALESCE(SUM(area_m2), 0),
			COALESCE(SUM(total_comp), 0),
			COALESCE(SUM(land_comp), 0),
			COALESCE(SUM(real_state_comp), 0),
			COALESCE(SUM(property_comp), 0)
		FROM parcel_agg
	`, args...).Scan(
		&summary.AcquisitionCount, &summary.ParcelCount, &summary.TotalAreaM2, &summary.TotalParcelAreaM2, &summary.TotalCompensation,
		&summary.LandCompensation, &summary.RealStateCompensation, &summary.PropertyCompensation,
	); err != nil {
		return nil, err
	}
	summary.OtherCompensation = summary.TotalCompensation - summary.LandCompensation -
		summary.RealStateCompensation - summary.PropertyCompensation

	statusRows, err := r.db.Query(ctx, cte+`
		SELECT status, status_name, COUNT(*)
		FROM parcel_agg
		GROUP BY status, status_name
		ORDER BY status
	`, args...)
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var s domain.ReportStatusStat
		if err := statusRows.Scan(&s.Status, &s.StatusName, &s.Count); err != nil {
			return nil, err
		}
		summary.StatusBreakdown = append(summary.StatusBreakdown, s)
	}
	if err := statusRows.Err(); err != nil {
		return nil, err
	}

	yearRows, err := r.db.Query(ctx, cte+`
		SELECT EXTRACT(YEAR FROM status_date)::int AS year, COUNT(DISTINCT acquisition_id)
		FROM parcel_agg
		WHERE status_date IS NOT NULL
		GROUP BY year
		ORDER BY year DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer yearRows.Close()
	for yearRows.Next() {
		var y domain.ReportYearStat
		if err := yearRows.Scan(&y.Year, &y.Count); err != nil {
			return nil, err
		}
		summary.YearBreakdown = append(summary.YearBreakdown, y)
	}
	if err := yearRows.Err(); err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *landAcqRepo) ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error) {
	where, args := buildReportParcelWhere(filter)
	i := len(args) + 1

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
			COALESCE(p.db_changed, ''),
			COALESCE(p.changed_parcel_id, ''),
			COALESCE(p.status, 0),
			COALESCE(ps.name, ''),
			(SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id),
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
		LEFT JOIN parcel_status ps ON ps.id = p.status
		LEFT JOIN compensation c ON c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
			AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
		LEFT JOIN assets a ON a.id = c.asset_id
		WHERE %s
		GROUP BY
			p.id, p.parcel_id, p.area_m2, p.acquisition_area_m2,
			p.remaining_area_m2, p.right_type, p.db_changed, p.changed_parcel_id, p.status, ps.name,
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
			&row.Status,
			&row.StatusName,
			&row.StatusDate,
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
		SELECT d.id, d.name, d.file_type, d.file_url, d.size_bytes, COALESCE(d.note,''), d.uploaded_by, d.uploaded_at,
		       COALESCE(d.document_type_id, 0)
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
	// Хавсралтын төрөл document.document_type_id-д нэгдсэн (migration 000050) —
	// хуучин land_acquisition_document.file_type багана устсан.
	_, err := r.db.Exec(ctx, `
		WITH ins AS (
			INSERT INTO document (id, name, file_type, file_url, size_bytes, note, uploaded_by, uploaded_at, document_type_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($10::int, 0))
			RETURNING id
		)
		INSERT INTO land_acquisition_document (land_acquisition_id, document_id)
		SELECT $9, id FROM ins
	`, doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note, doc.UploadedBy, doc.UploadedAt, acquisitionID, doc.DocumentTypeID)
	return err
}

// GetDocumentType — хавсралтын төрлийг ID-аар авна (файл нэрлэхэд ашиглана).
func (r *landAcqRepo) GetDocumentType(ctx context.Context, id int) (*domain.DocumentType, error) {
	dt := &domain.DocumentType{}
	err := r.db.QueryRow(ctx,
		`SELECT id, type, name FROM document_type WHERE id=$1`, id).
		Scan(&dt.ID, &dt.Type, &dt.Name)
	if err != nil {
		return nil, err
	}
	return dt, nil
}

func (r *landAcqRepo) ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.name, d.file_type, d.file_url, d.size_bytes, COALESCE(d.note,''), d.uploaded_by, d.uploaded_at,
		       COALESCE(d.document_type_id, 0),
		       COALESCE(d.source_doc_id, ''), COALESCE(d.source_ftp_id, ''),
		       COALESCE(d.source_doc_code, '')
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
		if err := rows.Scan(&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.Note, &d.UploadedBy, &d.UploadedAt, &d.DocumentTypeID,
			&d.SourceDocID, &d.SourceFtpID, &d.SourceDocCode); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) AddParcelDocument(ctx context.Context, parcelID uuid.UUID, doc *domain.Document) error {
	// "Хөрөнгийн үнэлгээний тайлан" (valuation_report) төрлийн хавсралтыг ЗӨВХӨН
	// үнэлгээ баталгаажсан (selected_valuation_type тавигдсан) нэгж талбарт оруулна.
	ct, err := r.db.Exec(ctx, `
		WITH ins AS (
			INSERT INTO document (id, name, file_type, file_url, size_bytes, note, uploaded_by, uploaded_at, document_type_id)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, NULLIF($10::int, 0)
			WHERE NOT EXISTS (
				SELECT 1 FROM document_type dt
				WHERE dt.id = $10::int AND dt.type = 'valuation_report'
				  AND NOT EXISTS (
					SELECT 1 FROM parcel p
					WHERE p.id = $9 AND p.selected_valuation_type IS NOT NULL
				  )
			)
			RETURNING id
		)
		INSERT INTO parcel_document (parcel_id, document_id)
		SELECT $9, id FROM ins
	`, doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note, doc.UploadedBy, doc.UploadedAt, parcelID, doc.DocumentTypeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrReportBeforeApproval
	}
	return nil
}

// AddSourceParcelDocuments — эх системээс татсан хавсралтуудыг бүртгэнэ.
//
// ЗӨВХӨН НЭМНЭ: аль хэдийн татагдсан баримт (source_doc_id давхцсан) алгасагдаж,
// хадгалагдсан хавсралт хөндөгдөхгүй. Тиймээс дахин дахин дуудахад аюулгүй.
//
// Баримт нь эх системд НЭГ л удаа үүсдэг тул document мөр давхардуулахгүй —
// өөр нэгж талбар мөн ижил баримтад холбогдвол одоо байгаа мөрөнд холбогдоно.
func (r *landAcqRepo) AddSourceParcelDocuments(ctx context.Context, parcelID uuid.UUID, docs []*domain.Document) (int, error) {
	if len(docs) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit хийгдсэн бол no-op

	saved := 0
	for _, doc := range docs {
		if doc == nil || strings.TrimSpace(doc.SourceDocID) == "" {
			continue
		}

		// 1. Баримтын мөр — эх дугаараар давхцвал одоо байгааг нь авна.
		var documentID uuid.UUID
		err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO document (id, name, file_type, file_url, size_bytes, note,
				                      uploaded_by, uploaded_at, document_type_id,
				                      source_doc_id, source_ftp_id, source_doc_code)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::int, 0), $10, NULLIF($11, ''), NULLIF($12, ''))
				-- Аль хэдийн татагдсан баримтыг ИРСЭН УТГААР ШИНЭЧИЛНЭ: эх системд
				-- нэр солигдох, файлын хаяг өөрчлөгдөх, урьд нь байгаагүй талбар
				-- (source_doc_code) нэмэгдэх боломжтой.
				--
				-- Хоосон ирсэн утгаар хуучныг ДАРЖ БИЧИХГҮЙ (NULLIF+COALESCE) —
				-- эх систем зарим талбарыг буцаахгүй байх нь хадгалсныг гээх
				-- шалтгаан болох ёсгүй. document_type_id нь ГУС-аас ирдэггүй,
				-- гараар тавигддаг тул ХӨНДӨХГҮЙ.
				ON CONFLICT (source_doc_id) WHERE source_doc_id IS NOT NULL DO UPDATE SET
					name            = COALESCE(NULLIF(EXCLUDED.name, ''), document.name),
					file_type       = COALESCE(NULLIF(EXCLUDED.file_type, ''), document.file_type),
					file_url        = COALESCE(NULLIF(EXCLUDED.file_url, ''), document.file_url),
					source_ftp_id   = COALESCE(NULLIF(EXCLUDED.source_ftp_id, ''), document.source_ftp_id),
					source_doc_code = COALESCE(NULLIF(EXCLUDED.source_doc_code, ''), document.source_doc_code)
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM document WHERE source_doc_id = $10
			LIMIT 1
		`, doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note,
			doc.UploadedBy, doc.UploadedAt, doc.DocumentTypeID,
			doc.SourceDocID, doc.SourceFtpID, doc.SourceDocCode).Scan(&documentID)
		if err != nil {
			return saved, err
		}

		// 2. Нэгж талбартай холбоно. Аль хэдийн холбогдсон бол ШИНЭ биш.
		ct, err := tx.Exec(ctx, `
			INSERT INTO parcel_document (parcel_id, document_id)
			VALUES ($1, $2)
			ON CONFLICT (parcel_id, document_id) DO NOTHING
		`, parcelID, documentID)
		if err != nil {
			return saved, err
		}
		if ct.RowsAffected() > 0 {
			saved++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
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

// GetParcelDocument — баримтыг ТУХАЙН нэгж талбарт холбогдсон эсэхийг нь
// шалгаж уншина. Зөвхөн document_id-гаар уншвал өөр талбарын баримтыг
// татах боломжтой болно (IDOR).
func (r *landAcqRepo) GetParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) (*domain.Document, error) {
	d := &domain.Document{}
	err := r.db.QueryRow(ctx, `
		SELECT d.id, d.name, d.file_type, d.file_url, d.size_bytes, COALESCE(d.note,''),
		       d.uploaded_by, d.uploaded_at, COALESCE(d.document_type_id, 0),
		       COALESCE(d.source_doc_id, ''), COALESCE(d.source_ftp_id, '')
		FROM document d
		JOIN parcel_document pd ON pd.document_id = d.id
		WHERE d.id = $1 AND pd.parcel_id = $2
	`, documentID, parcelID).Scan(
		&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.Note,
		&d.UploadedBy, &d.UploadedAt, &d.DocumentTypeID,
		&d.SourceDocID, &d.SourceFtpID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return d, nil
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
func (r *landAcqRepo) GetAssetAcquisition(ctx context.Context, assetID uuid.UUID) (uuid.UUID, error) {
	var acqID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT acquisition_id FROM assets WHERE id=$1`, assetID).Scan(&acqID)
	if err == pgx.ErrNoRows {
		// Үнэлгээ цуцлагдахад хөрөнгийн мөр УСТДАГ. Зураг нь хадгалалтад
		// үлдэж, snapshot дотроос харагддаг тул мөр нь алга болсон ч
		// архивын snapshot-оос эзнийг нь олно — эс бөгөөс "Хүчингүй
		// болгосон үнэлгээ" дэх зураг бүр 403 болно.
		return r.snapshotOwnerAcquisition(ctx, "assets", assetID)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return acqID, nil
}

// GetCompensationAcquisition — олговрын харьяалагдах чөлөөлөлт (файлын эрх
// шалгахад: compensation-report/<comp_id>). Устсан мөрийн хувьд архивын
// snapshot-оос олно — үнэлгээ цуцлахад олговрын мөрүүд устдаг.
func (r *landAcqRepo) GetCompensationAcquisition(ctx context.Context, compID uuid.UUID) (uuid.UUID, error) {
	var acqID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT acquisition_id FROM compensation WHERE id=$1`, compID).Scan(&acqID)
	if err == pgx.ErrNoRows {
		return r.snapshotOwnerAcquisition(ctx, "compensations", compID)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return acqID, nil
}

// snapshotOwnerAcquisition — цуцлагдсан үнэлгээний snapshot дотроос өгсөн
// ID-тай хөрөнгө/олговрыг агуулсан мөрийн чөлөөлөлтийг олно. `column` нь
// ЗӨВХӨН дотоод дуудагчийн тогтмол ("assets" | "compensations").
func (r *landAcqRepo) snapshotOwnerAcquisition(ctx context.Context, column string, id uuid.UUID) (uuid.UUID, error) {
	if column != "assets" && column != "compensations" {
		return uuid.Nil, domain.ErrNotFound
	}
	var acqID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT acquisition_id
		FROM parcel_valuation_snapshot
		WHERE `+column+` @> jsonb_build_array(jsonb_build_object('id', $1::text))
		ORDER BY cancelled_at DESC
		LIMIT 1`, id.String()).Scan(&acqID)
	if err == pgx.ErrNoRows {
		return uuid.Nil, domain.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return acqID, nil
}

// allowedStreams хоосон бол хязгаарлалтгүй (дотоод ахлах/админ); эс бөгөөс
// зөвхөн дуудагчийн үнэлгээний урсгалын мөр устана.
func (r *landAcqRepo) DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID, allowedStreams []string) (string, error) {
	var streams any
	if len(allowedStreams) > 0 {
		streams = allowedStreams
	}
	var photoURL string
	err := r.db.QueryRow(ctx, `
		DELETE FROM assets
		WHERE id=$1 AND acquisition_id=$2
		  AND ($3::text[] IS NULL OR COALESCE(NULLIF(valuation_type,''),'asset') = ANY($3::text[]))
		RETURNING COALESCE(photo_pdf_url,'')`,
		id, acquisitionID, streams).Scan(&photoURL)
	if err == pgx.ErrNoRows {
		// Мөр өөрөө байгаа хэрнээ урсгал таараагүй бол 404 биш 403 буцаана.
		if len(allowedStreams) > 0 {
			var exists bool
			if e := r.db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND acquisition_id=$2)`,
				id, acquisitionID).Scan(&exists); e == nil && exists {
				return "", domain.ErrForbidden
			}
		}
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

	// 0. Хуучин мэдээлэл цэвэрлэх — ЗӨВХӨН тухайн урсгалынхыг (бусад урсгал хэвээр үлдэнэ).
	// Устгагдсан мөрүүдийн файлуудын (тайлан, зураг) URL-ийг цуглуулж буцаана —
	// service давхарга хадгалалтаас цэвэрлэж, орфон файл үлдээхгүй.
	if imp.Replace {
		collect := func(query string) (int, error) {
			rows, e := tx.Query(ctx, query, acquisitionID, imp.ParcelID, vType)
			if e != nil {
				return 0, e
			}
			defer rows.Close()
			n := 0
			for rows.Next() {
				var fileURL string
				if e := rows.Scan(&fileURL); e != nil {
					return 0, e
				}
				n++
				if fileURL != "" {
					res.OrphanedFileURLs = append(res.OrphanedFileURLs, fileURL)
				}
			}
			return n, rows.Err()
		}
		n, e := collect("DELETE FROM compensation WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3 RETURNING COALESCE(valuation_report_url,'')")
		if e != nil {
			return nil, e
		}
		res.DeletedComps = n
		n, e = collect("DELETE FROM assets WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3 RETURNING COALESCE(photo_pdf_url,'')")
		if e != nil {
			return nil, e
		}
		res.DeletedAssets = n
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

// ListFilterOptions — шүүлтүүрийн dropdown-ы хөнгөн жагсаалт.
//
// List()-ЭЭС ЗӨРӨӨТЭЙ: geometry, parcel COUNT(*), acquisition_category болон
// sdplatform.sd_organization join БАЙХГҮЙ — зөвхөн land_acquisition хүснэгтийн
// 4 багана. Иймд нэгж талбар/үнэлгээний өгөгдөл өсөхөд ч хугацаа нь өсөхгүй.
//
// ШҮҮЛТГҮЙ (status/роль): жагсаалтын шүүлтүүрийг БҮХ нэвтэрсэн хэрэглэгч
// харна. Шүүлтүүрийн сонголт хязгаарлагдвал хэрэглэгч өөрт харагдах ёстой
// мөрүүдийг шүүж чадахгүй болдог. Жагсаалтын БОДИТ өгөгдлийн эрх нь
// List()/ListReportParcels() дээрээ хэвээр шалгагдана — энд зөвхөн нэр, план
// код л буцна.
func (r *landAcqRepo) ListFilterOptions(ctx context.Context) ([]*domain.LandAcquisitionOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT la.id, COALESCE(la.acquisition_name,''), COALESCE(la.plan_code,''),
		       COALESCE(la.plan_name,'')
		FROM land_acquisition la
		WHERE la.deleted_at IS NULL
		ORDER BY la.acquisition_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*domain.LandAcquisitionOption, 0, 64)
	for rows.Next() {
		o := &domain.LandAcquisitionOption{}
		if err := rows.Scan(&o.ID, &o.AcquisitionName, &o.PlanCode, &o.PlanName); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

// districtsAU1Code — шүүлтүүрийн дүүргийн жагсаалтад харуулах аймаг/нийслэлийн
// код. "011" = Улаанбаатар.
const districtsAU1Code = "011"

// ListAcquisitionDistricts — шүүлтүүрийн СУМ/ДҮҮРГИЙН сонголт.
//
// Эх сурвалж нь au2 ЛАВЛАХ хүснэгт: аймаг/нийслэлийн код нь
// districtsAU1Code ("011" = Улаанбаатар) бүх дүүрэг гарна. Чөлөөлөлт
// бүртгэгдсэн эсэхээс хамаарахгүй.
//
// ListFilterOptions-той ижил шалтгаанаар шүүлтгүй: сонголтыг бүх нэвтэрсэн
// хэрэглэгч харна, бодит өгөгдлийн эрх нь List()/GetDashboard() дээр шалгагдана.
func (r *landAcqRepo) ListAcquisitionDistricts(ctx context.Context) ([]domain.AU2Option, error) {
	rows, err := r.db.Query(ctx, `
		SELECT code, COALESCE(name, '')
		FROM au2
		WHERE au1_code = $1
		ORDER BY name, code
	`, districtsAU1Code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.AU2Option, 0, 32)
	for rows.Next() {
		var o domain.AU2Option
		if err := rows.Scan(&o.Code, &o.Name); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) Suggest(ctx context.Context, q string, status int) ([]*domain.LandAcquisition, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(acquisition_name,''), COALESCE(plan_code,'')
		FROM land_acquisition
		WHERE deleted_at IS NULL
		  AND (acquisition_name ILIKE $1 OR plan_code ILIKE $1)
		  AND ($2 = 0 OR status = $2)
		ORDER BY acquisition_name
		LIMIT 20
	`, "%"+q+"%", status)
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
	where := "la.deleted_at IS NULL" // soft delete: дашбордод устгасан чөлөөлөлт орохгүй
	i := 1
	profParam := 0

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
	// Сум/дүүрэг — жагсаалттай ИЖИЛ логик: чөлөөлөлтийн хилээр тодорхойлогдсон
	// нэгж (land_acquisition_au) ЭСВЭЛ нэгж талбаруудын au2 кодоор.
	if filter.AU2Code != "" {
		where += fmt.Sprintf(` AND (
			EXISTS (
				SELECT 1 FROM land_acquisition_au
				WHERE land_acquisition_id = la.id AND au2_code = $%d
			)
			OR EXISTS (
				SELECT 1 FROM parcel p_au
				WHERE p_au.acquisition_id = la.id AND p_au.deleted_at IS NULL
				  AND p_au.au2_code = $%d
			)
		)`, i, i)
		args = append(args, filter.AU2Code)
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
		where += fmt.Sprintf(` AND la.id IN (
			SELECT DISTINCT p.acquisition_id FROM parcel p
			WHERE EXTRACT(YEAR FROM (
				SELECT MAX(psh.status_date) FROM parcel_status_history psh WHERE psh.parcel_id = p.parcel_id
			)) IN (%s)
		)`, strings.Join(yearPlaceholders, ","))
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
	if filter.AssignedUserID != "" {
		where += fmt.Sprintf(` AND la.id IN (
			SELECT acquisition_id FROM land_acquisition_assignee WHERE user_id = $%d::int4
		)`, i)
		args = append(args, filter.AssignedUserID)
		i++
	}
	if filter.ProfessionalOrgID != "" {
		profParam = i
		where += fmt.Sprintf(` AND (
			la.professional_org_id = $%d::int4
			OR EXISTS (
				SELECT 1 FROM parcel p_org
				WHERE p_org.acquisition_id = la.id AND p_org.independent_org_id = $%d::int4
			)
		)`, i, i)
		args = append(args, filter.ProfessionalOrgID)
		i++
	}
	parcelScopeCond := ""
	acqParcelScopeCond := ""
	if filter.FilterByParcelStatus {
		parcelScopeCond += fmt.Sprintf(" AND p.status = %d", filter.ParcelStatusID)
		acqParcelScopeCond += fmt.Sprintf(" AND p_scope.status = %d", filter.ParcelStatusID)
	}
	if profParam != 0 {
		parcelScopeCond += fmt.Sprintf(` AND (
			EXISTS (
				SELECT 1 FROM land_acquisition la_scope
				WHERE la_scope.id = p.acquisition_id AND la_scope.deleted_at IS NULL
				  AND la_scope.professional_org_id = $%d::int4
			)
			OR p.independent_org_id = $%d::int4
		)`, profParam, profParam)
		acqParcelScopeCond += fmt.Sprintf(` AND (
			la.professional_org_id = $%d::int4 OR p_scope.independent_org_id = $%d::int4
		)`, profParam, profParam)
	}
	if acqParcelScopeCond != "" {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM parcel p_scope
			WHERE p_scope.acquisition_id = la.id%s
		)`, acqParcelScopeCond)
	}
	_ = i // suppress unused warning after last condition

	acqSubquery := fmt.Sprintf("SELECT la.id FROM land_acquisition la WHERE %s", where)

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
	type valuationRes struct {
		statuses []domain.ValuationStatusStat
		types    []domain.ValuationTypeStat
		err      error
	}

	acqCh := make(chan acqRes, 1)
	statusCh := make(chan statusRes, 1)
	statsCh := make(chan statsRes, 1)
	breakdownCh := make(chan breakdownRes, 1)
	timelineCh := make(chan timelineRes, 1)
	compCh := make(chan compRes, 1)
	valuationCh := make(chan valuationRes, 1)

	// 1. Шүүлтэд тохирох чөлөөлөлтүүд
	go func() {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT la.id, COALESCE(la.acquisition_name, ''), la.plan_code, la.status,
			       COALESCE(la.area_m2, 0), COALESCE(la.decree_number, ''), COUNT(p.id),
			       -- ЯВЦ: жагсаалттай ИЖИЛ шалгуур (эцсийн 3 төлөв).
			       COUNT(p.id) FILTER (WHERE COALESCE(p.status,0) IN (3,4,5))
			FROM land_acquisition la
			LEFT JOIN parcel p ON p.acquisition_id = la.id%s
			WHERE %s
			GROUP BY la.id, la.acquisition_name, la.plan_code, la.status, la.area_m2, la.decree_number, la.created_at
			-- ЭРЭМБЭ: явц БАГААС ИХ рүү — анхаарал хэрэгтэй (гүйцэтгэл султай)
			-- чөлөөлөлтүүд дээд талд гарна. Нэгж талбаргүй бол тэг гэж үзнэ.
			-- Ижил хувьтай үед шинэ нь дээр (created_at DESC).
			ORDER BY
			  CASE WHEN COUNT(p.id) = 0 THEN 0
			       ELSE COUNT(p.id) FILTER (WHERE COALESCE(p.status,0) IN (3,4,5))::numeric
			            / COUNT(p.id) END ASC,
			  la.created_at DESC
			LIMIT 1000
		`, parcelScopeCond, where), args...)
		if err != nil {
			acqCh <- acqRes{nil, err}
			return
		}
		defer rows.Close()
		var list []*domain.LandAcquisition
		for rows.Next() {
			a := &domain.LandAcquisition{}
			if err := rows.Scan(
				&a.ID,
				&a.AcquisitionName,
				&a.PlanCode,
				&a.Status,
				&a.AreaM2,
				&a.DecreeNumber,
				&a.ParcelCount,
				&a.FinalParcelCount,
			); err != nil {
				acqCh <- acqRes{nil, err}
				return
			}
			list = append(list, a)
		}
		acqCh <- acqRes{list, rows.Err()}
	}()

	// 2. Нэгж талбарын төлөвийн lookup
	go func() {
		list, err := r.ListParcelStatuses(ctx)
		statusCh <- statusRes{list, err}
	}()

	// 3. Нэгж талбарын нийт тоо, ЧӨЛӨӨЛӨГДСӨН тоо ба талбай.
	//
	// Шалгуур нь нэгж талбарын ТӨЛӨВ = "Чөлөөлсөн" (ParcelStatusReleased).
	// Өмнө нь `compensation_paid` тугаар тоолдог байсан нь ӨӨР ойлголт:
	// тэр нь олговор ОЛГОГДСОН эсэхийг заадаг ба хуучин системээс импортлох /
	// төлбөрийн үйлдлээр тавигддаг тул "Чөлөөлсөн" төлөвтэй талбар огт
	// байхгүй атал энэ хоёр карт 65% харуулж, дэргэдэх "Чөлөөлөлтийн
	// гүйцэтгэл"-ээс (эцсийн төлөвөөр бодогддог) зөрдөг байв.
	go func() {
		var total, freed int64
		var freedArea float64
		err := r.db.QueryRow(ctx, fmt.Sprintf(`
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE p.status = %d),
				COALESCE(SUM(p.acquisition_area_m2) FILTER (WHERE p.status = %d), 0)
			FROM parcel p
			WHERE p.acquisition_id IN (%s)%s%s
		`, domain.ParcelStatusReleased, domain.ParcelStatusReleased,
			acqSubquery, parcelYearCond, parcelScopeCond), args...).Scan(&total, &freed, &freedArea)
		statsCh <- statsRes{total, freed, freedArea, err}
	}()

	// 4. Төлөвийн задаргаа (нэгж талбар бүрийн тоо, талбай)
	go func() {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT p.status, COALESCE(ps.name, CAST(p.status AS text)),
			       COUNT(*), COALESCE(SUM(p.acquisition_area_m2), 0)
			FROM parcel p
			LEFT JOIN parcel_status ps ON ps.id = p.status
			WHERE p.acquisition_id IN (%s)%s%s
			GROUP BY p.status, ps.name
			ORDER BY p.status
		`, acqSubquery, parcelYearCond, parcelScopeCond), args...)
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
			  AND la.start_date IS NOT NULL%s%s
			GROUP BY la.start_date
			ORDER BY la.start_date
		`, acqSubquery, parcelYearCond, parcelScopeCond), args...)
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
					WHERE p.acquisition_id IN (%s)%s%s
				) x
			`, acqSubquery, parcelYearCond, parcelScopeCond)
		} else {
			compSQL = fmt.Sprintf(`
				SELECT COALESCE(SUM(c.amount), 0)
				FROM compensation c
				JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
				WHERE c.acquisition_id IN (%s)
				  AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
				  %s
			`, acqSubquery, parcelScopeCond)
		}
		err := r.db.QueryRow(ctx, compSQL, args...).Scan(&total)
		compCh <- compRes{total, err}
	}()

	// 7. Үнэлгээний илгээх/баталгаажуулах төлөв болон урсгалын нэгтгэл
	go func() {
		statusRows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT s.status, COUNT(*)
			FROM parcel_valuation_submission s
			JOIN parcel p ON p.acquisition_id = s.acquisition_id AND p.parcel_id = s.parcel_id
			WHERE p.acquisition_id IN (%s)%s%s
			GROUP BY s.status
			ORDER BY s.status
		`, acqSubquery, parcelYearCond, parcelScopeCond), args...)
		if err != nil {
			valuationCh <- valuationRes{err: err}
			return
		}
		var statuses []domain.ValuationStatusStat
		for statusRows.Next() {
			var s domain.ValuationStatusStat
			if err := statusRows.Scan(&s.Status, &s.Count); err != nil {
				statusRows.Close()
				valuationCh <- valuationRes{err: err}
				return
			}
			statuses = append(statuses, s)
		}
		if err := statusRows.Err(); err != nil {
			statusRows.Close()
			valuationCh <- valuationRes{err: err}
			return
		}
		statusRows.Close()

		typeRows, err := r.db.Query(ctx, fmt.Sprintf(`
			SELECT s.valuation_type,
			       COUNT(*),
			       COALESCE(SUM((
			           SELECT COALESCE(SUM(c.amount), 0)
			           FROM compensation c
			           WHERE c.acquisition_id = s.acquisition_id
			             AND c.parcel_id = s.parcel_id
			             AND c.valuation_type = s.valuation_type
			       )), 0)
			FROM parcel_valuation_submission s
			JOIN parcel p ON p.acquisition_id = s.acquisition_id AND p.parcel_id = s.parcel_id
			WHERE p.acquisition_id IN (%s)%s%s
			GROUP BY s.valuation_type
			ORDER BY s.valuation_type
		`, acqSubquery, parcelYearCond, parcelScopeCond), args...)
		if err != nil {
			valuationCh <- valuationRes{err: err}
			return
		}
		defer typeRows.Close()
		var types []domain.ValuationTypeStat
		for typeRows.Next() {
			var t domain.ValuationTypeStat
			if err := typeRows.Scan(&t.ValuationType, &t.Count, &t.Amount); err != nil {
				valuationCh <- valuationRes{err: err}
				return
			}
			types = append(types, t)
		}
		valuationCh <- valuationRes{statuses: statuses, types: types, err: typeRows.Err()}
	}()

	// 8. Шүүлтүүрт тохирох захиргааны нэгжийн код жагсаалт (au1/au2/au3 давхаргад ашиглана)
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
	valR := <-valuationCh
	auCodesR := <-auCodesCh

	for _, e := range []error{acqR.err, statusR.err, statsR.err, bkR.err, tlR.err, compR.err, valR.err, auCodesR.err} {
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
		ValuationStatuses: valR.statuses,
		ValuationTypes:    valR.types,
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

// IsAssignedToParcel — хэрэглэгч parcel-ийн харгалзах чөлөөлөлтөд хуваарилагдсан эсэх.
func (r *landAcqRepo) IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID domain.ID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM land_acquisition_assignee a
			JOIN parcel p ON p.acquisition_id = a.acquisition_id
			WHERE p.id = $1 AND a.user_id = $2
		)
	`, parcelID, userID).Scan(&exists)
	return exists, err
}

func (r *landAcqRepo) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID domain.ID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM land_acquisition_assignee
			WHERE acquisition_id = $1 AND user_id = $2
		)
	`, acquisitionID, userID).Scan(&exists)
	return exists, err
}

// IsProfMemberOfAcquisition — өгсөн БАЙГУУЛЛАГА нь тухайн чөлөөлөлтийн үндсэн
// гүйцэтгэгч (professional_org_id) эсвэл аль нэгэн нэгж талбарын хараат бус
// үнэлгээчин (independent_org_id) мөн эсэх.
func (r *landAcqRepo) IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, orgID domain.ID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM land_acquisition WHERE id = $1 AND professional_org_id = $2 AND deleted_at IS NULL
			UNION
			SELECT 1 FROM parcel WHERE acquisition_id = $1 AND independent_org_id = $2
		)
	`, acquisitionID, orgID).Scan(&exists)
	return exists, err
}

// IsProfMemberOfParcel — өгсөн БАЙГУУЛЛАГА нь parcel-ийн чөлөөлөлтийн үндсэн
// гүйцэтгэгч эсвэл тухайн parcel-ийн хараат бус үнэлгээчин мөн эсэх.
func (r *landAcqRepo) IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, orgID domain.ID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM parcel p
			JOIN land_acquisition la ON la.id = p.acquisition_id
			WHERE p.id = $1
			  AND (la.professional_org_id = $2 OR p.independent_org_id = $2)
		)
	`, parcelID, orgID).Scan(&exists)
	return exists, err
}

// OrgUserIDs returns app-owned login IDs, never HR user IDs from authdb.
func (r *landAcqRepo) OrgUserIDs(ctx context.Context, orgID domain.ID) ([]domain.ID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id
		FROM valuation_org_employee e
		JOIN valuation_org_profile o ON o.id = e.org_id
		WHERE e.org_id = $1 AND e.is_active AND o.is_active
		  AND COALESCE(e.password, '') <> ''`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ID
	for rows.Next() {
		var id domain.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
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

func (r *landAcqRepo) UpdateParcelStatus(ctx context.Context, ch domain.ParcelStatusChange) error {
	parcelID, acquisitionID, statusID := ch.ParcelID, ch.AcquisitionID, ch.StatusID
	// "Чөлөөлсөн" болгохын өмнө үнэлгээний тайлан байгаа эсэхийг шалгана
	var statusName string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(name,'') FROM parcel_status WHERE id=$1`, statusID).Scan(&statusName)
	if statusName == "Чөлөөлсөн" {
		var parcelCode string
		if err := r.db.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&parcelCode); err != nil {
			return domain.ErrNotFound
		}
		var hasReport bool
		// Сонгосон урсгалын зөвшөөрөгдсөн олговортой БӨГӨӨД үнэлгээний тайлан
		// хавсаргагдсан байхыг шаардана. Тайлан нь "Хөрөнгийн үнэлгээний тайлан"
		// (valuation_report) төрлийн parcel document (шинэ флоу) эсвэл олговрын
		// valuation_report_url (хуучин өгөгдөл) байж болно.
		_ = r.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM compensation c
				JOIN parcel p ON p.parcel_id = c.parcel_id AND p.acquisition_id = c.acquisition_id
				WHERE c.acquisition_id=$1 AND c.parcel_id=$2
				AND c.status='approved'
				AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
			) AND (
				EXISTS(
					SELECT 1 FROM parcel_document pd
					JOIN document d ON d.id = pd.document_id
					JOIN document_type dt ON dt.id = d.document_type_id
					WHERE pd.parcel_id=$3 AND dt.type='valuation_report'
				)
				OR EXISTS(
					SELECT 1 FROM compensation c
					JOIN parcel p ON p.parcel_id = c.parcel_id AND p.acquisition_id = c.acquisition_id
					WHERE c.acquisition_id=$1 AND c.parcel_id=$2
					AND c.status='approved' AND c.valuation_report_url<>''
					AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
				)
			)`, acquisitionID, parcelCode, parcelID).Scan(&hasReport)
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
	return r.insertStatusHistory(ctx, ch)
}

// InsertParcelStatusHistory — ХАВСРАЛТГҮЙ бүртгэл (ГУС-аас татах, системийн
// шилжилт г.м.). Хавсралттай шилжилт нь UpdateParcelStatus-аар явна.
func (r *landAcqRepo) InsertParcelStatusHistory(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy, reason string) error {
	return r.insertStatusHistory(ctx, domain.ParcelStatusChange{
		ParcelID: parcelID, AcquisitionID: acquisitionID, StatusID: statusID,
		CreatedBy: createdBy, Reason: reason,
	})
}

func (r *landAcqRepo) insertStatusHistory(ctx context.Context, ch domain.ParcelStatusChange) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO parcel_status_history (
			parcel_id, acquisition_id, status_id, created_by, reason,
			attachment_url, attachment_name
		)
		SELECT parcel_id, acquisition_id, $3, $4, NULLIF($5, ''), $6, $7
		FROM parcel
		WHERE id = $1 AND acquisition_id = $2`,
		ch.ParcelID, ch.AcquisitionID, ch.StatusID, ch.CreatedBy, ch.Reason,
		ch.AttachmentURL, ch.AttachmentName,
	)
	return err
}

func (r *landAcqRepo) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT h.id, h.parcel_id, h.acquisition_id, h.status_id, ps.name, h.status_date,
		       COALESCE(h.created_by, ''),
		       COALESCE(h.reason, ''), COALESCE(h.attachment_url, ''), COALESCE(h.attachment_name, '')
		FROM parcel_status_history h
		JOIN parcel_status ps ON ps.id = h.status_id
		JOIN parcel p ON p.parcel_id = h.parcel_id AND p.acquisition_id = h.acquisition_id
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
		if err := rows.Scan(&h.ID, &h.ParcelID, &h.AcquisitionID, &h.StatusID, &h.StatusName, &h.StatusDate, &h.CreatedBy, &h.Reason,
			&h.AttachmentURL, &h.AttachmentName); err != nil {
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
		&s.LastNote, &s.AttachmentURL, &s.AttachmentName, &s.CreatedAt, &s.UpdatedAt)
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
		SELECT s.id, s.acquisition_id, s.parcel_id, s.valuation_type, s.status,
		       COALESCE(s.submitted_by, ''),
		       s.submitted_at,
		       COALESCE(s.reviewed_by, ''),
		       s.reviewed_at, s.last_note,
		       COALESCE(s.attachment_url,''), COALESCE(s.attachment_name,''),
		       s.created_at, s.updated_at
		FROM parcel_valuation_submission s
		WHERE s.acquisition_id=$1 AND s.parcel_id=$2 AND s.valuation_type=$3`, acquisitionID, code, vType)
	return scanValuationSubmission(row, acquisitionID, code, vType)
}

// TransitionValuationSubmission — төлөв шилжүүлэх (submit|approve|return|cancel), түүх хадгална.
// approve үед тухайн урсгал нэгж талбарын "үндсэн" (selected_valuation_type) болно.
func (r *landAcqRepo) TransitionValuationSubmission(ctx context.Context, in domain.ValuationTransition) (*domain.ValuationSubmission, error) {
	acquisitionID, parcelID := in.AcquisitionID, in.ParcelID
	action, note, userID := in.Action, in.Note, in.UserID
	code, err := r.parcelCodeByID(ctx, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	vType := normalizeValuationType(in.ValuationType)

	// ХАВСРАЛТ нь буцаалт болон цуцлалтад хамаарна. Бусад үйлдэлд ирсэн ч
	// хадгалахгүй — эс бөгөөс "илгээх" дээрх файл буцаалт/цуцлалтын хавсралт
	// болж харагдана.
	attachURL, attachName := "", ""
	if action == "return" || action == "cancel" {
		attachURL, attachName = in.AttachmentURL, in.AttachmentName
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Одоогийн төлөв (мөр байхгүй бол Ноорог)
	// Hold the acquisition status stable and serialize all streams of this parcel.
	var acquisitionStatus int
	if err = tx.QueryRow(ctx, `SELECT status FROM land_acquisition WHERE id=$1 FOR SHARE`, acquisitionID).Scan(&acquisitionStatus); err != nil {
		return nil, err
	}
	if acquisitionStatus == domain.LandAcquisitionStatusVerified {
		return nil, domain.ErrAcquisitionLocked
	}
	var lockedParcelID uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM parcel WHERE id=$1 AND acquisition_id=$2 FOR UPDATE`, parcelID, acquisitionID).Scan(&lockedParcelID); err != nil {
		return nil, err
	}

	from := domain.ValuationStatusDraft
	err = tx.QueryRow(ctx,
		`SELECT status FROM parcel_valuation_submission WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`,
		acquisitionID, code, vType).Scan(&from)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

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
	case "cancel":
		if from != domain.ValuationStatusApproved {
			return nil, domain.ErrInvalidTransition
		}
		to = domain.ValuationStatusDraft
	default:
		return nil, domain.ErrInvalidInput
	}

	// Илгээхийн өмнө тухайн урсгалын БҮХ "Үл хөдлөх" төрлийн хөрөнгө зурагтай эсэхийг шалгана.
	if action == "submit" {
		var missingPhotos int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM assets
			WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3
			  AND asset_type='real_state' AND COALESCE(photo_pdf_url,'')=''`,
			acquisitionID, code, vType).Scan(&missingPhotos); err != nil {
			return nil, err
		}
		if missingPhotos > 0 {
			return nil, domain.ErrAssetPhotoRequired
		}
	}

	// Upsert төлөв. submit → submitted_by/at, approve/return → reviewed_by/at.
	// attachment_url/name нь ҮЙЛДЭЛ БҮРД дарж бичигдэнэ (буцаалт биш үед хоосон
	// болно) — `last_note`-той ижил зарчим: карт нь СҮҮЛИЙН үйлдлийг харуулна.
	if action == "submit" {
		_, err = tx.Exec(ctx, `
			INSERT INTO parcel_valuation_submission (acquisition_id, parcel_id, valuation_type, status, submitted_by, submitted_at, last_note, attachment_url, attachment_name)
			VALUES ($1,$2,$6,$3,$4,NOW(),$5,$7,$8)
			ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE
			SET status=$3, submitted_by=$4, submitted_at=NOW(), last_note=$5,
			    attachment_url=$7, attachment_name=$8, updated_at=NOW()`,
			acquisitionID, code, to, userID, note, vType, attachURL, attachName)
	} else if action != "cancel" {
		_, err = tx.Exec(ctx, `
			INSERT INTO parcel_valuation_submission (acquisition_id, parcel_id, valuation_type, status, reviewed_by, reviewed_at, last_note, attachment_url, attachment_name)
			VALUES ($1,$2,$6,$3,$4,NOW(),$5,$7,$8)
			ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE
			SET status=$3, reviewed_by=$4, reviewed_at=NOW(), last_note=$5,
			    attachment_url=$7, attachment_name=$8, updated_at=NOW()`,
			acquisitionID, code, to, userID, note, vType, attachURL, attachName)
	}
	if err != nil {
		return nil, err
	}

	if action == "cancel" {
		if attachURL == "" {
			return nil, domain.ErrValuationCancelAttachmentRequired
		}
		// ҮНЭЛГЭЭНИЙ ТАЙЛАН нь цуцлахад ЗААВАЛ байх ёстой ба түүхэнд үлдэнэ.
		// Тайлан ХОЁР флоугоор бүртгэгдсэн байж болно:
		//   1) "Хөрөнгийн үнэлгээний тайлан" төрлийн parcel_document (шинэ),
		//   2) баталгаажсан олговрын valuation_report_url (хуучин өгөгдөл).
		// Аль нь ч байсан ХОЛБООСЫГ нь snapshot дээр хуулж авна — цуцлалтын
		// дараа олговрын мөр устаж, хавсралт нь салгагдах тул тайлан зөвхөн
		// түүхээс татагдана.
		var reportDocID uuid.UUID
		var reportURL, reportName string
		if err = tx.QueryRow(ctx, `
			SELECT d.id, COALESCE(d.file_url,''), COALESCE(d.name,'')
			FROM parcel_document pd
			JOIN document d ON d.id = pd.document_id
			JOIN document_type dt ON dt.id = d.document_type_id
			WHERE pd.parcel_id=$1 AND dt.type='valuation_report'
			ORDER BY d.uploaded_at DESC
			LIMIT 1`, parcelID,
		).Scan(&reportDocID, &reportURL, &reportName); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if reportURL == "" {
			reportDocID = uuid.Nil
			if err = tx.QueryRow(ctx, `
				SELECT COALESCE(c.valuation_report_url,''), COALESCE(c.valuation_report_name,'')
				FROM compensation c
				WHERE c.acquisition_id=$1 AND c.parcel_id=$2
				  AND c.valuation_type=$3 AND c.status='approved'
				  AND COALESCE(c.valuation_report_url,'') <> ''
				ORDER BY c.updated_at DESC
				LIMIT 1`, acquisitionID, code, vType,
			).Scan(&reportURL, &reportName); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
		if reportURL == "" {
			return nil, domain.ErrValuationReportRequired
		}
		if reportName == "" {
			reportName = "Үнэлгээний тайлан"
		}
		var snapshotID uuid.UUID
		if err = tx.QueryRow(ctx, `
			INSERT INTO parcel_valuation_snapshot (
				acquisition_id, parcel_id, valuation_type, submission_id, note, cancelled_by,
				submitted_by, submitted_at, reviewed_by, reviewed_at,
				report_url, report_name,
				land_valuation, assets, compensations, total_amount
			)
			SELECT s.acquisition_id, s.parcel_id, s.valuation_type, s.id, $4, $5,
			       COALESCE(s.submitted_by,''), s.submitted_at,
			       COALESCE(s.reviewed_by,''), s.reviewed_at,
			       $7, $8,
			       COALESCE((
			         SELECT to_jsonb(lv)
			         FROM land_valuation lv
			         WHERE lv.acquisition_id=s.acquisition_id AND lv.parcel_id=s.parcel_id
			           AND lv.valuation_type=s.valuation_type
			       ), '{}'::jsonb),
			       COALESCE((
			         SELECT jsonb_agg(asset_json ORDER BY created_at DESC)
			         FROM (
			           SELECT a.created_at,
			                  to_jsonb(a)
			                  || jsonb_build_object(
			                    'specs', COALESCE((
			                      SELECT jsonb_agg(to_jsonb(sp) ORDER BY sp.id)
			                      FROM asset_spec sp
			                      WHERE sp.asset_id=a.id
			                    ), '[]'::jsonb),
			                    'calculations', COALESCE((
			                      SELECT jsonb_agg(
			                        to_jsonb(calc)
			                        || jsonb_build_object(
			                          'calc_code', COALESCE(t.code,''),
			                          'calc_name', COALESCE(t.name,''),
			                          'calc_group', COALESCE(t.grp,'')
			                        )
			                        ORDER BY calc.sort_index, calc.id
			                      )
			                      FROM asset_calculation calc
			                      JOIN asset_calc_type t ON t.id = calc.calc_type_id
			                      WHERE calc.asset_id=a.id
			                    ), '[]'::jsonb)
			                  ) AS asset_json
			           FROM assets a
			           WHERE a.acquisition_id=s.acquisition_id AND a.parcel_id=s.parcel_id
			             AND a.valuation_type=s.valuation_type
			         ) asset_rows
			       ), '[]'::jsonb),
			       COALESCE((
			         SELECT jsonb_agg(comp_json ORDER BY created_at DESC)
			         FROM (
			           SELECT c.created_at,
			                  to_jsonb(c)
			                  || jsonb_build_object(
			                    'grant', COALESCE((
			                      SELECT to_jsonb(cg)
			                      FROM compensation_grant cg
			                      WHERE cg.compensation_id=c.id
			                      LIMIT 1
			                    ), '{}'::jsonb)
			                  ) AS comp_json
			           FROM compensation c
			           WHERE c.acquisition_id=s.acquisition_id AND c.parcel_id=s.parcel_id
			             AND c.valuation_type=s.valuation_type
			         ) comp_rows
			       ), '[]'::jsonb),
			       COALESCE((
			         SELECT SUM(c.amount)
			         FROM compensation c
			         WHERE c.acquisition_id=s.acquisition_id AND c.parcel_id=s.parcel_id
			           AND c.valuation_type=s.valuation_type AND c.status='approved'
			       ), 0)
			FROM parcel_valuation_submission s
			WHERE s.acquisition_id=$1 AND s.parcel_id=$2 AND s.valuation_type=$3 AND s.status=$6
			RETURNING id`,
			acquisitionID, code, vType, note, userID, domain.ValuationStatusApproved,
			reportURL, reportName).Scan(&snapshotID); err != nil {
			if err == pgx.ErrNoRows {
				return nil, domain.ErrInvalidTransition
			}
			return nil, err
		}
		updated, updateErr := tx.Exec(ctx,
			`UPDATE parcel SET selected_valuation_type=NULL, status=$4 WHERE acquisition_id=$1 AND parcel_id=$2 AND (selected_valuation_type=$3 OR selected_valuation_type IS NULL)`,
			acquisitionID, code, vType, domain.ParcelStatusEvaluating)
		if updateErr != nil {
			return nil, updateErr
		}
		if updated.RowsAffected() != 1 {
			return nil, domain.ErrInvalidTransition
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO parcel_status_history (
				parcel_id, acquisition_id, status_id, created_by, reason,
				attachment_url, attachment_name
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			code, acquisitionID, domain.ParcelStatusEvaluating, userID, note, attachURL, attachName); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx,
			`DELETE FROM compensation WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`,
			acquisitionID, code, vType); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx,
			`DELETE FROM land_valuation WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`,
			acquisitionID, code, vType); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx,
			`DELETE FROM assets WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`,
			acquisitionID, code, vType); err != nil {
			return nil, err
		}
		// Үнэлгээний тайланг нэгж талбарын ХАВСРАЛТААС салгана — дараагийн
		// үнэлгээ өөрийн тайлантай ирэх тул хуучин нь "Баримт бичиг" табд
		// үлдвэл андуурагдана. Мөрийг нь л устгана: MinIO дээрх файл ХЭВЭЭР
		// үлдэх тул snapshot дээрх холбоосоор татагдана.
		if reportDocID != uuid.Nil {
			if _, err = tx.Exec(ctx,
				`DELETE FROM parcel_document WHERE parcel_id=$1 AND document_id=$2`,
				parcelID, reportDocID); err != nil {
				return nil, err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM document WHERE id=$1`, reportDocID); err != nil {
				return nil, err
			}
		}
		if _, err = tx.Exec(ctx, `
			UPDATE parcel_valuation_submission
			SET status=$4, submitted_by='', submitted_at=NULL,
			    reviewed_by='', reviewed_at=NULL,
			    last_note=$5, attachment_url=$6, attachment_name=$7, updated_at=NOW()
			WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3`,
			acquisitionID, code, vType, to, note, attachURL, attachName); err != nil {
			return nil, err
		}
	}

	// Зөвшөөрсөн урсгал нь нэгж талбарын үндсэн (selected) үнэлгээ болно.
	if action == "approve" {
		if _, err = tx.Exec(ctx,
			`UPDATE parcel SET selected_valuation_type=$1 WHERE acquisition_id=$2 AND parcel_id=$3`,
			vType, acquisitionID, code); err != nil {
			return nil, err
		}

		// Нэгж талбарт зөвхөн НЭГ баталгаажсан үнэлгээ байна — бусад урсгалын
		// илгээх төлөвийг ТАТГАЛЗСАН болгож, урсгал тус бүрийн түүхэд бичнэ.
		voidNote := "Өөр үнэлгээний урсгал (" + vType + ") баталгаажсан тул автоматаар татгалзагдав"
		rejRows, verr := tx.Query(ctx, `
			UPDATE parcel_valuation_submission s
			SET status=$4, last_note=$5, updated_at=NOW()
			FROM parcel_valuation_submission old
			WHERE old.id = s.id
			  AND s.acquisition_id=$1 AND s.parcel_id=$2
			  AND s.valuation_type<>$3 AND s.status<>$4
			RETURNING s.valuation_type, old.status`,
			acquisitionID, code, vType, domain.ValuationStatusRejected, voidNote)
		if verr != nil {
			return nil, verr
		}
		type rejected struct{ vt, fromStatus string }
		var rejectedList []rejected
		for rejRows.Next() {
			var v rejected
			if err := rejRows.Scan(&v.vt, &v.fromStatus); err != nil {
				rejRows.Close()
				return nil, err
			}
			rejectedList = append(rejectedList, v)
		}
		rejRows.Close()
		if err := rejRows.Err(); err != nil {
			return nil, err
		}
		for _, v := range rejectedList {
			if _, err = tx.Exec(ctx, `
				INSERT INTO parcel_valuation_submission_history (acquisition_id, parcel_id, valuation_type, action, from_status, to_status, note, created_by)
				VALUES ($1,$2,$3,'reject',$4,$5,$6,$7)`,
				acquisitionID, code, v.vt, v.fromStatus, domain.ValuationStatusRejected, voidNote, userID); err != nil {
				return nil, err
			}
		}

		// Олговрын мөрүүдийн статусыг мөн синк хийнэ: сонгогдсон урсгалын
		// хүлээгдэж буй олговрууд БАТАЛГААЖИЖ (approved), бусад урсгалынх
		// ТАТГАЛЗСАН (rejected) болно.
		if _, err = tx.Exec(ctx, `
			UPDATE compensation
			SET status='approved', reviewed_by=$4, reviewed_at=NOW(), review_note=$5, updated_at=NOW()
			WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type=$3 AND status='pending'`,
			acquisitionID, code, vType, userID, note); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `
			UPDATE compensation
			SET status='rejected', reviewed_by=$4, reviewed_at=NOW(), review_note=$5, updated_at=NOW()
			WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type<>$3 AND status<>'rejected'`,
			acquisitionID, code, vType, userID, voidNote); err != nil {
			return nil, err
		}
	}

	// Түүх
	if _, err = tx.Exec(ctx, `
		INSERT INTO parcel_valuation_submission_history (acquisition_id, parcel_id, valuation_type, action, from_status, to_status, note, created_by, attachment_url, attachment_name)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		acquisitionID, code, vType, action, from, to, note, userID, attachURL, attachName); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		SELECT s.id, s.acquisition_id, s.parcel_id, s.valuation_type, s.status,
		       COALESCE(s.submitted_by, ''),
		       s.submitted_at,
		       COALESCE(s.reviewed_by, ''),
		       s.reviewed_at, s.last_note,
		       COALESCE(s.attachment_url,''), COALESCE(s.attachment_name,''),
		       s.created_at, s.updated_at
		FROM parcel_valuation_submission s
		WHERE s.acquisition_id=$1 AND s.parcel_id=$2 AND s.valuation_type=$3`, acquisitionID, code, vType)
	submission, err := scanValuationSubmission(row, acquisitionID, code, vType)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return submission, nil
}

func (r *landAcqRepo) ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error) {
	code, err := r.parcelCodeByID(ctx, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	vType := normalizeValuationType(valuationType)
	rows, err := r.db.Query(ctx, `
		SELECT h.id, h.acquisition_id, h.parcel_id, h.valuation_type, h.action, h.from_status, h.to_status, h.note,
		       COALESCE(h.attachment_url,''), COALESCE(h.attachment_name,''),
		       COALESCE(h.created_by, ''),
		       h.created_at
		FROM parcel_valuation_submission_history h
		WHERE h.acquisition_id=$1 AND h.parcel_id=$2 AND h.valuation_type=$3
		ORDER BY h.created_at DESC, h.id DESC`, acquisitionID, code, vType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.ValuationSubmissionHistory
	for rows.Next() {
		h := &domain.ValuationSubmissionHistory{}
		if err := rows.Scan(&h.ID, &h.AcquisitionID, &h.ParcelID, &h.ValuationType, &h.Action,
			&h.FromStatus, &h.ToStatus, &h.Note,
			&h.AttachmentURL, &h.AttachmentName,
			&h.CreatedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

func (r *landAcqRepo) ListValuationSnapshots(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSnapshot, error) {
	code, err := r.parcelCodeByID(ctx, acquisitionID, parcelID)
	if err != nil {
		return nil, err
	}
	vType := normalizeValuationType(valuationType)
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.acquisition_id, s.parcel_id, s.valuation_type, s.submission_id,
		       s.note,
		       COALESCE(s.cancelled_by, ''),
		       s.cancelled_at,
		       COALESCE(s.submitted_by, ''),
		       s.submitted_at,
		       COALESCE(s.reviewed_by, ''),
		       s.reviewed_at,
		       s.land_valuation, s.assets, s.compensations, s.total_amount,
		       COALESCE(s.report_url, ''), COALESCE(s.report_name, '')
		FROM parcel_valuation_snapshot s
		WHERE s.acquisition_id=$1 AND s.parcel_id=$2 AND s.valuation_type=$3
		ORDER BY s.cancelled_at DESC, s.id DESC`, acquisitionID, code, vType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.ValuationSnapshot
	for rows.Next() {
		s := &domain.ValuationSnapshot{}
		if err := rows.Scan(
			&s.ID, &s.AcquisitionID, &s.ParcelID, &s.ValuationType, &s.SubmissionID,
			&s.Note, &s.CancelledBy, &s.CancelledAt,
			&s.SubmittedBy, &s.SubmittedAt, &s.ReviewedBy, &s.ReviewedAt,
			&s.LandValuation, &s.Assets, &s.Compensations, &s.TotalAmount,
			&s.ReportURL, &s.ReportName,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
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

// UpdateAssetPhoto — хөрөнгийн зургийн PDF-ийн URL-ийг хадгална (асеет уг чөлөөлөлтөд
// хамаарна). Солигдсон ХУУЧИН файлын URL-ийг буцаана — дуудагч хадгалалтаас цэвэрлэнэ.
func (r *landAcqRepo) UpdateAssetPhoto(ctx context.Context, acquisitionID, assetID uuid.UUID, photoURL, photoName string) (string, error) {
	var oldURL string
	err := r.db.QueryRow(ctx, `
		UPDATE assets a SET photo_pdf_url=$1, photo_pdf_name=$2, updated_at=NOW()
		FROM assets prev
		WHERE prev.id = a.id AND a.id=$3 AND a.acquisition_id=$4
		RETURNING COALESCE(prev.photo_pdf_url,'')`,
		photoURL, photoName, assetID, acquisitionID).Scan(&oldURL)
	if err == pgx.ErrNoRows {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return oldURL, nil
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
	// acquisition_id нь захирамжаас гараар үүсгэсэн бие даасан мөр дээр NULL
	// байж болно (000060) — uuid.UUID руу scan хийхийн тулд тэглэнэ.
	err := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(acquisition_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       organization_name, source_type, amount, currency, note, created_at, created_by
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
	rows, err := r.db.Query(ctx, `SELECT id, code, name, default_unit, sort_order, COALESCE(grp,'') FROM asset_calc_type ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AssetCalcType
	for rows.Next() {
		t := &domain.AssetCalcType{}
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.DefaultUnit, &t.SortOrder, &t.Grp); err != nil {
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

func (r *landAcqRepo) SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *domain.ID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE land_acquisition SET professional_org_id = $2, updated_at = NOW() WHERE id = $1`,
		acquisitionID, orgUserID,
	)
	return err
}

func (r *landAcqRepo) SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *domain.ID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE parcel SET independent_org_id = $3 WHERE id = $2 AND acquisition_id = $1`,
		acqID, parcelID, orgUserID,
	)
	return err
}

var _ repository.LandAcquisitionRepository = (*landAcqRepo)(nil)

// ── Дроны ортофото ───────────────────────────────────────────────────────────

const droneImageColumns = `id, acquisition_id, file_name, file_url, original_name, content_type,
	size_bytes, min_x, min_y, max_x, max_y, published, uploaded_at, uploaded_by`

func scanDroneImage(row pgx.Row) (*domain.DroneImage, error) {
	img := &domain.DroneImage{}
	if err := row.Scan(&img.ID, &img.AcquisitionID, &img.FileName, &img.FileURL, &img.OriginalName,
		&img.ContentType, &img.SizeBytes, &img.MinX, &img.MinY, &img.MaxX, &img.MaxY,
		&img.Published, &img.UploadedAt, &img.UploadedBy); err != nil {
		return nil, err
	}
	return img, nil
}

func (r *landAcqRepo) ListDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+droneImageColumns+`
		FROM drone_image
		WHERE acquisition_id = $1
		ORDER BY uploaded_at DESC
	`, acquisitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DroneImage
	for rows.Next() {
		img, err := scanDroneImage(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, img)
	}
	return list, rows.Err()
}

func (r *landAcqRepo) GetDroneImage(ctx context.Context, id uuid.UUID) (*domain.DroneImage, error) {
	img, err := scanDroneImage(r.db.QueryRow(ctx,
		`SELECT `+droneImageColumns+` FROM drone_image WHERE id = $1`, id))
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return img, nil
}

func (r *landAcqRepo) CreateDroneImage(ctx context.Context, img *domain.DroneImage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO drone_image (id, acquisition_id, file_name, file_url, original_name, content_type,
			size_bytes, min_x, min_y, max_x, max_y, published, uploaded_at, uploaded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, img.ID, img.AcquisitionID, img.FileName, img.FileURL, img.OriginalName, img.ContentType,
		img.SizeBytes, img.MinX, img.MinY, img.MaxX, img.MaxY, img.Published,
		img.UploadedAt, img.UploadedBy)
	return err
}

// UpdateDroneImageBounds — GeoServer-ээс уншсан давхаргын хүрээг тэмдэглэнэ.
func (r *landAcqRepo) UpdateDroneImageBounds(ctx context.Context, id uuid.UUID, minX, minY, maxX, maxY float64) error {
	res, err := r.db.Exec(ctx, `
		UPDATE drone_image
		SET min_x = $1, min_y = $2, max_x = $3, max_y = $4, published = TRUE
		WHERE id = $5
	`, minX, minY, maxX, maxY, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *landAcqRepo) DeleteDroneImage(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM drone_image WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
