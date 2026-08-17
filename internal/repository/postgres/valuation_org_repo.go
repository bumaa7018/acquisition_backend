package postgres

import (
	"context"
	"fmt"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ValuationOrgRepo — үнэлгээний (мэргэжлийн) байгууллага ба түүний ажилтнууд.
//
// Мэргэжлийн байгууллагын identity болон profile мэдээлэл БҮГД appdb-ийн
// valuation_org_profile хүснэгтэд байна. sdplatform нь ердийн HR лавлахын
// эх сурвалж хэвээр үлдэнэ; мэргэжлийн байгууллагын нэр, id-г тэндээс
// уншихгүй, тэнд байгууллагын мөр үүсгэх/засах/устгахгүй.
//
// Энэ repository нь зөвхөн байгууллагын profile-ийг хариуцна. Ажилтан,
// хэлтэс, хэрэглэгчийн HR өгөгдөл нь тусдаа HR модулийн хариуцлага бөгөөд
// энэ бүртгэлийн цонхоор sdplatform руу бичихгүй.
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
	       vp.created_at, vp.updated_at, 0
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

func (r *ValuationOrgRepo) GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error) {
	v := &domain.ValuationOrg{}
	if err := scanValuationOrg(r.db.QueryRow(ctx, valuationOrgSelectSQL+` WHERE vp.id = $1`, id), v); err != nil {
		return nil, mapHRError(err)
	}
	return v, nil
}

// Create — мэргэжлийн байгууллагын profile-ийг appdb-д үүсгэнэ. sdplatform-д
// байгууллагын мөр нэмэхгүй.
func (r *ValuationOrgRepo) Create(ctx context.Context, v *domain.ValuationOrg) error {
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
	return tx.Commit(ctx)
}

// Update — profile-ийн байгууллагын мэдээллийг appdb-д шинэчилнэ.
func (r *ValuationOrgRepo) Update(ctx context.Context, v *domain.ValuationOrg) error {
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
	return tx.Commit(ctx)
}

// Delete — чөлөөлөлт/нэгж талбарт холбогдоогүй байгууллагыг л устгана.
//
// Профайл нь appdb-ийн өөрийн хүснэгт тул зөвхөн profile мөрийг устгана.
// sdplatform-ийн байгууллага, хэлтэс, ажилтан, хэрэглэгчийн өгөгдөлд хүрэхгүй.
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

	res, err := tx.Exec(ctx, `DELETE FROM valuation_org_profile WHERE id = $1`, id)
	if err != nil {
		return mapHRError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit(ctx)
}
