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
// НЭВТРЭХ ЭРХ ч мөн appdb-д: valuation_org_employee.username / password
// (000024 миграц). sdplatform-д ЮУ Ч бичигдэхгүй — өмнө нь энд sd_user мөр
// үүсгэж, идэвхгүй болгодог байсныг ХАСАВ: authdb нь ажиллаж байгаа ГУС-ийн
// сан тул апп тэнд хэрэглэгч үүсгэх/өөрчлөх ёсгүй.
//
// РОЛЬ ОНООГДОХГҮЙ — хандалт нь токены `valuation_org` тэмдэг ба аппын
// allowlist-ээр шийдэгдэнэ (internal/middleware/valuation_org.go).
//
// Бүх зүйл НЭГ САНД болсон тул нэг гүйлгээнд багтана: ажилтны мөр өөрөө
// нэвтрэх нэгж бөгөөд is_active нь нэвтрэх эрхийн төлөв (тусад нь
// тэнцүүлэх алхам шаардлагагүй).
type ValuationOrgRepo struct {
	db *pgxpool.Pool
}

func NewValuationOrgRepo(db *pgxpool.Pool) *ValuationOrgRepo {
	return &ValuationOrgRepo{db: db}
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
		SELECT id, user_id, COALESCE(username, ''), last_name, first_name,
		       register_no, position_name,
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
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &lastName, &firstName,
			&registerNo, &position, &phone, &email, &e.IsActive,
			&e.CreatedAt, &e.UpdatedAt); err != nil {
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
	return out, nil
}

