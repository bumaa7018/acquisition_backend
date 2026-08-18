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
	menus, err := r.collect(ctx, menuSelectSQL+`
		WHERE EXISTS (
			SELECT 1
			FROM sdplatform.sd_form f
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
			WHERE f.code = m.code
		)
		ORDER BY COALESCE(m.sort_order, 0), m."name"`)
	if err != nil {
		return nil, err
	}
	if err := r.attachAvailablePermissions(ctx, menus); err != nil {
		return nil, err
	}
	return menus, nil
}

// ListByRole — тухайн роль дээр холбогдсон цэсүүд.
func (r *MenuRepo) ListByRole(ctx context.Context, roleID domain.ID) ([]*domain.Menu, error) {
	menus, err := r.collect(ctx, menuSelectSQL+`
		JOIN sdplatform.sd_menu_role mr ON mr.menu_id = m.menu_id
		WHERE mr.role_id = $1
		  AND EXISTS (
			SELECT 1
			FROM sdplatform.sd_form f
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
			WHERE f.code = m.code
		  )
		ORDER BY COALESCE(m.sort_order, 0), m."name"`, roleID)
	if err != nil {
		return nil, err
	}
	if err := r.attachAvailablePermissions(ctx, menus); err != nil {
		return nil, err
	}
	return menus, nil
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
		JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
		WHERE ur.user_id = $1
		  AND EXISTS (
			SELECT 1
			FROM sdplatform.sd_form f
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
			WHERE f.code = m.code
		  )
		ORDER BY COALESCE(m.sort_order, 0), COALESCE(m."name", '')`, userID)
}

func (r *MenuRepo) ListPermissions(ctx context.Context, menuID domain.ID) ([]domain.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT p.permission_id, COALESCE(p.code, ''), COALESCE(p.name, ''),
		       COALESCE(m.code, ''), COALESCE(p.created_at::timestamptz, NOW())
		FROM sdplatform.sd_menu m
		JOIN sdplatform.sd_form f ON f.code = m.code
		JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
		JOIN sdplatform.sd_permission p ON
			p.permission_id = fp.permission_id
			OR p.code = fp.permission_code
			OR p.name = fp.permission_code
		WHERE m.menu_id = $1
		ORDER BY COALESCE(p.name, '')`, menuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Action, &p.Resource, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Resource = domain.MenuResource(p.Resource)
		p.Name = domain.PermissionKey(p.Resource, p.Action)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *MenuRepo) attachAvailablePermissions(ctx context.Context, menus []*domain.Menu) error {
	if len(menus) == 0 {
		return nil
	}
	byID := make(map[domain.ID]*domain.Menu, len(menus))
	ids := make([]domain.ID, 0, len(menus))
	for _, m := range menus {
		byID[m.ID] = m
		ids = append(ids, m.ID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT m.menu_id, p.permission_id, COALESCE(p.code, ''), COALESCE(p.name, ''),
		       COALESCE(m.code, ''), COALESCE(p.created_at::timestamptz, NOW())
		FROM sdplatform.sd_menu m
		JOIN sdplatform.sd_form f ON f.code = m.code
		JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
		JOIN sdplatform.sd_permission p ON
			p.permission_id = fp.permission_id
			OR p.code = fp.permission_code
			OR p.name = fp.permission_code
		WHERE m.menu_id = ANY($1)
		ORDER BY COALESCE(p.name, '')`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var menuID domain.ID
		var p domain.Permission
		if err := rows.Scan(&menuID, &p.ID, &p.Code, &p.Action, &p.Resource, &p.CreatedAt); err != nil {
			return err
		}
		if m := byID[menuID]; m != nil {
			p.Resource = domain.MenuResource(p.Resource)
			p.Name = domain.PermissionKey(p.Resource, p.Action)
			m.Permissions = append(m.Permissions, p)
		}
	}
	return rows.Err()
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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx,
		`DELETE FROM sdplatform.sd_role_permission
		 WHERE role_id = $1 AND menu_id = $2 AND `+roleInSystem10SQL, roleID, menuID); err != nil {
		return err
	}
	res, err := tx.Exec(ctx,
		`DELETE FROM sdplatform.sd_menu_role
		 WHERE role_id = $1 AND menu_id = $2 AND `+roleInSystem10SQL, roleID, menuID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit(ctx)
}
