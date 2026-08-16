package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HR лавлах нь sdplatform схемийн хүснэгтүүд дээр ШУУД буудаг — нэмэлт хүснэгт,
// нэмэлт багана үүсгэдэггүй. Тиймээс энэ холболтыг өөр sdplatform DB рүү
// залгахад, эсвэл өөр систем тэдгээр хүснэгтийг өөрчлөхөд хэвийн ажиллана.
//
//	Organization -> sdplatform.sd_organization
//	Department   -> sdplatform.hr_department
//	Position     -> sdplatform.hr_position
//	Person       -> base.bs_person
//	Employee     -> sdplatform.hr_employee
//
// Схемд deleted_at багана байхгүй тул лавлахуудын устгал нь БОДИТ DELETE.
// Ажилтны хувьд hr_employee.is_active багана байгаа тул устгал нь is_active=0
// (идэвхгүй ажилтан жагсаалтад анхдагчаар харагдахгүй — өмнөх зан төлөв хэвээр).

type HRRepo struct {
	db *pgxpool.Pool
}

func NewHRRepo(db *pgxpool.Pool) *HRRepo {
	return &HRRepo{db: db}
}

type hrQuerier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func normalizeHRFilter(f domain.HRFilter) domain.HRFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	return f
}

func mapHRError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "duplicate key"):
		return domain.ErrAlreadyExists
	case strings.Contains(msg, "violates foreign key constraint"),
		strings.Contains(msg, "violates check constraint"):
		return domain.ErrInvalidInput
	default:
		return err
	}
}

// mapHRInsertError — INSERT..SELECT нь холбоос олдоогүйгээс мөр буцаахгүй бол
// pgx ErrNoRows өгнө; энэ нь буруу оролт гэсэн үг.
func mapHRInsertError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidInput
	}
	return mapHRError(err)
}

