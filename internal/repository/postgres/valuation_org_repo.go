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
// Байгууллага БОЛОН ажилтан appdb-д амьдарна:
//
//	valuation_org_profile   → байгууллага
//	valuation_org_employee  → ажилтан (org_id → profile.id)
//
// Ажилтныг sdplatform-ийн HR бүтцэд БИШ appdb-д хадгалах нь зориудын шийдэл:
// HR нь байгууллагыг sd_organization.id-гаар таньдаг бол чөлөөлөлт/нэгж талбар
// нь valuation_org_profile.id-гаар таньдаг. 000002 миграцаас хойш хоёр дараалал
// салсан тул HR-аар дамжуулбал токены org_id нь professional_org_id-тай хэзээ ч
// таарахгүй байв (ажилтан нэвтэрч чадах ч ажлаа харж чадахгүй).
//
// sdplatform-д ЗӨВХӨН нэвтрэлтийн МӨР нэмэгдэнэ (sd_user); хүснэгтийн бүтэц,
// HR-ийн байгууллага/хэлтэс/ажилтан хөндөгдөхгүй. РОЛЬ ОНООГДОХГҮЙ —
// хандалт нь токены `valuation_org` тэмдэг ба аппын allowlist-ээр шийдэгдэнэ
// (internal/middleware/valuation_org.go).
//
// Хоёр САНГИЙН хооронд нэг гүйлгээ байх боломжгүй тул дараалал нь:
//  1. authdb — нэвтрэх эрхийг бэлдэж user_id-г шийднэ;
//  2. appdb  — профайл + ажилтныг НЭГ гүйлгээнд бичнэ;
//  3. authdb — хасагдсан/сэргэсэн эрхийн төлөвийг тэнцүүлнэ.
//
// 2-т алдаа гарвал 1-д үүссэн sd_user мөр эзэнгүй үлдэж болно — дараагийн
// хадгалалт түүнийг username-ээр НЬ дахин ашиглана (шинэ мөр үүсгэхгүй).
type ValuationOrgRepo struct {
	db     *pgxpool.Pool
	authDB *pgxpool.Pool
}

func NewValuationOrgRepo(db, authDB *pgxpool.Pool) *ValuationOrgRepo {
	return &ValuationOrgRepo{db: db, authDB: authDB}
}

const valuationOrgSelectSQL = `
	SELECT vp.id, COALESCE(vp.name, ''), vp.short_name, vp.register_no,
	       vp.license_no, vp.license_issued_at, vp.license_expires_at,
	       vp.phone, vp.email, vp.address, vp.note, vp.is_active,
	       vp.created_at, vp.updated_at,
	       (SELECT COUNT(*) FROM valuation_org_employee ve
	         WHERE ve.org_id = vp.id AND ve.is_active)
	FROM valuation_org_profile vp`

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
		where += fmt.Sprintf(` AND (vp.name ILIKE $%d OR vp.short_name ILIKE $%d OR vp.register_no ILIKE $%d OR vp.license_no ILIKE $%d)`, arg, arg, arg, arg)
		args = append(args, "%"+f.Search+"%")
		arg++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND vp.is_active = $%d", arg)
		args = append(args, *f.IsActive)
		arg++
	}

	var total int64
	countSQL := `SELECT COUNT(*) FROM valuation_org_profile vp ` + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(valuationOrgSelectSQL+` %s ORDER BY vp.name LIMIT $%d OFFSET $%d`, where, arg, arg+1), args...)
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

// GetByID — байгууллагыг АЖИЛТНУУДЫНХ НЬ хамт буцаана (жагсаалтад ажилтан
// ирэхгүй; засварлах цонх үүнийг дуудна).
func (r *ValuationOrgRepo) GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error) {
	v := &domain.ValuationOrg{}
	if err := scanValuationOrg(r.db.QueryRow(ctx, valuationOrgSelectSQL+` WHERE vp.id = $1`, id), v); err != nil {
		return nil, mapHRError(err)
	}
	emps, err := r.listEmployees(ctx, id)
	if err != nil {
		return nil, err
	}
	v.Employees = emps
	return v, nil
}

