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

// Хэрэглэгчийн эх сурвалж: sdplatform DB (дотоод ажилтан) эсвэл аппын өөрийн
// DB (гадаад байгууллага). Хоёулаа тус тусын serial4 дараалалтай тул ID нь
// мөргөлдөж болзошгүй — аппын төлөвийн хүснэгтүүдэд энэ нь түлхүүрийн хэсэг.
const (
	userSourceAuth = "auth"
	userSourceMain = "main"
)

// UserRepo — sdplatform.sd_user дээр ШУУД ажиллана.
//
// sd_user-т албан тушаал болон ажилтны холбоос байхгүй тул тэдгээрийг
// hr_employee.user_id → hr_position-оор гаргаж авна (уншихад зориулсан).
//
// appDB — аппын ӨӨРИЙН өгөгдлийн сан. sd_user-т токены эх цэг хадгалах багана
// байхгүй бөгөөд sdplatform схемд хүснэгт нэмэхийг хориглосон тул сессийн
// эх цэг тэнд биш, энд хадгалагдана.
type UserRepo struct {
	db     *pgxpool.Pool
	appDB  *pgxpool.Pool
	source string
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db, appDB: db, source: userSourceAuth}
}

// NewUserRepoFor — sdplatform холболт (db) болон аппын өөрийн сан (appDB)-г
// тусад нь өгнө.
func NewUserRepoFor(source string, db, appDB *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db, appDB: appDB, source: source}
}

