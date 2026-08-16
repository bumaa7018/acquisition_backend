package postgres

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MenuRepo — sdplatform.sd_menu / sd_menu_role.
//
// Эрхийн загварын 1-р давхарга: хэрэглэгчид ямар цэс харагдахыг роль тус бүрт
// sd_menu_role-оор тохируулна. Цэс харагдсаны дараа роль (нөөц) ба эрх (үйлдэл)
// шалгагдана.
type MenuRepo struct {
	db *pgxpool.Pool
}

func NewMenuRepo(db *pgxpool.Pool) *MenuRepo {
	return &MenuRepo{db: db}
}

const menuSelectSQL = `
	SELECT m.menu_id, COALESCE(m.code, ''), COALESCE(m."name", ''), m.parent_id,
	       COALESCE(m.parent_code, ''), COALESCE(m.menu_url, ''), COALESCE(m.menu_icon, ''),
	       COALESCE(m.sort_order, 0),
	       COALESCE(m.created_at::timestamptz, NOW()), COALESCE(m.updated_at::timestamptz, NOW())
	FROM sdplatform.sd_menu m`

func scanMenu(row rowScanner, m *domain.Menu) error {
	return row.Scan(&m.ID, &m.Code, &m.Name, &m.ParentID, &m.ParentCode, &m.URL, &m.Icon,
		&m.SortOrder, &m.CreatedAt, &m.UpdatedAt)
}

func (r *MenuRepo) collect(ctx context.Context, query string, args ...any) ([]*domain.Menu, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Menu
	for rows.Next() {
		m := &domain.Menu{}
		if err := scanMenu(rows, m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// List — системийн бүх цэс.
func (r *MenuRepo) List(ctx context.Context) ([]*domain.Menu, error) {
	return r.collect(ctx, menuSelectSQL+` ORDER BY COALESCE(m.sort_order, 0), m."name"`)
}

// ListByRole — тухайн роль дээр холбогдсон цэсүүд.
func (r *MenuRepo) ListByRole(ctx context.Context, roleID domain.ID) ([]*domain.Menu, error) {
	return r.collect(ctx, menuSelectSQL+`
		JOIN sdplatform.sd_menu_role mr ON mr.menu_id = m.menu_id
		WHERE mr.role_id = $1
		ORDER BY COALESCE(m.sort_order, 0), m."name"`, roleID)
}

// ListByUser — хэрэглэгчийн ролиудаар дамжин харагдах цэсүүд (давхардалгүй).
func (r *MenuRepo) ListByUser(ctx context.Context, userID domain.ID) ([]*domain.Menu, error) {
	return r.collect(ctx, `
		SELECT DISTINCT m.menu_id, COALESCE(m.code, ''), COALESCE(m."name", ''), m.parent_id,
		       COALESCE(m.parent_code, ''), COALESCE(m.menu_url, ''), COALESCE(m.menu_icon, ''),
		       COALESCE(m.sort_order, 0),
		       COALESCE(m.created_at::timestamptz, NOW()), COALESCE(m.updated_at::timestamptz, NOW())
		FROM sdplatform.sd_menu m
		JOIN sdplatform.sd_menu_role mr ON mr.menu_id = m.menu_id
		JOIN sdplatform.sd_user_role ur ON ur.role_id = mr.role_id
		WHERE ur.user_id = $1
		ORDER BY COALESCE(m.sort_order, 0), COALESCE(m."name", '')`, userID)
}

// AttachToRole — цэсийг роль дээр холбоно (идемпотент).
func (r *MenuRepo) AttachToRole(ctx context.Context, roleID, menuID domain.ID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT DO NOTHING`, menuID, roleID)
	return mapHRError(err)
}

// DetachFromRole — холбоосыг салгана.
func (r *MenuRepo) DetachFromRole(ctx context.Context, roleID, menuID domain.ID) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM sdplatform.sd_menu_role WHERE role_id = $1 AND menu_id = $2`, roleID, menuID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
