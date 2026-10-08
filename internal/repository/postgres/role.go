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

// RoleRepo — sdplatform.sd_role дээр ШУУД ажиллана.
//
// sd_role.role_type_code нь НӨӨЦ (land, compensation, decision, config ...),
// sd_permission нь ҮЙЛДЭЛ (read/create/update/delete). Үр дүнгийн эрх нь
// role_type_code + ":" + permission.name.
type RoleRepo struct {
	db *pgxpool.Pool
}

func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

const roleSelectSQL = `
	SELECT r.role_id, COALESCE(r.code, ''), COALESCE(r.name, ''), COALESCE(r.role_type_code, ''),
	       COALESCE(r.description, ''),
	       COALESCE(r.created_at::timestamptz, NOW()), COALESCE(r.updated_at::timestamptz, NOW())
	FROM sdplatform.sd_role r`

// Зөвхөн энэ системд (system_id=10) хамаарах ролиуд.
// roleInSystem10SQL — тухайн role_id нь систем 10-д харьяалагдах эсэх.
// УСТГАХ үйлдэл бүр үүгээр хязгаарлагдана: sdplatform схем нь бусад системтэй
// ХУВААЛЦСАН тул зөвхөн энэ системийн мөрийг л хөндөх ёстой.
const roleInSystem10SQL = `
	EXISTS (
		SELECT 1 FROM sdplatform.sd_system_role sr
		WHERE sr.role_id = $1 AND sr.system_id = 10
	)`

const roleSystemFilter = `
	EXISTS (
		SELECT 1 FROM sdplatform.sd_system_role sr
		WHERE sr.role_id = r.role_id AND sr.system_id = 10
	)`

func scanRole(row rowScanner, role *domain.Role) error {
	return row.Scan(&role.ID, &role.Code, &role.Name, &role.Resource, &role.Description, &role.CreatedAt, &role.UpdatedAt)
}

func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	if role.CreatedAt.IsZero() {
		role.CreatedAt = time.Now()
	}
	if role.UpdatedAt.IsZero() {
		role.UpdatedAt = time.Now()
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_role (code, name, role_type_code, description, created_at, updated_at)
		VALUES (LEFT(NULLIF($1, ''), 10), LEFT($2, 250), LEFT(NULLIF($3, ''), 45), $4, $5, $6)
		RETURNING role_id`,
		role.Code, role.Name, role.Resource, role.Description, role.CreatedAt, role.UpdatedAt).Scan(&role.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	// Шинэ роль энэ системд харьяалагдана.
	_, err = r.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
		SELECT 10, $1, NOW()
		WHERE NOT EXISTS (
			SELECT 1 FROM sdplatform.sd_system_role WHERE system_id = 10 AND role_id = $1
		)`, role.ID)
	return err
}