// Схемд unique индекс байхгүй давхардлын дүрмүүдийг (хүний регистр, албан
// тушаалын нэр, хэлтсийн нэр, нэг хүн = нэг ажилтан) хүснэгт өөрчлөхгүйгээр
// хангахын тулд бичихийн өмнө шалгана.
func hrExists(ctx context.Context, q hrQuerier, query string, args ...any) (bool, error) {
	var ok bool
	if err := q.QueryRow(ctx, query, args...).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

// ---------------------------------------------------------------------------
// Organization — sd_organization (id, name)
// ---------------------------------------------------------------------------

func (r *HRRepo) ListOrganizations(ctx context.Context, f domain.HRFilter) ([]*domain.Organization, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := ""
	if f.Search != "" {
		where = fmt.Sprintf("WHERE o.name ILIKE $%d", arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM sdplatform.sd_organization o "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(
		`SELECT o.id, COALESCE(o.name, '') FROM sdplatform.sd_organization o %s ORDER BY o.name LIMIT $%d OFFSET $%d`,
		where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Organization
	for rows.Next() {
		o := &domain.Organization{}
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetOrganizationByID(ctx context.Context, id domain.ID) (*domain.Organization, error) {
	o := &domain.Organization{}
	err := r.db.QueryRow(ctx, `SELECT o.id, COALESCE(o.name, '') FROM sdplatform.sd_organization o WHERE o.id=$1`, id).
		Scan(&o.ID, &o.Name)
	return o, mapHRError(err)
}

func (r *HRRepo) CreateOrganization(ctx context.Context, o *domain.Organization) error {
	dup, err := hrExists(ctx, r.db, `SELECT EXISTS(SELECT 1 FROM sdplatform.sd_organization WHERE name = LEFT($1, 250))`, o.Name)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	err = r.db.QueryRow(ctx,
		`INSERT INTO sdplatform.sd_organization (name) VALUES (LEFT($1, 250)) RETURNING id`, o.Name).Scan(&o.ID)
	return mapHRInsertError(err)
}

func (r *HRRepo) UpdateOrganization(ctx context.Context, o *domain.Organization) error {
	dup, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_organization WHERE name = LEFT($1, 250) AND id <> $2)`, o.Name, o.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `UPDATE sdplatform.sd_organization SET name = LEFT($1, 250) WHERE id = $2`, o.Name, o.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeleteOrganization(ctx context.Context, id domain.ID) error {
	linked, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_department WHERE organization = $1)`, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `DELETE FROM sdplatform.sd_organization WHERE id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Department — hr_department
// ---------------------------------------------------------------------------

const departmentSelectSQL = `
	SELECT d.department_id, d.organization, COALESCE(o.name, ''), d.parent_id, d.code, COALESCE(d.name, ''),
	       COALESCE(d.created_at::timestamptz, NOW()), COALESCE(d.updated_at::timestamptz, NOW())` + departmentFromSQL

const departmentFromSQL = `
	FROM sdplatform.hr_department d
	LEFT JOIN sdplatform.sd_organization o ON o.id = d.organization`

func scanDepartment(row rowScanner, d *domain.Department) error {
	return row.Scan(&d.ID, &d.OrganizationID, &d.OrganizationName, &d.ParentID, &d.Code, &d.Name, &d.CreatedAt, &d.UpdatedAt)
}

func (r *HRRepo) ListDepartments(ctx context.Context, f domain.HRFilter) ([]*domain.Department, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := "WHERE true"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (d.name ILIKE $%d OR d.code ILIKE $%d)", arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.OrganizationID != nil {
		where += fmt.Sprintf(" AND d.organization = $%d", arg)
		args = append(args, *f.OrganizationID)
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+departmentFromSQL+" "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(departmentSelectSQL+` %s ORDER BY d.name LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Department
	for rows.Next() {
		d := &domain.Department{}
		if err := scanDepartment(rows, d); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetDepartmentByID(ctx context.Context, id domain.ID) (*domain.Department, error) {
	d := &domain.Department{}
	err := scanDepartment(r.db.QueryRow(ctx, departmentSelectSQL+` WHERE d.department_id=$1`, id), d)
	return d, mapHRError(err)
}

func (r *HRRepo) CreateDepartment(ctx context.Context, d *domain.Department) error {
	dup, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_department WHERE organization = $1 AND name = LEFT($2, 250))`,
		d.OrganizationID, d.Name)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now
	err = r.db.QueryRow(ctx, `
		INSERT INTO sdplatform.hr_department (organization, parent_id, code, name, created_at, updated_at)
		VALUES ($1, $2, LEFT(NULLIF($3, ''), 45), LEFT($4, 250), $5, $6)
		RETURNING department_id`,
		d.OrganizationID, d.ParentID, d.Code, d.Name, d.CreatedAt, d.UpdatedAt).Scan(&d.ID)
	return mapHRInsertError(err)
}

func (r *HRRepo) UpdateDepartment(ctx context.Context, d *domain.Department) error {
	dup, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_department WHERE organization = $1 AND name = LEFT($2, 250) AND department_id <> $3)`,
		d.OrganizationID, d.Name, d.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	d.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE sdplatform.hr_department
		SET organization=$1, parent_id=$2, code=LEFT(NULLIF($3, ''), 45), name=LEFT($4, 250), updated_at=$5
		WHERE department_id=$6`,
		d.OrganizationID, d.ParentID, d.Code, d.Name, d.UpdatedAt, d.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeleteDepartment(ctx context.Context, id domain.ID) error {
	linked, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_employee WHERE department_id = $1 AND COALESCE(is_active, 1) <> 0)`, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `DELETE FROM sdplatform.hr_department WHERE department_id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Position — hr_position
// ---------------------------------------------------------------------------

const positionSelectSQL = `
	SELECT p.position_id, p.code, COALESCE(p.name, ''),
	       COALESCE(p.created_at::timestamptz, NOW()), COALESCE(p.updated_at::timestamptz, NOW())
	FROM sdplatform.hr_position p`

func scanPosition(row rowScanner, p *domain.Position) error {
	return row.Scan(&p.ID, &p.Code, &p.Name, &p.CreatedAt, &p.UpdatedAt)
}

func (r *HRRepo) ListPositions(ctx context.Context, f domain.HRFilter) ([]*domain.Position, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := ""
	if f.Search != "" {
		where = fmt.Sprintf("WHERE (p.name ILIKE $%d OR p.code ILIKE $%d)", arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM sdplatform.hr_position p "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(positionSelectSQL+` %s ORDER BY p.name LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Position
	for rows.Next() {
		p := &domain.Position{}
		if err := scanPosition(rows, p); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetPositionByID(ctx context.Context, id domain.ID) (*domain.Position, error) {
	p := &domain.Position{}
	err := scanPosition(r.db.QueryRow(ctx, positionSelectSQL+` WHERE p.position_id=$1`, id), p)
	return p, mapHRError(err)
}

func (r *HRRepo) CreatePosition(ctx context.Context, p *domain.Position) error {
	dup, err := hrExists(ctx, r.db, `SELECT EXISTS(SELECT 1 FROM sdplatform.hr_position WHERE name = LEFT($1, 250))`, p.Name)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	err = r.db.QueryRow(ctx, `
		INSERT INTO sdplatform.hr_position (code, name, created_at, updated_at)
		VALUES (LEFT(NULLIF($1, ''), 45), LEFT($2, 250), $3, $4)
		RETURNING position_id`,
		p.Code, p.Name, p.CreatedAt, p.UpdatedAt).Scan(&p.ID)
	return mapHRInsertError(err)
}

func (r *HRRepo) UpdatePosition(ctx context.Context, p *domain.Position) error {
	dup, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_position WHERE name = LEFT($1, 250) AND position_id <> $2)`, p.Name, p.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	p.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE sdplatform.hr_position
		SET code=LEFT(NULLIF($1, ''), 45), name=LEFT($2, 250), updated_at=$3
		WHERE position_id=$4`,
		p.Code, p.Name, p.UpdatedAt, p.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeletePosition(ctx context.Context, id domain.ID) error {
	linked, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_employee WHERE position_id = $1 AND COALESCE(is_active, 1) <> 0)`, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `DELETE FROM sdplatform.hr_position WHERE position_id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Person — base.bs_person
// ---------------------------------------------------------------------------

// bs_person нь API-ийн загвараас өөр: төрөл нь cl_person_type код (1=иргэн,
// 2=хуулийн этгээд), овог/байгууллагын нэр нь нэг "name" багана, хүйс нь int.
const personTypeExpr = `CASE p."type" WHEN 2 THEN 'legal'::text ELSE 'citizen'::text END`
const personLastNameExpr = `CASE WHEN p."type" = 2 THEN NULL ELSE p."name" END`
const personLegalNameExpr = `CASE WHEN p."type" = 2 THEN p."name" ELSE NULL END`
const personGenderExpr = `CASE p.gender WHEN 1 THEN 'M'::text WHEN 2 THEN 'F'::text ELSE NULL END`
const personDisplayNameExpr = `CASE WHEN p."type" = 2 THEN COALESCE(p."name", '')
	ELSE trim(COALESCE(p."name", '') || ' ' || COALESCE(p.first_name, '')) END`

const personColumnsSQL = `p.person_id, ` + personTypeExpr + `, p.person_register,
	       ` + personLastNameExpr + `, p.first_name, ` + personLegalNameExpr + `,
	       p.civil_id::text, p.date_of_birth, ` + personGenderExpr + `,
	       p.phone, p.email_address, p.address_town_or_local_name,
	       COALESCE(p.created_at::timestamptz, NOW()), COALESCE(p.updated_at::timestamptz, NOW())`

const personFromSQL = ` FROM base.bs_person p`

const personSelectSQL = `SELECT ` + personColumnsSQL + personFromSQL

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPersonRow(row rowScanner, p *domain.Person) error {
	return row.Scan(&p.ID, &p.PersonType, &p.RegisterNo, &p.LastName, &p.FirstName, &p.LegalName,
		&p.CivilID, &p.BirthDate, &p.Gender, &p.Phone, &p.Email, &p.Address, &p.CreatedAt, &p.UpdatedAt)
}

func (r *HRRepo) ListPersons(ctx context.Context, f domain.HRFilter) ([]*domain.Person, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := ""
	if f.Search != "" {
		where = fmt.Sprintf(`WHERE (p.person_register ILIKE $%d OR p.first_name ILIKE $%d OR p."name" ILIKE $%d)`, arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*)"+personFromSQL+" "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(personSelectSQL+` %s ORDER BY p.created_at DESC NULLS LAST LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Person
	for rows.Next() {
		p := &domain.Person{}
		if err := scanPersonRow(rows, p); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetPersonByID(ctx context.Context, id domain.ID) (*domain.Person, error) {
	p := &domain.Person{}
	err := scanPersonRow(r.db.QueryRow(ctx, personSelectSQL+` WHERE p.person_id=$1`, id), p)
	return p, mapHRError(err)
}

// personWriteArgs — bs_person руу бичих утгуудыг API загвараас хөрвүүлнэ.
func personWriteArgs(p *domain.Person) (legacyName *string, firstName *string) {
	legacyName = p.LastName
	firstName = p.FirstName
	if p.PersonType == "legal" {
		legacyName = p.LegalName
		firstName = nil
	}
	return
}

func insertPerson(ctx context.Context, q hrQuerier, p *domain.Person) error {
	// person_register-т unique индекс байхгүй тул бичихийн өмнө шалгана.
	dup, err := hrExists(ctx, q,
		`SELECT EXISTS(SELECT 1 FROM base.bs_person WHERE person_register = LEFT($1, 10))`, p.RegisterNo)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	legacyName, firstName := personWriteArgs(p)
	err = q.QueryRow(ctx, `
		INSERT INTO base.bs_person
			("type", "name", first_name, gender, date_of_birth, person_register,
			 phone, email_address, address_town_or_local_name, civil_id, created_at, updated_at)
		VALUES (
			CASE $1::text WHEN 'legal' THEN 2 ELSE 1 END,
			LEFT(COALESCE($2::text, ''), 250),
			LEFT($3::text, 50),
			CASE $4::text WHEN 'M' THEN 1 WHEN 'F' THEN 2 ELSE NULL END,
			$5, LEFT($6::text, 10), LEFT($7::text, 256), LEFT($8::text, 256), LEFT($9::text, 256),
			CASE WHEN $10::text ~ '^[0-9]+$' THEN $10::int8 ELSE NULL END,
			$11, $12
		)
		RETURNING person_id`,
		p.PersonType, legacyName, firstName, p.Gender, p.BirthDate, p.RegisterNo,
		p.Phone, p.Email, p.Address, p.CivilID, p.CreatedAt, p.UpdatedAt).Scan(&p.ID)
	return mapHRInsertError(err)
}

func (r *HRRepo) CreatePerson(ctx context.Context, p *domain.Person) error {
	return insertPerson(ctx, r.db, p)
}

func (r *HRRepo) UpdatePerson(ctx context.Context, p *domain.Person) error {
	dup, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM base.bs_person WHERE person_register = LEFT($1, 10) AND person_id <> $2)`,
		p.RegisterNo, p.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	p.UpdatedAt = time.Now()
	legacyName, firstName := personWriteArgs(p)
	res, err := r.db.Exec(ctx, `
		WITH upd AS (
			UPDATE base.bs_person p
			SET "type" = CASE $1::text WHEN 'legal' THEN 2 ELSE 1 END,
			    "name" = LEFT(COALESCE($2::text, ''), 250),
			    first_name = LEFT($3::text, 50),
			    gender = CASE $4::text WHEN 'M' THEN 1 WHEN 'F' THEN 2 ELSE NULL END,
			    date_of_birth = $5,
			    person_register = LEFT($6::text, 10),
			    phone = LEFT($7::text, 256),
			    email_address = LEFT($8::text, 256),
			    address_town_or_local_name = LEFT($9::text, 256),
			    civil_id = CASE WHEN $10::text ~ '^[0-9]+$' THEN $10::int8 ELSE NULL END,
			    updated_at = $11
			WHERE p.person_id = $12
			RETURNING p.person_id, p."name", p.first_name
		)
		-- hr_employee нь ажилтны нэрийг давхардуулан хадгалдаг (legacy
		-- уншигчдад зориулж) — хүн засагдахад тэднийг мөн шинэчилнэ.
		UPDATE sdplatform.hr_employee e
		SET firstname = LEFT(u.first_name, 250), lastname = LEFT(u."name", 250)
		FROM upd u
		WHERE e.person_id = u.person_id`,
		p.PersonType, legacyName, firstName, p.Gender, p.BirthDate, p.RegisterNo,
		p.Phone, p.Email, p.Address, p.CivilID, p.UpdatedAt, p.ID)
	if err != nil {
		return mapHRError(err)
	}
	// Ажилтангүй хүн засагдвал сүүлийн UPDATE 0 мөр хөндөнө — хүн өөрөө
	// шинэчлэгдсэн эсэхийг тусад нь шалгана.
	if res.RowsAffected() == 0 {
		found, err := hrExists(ctx, r.db, `SELECT EXISTS(SELECT 1 FROM base.bs_person WHERE person_id = $1)`, p.ID)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrNotFound
		}
	}
	return nil
}

func (r *HRRepo) DeletePerson(ctx context.Context, id domain.ID) error {
	linked, err := hrExists(ctx, r.db,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_employee WHERE person_id = $1 AND COALESCE(is_active, 1) <> 0)`, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `DELETE FROM base.bs_person WHERE person_id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Employee — hr_employee
// ---------------------------------------------------------------------------

func employeeSelectSQL() string {
	return `SELECT e.employee_id, e.person_id, d.organization, COALESCE(o.name, ''),
		e.department_id, COALESCE(d.name, ''), e.position_id, COALESCE(pos.name, ''),
		e.email, e.mobile_phone, e.hired_date, e.terminated_date, COALESCE(e.is_active, 1) <> 0,
		COALESCE(e.created_at::timestamptz, NOW()), COALESCE(e.updated_at::timestamptz, NOW()),
		` + personColumnsSQL + `,
		` + personDisplayNameExpr + employeeFromSQL
}

const employeeFromSQL = `
	FROM sdplatform.hr_employee e
	JOIN base.bs_person p ON p.person_id = e.person_id
	JOIN sdplatform.hr_position pos ON pos.position_id = e.position_id
	LEFT JOIN sdplatform.hr_department d ON d.department_id = e.department_id
	LEFT JOIN sdplatform.sd_organization o ON o.id = d.organization`

func scanEmployeeRow(row rowScanner, e *domain.Employee, p *domain.Person) error {
	return row.Scan(&e.ID, &e.PersonID, &e.OrganizationID, &e.OrganizationName,
		&e.DepartmentID, &e.DepartmentName, &e.PositionID, &e.PositionName,
		&e.WorkEmail, &e.WorkPhone, &e.HiredAt, &e.ReleasedAt, &e.IsActive, &e.CreatedAt, &e.UpdatedAt,
		&p.ID, &p.PersonType, &p.RegisterNo, &p.LastName, &p.FirstName, &p.LegalName,
		&p.CivilID, &p.BirthDate, &p.Gender, &p.Phone, &p.Email, &p.Address, &p.CreatedAt, &p.UpdatedAt,
		&e.PersonName)
}

func (r *HRRepo) ListEmployees(ctx context.Context, f domain.HRFilter) ([]*domain.Employee, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	// Устгагдсан ажилтан is_active=0 болдог тул тусгайлан асуугаагүй бол
	// жагсаалтад харагдахгүй (өмнөх deleted_at шүүлттэй ижил үр дүн).
	where := "WHERE true"
	if f.IsActive == nil {
		where += " AND COALESCE(e.is_active, 1) <> 0"
	}
	if f.Search != "" {
		where += fmt.Sprintf(` AND (e.email ILIKE $%d OR p.person_register ILIKE $%d OR p.first_name ILIKE $%d OR p."name" ILIKE $%d)`, arg, arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.OrganizationID != nil {
		where += fmt.Sprintf(" AND d.organization = $%d", arg)
		args = append(args, *f.OrganizationID)
		arg++
	}
	if f.DepartmentID != nil {
		where += fmt.Sprintf(" AND e.department_id = $%d", arg)
		args = append(args, *f.DepartmentID)
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND (COALESCE(e.is_active, 1) <> 0) = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+employeeFromSQL+" "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(employeeSelectSQL()+` %s ORDER BY e.created_at DESC NULLS LAST LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Employee
	for rows.Next() {
		e := &domain.Employee{}
		p := &domain.Person{}
		if err := scanEmployeeRow(rows, e, p); err != nil {
			return nil, 0, err
		}
		e.Person = p
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetEmployeeByID(ctx context.Context, id domain.ID) (*domain.Employee, error) {
	e := &domain.Employee{}
	p := &domain.Person{}
	err := scanEmployeeRow(
		r.db.QueryRow(ctx, employeeSelectSQL()+` WHERE e.employee_id=$1 AND COALESCE(e.is_active, 1) <> 0`, id), e, p)
	if err != nil {
		return nil, mapHRError(err)
	}
	e.Person = p
	return e, nil
}

func insertEmployee(ctx context.Context, q hrQuerier, e *domain.Employee) error {
	// Нэг хүн = нэг идэвхтэй ажилтан (схемд unique индекс байхгүй тул шалгана).
	dup, err := hrExists(ctx, q,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_employee WHERE person_id = $1 AND COALESCE(is_active, 1) <> 0)`, e.PersonID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	err = q.QueryRow(ctx, `
		INSERT INTO sdplatform.hr_employee (
			person_id, department_id, position_id, firstname, lastname,
			email, mobile_phone, hired_date, terminated_date, is_active, created_at, updated_at
		)
		SELECT $1::int4, $2::int4, $3::int4, LEFT(p.first_name, 250), LEFT(p."name", 250),
		       LEFT($4::text, 250), LEFT($5::text, 50), $6, $7, 1, $8, $9
		FROM base.bs_person p
		WHERE p.person_id = $1::int4
		RETURNING employee_id`,
		e.PersonID, e.DepartmentID, e.PositionID, e.WorkEmail, e.WorkPhone,
		e.HiredAt, e.ReleasedAt, e.CreatedAt, e.UpdatedAt).Scan(&e.ID)
	if err != nil {
		return mapHRInsertError(err)
	}
	e.IsActive = true
	return nil
}

func (r *HRRepo) CreateEmployee(ctx context.Context, e *domain.Employee) error {
	return insertEmployee(ctx, r.db, e)
}

func (r *HRRepo) CreateEmployeeWithPerson(ctx context.Context, e *domain.Employee, p *domain.Person) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if p != nil {
		if err := insertPerson(ctx, tx, p); err != nil {
			return err
		}
		e.PersonID = p.ID
	}
	if err := insertEmployee(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *HRRepo) UpdateEmployee(ctx context.Context, e *domain.Employee) error {
	e.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE sdplatform.hr_employee e
		SET person_id=$1, department_id=$2, position_id=$3,
		    email=LEFT($4::text, 250), mobile_phone=LEFT($5::text, 50),
		    hired_date=$6, terminated_date=$7,
		    is_active=CASE WHEN $8 THEN 1 ELSE 0 END,
		    updated_at=$9,
		    firstname=LEFT((SELECT p.first_name FROM base.bs_person p WHERE p.person_id=$1), 250),
		    lastname=LEFT((SELECT p."name" FROM base.bs_person p WHERE p.person_id=$1), 250)
		WHERE e.employee_id=$10 AND COALESCE(e.is_active, 1) <> 0`,
		e.PersonID, e.DepartmentID, e.PositionID, e.WorkEmail, e.WorkPhone,
		e.HiredAt, e.ReleasedAt, e.IsActive, e.UpdatedAt, e.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteEmployee — схемд deleted_at байхгүй тул is_active=0 болгоно.
func (r *HRRepo) DeleteEmployee(ctx context.Context, id domain.ID) error {
	linked, err := hrExists(ctx, r.db, `
		SELECT EXISTS(
			SELECT 1 FROM sdplatform.sd_user u
			JOIN sdplatform.hr_employee e ON e.user_id = u.user_id
			WHERE e.employee_id = $1 AND u.deleted_at IS NULL
		)`, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx,
		`UPDATE sdplatform.hr_employee SET is_active = 0, updated_at = NOW()
		 WHERE employee_id = $1 AND COALESCE(is_active, 1) <> 0`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
