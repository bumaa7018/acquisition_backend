package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func scanOrganization(row pgx.Row) (*domain.Organization, error) {
	o := &domain.Organization{}
	err := row.Scan(&o.ID, &o.ParentID, &o.Name, &o.ShortName, &o.RegisterNo, &o.Address, &o.Phone, &o.Email, &o.ExternalID, &o.Source, &o.IsActive, &o.CreatedAt, &o.UpdatedAt, &o.DeletedAt)
	return o, mapHRError(err)
}

func (r *HRRepo) ListOrganizations(ctx context.Context, f domain.HRFilter) ([]*domain.Organization, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := "WHERE deleted_at IS NULL"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR short_name ILIKE $%d OR register_no ILIKE $%d)", arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM organization "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id, parent_id, name, short_name, register_no, address, phone, email, external_id, source, is_active, created_at, updated_at, deleted_at
		FROM organization %s ORDER BY name LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Organization
	for rows.Next() {
		o := &domain.Organization{}
		if err := rows.Scan(&o.ID, &o.ParentID, &o.Name, &o.ShortName, &o.RegisterNo, &o.Address, &o.Phone, &o.Email, &o.ExternalID, &o.Source, &o.IsActive, &o.CreatedAt, &o.UpdatedAt, &o.DeletedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	return scanOrganization(r.db.QueryRow(ctx, `SELECT id, parent_id, name, short_name, register_no, address, phone, email, external_id, source, is_active, created_at, updated_at, deleted_at FROM organization WHERE id=$1 AND deleted_at IS NULL`, id))
}

func (r *HRRepo) CreateOrganization(ctx context.Context, o *domain.Organization) error {
	now := time.Now()
	o.ID = uuid.New()
	o.CreatedAt = now
	o.UpdatedAt = now
	if o.Source == "" {
		o.Source = "local"
	}
	if !o.IsActive {
		o.IsActive = true
	}
	err := r.db.QueryRow(ctx, `INSERT INTO organization (id, parent_id, name, short_name, register_no, address, phone, email, external_id, source, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`,
		o.ID, o.ParentID, o.Name, o.ShortName, o.RegisterNo, o.Address, o.Phone, o.Email, o.ExternalID, o.Source, o.IsActive, o.CreatedAt, o.UpdatedAt).Scan(&o.ID)
	return mapHRError(err)
}

func (r *HRRepo) UpdateOrganization(ctx context.Context, o *domain.Organization) error {
	o.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `UPDATE organization SET parent_id=$1, name=$2, short_name=$3, register_no=$4, address=$5, phone=$6, email=$7, external_id=$8, source=$9, is_active=$10, updated_at=$11 WHERE id=$12 AND deleted_at IS NULL`,
		o.ParentID, o.Name, o.ShortName, o.RegisterNo, o.Address, o.Phone, o.Email, o.ExternalID, o.Source, o.IsActive, o.UpdatedAt, o.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	var linked bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employee WHERE organization_id=$1 AND deleted_at IS NULL)`, id).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `UPDATE organization SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) ListDepartments(ctx context.Context, f domain.HRFilter) ([]*domain.Department, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := "WHERE d.deleted_at IS NULL"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (d.name ILIKE $%d OR d.code ILIKE $%d)", arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.OrganizationID != nil {
		where += fmt.Sprintf(" AND d.organization_id = $%d", arg)
		args = append(args, *f.OrganizationID)
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND d.is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM department d "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT d.id, d.organization_id, COALESCE(o.name,''), d.parent_id, d.code, d.name, d.sort_order, d.is_active, d.created_at, d.updated_at, d.deleted_at
		FROM department d JOIN organization o ON o.id=d.organization_id %s ORDER BY d.sort_order, d.name LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Department
	for rows.Next() {
		d := &domain.Department{}
		if err := rows.Scan(&d.ID, &d.OrganizationID, &d.OrganizationName, &d.ParentID, &d.Code, &d.Name, &d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetDepartmentByID(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	d := &domain.Department{}
	err := r.db.QueryRow(ctx, `SELECT d.id, d.organization_id, COALESCE(o.name,''), d.parent_id, d.code, d.name, d.sort_order, d.is_active, d.created_at, d.updated_at, d.deleted_at
		FROM department d JOIN organization o ON o.id=d.organization_id WHERE d.id=$1 AND d.deleted_at IS NULL`, id).
		Scan(&d.ID, &d.OrganizationID, &d.OrganizationName, &d.ParentID, &d.Code, &d.Name, &d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt)
	return d, mapHRError(err)
}

func (r *HRRepo) CreateDepartment(ctx context.Context, d *domain.Department) error {
	now := time.Now()
	d.ID = uuid.New()
	d.CreatedAt = now
	d.UpdatedAt = now
	if !d.IsActive {
		d.IsActive = true
	}
	err := r.db.QueryRow(ctx, `INSERT INTO department (id, organization_id, parent_id, code, name, sort_order, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, d.ID, d.OrganizationID, d.ParentID, d.Code, d.Name, d.SortOrder, d.IsActive, d.CreatedAt, d.UpdatedAt).Scan(&d.ID)
	return mapHRError(err)
}

func (r *HRRepo) UpdateDepartment(ctx context.Context, d *domain.Department) error {
	d.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `UPDATE department SET organization_id=$1, parent_id=$2, code=$3, name=$4, sort_order=$5, is_active=$6, updated_at=$7 WHERE id=$8 AND deleted_at IS NULL`,
		d.OrganizationID, d.ParentID, d.Code, d.Name, d.SortOrder, d.IsActive, d.UpdatedAt, d.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	var linked bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employee WHERE department_id=$1 AND deleted_at IS NULL)`, id).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `UPDATE department SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) ListPositions(ctx context.Context, f domain.HRFilter) ([]*domain.Position, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := "WHERE deleted_at IS NULL"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d)", arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM position "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id, code, name, sort_order, is_active, created_at, updated_at, deleted_at FROM position %s ORDER BY sort_order, name LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Position
	for rows.Next() {
		p := &domain.Position{}
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *HRRepo) GetPositionByID(ctx context.Context, id uuid.UUID) (*domain.Position, error) {
	p := &domain.Position{}
	err := r.db.QueryRow(ctx, `SELECT id, code, name, sort_order, is_active, created_at, updated_at, deleted_at FROM position WHERE id=$1 AND deleted_at IS NULL`, id).
		Scan(&p.ID, &p.Code, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	return p, mapHRError(err)
}

func (r *HRRepo) CreatePosition(ctx context.Context, p *domain.Position) error {
	now := time.Now()
	p.ID = uuid.New()
	p.CreatedAt = now
	p.UpdatedAt = now
	if !p.IsActive {
		p.IsActive = true
	}
	err := r.db.QueryRow(ctx, `INSERT INTO position (id, code, name, sort_order, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		p.ID, p.Code, p.Name, p.SortOrder, p.IsActive, p.CreatedAt, p.UpdatedAt).Scan(&p.ID)
	return mapHRError(err)
}

func (r *HRRepo) UpdatePosition(ctx context.Context, p *domain.Position) error {
	p.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `UPDATE position SET code=$1, name=$2, sort_order=$3, is_active=$4, updated_at=$5 WHERE id=$6 AND deleted_at IS NULL`,
		p.Code, p.Name, p.SortOrder, p.IsActive, p.UpdatedAt, p.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeletePosition(ctx context.Context, id uuid.UUID) error {
	var linked bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employee WHERE position_id=$1 AND deleted_at IS NULL)`, id).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `UPDATE position SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) ListPersons(ctx context.Context, f domain.HRFilter) ([]*domain.Person, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := "WHERE deleted_at IS NULL"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (register_no ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d OR legal_name ILIKE $%d)", arg, arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM person "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id, person_type, register_no, last_name, first_name, legal_name, civil_id, birth_date, gender, phone, email, address, external_id, source, is_active, created_at, updated_at, deleted_at, created_by, updated_by
		FROM person %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.Person
	for rows.Next() {
		p := &domain.Person{}
		if err := rows.Scan(&p.ID, &p.PersonType, &p.RegisterNo, &p.LastName, &p.FirstName, &p.LegalName, &p.CivilID, &p.BirthDate, &p.Gender, &p.Phone, &p.Email, &p.Address, &p.ExternalID, &p.Source, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.CreatedBy, &p.UpdatedBy); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func scanPerson(row pgx.Row) (*domain.Person, error) {
	p := &domain.Person{}
	err := row.Scan(&p.ID, &p.PersonType, &p.RegisterNo, &p.LastName, &p.FirstName, &p.LegalName, &p.CivilID, &p.BirthDate, &p.Gender, &p.Phone, &p.Email, &p.Address, &p.ExternalID, &p.Source, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.CreatedBy, &p.UpdatedBy)
	return p, mapHRError(err)
}

func (r *HRRepo) GetPersonByID(ctx context.Context, id uuid.UUID) (*domain.Person, error) {
	return scanPerson(r.db.QueryRow(ctx, `SELECT id, person_type, register_no, last_name, first_name, legal_name, civil_id, birth_date, gender, phone, email, address, external_id, source, is_active, created_at, updated_at, deleted_at, created_by, updated_by FROM person WHERE id=$1 AND deleted_at IS NULL`, id))
}

func insertPerson(ctx context.Context, q hrQuerier, p *domain.Person) error {
	now := time.Now()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Source == "" {
		p.Source = "local"
	}
	if !p.IsActive {
		p.IsActive = true
	}
	err := q.QueryRow(ctx, `INSERT INTO person (id, person_type, register_no, last_name, first_name, legal_name, civil_id, birth_date, gender, phone, email, address, external_id, source, is_active, created_at, updated_at, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) RETURNING id`,
		p.ID, p.PersonType, p.RegisterNo, p.LastName, p.FirstName, p.LegalName, p.CivilID, p.BirthDate, p.Gender, p.Phone, p.Email, p.Address, p.ExternalID, p.Source, p.IsActive, p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy).Scan(&p.ID)
	return mapHRError(err)
}

func (r *HRRepo) CreatePerson(ctx context.Context, p *domain.Person) error {
	return insertPerson(ctx, r.db, p)
}

func (r *HRRepo) UpdatePerson(ctx context.Context, p *domain.Person) error {
	p.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `UPDATE person SET person_type=$1, register_no=$2, last_name=$3, first_name=$4, legal_name=$5, civil_id=$6, birth_date=$7, gender=$8, phone=$9, email=$10, address=$11, external_id=$12, source=$13, is_active=$14, updated_at=$15, updated_by=$16 WHERE id=$17 AND deleted_at IS NULL`,
		p.PersonType, p.RegisterNo, p.LastName, p.FirstName, p.LegalName, p.CivilID, p.BirthDate, p.Gender, p.Phone, p.Email, p.Address, p.ExternalID, p.Source, p.IsActive, p.UpdatedAt, p.UpdatedBy, p.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeletePerson(ctx context.Context, id uuid.UUID) error {
	var linked bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employee WHERE person_id=$1 AND deleted_at IS NULL)`, id).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `UPDATE person SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) ListEmployees(ctx context.Context, f domain.HRFilter) ([]*domain.Employee, int64, error) {
	f = normalizeHRFilter(f)
	args := []any{}
	arg := 1
	where := "WHERE e.deleted_at IS NULL"
	if f.Search != "" {
		where += fmt.Sprintf(" AND (e.employee_no ILIKE $%d OR e.work_email ILIKE $%d OR p.register_no ILIKE $%d OR p.first_name ILIKE $%d OR p.last_name ILIKE $%d OR p.legal_name ILIKE $%d)", arg, arg, arg, arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.OrganizationID != nil {
		where += fmt.Sprintf(" AND e.organization_id = $%d", arg)
		args = append(args, *f.OrganizationID)
		arg++
	}
	if f.DepartmentID != nil {
		where += fmt.Sprintf(" AND e.department_id = $%d", arg)
		args = append(args, *f.DepartmentID)
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND e.is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}
	var total int64
	countQ := `SELECT COUNT(*) FROM employee e JOIN person p ON p.id=e.person_id ` + where
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(employeeSelectSQL()+` %s ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
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

func employeeSelectSQL() string {
	return `SELECT e.id, e.person_id, e.organization_id, COALESCE(o.name,''), e.department_id, COALESCE(d.name,''), e.position_id, COALESCE(pos.name,''), e.employee_no, e.work_email, e.work_phone, e.hired_at, e.released_at, e.is_active, e.created_at, e.updated_at, e.deleted_at, e.created_by, e.updated_by,
		p.id, p.person_type, p.register_no, p.last_name, p.first_name, p.legal_name, p.civil_id, p.birth_date, p.gender, p.phone, p.email, p.address, p.external_id, p.source, p.is_active, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by,
		CASE WHEN p.person_type='legal' THEN COALESCE(p.legal_name,'') ELSE trim(COALESCE(p.last_name,'') || ' ' || COALESCE(p.first_name,'')) END
		FROM employee e
		JOIN person p ON p.id=e.person_id
		JOIN organization o ON o.id=e.organization_id
		JOIN position pos ON pos.id=e.position_id
		LEFT JOIN department d ON d.id=e.department_id`
}

func scanEmployeeRow(rows pgx.Rows, e *domain.Employee, p *domain.Person) error {
	return rows.Scan(&e.ID, &e.PersonID, &e.OrganizationID, &e.OrganizationName, &e.DepartmentID, &e.DepartmentName, &e.PositionID, &e.PositionName, &e.EmployeeNo, &e.WorkEmail, &e.WorkPhone, &e.HiredAt, &e.ReleasedAt, &e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.DeletedAt, &e.CreatedBy, &e.UpdatedBy,
		&p.ID, &p.PersonType, &p.RegisterNo, &p.LastName, &p.FirstName, &p.LegalName, &p.CivilID, &p.BirthDate, &p.Gender, &p.Phone, &p.Email, &p.Address, &p.ExternalID, &p.Source, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.CreatedBy, &p.UpdatedBy, &e.PersonName)
}

func (r *HRRepo) GetEmployeeByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	rows, err := r.db.Query(ctx, employeeSelectSQL()+` WHERE e.id=$1 AND e.deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, domain.ErrNotFound
	}
	e := &domain.Employee{}
	p := &domain.Person{}
	if err := scanEmployeeRow(rows, e, p); err != nil {
		return nil, err
	}
	e.Person = p
	return e, rows.Err()
}

func insertEmployee(ctx context.Context, q hrQuerier, e *domain.Employee) error {
	now := time.Now()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.CreatedAt = now
	e.UpdatedAt = now
	if !e.IsActive {
		e.IsActive = true
	}
	err := q.QueryRow(ctx, `INSERT INTO employee (id, person_id, organization_id, department_id, position_id, employee_no, work_email, work_phone, hired_at, released_at, is_active, created_at, updated_at, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`,
		e.ID, e.PersonID, e.OrganizationID, e.DepartmentID, e.PositionID, e.EmployeeNo, e.WorkEmail, e.WorkPhone, e.HiredAt, e.ReleasedAt, e.IsActive, e.CreatedAt, e.UpdatedAt, e.CreatedBy, e.UpdatedBy).Scan(&e.ID)
	return mapHRError(err)
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
	res, err := r.db.Exec(ctx, `UPDATE employee SET person_id=$1, organization_id=$2, department_id=$3, position_id=$4, employee_no=$5, work_email=$6, work_phone=$7, hired_at=$8, released_at=$9, is_active=$10, updated_at=$11, updated_by=$12 WHERE id=$13 AND deleted_at IS NULL`,
		e.PersonID, e.OrganizationID, e.DepartmentID, e.PositionID, e.EmployeeNo, e.WorkEmail, e.WorkPhone, e.HiredAt, e.ReleasedAt, e.IsActive, e.UpdatedAt, e.UpdatedBy, e.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HRRepo) DeleteEmployee(ctx context.Context, id uuid.UUID) error {
	var linked bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE employee_id=$1 AND deleted_at IS NULL)`, id).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return domain.ErrAlreadyExists
	}
	res, err := r.db.Exec(ctx, `UPDATE employee SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