func (r *RoleRepo) GetByID(ctx context.Context, id domain.ID) (*domain.Role, error) {
	role := &domain.Role{}
	err := scanRole(r.db.QueryRow(ctx, roleSelectSQL+` WHERE r.role_id = $1 AND `+roleSystemFilter, id), role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	perms, err := r.getPermissions(ctx, role.ID, role.Resource)
	if err != nil {
		return nil, err
	}
	role.Permissions = perms
	menus, err := r.getMenus(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	role.Menus = menus
	return role, nil
}

func (r *RoleRepo) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	role := &domain.Role{}
	err := scanRole(r.db.QueryRow(ctx, roleSelectSQL+` WHERE r.name = $1`, name), role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *RoleRepo) Update(ctx context.Context, role *domain.Role) error {
	role.UpdatedAt = time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE sdplatform.sd_role
		SET name=LEFT($1, 250), description=$2, updated_at=$3
		WHERE role_id=$4`,
		role.Name, role.Description, role.UpdatedAt, role.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *RoleRepo) Delete(ctx context.Context, id domain.ID) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM sdplatform.sd_role WHERE role_id = $1 AND `+roleInSystem10SQL, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *RoleRepo) List(ctx context.Context) ([]*domain.Role, error) {
	rows, err := r.db.Query(ctx, roleSelectSQL+` WHERE `+roleSystemFilter+` ORDER BY r.created_at DESC NULLS LAST, r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role := &domain.Role{}
		if err := scanRole(rows, role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachPermissions(ctx, roles); err != nil {
		return nil, err
	}
	if err := r.attachMenus(ctx, roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// attachPermissions — ролиудын эрхийг НЭГ хүсэлтээр татна.
func (r *RoleRepo) attachPermissions(ctx context.Context, roles []*domain.Role) error {
	if len(roles) == 0 {
		return nil
	}
	byID := make(map[domain.ID]*domain.Role, len(roles))
	ids := make([]domain.ID, 0, len(roles))
	for _, role := range roles {
		byID[role.ID] = role
		ids = append(ids, role.ID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT role_id, permission_id, permission_code, permission_name, resource, created_at
		FROM (
			SELECT rp.role_id, p.permission_id, COALESCE(p.code, '') AS permission_code,
			       COALESCE(p.name, '') AS permission_name, COALESCE(m.code, '') AS resource,
			       COALESCE(p.created_at::timestamptz, NOW()) AS created_at
			FROM sdplatform.sd_role_permission rp
			JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
			JOIN sdplatform.sd_menu m ON m.menu_id = rp.menu_id
			JOIN sdplatform.sd_form f ON f.code = m.code
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
				AND (
					fp.permission_id = p.permission_id
					OR fp.permission_code = p.code
					OR fp.permission_code = p.name
				)
			WHERE rp.role_id = ANY($1)
			UNION ALL
			SELECT rp.role_id, p.permission_id, COALESCE(p.code, '') AS permission_code,
			       COALESCE(p.name, '') AS permission_name, COALESCE(r.role_type_code, '') AS resource,
			       COALESCE(p.created_at::timestamptz, NOW()) AS created_at
			FROM sdplatform.sd_role_permission rp
			JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
			JOIN sdplatform.sd_role r ON r.role_id = rp.role_id
			WHERE rp.role_id = ANY($1) AND rp.menu_id IS NULL
		) q
		ORDER BY resource, permission_name`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var roleID domain.ID
		var p domain.Permission
		if err := rows.Scan(&roleID, &p.ID, &p.Code, &p.Action, &p.Resource, &p.CreatedAt); err != nil {
			return err
		}
		role, ok := byID[roleID]
		if !ok {
			continue
		}
		p.Resource = domain.MenuResource(p.Resource)
		p.Name = domain.PermissionKey(p.Resource, p.Action)
		role.Permissions = append(role.Permissions, p)
	}
	return rows.Err()
}

func (r *RoleRepo) getPermissions(ctx context.Context, roleID domain.ID, resource string) ([]domain.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT permission_id, permission_code, permission_name, resource, created_at
		FROM (
			SELECT p.permission_id, COALESCE(p.code, '') AS permission_code,
			       COALESCE(p.name, '') AS permission_name, COALESCE(m.code, '') AS resource,
			       COALESCE(p.created_at::timestamptz, NOW()) AS created_at
			FROM sdplatform.sd_permission p
			JOIN sdplatform.sd_role_permission rp ON rp.permission_id = p.permission_id
			JOIN sdplatform.sd_menu m ON m.menu_id = rp.menu_id
			JOIN sdplatform.sd_form f ON f.code = m.code
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
				AND (
					fp.permission_id = p.permission_id
					OR fp.permission_code = p.code
					OR fp.permission_code = p.name
				)
			WHERE rp.role_id = $1
			UNION ALL
			SELECT p.permission_id, COALESCE(p.code, '') AS permission_code,
			       COALESCE(p.name, '') AS permission_name, COALESCE($2::text, '') AS resource,
			       COALESCE(p.created_at::timestamptz, NOW()) AS created_at
			FROM sdplatform.sd_permission p
			JOIN sdplatform.sd_role_permission rp ON rp.permission_id = p.permission_id
			WHERE rp.role_id = $1 AND rp.menu_id IS NULL
		) q
		ORDER BY resource, permission_name`, roleID, resource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		p := domain.Permission{}
		if err := rows.Scan(&p.ID, &p.Code, &p.Action, &p.Resource, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Resource = domain.MenuResource(p.Resource)
		p.Name = domain.PermissionKey(p.Resource, p.Action)
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *RoleRepo) AssignPermission(ctx context.Context, roleID, permissionID domain.ID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at)
		SELECT $1, $2, m.menu_id, NOW()
		FROM sdplatform.sd_role role
		LEFT JOIN sdplatform.sd_menu m ON m.code = role.role_type_code
		WHERE role.role_id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM sdplatform.sd_role_permission
			WHERE role_id = $1 AND permission_id = $2 AND menu_id IS NOT DISTINCT FROM m.menu_id
		)`, roleID, permissionID)
	return err
}

func (r *RoleRepo) RemovePermission(ctx context.Context, roleID, permissionID domain.ID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM sdplatform.sd_role_permission
		 WHERE role_id = $1 AND permission_id = $2 AND `+roleInSystem10SQL,
		roleID, permissionID)
	return err
}

func (r *RoleRepo) SetMenuPermissions(ctx context.Context, roleID, menuID domain.ID, permissionIDs []domain.ID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT DO NOTHING`, menuID, roleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM sdplatform.sd_role_permission
		 WHERE role_id = $1 AND menu_id = $2 AND `+roleInSystem10SQL,
		roleID, menuID); err != nil {
		return err
	}
	if len(permissionIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO sdplatform.sd_role_permission (role_id, menu_id, permission_id, created_at, updated_at)
			SELECT $1, $2, p.permission_id, NOW(), NOW()
			FROM sdplatform.sd_permission p
			JOIN sdplatform.sd_menu m ON m.menu_id = $2
			JOIN sdplatform.sd_form f ON f.code = m.code
			JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
				AND (
					fp.permission_id = p.permission_id
					OR fp.permission_code = p.code
					OR fp.permission_code = p.name
				)
			WHERE p.permission_id = ANY($3)
			GROUP BY p.permission_id`, roleID, menuID, permissionIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *RoleRepo) RemoveMenuPermissions(ctx context.Context, roleID, menuID domain.ID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx,
		`DELETE FROM sdplatform.sd_role_permission
		 WHERE role_id = $1 AND menu_id = $2 AND `+roleInSystem10SQL,
		roleID, menuID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM sdplatform.sd_menu_role
		 WHERE role_id = $1 AND menu_id = $2 AND `+roleInSystem10SQL,
		roleID, menuID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *RoleRepo) attachMenus(ctx context.Context, roles []*domain.Role) error {
	if len(roles) == 0 {
		return nil
	}
	byID := make(map[domain.ID]*domain.Role, len(roles))
	ids := make([]domain.ID, 0, len(roles))
	for _, role := range roles {
		byID[role.ID] = role
		ids = append(ids, role.ID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT rp.role_id, m.menu_id, COALESCE(m.code, ''), COALESCE(m."name", ''),
		       m.parent_id, COALESCE(m.parent_code, ''), COALESCE(m.menu_url, ''),
		       COALESCE(m.menu_icon, ''), COALESCE(m.sort_order, 0),
		       COALESCE(m.created_at::timestamptz, NOW()), COALESCE(m.updated_at::timestamptz, NOW()),
		       p.permission_id, COALESCE(p.code, ''), COALESCE(p.name, ''),
		       COALESCE(p.created_at::timestamptz, NOW())
		FROM sdplatform.sd_role_permission rp
		JOIN sdplatform.sd_menu m ON m.menu_id = rp.menu_id
		JOIN sdplatform.sd_form f ON f.code = m.code
		JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
		JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
			AND (
				fp.permission_id = p.permission_id
				OR fp.permission_code = p.code
				OR fp.permission_code = p.name
			)
		WHERE rp.role_id = ANY($1)
		ORDER BY COALESCE(m.sort_order, 0), COALESCE(m."name", ''), COALESCE(p.name, '')`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	menusByRole := make(map[domain.ID]map[domain.ID]*domain.Menu)
	for rows.Next() {
		var roleID domain.ID
		var menu domain.Menu
		var perm domain.Permission
		if err := rows.Scan(&roleID, &menu.ID, &menu.Code, &menu.Name, &menu.ParentID,
			&menu.ParentCode, &menu.URL, &menu.Icon, &menu.SortOrder, &menu.CreatedAt,
			&menu.UpdatedAt, &perm.ID, &perm.Code, &perm.Action, &perm.CreatedAt); err != nil {
			return err
		}
		role := byID[roleID]
		if role == nil {
			continue
		}
		if menusByRole[roleID] == nil {
			menusByRole[roleID] = map[domain.ID]*domain.Menu{}
		}
		existing := menusByRole[roleID][menu.ID]
		if existing == nil {
			role.Menus = append(role.Menus, menu)
			existing = &role.Menus[len(role.Menus)-1]
			menusByRole[roleID][menu.ID] = existing
		}
		perm.Resource = domain.MenuResource(menu.Code)
		perm.Name = domain.PermissionKey(perm.Resource, perm.Action)
		existing.Permissions = append(existing.Permissions, perm)
	}
	return rows.Err()
}

func (r *RoleRepo) getMenus(ctx context.Context, roleID domain.ID) ([]domain.Menu, error) {
	role := &domain.Role{ID: roleID}
	if err := r.attachMenus(ctx, []*domain.Role{role}); err != nil {
		return nil, err
	}
	return role.Menus, nil
}

func (r *RoleRepo) AssignToUser(ctx context.Context, userID, roleID domain.ID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT DO NOTHING`, userID, roleID)
	return err
}

func (r *RoleRepo) RemoveFromUser(ctx context.Context, userID, roleID domain.ID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM sdplatform.sd_user_role WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	return err
}
