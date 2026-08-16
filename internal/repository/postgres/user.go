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

// UserRepo — sdplatform.sd_user дээр ШУУД ажиллана.
//
// sd_user-т албан тушаал болон ажилтны холбоос байхгүй тул тэдгээрийг
// hr_employee.user_id → hr_position-оор гаргаж авна (уншихад зориулсан).
type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

const userSelectSQL = `
	SELECT u.user_id, COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(u.password, ''),
	       COALESCE(u.firstname, ''), COALESCE(u.lastname, ''),
	       COALESCE(pos.name, ''), e.employee_id, COALESCE(u.is_active, false),
	       COALESCE(u.created_at::timestamptz, NOW()), COALESCE(u.updated_at::timestamptz, NOW()),
	       u.deleted_at::timestamptz` + userFromSQL

const userFromSQL = `
	FROM sdplatform.sd_user u
	LEFT JOIN sdplatform.hr_employee e ON e.user_id = u.user_id AND COALESCE(e.is_active, 1) <> 0
	LEFT JOIN sdplatform.hr_position pos ON pos.position_id = e.position_id`

func scanUser(row rowScanner, u *domain.User) error {
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.Position, &u.EmployeeID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
}

func (r *UserRepo) getBy(ctx context.Context, cond string, arg any) (*domain.User, error) {
	u := &domain.User{}
	err := scanUser(r.db.QueryRow(ctx, userSelectSQL+" WHERE "+cond+" AND u.deleted_at IS NULL", arg), u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	dup, err := hrExists(ctx, r.db, `
		SELECT EXISTS(
			SELECT 1 FROM sdplatform.sd_user
			WHERE deleted_at IS NULL AND (username = LEFT($1, 50) OR email = LEFT($2, 250))
		)`, u.Username, u.Email)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = time.Now()
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_user (username, email, password, firstname, lastname, created_at, updated_at, is_active)
		VALUES (LEFT($1,50), LEFT($2,250), LEFT($3,150), LEFT($4,50), LEFT($5,50), $6::date, $7::timestamp, $8)
		RETURNING user_id`,
		u.Username, u.Email, u.Password, u.FirstName, u.LastName, u.CreatedAt, u.UpdatedAt, u.IsActive).Scan(&u.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return r.linkEmployee(ctx, r.db, u.ID, u.EmployeeID)
}

// linkEmployee — хэрэглэгч ↔ ажилтны холбоос нь hr_employee.user_id дээр байна.
func (r *UserRepo) linkEmployee(ctx context.Context, q hrQuerier, userID domain.ID, employeeID *domain.ID) error {
	if _, err := q.Exec(ctx,
		`UPDATE sdplatform.hr_employee SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if employeeID == nil {
		return nil
	}
	res, err := q.Exec(ctx,
		`UPDATE sdplatform.hr_employee SET user_id = $1 WHERE employee_id = $2`, userID, *employeeID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrInvalidInput
	}
	return nil
}

func (r *UserRepo) CreateWithEmployee(ctx context.Context, u *domain.User, e *domain.Employee, p *domain.Person) error {
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
	if e != nil {
		if err := insertEmployee(ctx, tx, e); err != nil {
			return err
		}
		u.EmployeeID = &e.ID
		if err := fillUserFromEmployee(ctx, tx, u, e.ID); err != nil {
			return err
		}
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = time.Now()
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_user (username, email, password, firstname, lastname, created_at, updated_at, is_active)
		VALUES (LEFT($1,50), LEFT($2,250), LEFT($3,150), LEFT($4,50), LEFT($5,50), $6::date, $7::timestamp, $8)
		RETURNING user_id`,
		u.Username, u.Email, u.Password, u.FirstName, u.LastName, u.CreatedAt, u.UpdatedAt, u.IsActive).Scan(&u.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return mapHRError(err)
	}
	if u.EmployeeID != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE sdplatform.hr_employee SET user_id = $1 WHERE employee_id = $2`, u.ID, *u.EmployeeID); err != nil {
			return mapHRError(err)
		}
	}
	return tx.Commit(ctx)
}