// Create — байгууллагыг (ба ажилтныг) appdb-д үүсгэнэ.
func (r *ValuationOrgRepo) Create(ctx context.Context, v *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	if err := validateLogins(emps); err != nil {
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

	if err := syncValuationEmployees(ctx, tx, v.ID, emps); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Update — байгууллагын мэдээлэл (ба emps != nil бол ажилтнууд) шинэчилнэ.
//
// ЭНЭ ФУНКЦ ШИНЭ БАЙГУУЛЛАГА ҮҮСГЭХГҮЙ: мөр олдохгүй бол ErrNotFound.
func (r *ValuationOrgRepo) Update(ctx context.Context, v *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	if err := validateLogins(emps); err != nil {
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

	if err := syncValuationEmployees(ctx, tx, v.ID, emps); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Delete — чөлөөлөлт/нэгж талбарт холбогдоогүй байгууллагыг л устгана.
//
// Ажилтнууд FK-ийн ON DELETE CASCADE-ээр устах тул нэвтрэх эрх нь тэдэнтэй
// хамт алга болно. Гэхдээ олгогдсон ACCESS токен хүчинтэй хэвээр байх тул
// эхлээд токены эх цэгийг зөөж, тэднийг хүчингүй болгоно.
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

	// Нэвтрэх эрхтэй ажилтнуудын ID — тэдгээр нь ӨӨРСДӨӨ user_id (эх сурвалж
	// нь valuation_org).
	userIDs, err := collectEmployeeUserIDs(ctx, tx, `
		SELECT id FROM valuation_org_employee
		WHERE org_id = $1 AND username IS NOT NULL`, id)
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
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Ажилтан
// ---------------------------------------------------------------------------

// validateLogins — нэвтрэх эрхийн оролтыг УРЬДЧИЛАН шалгана.
//
// Нууц үг нь ШИНЭ эрх олгоход заавал шаардлагатай. Байгаа ажилтны нууц үгийг
// солихгүй бол хоосон үлдээж болно (тэр үед хуучин hash хэвээр).
func validateLogins(emps []domain.ValuationOrgEmployee) error {
	for i := range emps {
		e := &emps[i]
		if strings.TrimSpace(e.Username) == "" && strings.TrimSpace(e.Password) != "" {
			// Нэргүй нууц үг утгагүй — дуудагчийн алдаа.
			return domain.ErrInvalidInput
		}
	}
	return nil
}

// syncValuationEmployees — байгууллагын ажилтнуудын БҮРЭН төлөвийг тавина.
//
// emps == nil бол ажилтнуудад ХҮРЭХГҮЙ (зөвхөн байгууллагын мэдээлэл засварласан
// тохиолдол). Хоосон биш бол жагсаалтад ороогүй идэвхтэй ажилтан идэвхгүй болно
// — мөр УСТГАХГҮЙ тул түүх, баримт дээрх нэр хадгалагдана.
//
// Нэвтрэх эрх нь мөрийн өөрийнх нь username/password тул тусад нь authdb-д
// тэнцүүлэх алхам БАЙХГҮЙ: бүх зүйл энэ нэг гүйлгээнд багтана.
func syncValuationEmployees(
	ctx context.Context,
	tx pgx.Tx,
	orgID domain.ID,
	emps []domain.ValuationOrgEmployee,
) error {
	if emps == nil {
		return nil
	}

	keep := make([]domain.ID, 0, len(emps))
	for i := range emps {
		id, err := saveValuationEmployee(ctx, tx, orgID, &emps[i])
		if err != nil {
			return err
		}
		keep = append(keep, id)
	}

	// Жагсаалтад ороогүй ажилтныг идэвхгүй болгоно. keep хоосон байж болзошгүй
	// тул query-г тусад нь салгана (<> ALL('{}') нь ҮРГЭЛЖ true болох тул зөв,
	// гэхдээ уншихад бүрхэг).
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
			RETURNING id`, orgID, now)
	} else {
		rows, err = tx.Query(ctx, `
			UPDATE valuation_org_employee
			SET is_active = false, updated_at = $3
			WHERE org_id = $1 AND is_active AND id <> ALL($2::int4[])
			RETURNING id`, orgID, keep, now)
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

	// Байгууллагаас хасагдсан ажилтны олгогдсон токен ч хүчингүй болно
	// (is_active=false нь refresh-ийг л зогсооно, access токен хэвээр).
	return invalidateUserSessions(ctx, tx, removed)
}

// saveValuationEmployee — нэг ажилтныг бичиж, valuation_org_employee.id-г
// буцаана. Тэр id нь ажилтны user_id мөн болно (user_source=valuation_org).
//
// НЭВТРЭХ ЭРХ: username хоосон бол БАЙГАА нэр/нууц үгийг хөндөхгүй — эрх хасах
// нь тусдаа зориудын үйлдэл байх ёстой. Нууц үг хоосон бол хуучин hash хэвээр.
//
// uq_valuation_org_employee_username нь нэг нэрийг зөвхөн нэг ИДЭВХТЭЙ
// ажилтанд зөвшөөрдөг тул давхардвал ErrAlreadyExists болно (409) — зориуд.
func saveValuationEmployee(
	ctx context.Context,
	tx pgx.Tx,
	orgID domain.ID,
	in *domain.ValuationOrgEmployee,
) (domain.ID, error) {
	position := strings.TrimSpace(in.PositionName)
	if position == "" {
		position = domain.ValuationOrgDefaultPosition
	}
	username := optionalText(in.Username)
	password := optionalText(in.Password)

	// 1) ID өгсөн бол ЯГ тэр мөрийг засна. org_id-г нөхцөлд шалгах нь өөр
	//    байгууллагын ажилтныг ID-гаар нь өөрлөхөөс сэргийлнэ.
	if in.ID != nil {
		var id domain.ID
		err := tx.QueryRow(ctx, `
			UPDATE valuation_org_employee
			SET last_name = LEFT($1, 250), first_name = LEFT($2, 250),
			    register_no = LEFT($3, 20), position_name = LEFT($4, 250),
			    phone = LEFT($5, 50), email = LEFT($6, 250),
			    username = COALESCE(LEFT($7, 50), username),
			    "password" = COALESCE(LEFT($8, 150), "password"),
			    is_active = true, updated_at = $9
			WHERE id = $10 AND org_id = $11
			RETURNING id`,
			in.LastName, in.FirstName, in.RegisterNo, position, in.Phone, in.Email,
			username, password, time.Now(), *in.ID, orgID).Scan(&id)
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
		    username = COALESCE(LEFT($6, 50), username),
		    "password" = COALESCE(LEFT($7, 150), "password"),
		    is_active = true, updated_at = $8
		WHERE org_id = $9 AND register_no = LEFT($10, 20) AND register_no <> ''
		RETURNING id`,
		in.LastName, in.FirstName, position, in.Phone, in.Email, username, password, now,
		orgID, in.RegisterNo).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, mapHRError(err)
	}

	// 3) Шинэ ажилтан. Нэвтрэх нэр өгсөн мөртлөө нууц үггүй бол эрх олгох
	//    боломжгүй (hash хоосон мөр нэвтрэлтэд ОГТ олдохгүй — GetActiveByLogin).
	if username != nil && password == nil {
		return 0, domain.ErrInvalidInput
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO valuation_org_employee (
			org_id, last_name, first_name, register_no, position_name,
			phone, email, username, "password", is_active, created_at, updated_at)
		VALUES ($1, LEFT($2, 250), LEFT($3, 250), LEFT($4, 20), LEFT($5, 250),
			LEFT($6, 50), LEFT($7, 250), LEFT($8, 50), LEFT($9, 150), true, $10, $10)
		RETURNING id`,
		orgID, in.LastName, in.FirstName, in.RegisterNo, position,
		in.Phone, in.Email, username, password, now).Scan(&id); err != nil {
		return 0, mapHRError(err)
	}
	return id, nil
}

// invalidateUserSessions — олгогдсон токеныг хүчингүй болгоно (appdb-ийн
// user_token_epoch; jwt.Manager унших үедээ шалгана). is_active=false нь
// refresh-ийг л зогсоох тул access token дуусах хүртэл хүчинтэй байдаг —
// эрх хассан ажилтан тэр хугацаанд ажиллаж болохгүй.
//
// userIDs нь valuation_org_employee.id — иймд эх сурвалж 'valuation_org'.
func invalidateUserSessions(ctx context.Context, q hrQuerier, userIDs []domain.ID) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO user_token_epoch (user_source, user_id, valid_from)
		SELECT $2, u, date_trunc('second', NOW())
		FROM UNNEST($1::int4[]) AS t(u)
		ON CONFLICT (user_source, user_id) DO UPDATE SET valid_from = EXCLUDED.valid_from`,
		userIDs, domain.UserSourceValuationOrg)
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
// Туслахууд
// ---------------------------------------------------------------------------

// optionalText — хоосон мөрийг NULL болгоно (JSON-д талбар огт гарахгүй).
func optionalText(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
