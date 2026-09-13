package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValuationOrgLoginRepo — ҮНЭЛГЭЭНИЙ БАЙГУУЛЛАГЫН ажилтны нэвтрэлт.
//
// Эдгээр хэрэглэгч sdplatform-д БҮРТГЭГДЭХГҮЙ: бүртгэл нь appdb-ийн
// valuation_org_employee мөр өөрөө бөгөөд тэр мөрийн id нь user_id болно
// (user_source = domain.UserSourceValuationOrg).
//
// ЯАГААД ТУСДАА ФАЙЛ/РЕПО ВЭ: authdb дахь хэрэглэгчийн удирдлага (UserRepo,
// sd_user, роль/эрх) ба үнэлгээний байгууллагын ажилтны удирдлага хоёр нь
// ӨӨР ӨӨР эх сурвалжтай, өөр эрхийн загвартай. Нэг репод хольбол аль сангаас
// уншиж байгаа нь балартаж, user_id мөргөлдөх алдаа эргэж ирнэ.
//
// ЭРХ: роль/эрх ОГТ БАЙХГҮЙ. GetRoles / GetPermissions нь ҮРГЭЛЖ хоосон
// буцаана — эдгээр хэрэглэгчийн хандалт зөвхөн токены `valuation_org` тэмдэг
// ба маршрутын allowlist-ээр шийдэгдэнэ (internal/middleware/valuation_org.go).
type ValuationOrgLoginRepo struct {
	db *pgxpool.Pool
}

func NewValuationOrgLoginRepo(db *pgxpool.Pool) *ValuationOrgLoginRepo {
	return &ValuationOrgLoginRepo{db: db}
}

// Source — domain.UserSourceValuationOrg. Токен болон user_token_epoch-д бичнэ.
func (r *ValuationOrgLoginRepo) Source() string { return domain.UserSourceValuationOrg }

const valuationOrgLoginSelectSQL = `
	SELECT ve.id,
	       COALESCE(ve.username, ''),
	       COALESCE(ve.email, ''),
	       COALESCE(ve."password", ''),
	       ve.first_name, ve.last_name,
	       COALESCE(ve.position_name, ''),
	       ve.is_active, ve.created_at, ve.updated_at
	FROM valuation_org_employee ve`

func scanValuationOrgLogin(row rowScanner, u *domain.User) error {
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.Password,
		&u.FirstName, &u.LastName, &u.Position, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt)
}

// GetActiveByLogin — нэвтрэх нэр ЭСВЭЛ имэйлээр, ЗӨВХӨН идэвхтэй ажилтнаас.
//
// Нууц үггүй (эрх олгоогүй) ажилтныг ОГТ буцаахгүй: эс бөгөөс hash хоосон
// байхад ComparePassword-д найдах болно.
func (r *ValuationOrgLoginRepo) GetActiveByLogin(ctx context.Context, login string) (*domain.User, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, domain.ErrNotFound
	}
	u := &domain.User{}
	err := scanValuationOrgLogin(r.db.QueryRow(ctx, valuationOrgLoginSelectSQL+`
		WHERE ve.is_active
		  AND COALESCE(ve."password", '') <> ''
		  AND (lower(ve.username) = lower($1) OR lower(ve.email) = lower($1))
		ORDER BY ve.id
		LIMIT 1`, login), u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *ValuationOrgLoginRepo) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	u := &domain.User{}
	err := scanValuationOrgLogin(r.db.QueryRow(ctx, valuationOrgLoginSelectSQL+`
		WHERE ve.id = $1`, id), u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

// GetRoles — ҮРГЭЛЖ хоосон. Роль энэ эх сурвалжид байхгүй (дээрх тайлбарыг үз).
func (r *ValuationOrgLoginRepo) GetRoles(_ context.Context, _ domain.ID) ([]string, error) {
	return nil, nil
}

// GetPermissions — ҮРГЭЛЖ хоосон. Эрх энэ эх сурвалжид байхгүй.
func (r *ValuationOrgLoginRepo) GetPermissions(_ context.Context, _ domain.ID) ([]string, error) {
	return nil, nil
}

// ValuationOrgIDForUser — ажилтны харьяа байгууллага (valuation_org_profile.id).
// Токены org_id мөн `valuation_org` тэмдэг үүн дээр тогтоно.
func (r *ValuationOrgLoginRepo) ValuationOrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error) {
	var orgID domain.ID
	err := r.db.QueryRow(ctx, `
		SELECT ve.org_id
		FROM valuation_org_employee ve
		JOIN valuation_org_profile vp ON vp.id = ve.org_id
		WHERE ve.id = $1 AND ve.is_active AND vp.is_active`, userID).Scan(&orgID)
	switch {
	case err == nil:
		return &orgID, nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	default:
		return nil, err
	}
}

// OrgIDForUser — HR-ийн байгууллага. Эдгээр хэрэглэгч HR бүтцэд байхгүй тул nil.
func (r *ValuationOrgLoginRepo) OrgIDForUser(_ context.Context, _ domain.ID) (*domain.ID, error) {
	return nil, nil
}

func (r *ValuationOrgLoginRepo) UpdatePassword(ctx context.Context, id domain.ID, hashed string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE valuation_org_employee
		SET "password" = LEFT($2, 150), updated_at = NOW()
		WHERE id = $1`, id, hashed)
	return err
}

// InvalidateSessions — нууц үг солигдоход өмнө олгогдсон бүх токеныг таслана.
func (r *ValuationOrgLoginRepo) InvalidateSessions(ctx context.Context, userID domain.ID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_token_epoch (user_source, user_id, valid_from)
		VALUES ($2, $1, date_trunc('second', NOW()))
		ON CONFLICT (user_source, user_id) DO UPDATE SET valid_from = EXCLUDED.valid_from`,
		userID, domain.UserSourceValuationOrg)
	return err
}

// IsSessionValid — UserRepo-гийнхтэй ижил утгатай (user_token_epoch нь appdb-д).
func (r *ValuationOrgLoginRepo) IsSessionValid(ctx context.Context, source string, userID domain.ID, issuedAt time.Time) (bool, error) {
	if source == "" {
		source = domain.UserSourceValuationOrg
	}
	var invalid bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_token_epoch
			WHERE user_source = $3 AND user_id = $1 AND valid_from > $2
		)`, userID, issuedAt, source).Scan(&invalid)
	if err != nil {
		return false, err
	}
	return !invalid, nil
}
