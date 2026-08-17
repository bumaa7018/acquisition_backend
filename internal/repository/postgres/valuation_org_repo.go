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

// ValuationOrgRepo — үнэлгээний (мэргэжлийн) байгууллага ба түүний ажилтнууд.
//
// БҮГД НЭГ САНД (main DB) байрлана. Үнэлгээний байгууллага бол ГАДААД тал
// бөгөөд гадаад талууд (мэргэжлийн байгууллага, МИКА, санхүү) нь 000093-ын
// дагуу main DB-ийн sdplatform схем дээр амьдардаг — auth DB руу ХЭЗЭЭ Ч
// холбогдохгүй. Иймд:
//
//   - sdplatform.sd_organization — байгууллага өөрөө (id, name);
//   - valuation_org_profile — регистр, тусгай зөвшөөрөл зэрэг МАНАЙ системд л
//     хэрэгтэй талбарууд. sdplatform схемд хүснэгт нэмэхийг хориглосон (000015)
//     тул аппын өөрийн хүснэгт болгосон (000094-ийн адил загвар).
//
// Нэг санд байгаа тул жинхэнэ SQL join, SQL хуудаслалт, нэг гүйлгээ ажиллана.
//
// profile мөртэй байгууллага л ҮНЭЛГЭЭНИЙ байгууллага гэж тооцогдоно —
// жирийн HR лавлахын байгууллагаас ингэж ялгарна.
//
// Ажилтан нь sdplatform-ын төрөлх бүтцээр:
//
//	sd_organization → hr_department → hr_employee → bs_person
//	hr_employee.user_id → sd_user (нэвтрэх эрх, заавал бус)
//
// hr_employee-д organization багана байхгүй тул байгууллага бүрд үүсэх үедээ
// нэг "үндсэн нэгж" (hr_department) автоматаар үүсгэж, ажилтныг түүнд холбоно.
type ValuationOrgRepo struct {
	db *pgxpool.Pool
}

func NewValuationOrgRepo(db *pgxpool.Pool) *ValuationOrgRepo {
	return &ValuationOrgRepo{db: db}
}

// roleLevelExpr — domain.Role.Level()-ийн SQL хувилбар: нэрнээс нөөцийн
// угтварыг таслаад түвшинг үлдээнэ ("land_professional_org" → "professional_org").
const roleLevelExpr = `CASE
	WHEN COALESCE(r.role_type_code, '') <> '' AND r."name" LIKE r.role_type_code || '\_%'
	THEN substring(r."name" from length(r.role_type_code) + 2)
	ELSE r."name"
END`

const valuationOrgSelectSQL = `
	SELECT o.id, COALESCE(o."name", ''), vp.short_name, vp.register_no,
	       vp.license_no, vp.license_issued_at, vp.license_expires_at,
	       vp.phone, vp.email, vp.address, vp.note, vp.is_active,
	       vp.created_at, vp.updated_at,
	       (SELECT COUNT(*)
	          FROM sdplatform.hr_employee e
	          JOIN sdplatform.hr_department d ON d.department_id = e.department_id
	         WHERE d.organization = o.id AND COALESCE(e.is_active, 1) <> 0)
	FROM valuation_org_profile vp
	JOIN sdplatform.sd_organization o ON o.id = vp.organization_id`

func scanValuationOrg(row rowScanner, v *domain.ValuationOrg) error {
	return row.Scan(&v.ID, &v.Name, &v.ShortName, &v.RegisterNo,
		&v.LicenseNo, &v.LicenseIssuedAt, &v.LicenseExpiresAt,
		&v.Phone, &v.Email, &v.Address, &v.Note, &v.IsActive,
		&v.CreatedAt, &v.UpdatedAt, &v.EmployeeCount)
}