// listEmployees — байгууллагын ажилтнууд. Идэвхгүй болсон ажилтан ч ирнэ
// (мөрийг устгадаггүй тул түүх хадгалагдана) — идэвхтэй нь эхэлж жагсана.
func (r *ValuationOrgRepo) listEmployees(ctx context.Context, orgID domain.ID) ([]*domain.Employee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, last_name, first_name, register_no, position_name,
		       phone, email, is_active, created_at, updated_at
		FROM valuation_org_employee
		WHERE org_id = $1
		ORDER BY is_active DESC, last_name, first_name, id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.Employee, 0)
	for rows.Next() {
		var (
			e                                         domain.Employee
			lastName, firstName, registerNo, position string
			phone, email                              string
		)
		if err := rows.Scan(&e.ID, &e.UserID, &lastName, &firstName, &registerNo,
			&position, &phone, &email, &e.IsActive, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		org := orgID
		e.OrganizationID = &org
		e.PositionName = position
		e.WorkPhone = optionalText(phone)
		e.WorkEmail = optionalText(email)
		// Овог/нэр/регистрийг Person дотор буцаана — HR-ийн ажилтны хариулттай
		// ижил хэлбэртэй байх нь UI-д нэг л mapper хэрэглэх боломж өгнө.
		e.Person = &domain.Person{
			PersonType: "citizen",
			RegisterNo: registerNo,
			LastName:   optionalText(lastName),
			FirstName:  optionalText(firstName),
			Phone:      optionalText(phone),
			Email:      optionalText(email),
		}
		e.PersonName = e.Person.DisplayName()
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.fillUsernames(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// fillUsernames — ажилтнуудын нэвтрэх нэрийг authdb-аас нөхнө.
//
// sd_user нь ӨӨР САНД байдаг тул join хийх боломжгүй — user_id-уудаар нэг
// хүсэлт болгож татна. Нэр олдохгүй (эрх устсан) бол хоосон үлдэнэ: ажилтны
// жагсаалт үүнээс болж унах ёсгүй.
func (r *ValuationOrgRepo) fillUsernames(ctx context.Context, emps []*domain.Employee) error {
	ids := make([]domain.ID, 0, len(emps))
	for _, e := range emps {
		if e.UserID != nil {
			ids = append(ids, *e.UserID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := r.authDB.Query(ctx,
		`SELECT user_id, COALESCE(username, '') FROM sdplatform.sd_user WHERE user_id = ANY($1::int4[])`, ids)
	if err != nil {
		return mapHRError(err)
	}
	defer rows.Close()

	names := make(map[domain.ID]string, len(ids))
	for rows.Next() {
		var (
			id       domain.ID
			username string
		)
		if err := rows.Scan(&id, &username); err != nil {
			return err
		}
		names[id] = username
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range emps {
		if e.UserID != nil {
			e.Username = names[*e.UserID]
		}
	}
	return nil
}

// Create — байгууллагыг (ба ажилтныг) appdb-д үүсгэнэ.
func (r *ValuationOrgRepo) Create(ctx context.Context, v *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	logins, err := r.ensureLogins(ctx, emps)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	dup, err := hrExists(ctx, tx,
		`SELECT EXISTS(SELECT 1 FROM valuation_org_profile WHERE name = LEFT($1, 250))`, v.Name)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}

	now := time.Now()
	if err := tx.QueryRow(ctx, `
		INSERT INTO valuation_org_profile (
			name, short_name, register_no, license_no, license_issued_at,
			license_expires_at, phone, email, address, note, is_active,
			created_at, updated_at)
		VALUES (LEFT($1, 250), LEFT($2, 250), LEFT($3, 20), LEFT($4, 50), $5,
			$6, LEFT($7, 50), LEFT($8, 250), LEFT($9, 500), $10, $11, $12, $12)
		RETURNING id`, v.Name, v.ShortName, v.RegisterNo, v.LicenseNo,
		v.LicenseIssuedAt, v.LicenseExpiresAt, v.Phone, v.Email, v.Address, v.Note,
		v.IsActive, now).Scan(&v.ID); err != nil {
		return mapHRError(err)
	}
	v.CreatedAt = now
	v.UpdatedAt = now

	sync, err := syncValuationEmployees(ctx, tx, v.ID, emps, logins)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return r.applyLoginState(ctx, sync)
}

// Update — байгууллагын мэдээлэл (ба emps != nil бол ажилтнууд) шинэчилнэ.
//
// ЭНЭ ФУНКЦ ШИНЭ БАЙГУУЛЛАГА ҮҮСГЭХГҮЙ: мөр олдохгүй бол ErrNotFound.
func (r *ValuationOrgRepo) Update(ctx context.Context, v *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	logins, err := r.ensureLogins(ctx, emps)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	dup, err := hrExists(ctx, tx,
		`SELECT EXISTS(SELECT 1 FROM valuation_org_profile WHERE name = LEFT($1, 250) AND id <> $2)`, v.Name, v.ID)
	if err != nil {
		return err
	}
	if dup {
		return domain.ErrAlreadyExists
	}

	now := time.Now()
	res, err := tx.Exec(ctx, `
		UPDATE valuation_org_profile
		SET name = LEFT($1, 250), short_name = LEFT($2, 250), register_no = LEFT($3, 20),
			license_no = LEFT($4, 50), license_issued_at = $5, license_expires_at = $6,
			phone = LEFT($7, 50), email = LEFT($8, 250), address = LEFT($9, 500),
			note = $10, is_active = $11, updated_at = $12
		WHERE id = $13`, v.Name, v.ShortName, v.RegisterNo, v.LicenseNo,
		v.LicenseIssuedAt, v.LicenseExpiresAt, v.Phone, v.Email, v.Address, v.Note,
		v.IsActive, now, v.ID)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	v.UpdatedAt = now

	sync, err := syncValuationEmployees(ctx, tx, v.ID, emps, logins)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return r.applyLoginState(ctx, sync)
}

// Delete — чөлөөлөлт/нэгж талбарт холбогдоогүй байгууллагыг л устгана.
//
// Ажилтнууд FK-ийн ON DELETE CASCADE-ээр устах тул нэвтрэх эрхийг нь ЭХЛЭЭД
// хааж, токеныг хүчингүй болгоно — эс бөгөөс байгууллага байхгүй болсон ч
// ажилтан нэвтэрсээр байна.
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

	userIDs, err := collectEmployeeUserIDs(ctx, tx, `
		SELECT user_id FROM valuation_org_employee
		WHERE org_id = $1 AND user_id IS NOT NULL`, id)
	if err != nil {
		return err
	}
	if err := invalidateUserSessions(ctx, tx, userIDs); err != nil {
		return err
	}

	res, err := tx.Exec(ctx, `DELETE FROM valuation_org_profile WHERE id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return r.applyLoginState(ctx, loginSync{deactivate: userIDs})
}

// ---------------------------------------------------------------------------
// Ажилтан
// ---------------------------------------------------------------------------

// loginSync — appdb-ийн гүйлгээ commit болсны дараа authdb-д тэнцүүлэх төлөв.
type loginSync struct {
	activate   []domain.ID
	deactivate []domain.ID
}

// syncValuationEmployees — байгууллагын ажилтнуудын БҮРЭН төлөвийг тавина.
//
// emps == nil бол ажилтнуудад ХҮРЭХГҮЙ (зөвхөн байгууллагын мэдээлэл засварласан
// тохиолдол). Хоосон биш бол жагсаалтад ороогүй идэвхтэй ажилтан идэвхгүй болно
// — мөр УСТГАХГҮЙ тул түүх, баримт дээрх нэр хадгалагдана.
func syncValuationEmployees(
	ctx context.Context,
	tx pgx.Tx,
	orgID domain.ID,
	emps []domain.ValuationOrgEmployee,
	logins map[int]domain.ID,
) (loginSync, error) {
	var sync loginSync
	if emps == nil {
		return sync, nil
	}

	keep := make([]domain.ID, 0, len(emps))
	for i := range emps {
		id, err := saveValuationEmployee(ctx, tx, orgID, &emps[i], logins[i])
		if err != nil {
			return sync, err
		}
		keep = append(keep, id)
		if uid, ok := logins[i]; ok {
			sync.activate = append(sync.activate, uid)
		}
	}

	// Жагсаалтад ороогүй ажилтныг идэвхгүй болгож, нэвтрэх эрхийг нь хаана.
	// keep хоосон байж болзошгүй тул query-г тусад нь салгана (<> ALL('{}')
	// нь ҮРГЭЛЖ true болох тул зөв, гэхдээ уншихад бүрхэг).
	var (
		rows pgx.Rows
		err  error
		now  = time.Now()
	)
	if len(keep) == 0 {
		rows, err = tx.Query(ctx, `
			UPDATE valuation_org_employee
			SET is_active = false, updated_at = $2
			WHERE org_id = $1 AND is_active
			RETURNING user_id`, orgID, now)
	} else {
		rows, err = tx.Query(ctx, `
			UPDATE valuation_org_employee
			SET is_active = false, updated_at = $3
			WHERE org_id = $1 AND is_active AND id <> ALL($2::int4[])
			RETURNING user_id`, orgID, keep, now)
	}
	if err != nil {
		return sync, mapHRError(err)
	}
	for rows.Next() {
		var uid *domain.ID
		if err := rows.Scan(&uid); err != nil {
			rows.Close()
			return sync, err
		}
		if uid != nil {
			sync.deactivate = append(sync.deactivate, *uid)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sync, err
	}

	// Байгууллагаас хасагдсан ажилтны олгогдсон токен ч хүчингүй болно.
	if err := invalidateUserSessions(ctx, tx, sync.deactivate); err != nil {
		return sync, err
	}
	return sync, nil
}

// saveValuationEmployee — нэг ажилтныг бичиж, valuation_org_employee.id-г
// буцаана.
//
// userID нь 0 бол нэвтрэх эрх шийдэгдээгүй (username хоосон) — тэр үед байгаа
// user_id-г ХЭВЭЭР үлдээнэ, эрх хасах нь тусдаа зориудын үйлдэл байх ёстой.
//
// uq_valuation_org_employee_user нь нэг нэвтрэх эрхийг зөвхөн нэг ИДЭВХТЭЙ
// ажилтанд зөвшөөрдөг тул хэрэглэгчийг өөр ажилтан/байгууллага руу зэрэг
// холбох гэвэл ErrAlreadyExists болно (409) — зориуд.
func saveValuationEmployee(
	ctx context.Context,
	tx pgx.Tx,
	orgID domain.ID,
	in *domain.ValuationOrgEmployee,
	userID domain.ID,
) (domain.ID, error) {
	position := strings.TrimSpace(in.PositionName)
	if position == "" {
		position = domain.ValuationOrgDefaultPosition
	}
	var newUser *domain.ID
	if userID != 0 {
		uid := userID
		newUser = &uid
	}

	// 1) ID өгсөн бол ЯГ тэр мөрийг засна. org_id-г нөхцөлд шалгах нь өөр
	//    байгууллагын ажилтныг ID-гаар нь өөрлөхөөс сэргийлнэ.
	if in.ID != nil {
		var id domain.ID
		err := tx.QueryRow(ctx, `
			UPDATE valuation_org_employee
			SET last_name = LEFT($1, 250), first_name = LEFT($2, 250),
			    register_no = LEFT($3, 20), position_name = LEFT($4, 250),
			    phone = LEFT($5, 50), email = LEFT($6, 250),
			    user_id = COALESCE($7, user_id), is_active = true, updated_at = $8
			WHERE id = $9 AND org_id = $10
			RETURNING id`,
			in.LastName, in.FirstName, in.RegisterNo, position, in.Phone, in.Email,
			newUser, time.Now(), *in.ID, orgID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrNotFound
		}
		if err != nil {
			return 0, mapHRError(err)
		}
		return id, nil
	}

	// 2) ID байхгүй — регистрээр нь ижил ажилтан байвал ШИНЭ мөр үүсгэхгүй,
	//    түүнийг сэргээнэ (идэвхгүй болгосон ажилтныг дахин нэмэх нь түгээмэл).
	now := time.Now()
	var id domain.ID
	err := tx.QueryRow(ctx, `
		UPDATE valuation_org_employee
		SET last_name = LEFT($1, 250), first_name = LEFT($2, 250),
		    position_name = LEFT($3, 250), phone = LEFT($4, 50), email = LEFT($5, 250),
		    user_id = COALESCE($6, user_id), is_active = true, updated_at = $7
		WHERE org_id = $8 AND register_no = LEFT($9, 20) AND register_no <> ''
		RETURNING id`,
		in.LastName, in.FirstName, position, in.Phone, in.Email, newUser, now,
		orgID, in.RegisterNo).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, mapHRError(err)
	}

	// 3) Шинэ ажилтан.
	if err := tx.QueryRow(ctx, `
		INSERT INTO valuation_org_employee (
			org_id, user_id, last_name, first_name, register_no, position_name,
			phone, email, is_active, created_at, updated_at)
		VALUES ($1, $2, LEFT($3, 250), LEFT($4, 250), LEFT($5, 20), LEFT($6, 250),
			LEFT($7, 50), LEFT($8, 250), true, $9, $9)
		RETURNING id`,
		orgID, newUser, in.LastName, in.FirstName, in.RegisterNo, position,
		in.Phone, in.Email, now).Scan(&id); err != nil {
		return 0, mapHRError(err)
	}
	return id, nil
}

// invalidateUserSessions — олгогдсон токеныг хүчингүй болгоно (appdb-ийн
// user_token_epoch; jwt.Manager унших үедээ шалгана). sd_user-ийг идэвхгүй
// болгох нь refresh-ийг л зогсоох тул access token дуусах хүртэл хүчинтэй
// байдаг — эрх хассан ажилтан тэр хугацаанд ажиллаж болохгүй.
func invalidateUserSessions(ctx context.Context, q hrQuerier, userIDs []domain.ID) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO user_token_epoch (user_source, user_id, valid_from)
		SELECT 'auth', u, date_trunc('second', NOW())
		FROM UNNEST($1::int4[]) AS t(u)
		ON CONFLICT (user_source, user_id) DO UPDATE SET valid_from = EXCLUDED.valid_from`,
		userIDs)
	return mapHRError(err)
}

func collectEmployeeUserIDs(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]domain.ID, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, mapHRError(err)
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

// ---------------------------------------------------------------------------
// Нэвтрэх эрх — sdplatform.sd_user (authdb). ЗӨВХӨН МӨР нэмнэ/шинэчилнэ.
// ---------------------------------------------------------------------------

// ensureLogins — username өгсөн ажилтан бүрт sd_user-ийг бэлдэж (олдвол
// холбоно, эс бөгөөс үүсгэнэ) professional_org ролиудыг олгоно.
//
// Буцах map нь emps-ийн ИНДЕКС → user_id. Username хоосон ажилтан map-д орохгүй
// — нэвтрэх эрх ЗААВАЛ БИШ.
//
// in.Password нь ЭНД ирэхдээ аль хэдийн hash хийгдсэн байх ёстой (service).
func (r *ValuationOrgRepo) ensureLogins(ctx context.Context, emps []domain.ValuationOrgEmployee) (map[int]domain.ID, error) {
	out := make(map[int]domain.ID, len(emps))
	wanted := false
	for i := range emps {
		if strings.TrimSpace(emps[i].Username) != "" {
			wanted = true
			break
		}
	}
	if !wanted {
		return out, nil
	}

	tx, err := r.authDB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i := range emps {
		e := &emps[i]
		username := strings.TrimSpace(e.Username)
		if username == "" {
			continue
		}

		var userID domain.ID
		err := tx.QueryRow(ctx,
			`SELECT user_id FROM sdplatform.sd_user WHERE username = LEFT($1, 50) AND deleted_at IS NULL`,
			username).Scan(&userID)
		switch {
		case err == nil:
			if e.Password != "" {
				if _, err := tx.Exec(ctx,
					`UPDATE sdplatform.sd_user SET "password" = LEFT($1, 150), updated_at = $2 WHERE user_id = $3`,
					e.Password, time.Now(), userID); err != nil {
					return nil, mapHRError(err)
				}
			}
		case errors.Is(err, pgx.ErrNoRows):
			if e.Password == "" {
				// Шинэ эрх үүсгэхэд нууц үг заавал шаардлагатай.
				return nil, domain.ErrInvalidInput
			}
			now := time.Now()
			// email-д UNIQUE индекс (ux_sd_user_email_active) байдаг тул
			// имэйлгүй ажилтанд ХООСОН МӨР биш NULL бичнэ — эс бөгөөс
			// имэйлгүй хоёр дахь ажилтан "давхардсан" гэж татгалзагдана.
			if err := tx.QueryRow(ctx, `
				INSERT INTO sdplatform.sd_user (username, email, "password", firstname, lastname, created_at, updated_at, is_active)
				VALUES (LEFT($1, 50), LEFT(NULLIF($2::text, ''), 250), LEFT($3, 150), LEFT($4::text, 50), LEFT($5::text, 50), $6::date, $6, true)
				RETURNING user_id`,
				username, e.Email, e.Password, e.FirstName, e.LastName, now).Scan(&userID); err != nil {
				return nil, mapHRError(err)
			}
		default:
			return nil, err
		}

		// РОЛЬ ОНООХГҮЙ. Үнэлгээний байгууллага системийн үндсэн хэрэглэгчээс
		// тусдаа явдаг тул sdplatform-ийн роль/эрх (professional_org,
		// compensation:*) түүний хандалтад ямар ч үүрэг хүлээхгүй: ажилтан
		// valuation_org_employee дээр байснаар токенд `valuation_org` тэмдэг
		// авч, /prof болон дашбоардад ТЭР тэмдгээр нэвтэрнэ
		// (middleware.ValuationOrgScope / RequireValuationOrgCaller).
		//
		// Роль олговол тэр нь дотоод маршрутууд руу санамсаргүй эрх нэмэх
		// эрсдэлтэй — allowlist хаадаг ч эрхийн эх сурвалж хоёр болох нь өөрөө
		// эмзэг байдал.
		out[i] = userID
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// applyLoginState — ажилтны төлөвийг нэвтрэх эрх дээр тусгана.
//
// ЭНЭ ЦОНХ нь тухайн байгууллагын ажилтны эрхийн ЭХ СУРВАЛЖ: жагсаалтад байгаа
// ажилтан нэвтэрч чадна, хасагдсан нь чадахгүй. Тиймээс /users цонхоор гараар
// хаасан эрх энд дахин нээгдэж болно — байгууллагын ажилтны эрхийг энэ цонхоор
// удирдана гэсэн зориудын шийдэл.
func (r *ValuationOrgRepo) applyLoginState(ctx context.Context, s loginSync) error {
	if err := r.setLoginActive(ctx, s.deactivate, false); err != nil {
		return err
	}
	return r.setLoginActive(ctx, s.activate, true)
}

// optionalText — хоосон мөрийг NULL болгоно (JSON-д талбар огт гарахгүй).
func optionalText(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func (r *ValuationOrgRepo) setLoginActive(ctx context.Context, userIDs []domain.ID, active bool) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := r.authDB.Exec(ctx, `
		UPDATE sdplatform.sd_user
		SET is_active = $2, updated_at = NOW()
		WHERE user_id = ANY($1::int4[]) AND is_active IS DISTINCT FROM $2`,
		userIDs, active)
	return mapHRError(err)
}