// Source — токен болон user_token_epoch-д бичигдэх эх сурвалжийн тэмдэг.
func (r *UserRepo) Source() string {
	if r.source == "" {
		return domain.UserSourceAuth
	}
	return r.source
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
				WHERE deleted_at IS NULL
				  AND (
				      username = LEFT($1, 50)
				      OR (NULLIF(BTRIM($2::text), '') IS NOT NULL AND email = LEFT(BTRIM($2::text), 250))
				  )
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
			VALUES (LEFT($1,50), LEFT(NULLIF(BTRIM($2::text), ''),250), LEFT($3,150), LEFT($4,50), LEFT($5,50), $6::date, $7::timestamp, $8)
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
			VALUES (LEFT($1,50), LEFT(NULLIF(BTRIM($2::text), ''),250), LEFT($3,150), LEFT($4,50), LEFT($5,50), $6::date, $7::timestamp, $8)
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

// GetActiveByLogin — НЭВТРЭХЭД зориулсан хайлт: нэвтрэх нэр ЭСВЭЛ имэйлээр
// хайх ба ЗӨВХӨН идэвхтэй хэрэглэгчийг олно.
//
// Яагаад тусдаа метод вэ: GetByUsername/GetByEmail нь seed-ийн давхардлын
// шалгалт, хэрэглэгч үүсгэхийн өмнөх шалгалт, нууц үг сэргээх зэрэгт
// хэрэглэгддэг бөгөөд тэдгээрт ИДЭВХГҮЙ хэрэглэгч ч олдох ёстой (эс бөгөөс
// идэвхгүй хүнтэй ижил нэрээр давхардсан бүртгэл үүсгэх, эсвэл идэвхгүй
// болгосон хүнээ буцааж идэвхжүүлэх боломжгүй болно).
//
// Нэвтрэх үед идэвхгүй хэрэглэгч ОГТ ОЛДОХГҮЙ байх нь "бүртгэл байгаа эсэх"-ийг
// гаднаас таамаглах боломжийг ч хаана (403 биш 401 буцна).
func (r *UserRepo) GetActiveByLogin(ctx context.Context, login string) (*domain.User, error) {
	return r.getBy(ctx, "(u.username = $1 OR u.email = $1) AND COALESCE(u.is_active, false) = true", login)
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
		JOIN sdplatform.sd_system_role sr ON sr.role_id = r.role_id AND sr.system_id = 10
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
				  AND (
				      username = LEFT($1, 50)
				      OR (NULLIF(BTRIM($2::text), '') IS NOT NULL AND email = LEFT(BTRIM($2::text), 250))
				  )
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
			SET username=LEFT($1,50), email=LEFT(NULLIF(BTRIM($2::text), ''),250), firstname=LEFT($3,50), lastname=LEFT($4,50),
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
			JOIN sdplatform.sd_system_role sr ON sr.role_id = r.role_id AND sr.system_id = 10
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
	if f.DepartmentID != nil {
		where += fmt.Sprintf(" AND e.department_id = $%d", argIdx)
		args = append(args, *f.DepartmentID)
		argIdx++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+userFromSQL+" "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(userSelectSQL+` %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		where,
		orderClause(userSortColumns, f.SortBy, f.SortDir, "u.created_at DESC NULLS LAST", "u.user_id"),
		argIdx, argIdx+1), args...)
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

// OrgIDForUser — хэрэглэгчийн ХАРЬЯА байгууллага (JWT-ийн org_id).
//
// ЭХЛЭЭД appdb-ийн valuation_org_employee-г хайна. Энэ нэхэмжлэл нь
// land_acquisition.professional_org_id / parcel.independent_org_id-тай
// тулгагддаг ба тэдгээр нь valuation_org_profile.id-г заадаг тул org_id нь ЯГ
// ТЭР дараалалаас гарах ёстой.
//
// Олдохгүй бол sdplatform-ийн HR бүтэц (hr_employee → hr_department.
// organization = sd_organization.id) руу шилжинэ — mika / finance_specialist
// зэрэг үнэлгээний байгууллагын бүртгэлд байхгүй гадаад ролиудын хувьд өмнөх
// зан төлөв хэвээр үлдэнэ.
//
// АНХААР: HR-ийн замаар гарсан ID нь ӨӨР дараалалынх тул professional_org_id-
// тай санамсаргүй тааралдаж болзошгүй. Үнэлгээний байгууллагын ажилтныг
// ҮРГЭЛЖ /valuation-orgs цонхоор бүртгэ — тэгвэл дээрх нэгдүгээр зам ажиллаж,
// HR fallback хэрэглэгдэхгүй.
//
// Байгууллагад харьяалагдаагүй хэрэглэгчид nil (нэвтрэлт ЗОГСОХГҮЙ).
func (r *UserRepo) OrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error) {
	if orgID, err := r.ValuationOrgIDForUser(ctx, userID); err != nil {
		return nil, err
	} else if orgID != nil {
		return orgID, nil
	}

	var orgID *domain.ID
	err := r.db.QueryRow(ctx, `
		SELECT d.organization
		FROM sdplatform.hr_employee e
		JOIN sdplatform.hr_department d ON d.department_id = e.department_id
		WHERE e.user_id = $1 AND COALESCE(e.is_active, 1) <> 0
		ORDER BY e.employee_id
		LIMIT 1`, userID).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return orgID, nil
}

// DepartmentIDForUser — хэрэглэгчийн ХЭЛТЭС (JWT-ийн department_id).
// Хэлтсийн дарга болон ажилтны хандалт (зөвхөн өөрийн хэлтсийн хэрэглэгч,
// мэдээлэл) үүн дээр тулгуурлана. Хэлтэсгүй бол nil.
func (r *UserRepo) DepartmentIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error) {
	var deptID *domain.ID
	err := r.db.QueryRow(ctx, `
		SELECT e.department_id
		FROM sdplatform.hr_employee e
		WHERE e.user_id = $1 AND COALESCE(e.is_active, 1) <> 0
		ORDER BY e.employee_id
		LIMIT 1`, userID).Scan(&deptID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return deptID, nil
}

// CountSystemAdmins — системийн админ түвшний (Role.Level) роль-той,
// устгагдаагүй хэрэглэгчийн тоо.
func (r *UserRepo) CountSystemAdmins(ctx context.Context, excludeUserID *domain.ID) (int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ur.user_id, COALESCE(ro.name, ''), COALESCE(ro.role_type_code, '')
		FROM sdplatform.sd_user_role ur
		JOIN sdplatform.sd_user u ON u.user_id = ur.user_id AND u.deleted_at IS NULL
		JOIN sdplatform.sd_role ro ON ro.role_id = ur.role_id
		JOIN sdplatform.sd_system_role sr ON sr.role_id = ro.role_id AND sr.system_id = 10`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	admins := map[domain.ID]bool{}
	for rows.Next() {
		var userID domain.ID
		var name, resource string
		if err := rows.Scan(&userID, &name, &resource); err != nil {
			return 0, err
		}
		if excludeUserID != nil && userID == *excludeUserID {
			continue
		}
		if (domain.Role{Name: name, Resource: resource}).IsSystemAdmin() {
			admins[userID] = true
		}
	}
	return len(admins), rows.Err()
}

// ValuationOrgIDForUser — ЗӨВХӨН appdb-ийн valuation_org_employee-г хайна
// (HR fallback БАЙХГҮЙ). Иймд буцах ID нь ҮРГЭЛЖ valuation_org_profile.id
// бөгөөд professional_org_id / independent_org_id-тай шууд тулгагдана.
//
// Токены `valuation_org` нэхэмжлэл ба түүнээс хамаарах бүх хаалт (цэс,
// маршрутын allowlist, /prof) ЯГ ЭНЭ нэхэмжлэл дээр тогтоно — тиймээс HR-ийн
// замаар гарсан санамсаргүй таарсан ID эндээс ХЭЗЭЭ Ч гарахгүй байх ёстой.
func (r *UserRepo) ValuationOrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error) {
	if r.appDB == nil {
		return nil, nil
	}
	var orgID *domain.ID
	err := r.appDB.QueryRow(ctx, `
		SELECT org_id
		FROM valuation_org_employee
		WHERE user_id = $1 AND is_active
		ORDER BY id
		LIMIT 1`, userID).Scan(&orgID)
	switch {
	case err == nil:
		return orgID, nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	default:
		return nil, err
	}
}

// InvalidateSessions — sd_user-т токены эх цэг хадгалах багана байхгүй тул
// аппын өөрийн сан дахь user_token_epoch хүснэгтэд бичнэ.
func (r *UserRepo) InvalidateSessions(ctx context.Context, userID domain.ID) error {
	_, err := r.appDB.Exec(ctx, `
		INSERT INTO user_token_epoch (user_source, user_id, valid_from)
		VALUES ($2, $1, date_trunc('second', NOW()))
		ON CONFLICT (user_source, user_id) DO UPDATE SET valid_from = EXCLUDED.valid_from`,
		userID, r.source)
	return err
}

// IsSessionValid — эх цэг тавиагүй бол true; эх цэгээс өмнө олгогдсон токенд false.
//
// source нь ТОКЕНООС ирнэ (r.source БИШ): user_token_epoch нь appdb-д байрладаг
// тул энэ нэг шалгагч хоёр эх сурвалжийг (auth / valuation_org) хоёуланг нь
// үйлчилнэ. Хоёр сангийн user_id мөргөлддөг тул source-ыг орхиж БОЛОХГҮЙ.
func (r *UserRepo) IsSessionValid(ctx context.Context, source string, userID domain.ID, issuedAt time.Time) (bool, error) {
	if source == "" {
		source = domain.UserSourceAuth
	}
	var invalid bool
	err := r.appDB.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_token_epoch
			WHERE user_source = $3 AND user_id = $1 AND valid_from > $2
		)`, userID, issuedAt, source).Scan(&invalid)
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
		JOIN sdplatform.sd_system_role sr ON sr.role_id = r.role_id AND sr.system_id = 10
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
		SELECT DISTINCT resource, action
		FROM (
			SELECT COALESCE(m.code, '') AS resource, COALESCE(p.name, '') AS action
			FROM sdplatform.sd_user_role ur
			JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
			JOIN sdplatform.sd_role_permission rp ON rp.role_id = ur.role_id
			JOIN sdplatform.sd_menu m ON m.menu_id = rp.menu_id
			JOIN sdplatform.sd_menu_form mf ON mf.menu_id = m.menu_id OR mf.menu_code = m.code
			JOIN sdplatform.sd_form f ON f.form_id = mf.form_id OR f.code = mf.form_code
			JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
				AND (
					fp.permission_id = p.permission_id
					OR fp.permission_code = p.code
					OR fp.permission_code = p.name
				)
			WHERE ur.user_id = $1
			UNION ALL
			SELECT COALESCE(r.role_type_code, '') AS resource, COALESCE(p.name, '') AS action
			FROM sdplatform.sd_user_role ur
			JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
			JOIN sdplatform.sd_role r ON r.role_id = ur.role_id
			JOIN sdplatform.sd_role_permission rp ON rp.role_id = ur.role_id AND rp.menu_id IS NULL
			JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
			WHERE ur.user_id = $1
		) q`, userID)
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
		key := domain.MenuPermissionKey(resource, action)
		if !seen[key] {
			seen[key] = true
			perms = append(perms, key)
		}
	}
	return perms, rows.Err()
}

// InvalidateRoleSessions — роль-той бүх хэрэглэгчийн токеныг хүчингүй болгоно.
//
// Эрх нь JWT-д агуулагддаг тул ролоос эрх хассаны дараа эзэмшигчид хуучин
// токеноороо (хугацаа дуустал) тэр эрхийг хэрэглэсээр байдаг байв. Гишүүнчлэл
// authdb-ээс УНШИГДАНА (өөрчлөхгүй); эх цэг нь appdb-ийн user_token_epoch.
func (r *UserRepo) InvalidateRoleSessions(ctx context.Context, roleID domain.ID) error {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT user_id FROM sdplatform.sd_user_role WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	var ids []int32
	for rows.Next() {
		var id domain.ID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, int32(id))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	_, err = r.appDB.Exec(ctx, `
		INSERT INTO user_token_epoch (user_source, user_id, valid_from)
		SELECT $2, u, date_trunc('second', NOW()) FROM unnest($1::int[]) AS u
		ON CONFLICT (user_source, user_id) DO UPDATE SET valid_from = EXCLUDED.valid_from`,
		ids, r.source)
	return err
}