func (r *ValuationOrgRepo) List(ctx context.Context, f domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 20
	}
	where := "WHERE true"
	args := []any{}
	arg := 1
	if f.Search != "" {
		where += fmt.Sprintf(` AND (o."name" ILIKE $%d OR vp.short_name ILIKE $%d OR vp.register_no ILIKE $%d OR vp.license_no ILIKE $%d)`, arg, arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND vp.is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}

	var total int64
	countSQL := `SELECT COUNT(*) FROM valuation_org_profile vp
		JOIN sdplatform.sd_organization o ON o.id = vp.organization_id ` + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(valuationOrgSelectSQL+` %s ORDER BY o."name" LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []*domain.ValuationOrg
	for rows.Next() {
		v := &domain.ValuationOrg{}
		if err := scanValuationOrg(rows, v); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *ValuationOrgRepo) GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error) {
	v := &domain.ValuationOrg{}
	if err := scanValuationOrg(r.db.QueryRow(ctx, valuationOrgSelectSQL+` WHERE o.id = $1`, id), v); err != nil {
		return nil, mapHRError(err)
	}
	emps, err := r.ListEmployees(ctx, id)
	if err != nil {
		return nil, err
	}
	v.Employees = emps
	return v, nil
}

// ListEmployees — байгууллагын ИДЭВХТЭЙ ажилтнууд, нэвтрэх эрхтэй эсэхийн хамт.
func (r *ValuationOrgRepo) ListEmployees(ctx context.Context, orgID domain.ID) ([]*domain.Employee, error) {
	rows, err := r.db.Query(ctx, employeeSelectSQL()+`
		WHERE d.organization = $1 AND COALESCE(e.is_active, 1) <> 0
		ORDER BY p."name", p.first_name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Employee
	for rows.Next() {
		e := &domain.Employee{}
		p := &domain.Person{}
		if err := scanEmployeeRow(rows, e, p); err != nil {
			return nil, err
		}
		e.Person = p
		out = append(out, e)
	}
	return out, rows.Err()
}

// Create — байгууллага, профайл, үндсэн нэгж болон ажилтнуудыг НЭГ гүйлгээгээр
// үүсгэнэ. Аль нэг алхам унавал юу ч үлдэхгүй.
func (r *ValuationOrgRepo) Create(ctx context.Context, v *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	dup, err := hrExists(ctx, tx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_organization WHERE "name" = LEFT($1, 250))`, v.Name)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}

	if err := tx.QueryRow(ctx,
		`INSERT INTO sdplatform.sd_organization ("name") VALUES (LEFT($1, 250)) RETURNING id`, v.Name).
		Scan(&v.ID); err != nil {
		return mapHRError(err)
	}
	if err := upsertValuationProfile(ctx, tx, v, true); err != nil {
		return err
	}
	if _, err := ensureOrgDepartment(ctx, tx, v.ID); err != nil {
		return err
	}
	if err := saveValuationEmployees(ctx, tx, v.ID, emps); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Update — байгууллагын мэдээлэл ба ажилтнуудыг ХАМТ шинэчилнэ. emps нь nil
// бол ажилтнуудад хүрэхгүй; nil биш бол жагсаалт нь БҮРЭН төлөвийг илэрхийлнэ
// (дотор нь байхгүй ажилтан идэвхгүй болно).
func (r *ValuationOrgRepo) Update(ctx context.Context, v *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	dup, err := hrExists(ctx, tx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_organization WHERE "name" = LEFT($1, 250) AND id <> $2)`, v.Name, v.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}

	res, err := tx.Exec(ctx, `UPDATE sdplatform.sd_organization SET "name" = LEFT($1, 250) WHERE id = $2`, v.Name, v.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err := upsertValuationProfile(ctx, tx, v, false); err != nil {
		return err
	}
	if emps != nil {
		if _, err := ensureOrgDepartment(ctx, tx, v.ID); err != nil {
			return err
		}
		if err := saveValuationEmployees(ctx, tx, v.ID, emps); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Delete — чөлөөлөлт/нэгж талбарт холбогдоогүй байгууллагыг л устгана.
//
// sd_organization устахад hr_department (ON DELETE CASCADE) мөн устаж,
// ажилтнуудын hr_employee.department_id NULL болно. Профайл нь sdplatform
// схемийн гадна байдаг тул (тэнд FK тавихыг хориглосон) тусад нь устгах ба
// өнчин профайл үлдэхээс сэргийлж хоёуланг НЭГ гүйлгээнд гүйцэтгэнэ.
func (r *ValuationOrgRepo) Delete(ctx context.Context, id domain.ID) error {
	linked, err := hrExists(ctx, r.db, `SELECT EXISTS(
		SELECT 1 FROM land_acquisition WHERE professional_org_id = $1
		UNION ALL
		SELECT 1 FROM parcel WHERE independent_org_id = $1
	)`, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ажилтныг ЭХЛЭЭД салгана. hr_employee.department_id-д гадаад түлхүүр
	// БАЙХГҮЙ тул хэлтэс устахад ажилтан устсан дугаар руу зааж үлддэг —
	// serial дугаар дахин ашиглагдвал тэр ажилтан ӨӨР байгууллагад чимээгүй
	// наалдаж, тэр байгууллагын ажлыг харах эрсдэлтэй.
	detached, err := tx.Query(ctx, `
		UPDATE sdplatform.hr_employee e
		SET department_id = NULL, is_active = 0,
		    terminated_date = COALESCE(e.terminated_date, $2::date), updated_at = $2
		WHERE e.department_id IN (
			SELECT department_id FROM sdplatform.hr_department WHERE organization = $1
		)
		RETURNING e.employee_id`, id, time.Now())
	if err != nil {
		return mapHRError(err)
	}
	var detachedIDs []domain.ID
	for detached.Next() {
		var empID domain.ID
		if err := detached.Scan(&empID); err != nil {
			detached.Close()
			return err
		}
		detachedIDs = append(detachedIDs, empID)
	}
	detached.Close()
	if err := detached.Err(); err != nil {
		return err
	}
	// Байгууллага байхгүй болсон тул ажилтнуудын нэвтрэх эрхийг мөн хаана.
	if err := deactivateEmployeeUsers(ctx, tx, detachedIDs); err != nil {
		return err
	}

	res, err := tx.Exec(ctx, `DELETE FROM sdplatform.sd_organization WHERE id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM valuation_org_profile WHERE organization_id = $1`, id); err != nil {
		return mapHRError(err)
	}
	return tx.Commit(ctx)
}

// upsertValuationProfile — аппын өөрийн талбарууд (гүйлгээний дотор).
func upsertValuationProfile(ctx context.Context, q hrQuerier, v *domain.ValuationOrg, insert bool) error {
	now := time.Now()
	v.UpdatedAt = now
	if insert {
		v.CreatedAt = now
	}
	_, err := q.Exec(ctx, `
		INSERT INTO valuation_org_profile (
			organization_id, short_name, register_no, license_no,
			license_issued_at, license_expires_at, phone, email, address, note,
			is_active, created_at, updated_at)
		VALUES ($1, LEFT($2, 250), LEFT($3, 20), LEFT($4, 50), $5, $6,
		        LEFT($7, 50), LEFT($8, 250), LEFT($9, 500), $10, $11, $12, $12)
		ON CONFLICT (organization_id) DO UPDATE SET
			short_name = EXCLUDED.short_name,
			register_no = EXCLUDED.register_no,
			license_no = EXCLUDED.license_no,
			license_issued_at = EXCLUDED.license_issued_at,
			license_expires_at = EXCLUDED.license_expires_at,
			phone = EXCLUDED.phone,
			email = EXCLUDED.email,
			address = EXCLUDED.address,
			note = EXCLUDED.note,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at`,
		v.ID, v.ShortName, v.RegisterNo, v.LicenseNo,
		v.LicenseIssuedAt, v.LicenseExpiresAt, v.Phone, v.Email, v.Address, v.Note,
		v.IsActive, now)
	return mapHRError(err)
}

// ensureOrgDepartment — байгууллагын үндсэн нэгжийг олж, байхгүй бол үүсгэнэ.
func ensureOrgDepartment(ctx context.Context, q hrQuerier, orgID domain.ID) (domain.ID, error) {
	var deptID domain.ID
	err := q.QueryRow(ctx, `
		SELECT department_id FROM sdplatform.hr_department
		WHERE organization = $1 ORDER BY department_id LIMIT 1`, orgID).Scan(&deptID)
	if err == nil {
		return deptID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	now := time.Now()
	err = q.QueryRow(ctx, `
		INSERT INTO sdplatform.hr_department (organization, "name", created_at, updated_at)
		VALUES ($1, LEFT($2, 250), $3, $3)
		RETURNING department_id`, orgID, domain.ValuationOrgDepartmentName, now).Scan(&deptID)
	if err != nil {
		return 0, mapHRError(err)
	}
	return deptID, nil
}

// ensurePosition — албан тушаалыг нэрээр нь олж, байхгүй бол үүсгэнэ.
func ensurePosition(ctx context.Context, q hrQuerier, name string) (domain.ID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = domain.ValuationOrgPositionName
	}
	var id domain.ID
	err := q.QueryRow(ctx,
		`SELECT position_id FROM sdplatform.hr_position WHERE "name" = LEFT($1, 250) ORDER BY position_id LIMIT 1`, name).
		Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	now := time.Now()
	err = q.QueryRow(ctx, `
		INSERT INTO sdplatform.hr_position ("name", created_at, updated_at)
		VALUES (LEFT($1, 250), $2, $2) RETURNING position_id`, name, now).Scan(&id)
	if err != nil {
		return 0, mapHRError(err)
	}
	return id, nil
}

// saveValuationEmployees — байгууллагын ажилтнуудын БҮРЭН төлөвийг тавина.
// Жагсаалтад ороогүй идэвхтэй ажилтан идэвхгүй болно (мөр устгахгүй — түүх,
// баримт дээрх нэр хадгалагдана).
func saveValuationEmployees(ctx context.Context, tx pgx.Tx, orgID domain.ID, emps []domain.ValuationOrgEmployee) error {
	deptID, err := ensureOrgDepartment(ctx, tx, orgID)
	if err != nil {
		return err
	}

	keep := make([]domain.ID, 0, len(emps))
	for i := range emps {
		id, err := saveValuationEmployee(ctx, tx, orgID, deptID, &emps[i])
		if err != nil {
			return err
		}
		keep = append(keep, id)
	}

	// Үлдсэнийг идэвхгүй болгоно. keep хоосон байж болзошгүй тул ANY(NULL)
	// биш, тусад нь салгаж бичнэ. Идэвхгүй болсон ажилтны employee_id-г
	// буцааж авах нь чухал — нэвтрэх эрхийг нь мөн хаах хэрэгтэй.
	now := time.Now()
	var rows pgx.Rows
	if len(keep) == 0 {
		rows, err = tx.Query(ctx, `
			UPDATE sdplatform.hr_employee e
			SET is_active = 0, terminated_date = COALESCE(e.terminated_date, $2::date), updated_at = $2
			WHERE e.department_id IN (SELECT department_id FROM sdplatform.hr_department WHERE organization = $1)
			  AND COALESCE(e.is_active, 1) <> 0
			RETURNING e.employee_id`, orgID, now)
	} else {
		rows, err = tx.Query(ctx, `
			UPDATE sdplatform.hr_employee e
			SET is_active = 0, terminated_date = COALESCE(e.terminated_date, $3::date), updated_at = $3
			WHERE e.department_id IN (SELECT department_id FROM sdplatform.hr_department WHERE organization = $1)
			  AND COALESCE(e.is_active, 1) <> 0
			  AND e.employee_id <> ALL($2::int4[])
			RETURNING e.employee_id`, orgID, keep, now)
	}
	if err != nil {
		return mapHRError(err)
	}
	var removed []domain.ID
	for rows.Next() {
		var id domain.ID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		removed = append(removed, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Байгууллагаас хасагдсан ажилтан нэвтэрсээр байх ёсгүй.
	return deactivateEmployeeUsers(ctx, tx, removed)
}

func saveValuationEmployee(ctx context.Context, tx pgx.Tx, orgID, deptID domain.ID, in *domain.ValuationOrgEmployee) (domain.ID, error) {
	positionID, err := ensurePosition(ctx, tx, in.PositionName)
	if err != nil {
		return 0, err
	}

	person := &domain.Person{
		PersonType: "citizen",
		RegisterNo: strings.TrimSpace(in.RegisterNo),
		LastName:   optionalText(in.LastName),
		FirstName:  optionalText(in.FirstName),
		Phone:      optionalText(in.Phone),
		Email:      optionalText(in.Email),
	}

	var employeeID domain.ID
	if in.ID != nil {
		employeeID = *in.ID
		// Ажилтан ЭНЭ байгууллагынх мөн эсэхийг батална — өөр байгууллагын
		// ажилтныг ID-гаар нь өөрлөх боломжгүй байх ёстой.
		var personID domain.ID
		err := tx.QueryRow(ctx, `
			SELECT e.person_id
			FROM sdplatform.hr_employee e
			JOIN sdplatform.hr_department d ON d.department_id = e.department_id
			WHERE e.employee_id = $1 AND d.organization = $2`, employeeID, orgID).Scan(&personID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, domain.ErrNotFound
			}
			return 0, err
		}
		person.ID = personID
		if err := updateValuationPerson(ctx, tx, person); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE sdplatform.hr_employee
			SET department_id = $1, position_id = $2, email = LEFT($3::text, 250),
			    mobile_phone = LEFT($4::text, 50), firstname = LEFT($5::text, 250),
			    lastname = LEFT($6::text, 250), is_active = 1, updated_at = $7
			WHERE employee_id = $8`,
			deptID, positionID, in.Email, in.Phone, in.FirstName, in.LastName,
			time.Now(), employeeID); err != nil {
			return 0, mapHRError(err)
		}
	} else {
		// Регистрээр нь одоо байгаа хүнийг дахин ашиглана (bs_person нь
		// системийн хэмжээнд нэгдсэн лавлах — давхардуулж болохгүй).
		var existing domain.ID
		err := tx.QueryRow(ctx,
			`SELECT person_id FROM base.bs_person WHERE person_register = LEFT($1, 10)`, person.RegisterNo).
			Scan(&existing)
		switch {
		case err == nil:
			person.ID = existing
			if err := updateValuationPerson(ctx, tx, person); err != nil {
				return 0, err
			}
		case errors.Is(err, pgx.ErrNoRows):
			if err := insertPerson(ctx, tx, person); err != nil {
				return 0, err
			}
		default:
			return 0, err
		}

		// Тухайн хүн аль хэдийн ажилтан бол шинэ мөр үүсгэхгүй, сэргээнэ.
		now := time.Now()
		err = tx.QueryRow(ctx, `
			UPDATE sdplatform.hr_employee
			SET department_id = $1, position_id = $2, email = LEFT($3::text, 250),
			    mobile_phone = LEFT($4::text, 50), firstname = LEFT($5::text, 250),
			    lastname = LEFT($6::text, 250), is_active = 1, terminated_date = NULL, updated_at = $7
			WHERE person_id = $8
			RETURNING employee_id`,
			deptID, positionID, in.Email, in.Phone, in.FirstName, in.LastName, now, person.ID).
			Scan(&employeeID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
				INSERT INTO sdplatform.hr_employee (
					person_id, department_id, position_id, firstname, lastname,
					email, mobile_phone, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, LEFT($4::text, 250), LEFT($5::text, 250),
				        LEFT($6::text, 250), LEFT($7::text, 50), 1, $8, $8)
				RETURNING employee_id`,
				person.ID, deptID, positionID, in.FirstName, in.LastName,
				in.Email, in.Phone, now).Scan(&employeeID)
		}
		if err != nil {
			return 0, mapHRError(err)
		}
	}

	if err := linkValuationEmployeeLogin(ctx, tx, employeeID, in); err != nil {
		return 0, err
	}
	return employeeID, nil
}

// updateValuationPerson — bs_person-ийг зөвхөн энэ урсгалд хэрэгтэй талбараар
// шинэчилнэ (HRRepo.UpdatePerson нь өөрийн pool дээр ажилладаг тул гүйлгээнд
// оруулж болохгүй).
func updateValuationPerson(ctx context.Context, q hrQuerier, p *domain.Person) error {
	dup, err := hrExists(ctx, q,
		`SELECT EXISTS(SELECT 1 FROM base.bs_person WHERE person_register = LEFT($1, 10) AND person_id <> $2)`,
		p.RegisterNo, p.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	_, err = q.Exec(ctx, `
		UPDATE base.bs_person
		SET "type" = 1,
		    "name" = LEFT(COALESCE($1::text, ''), 250),
		    first_name = LEFT($2::text, 50),
		    person_register = LEFT($3::text, 10),
		    phone = LEFT($4::text, 256),
		    email_address = LEFT($5::text, 256),
		    updated_at = $6
		WHERE person_id = $7`,
		p.LastName, p.FirstName, p.RegisterNo, p.Phone, p.Email, time.Now(), p.ID)
	return mapHRError(err)
}

// linkValuationEmployeeLogin — ажилтанд нэвтрэх эрх (sd_user) холбоно.
//
// Username хоосон бол ямар ч үйлдэл хийхгүй — өмнө холбогдсон эрх хэвээр
// үлдэнэ (эрх хасах нь тусдаа зориудын үйлдэл байх ёстой).
// in.Password нь ЭНД ирэхдээ аль хэдийн hash хийгдсэн байх ёстой.
func linkValuationEmployeeLogin(ctx context.Context, tx pgx.Tx, employeeID domain.ID, in *domain.ValuationOrgEmployee) error {
	username := strings.TrimSpace(in.Username)
	if username == "" {
		return nil
	}

	var userID domain.ID
	err := tx.QueryRow(ctx,
		`SELECT user_id FROM sdplatform.sd_user WHERE username = LEFT($1, 50) AND deleted_at IS NULL`, username).
		Scan(&userID)
	switch {
	case err == nil:
		if in.Password != "" {
			if _, err := tx.Exec(ctx,
				`UPDATE sdplatform.sd_user SET "password" = LEFT($1, 150), updated_at = $2 WHERE user_id = $3`,
				in.Password, time.Now(), userID); err != nil {
				return mapHRError(err)
			}
		}
	case errors.Is(err, pgx.ErrNoRows):
		if in.Password == "" {
			// Шинэ эрх үүсгэхэд нууц үг заавал шаардлагатай.
			return domain.ErrInvalidInput
		}
		now := time.Now()
		// email-д UNIQUE индекс (ux_sd_user_email_active) байдаг тул имэйлгүй
		// ажилтанд ХООСОН МӨР биш NULL бичнэ — эс бөгөөс имэйлгүй хоёр дахь
		// ажилтан "давхардсан бичлэг" гэж татгалзагдана.
		if err := tx.QueryRow(ctx, `
			INSERT INTO sdplatform.sd_user (username, email, "password", firstname, lastname, created_at, updated_at, is_active)
			VALUES (LEFT($1, 50), LEFT(NULLIF($2::text, ''), 250), LEFT($3, 150), LEFT($4::text, 50), LEFT($5::text, 50), $6::date, $6, true)
			RETURNING user_id`,
			username, in.Email, in.Password, in.FirstName, in.LastName, now).Scan(&userID); err != nil {
			return mapHRError(err)
		}
	default:
		return err
	}

	// Мэргэжлийн байгууллагын түвшний бүх роль.
	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
		SELECT $1, r.role_id, $2, $2
		FROM sdplatform.sd_role r
		WHERE `+roleLevelExpr+` = 'professional_org'
		ON CONFLICT (user_id, role_id) DO NOTHING`, userID, time.Now()); err != nil {
		return mapHRError(err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE sdplatform.hr_employee SET user_id = $1, updated_at = $2 WHERE employee_id = $3`,
		userID, time.Now(), employeeID); err != nil {
		return mapHRError(err)
	}
	return nil
}

func optionalText(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