func fillUserFromEmployee(ctx context.Context, q hrQuerier, u *domain.User, employeeID domain.ID) error {
	var personType, positionName string
	var lastName, firstName, legalName *string
	err := q.QueryRow(ctx, `
		SELECT `+personTypeExpr+`, `+personLastNameExpr+`, p.first_name, `+personLegalNameExpr+`, COALESCE(pos.name, '')
		FROM sdplatform.hr_employee e
		JOIN base.bs_person p ON p.person_id = e.person_id
		JOIN sdplatform.hr_position pos ON pos.position_id = e.position_id
		WHERE e.employee_id = $1`, employeeID).
		Scan(&personType, &lastName, &firstName, &legalName, &positionName)
	if err != nil {
		return mapHRError(err)
	}
	u.Position = positionName
	if personType == "legal" {
		u.LastName = "Хуулийн этгээд"
		if legalName != nil {
			u.FirstName = *legalName
		}
		return nil
	}
	if lastName != nil {
		u.LastName = *lastName
	}
	if firstName != nil {
		u.FirstName = *firstName
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	u, err := r.getBy(ctx, "u.user_id = $1", id)
	if err != nil {
		return nil, err
	}
	if err := r.attachRoles(ctx, []*domain.User{u}); err != nil {
		return nil, err
	}
	if err := r.attachEmployees(ctx, []*domain.User{u}); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getBy(ctx, "u.email = $1", email)
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.getBy(ctx, "u.username = $1", username)
}

func (r *UserRepo) GetByEmployeeID(ctx context.Context, employeeID domain.ID) (*domain.User, error) {
	return r.getBy(ctx, "e.employee_id = $1", employeeID)
}

// attachRoles — хэрэглэгчдийн ролиудыг НЭГ хүсэлтээр татна (N+1-ээс сэргийлнэ).
func (r *UserRepo) attachRoles(ctx context.Context, users []*domain.User) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[domain.ID]*domain.User, len(users))
	ids := make([]domain.ID, 0, len(users))
	for _, u := range users {
		byID[u.ID] = u
		ids = append(ids, u.ID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT ur.user_id, r.role_id, COALESCE(r.code, ''), COALESCE(r.name, ''),
		       COALESCE(r.role_type_code, ''), COALESCE(r.description, '')
		FROM sdplatform.sd_user_role ur
		JOIN sdplatform.sd_role r ON r.role_id = ur.role_id
		WHERE ur.user_id = ANY($1)
		ORDER BY r.name`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID domain.ID
		var role domain.Role
		if err := rows.Scan(&userID, &role.ID, &role.Code, &role.Name, &role.Resource, &role.Description); err != nil {
			return err
		}
		if u, ok := byID[userID]; ok {
			u.Roles = append(u.Roles, role)
		}
	}
	return rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	dup, err := hrExists(ctx, r.db, `
		SELECT EXISTS(
			SELECT 1 FROM sdplatform.sd_user
			WHERE deleted_at IS NULL AND user_id <> $3
			  AND (username = LEFT($1, 50) OR email = LEFT($2, 250))
		)`, u.Username, u.Email, u.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}
	u.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE sdplatform.sd_user
		SET username=LEFT($1,50), email=LEFT($2,250), firstname=LEFT($3,50), lastname=LEFT($4,50),
		    is_active=$5, updated_at=$6::timestamp
		WHERE user_id=$7 AND deleted_at IS NULL`,
		u.Username, u.Email, u.FirstName, u.LastName, u.IsActive, u.UpdatedAt, u.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return r.linkEmployee(ctx, r.db, u.ID, u.EmployeeID)
}

// UpdatePassword — зөвхөн нууц үгийн hash-ийг шинэчилнэ.
func (r *UserRepo) UpdatePassword(ctx context.Context, id domain.ID, hashed string) error {
	res, err := r.db.Exec(ctx,
		`UPDATE sdplatform.sd_user SET password=LEFT($1,150), updated_at=$2::timestamp
		 WHERE user_id=$3 AND deleted_at IS NULL`, hashed, time.Now(), id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id domain.ID) error {
	res, err := r.db.Exec(ctx,
		`UPDATE sdplatform.sd_user SET deleted_at=$1::timestamp WHERE user_id=$2 AND deleted_at IS NULL`,
		time.Now(), id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, f domain.UserFilter) ([]*domain.User, int64, error) {
	args := []any{}
	argIdx := 1
	where := "WHERE u.deleted_at IS NULL"

	if f.Search != "" {
		where += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.email ILIKE $%d OR u.firstname ILIKE $%d OR u.lastname ILIKE $%d OR pos.name ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	if f.Role != "" {
		// Роль нь "<resource>_<түвшин>" нэртэй тул нэрээр ч, түвшингээр ч хайж болно.
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM sdplatform.sd_user_role ur
			JOIN sdplatform.sd_role r ON r.role_id = ur.role_id
			WHERE ur.user_id = u.user_id
			  AND (r.name = $%d OR r.name = COALESCE(r.role_type_code, '') || '_' || $%d)
		)`, argIdx, argIdx)
		args = append(args, f.Role)
		argIdx++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND COALESCE(u.is_active, false) = $%d", argIdx)
		args = append(args, *f.IsActive)
		argIdx++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+userFromSQL+" "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(userSelectSQL+` %s ORDER BY u.created_at DESC NULLS LAST LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := scanUser(rows, u); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.attachRoles(ctx, users); err != nil {
		return nil, 0, err
	}
	if err := r.attachEmployees(ctx, users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepo) attachEmployees(ctx context.Context, users []*domain.User) error {
	if len(users) == 0 {
		return nil
	}
	byEmployeeID := make(map[domain.ID]*domain.User)
	ids := make([]domain.ID, 0, len(users))
	for _, u := range users {
		if u.EmployeeID != nil {
			byEmployeeID[*u.EmployeeID] = u
			ids = append(ids, *u.EmployeeID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.db.Query(ctx, employeeSelectSQL()+` WHERE e.employee_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		e := &domain.Employee{}
		p := &domain.Person{}
		if err := scanEmployeeRow(rows, e, p); err != nil {
			return err
		}
		e.Person = p
		if u := byEmployeeID[e.ID]; u != nil {
			u.Employee = e
		}
	}
	return rows.Err()
}

// InvalidateSessions — sd_user-т токены эх цэг хадгалах багана байхгүй тул
// энэ нь аппын өөрийн туслах хүснэгт (revoked_tokens-той ижил ангилал).
func (r *UserRepo) InvalidateSessions(ctx context.Context, userID domain.ID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sdplatform.user_token_epoch (user_id, valid_from)
		VALUES ($1, date_trunc('second', NOW()))
		ON CONFLICT (user_id) DO UPDATE SET valid_from = EXCLUDED.valid_from`, userID)
	return err
}

// IsSessionValid — эх цэг тавиагүй бол true; эх цэгээс өмнө олгогдсон токенд false.
func (r *UserRepo) IsSessionValid(ctx context.Context, userID domain.ID, issuedAt time.Time) (bool, error) {
	var invalid bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sdplatform.user_token_epoch
			WHERE user_id = $1 AND valid_from > $2
		)`, userID, issuedAt).Scan(&invalid)
	if err != nil {
		return false, err
	}
	return !invalid, nil
}

// GetRoles — JWT-ийн `roles` нэхэмжлэлд ролийн ТҮВШИН орно (land_admin → admin)
// тул "admin" эсэхийг шалгадаг маршрутууд өмнөх шигээ ажиллана.
func (r *UserRepo) GetRoles(ctx context.Context, userID domain.ID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(r.name, ''), COALESCE(r.role_type_code, '')
		FROM sdplatform.sd_role r
		JOIN sdplatform.sd_user_role ur ON ur.role_id = r.role_id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	var levels []string
	for rows.Next() {
		var name, resource string
		if err := rows.Scan(&name, &resource); err != nil {
			return nil, err
		}
		level := domain.Role{Name: name, Resource: resource}.Level()
		if level != "" && !seen[level] {
			seen[level] = true
			levels = append(levels, level)
		}
	}
	return levels, rows.Err()
}

// GetPermissions — эрх нь ролийн нөөц + эрхийн үйлдэл: "land:read".
func (r *UserRepo) GetPermissions(ctx context.Context, userID domain.ID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT COALESCE(r.role_type_code, ''), COALESCE(p.name, '')
		FROM sdplatform.sd_permission p
		JOIN sdplatform.sd_role_permission rp ON rp.permission_id = p.permission_id
		JOIN sdplatform.sd_role r ON r.role_id = rp.role_id
		JOIN sdplatform.sd_user_role ur ON ur.role_id = r.role_id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	var perms []string
	for rows.Next() {
		var resource, action string
		if err := rows.Scan(&resource, &action); err != nil {
			return nil, err
		}
		if action == "" {
			continue
		}
		key := domain.PermissionKey(resource, action)
		if !seen[key] {
			seen[key] = true
			perms = append(perms, key)
		}
	}
	return perms, rows.Err()
}
