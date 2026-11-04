package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/landreleasesql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LandReleaseRepo — «Буулгалтын мэдээлэл» (land_release* хүснэгтүүд).
type LandReleaseRepo struct {
	db *pgxpool.Pool
}

func NewLandReleaseRepo(db *pgxpool.Pool) *LandReleaseRepo {
	return &LandReleaseRepo{db: db}
}

// releaseGeomSQL — оруулсан WKT-г geometry(MultiPolygon,4326) болгоно (засаж,
// зөвхөн полигон хэсгийг авна).
const releaseGeomSQL = `ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_GeomFromText(%s, 4326)), 3))`

// AreasOf — WKT бүрийн талбай (м², public.calculate_area_utm — нэгж талбартай ИЖИЛ).
func (r *LandReleaseRepo) AreasOf(ctx context.Context, wkts []string) ([]float64, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT COALESCE(public.calculate_area_utm(`+releaseGeomSQL+`), 0)::float8
		  FROM unnest($1::text[]) WITH ORDINALITY AS t(w, n)
		 ORDER BY n`, "t.w"), wkts)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[float64])
}

// insertDecree — захирамжийн мөр (гүйлгээ дотор).
func insertDecree(ctx context.Context, tx pgx.Tx, d *domain.LandReleaseDecree) error {
	return tx.QueryRow(ctx, `
		INSERT INTO land_release_decree
		    (decree_number, decree_date, file_name, file_type, file_url, size_bytes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, created_at`,
		d.DecreeNumber, d.DecreeDate, d.FileName, d.FileType, d.FileURL, d.SizeBytes, d.CreatedBy,
	).Scan(&d.ID, &d.CreatedAt)
}

// CreateImport — оруулалт, газрууд, (байвал) захирамж ба холбоос, түүхийг НЭГ
// гүйлгээнд бичнэ. Үүссэн газрын тоог буцаана.
func (r *LandReleaseRepo) CreateImport(
	ctx context.Context, imp *domain.LandReleaseImport, features []domain.LandReleaseFeature, decree *domain.LandReleaseDecree,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := resolveStructureType(ctx, tx, imp); err != nil {
		return 0, err
	}
	var decreeID *string
	if decree != nil {
		if err := insertDecree(ctx, tx, decree); err != nil {
			return 0, err
		}
		decreeID = &decree.ID
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO land_release_import (name, description, initial_status, structure_type_id, file_names, feature_count, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		imp.Name, imp.Description, imp.InitialStatus, imp.StructureTypeID, imp.FileNames, len(features), imp.CreatedBy,
	).Scan(&imp.ID); err != nil {
		return 0, err
	}

	wkts := make([]string, len(features))
	files := make([]string, len(features))
	idx := make([]int32, len(features))
	for i, f := range features {
		wkts[i], files[i], idx[i] = f.GeometryWKT, f.SourceFile, int32(f.SourceIndex)
	}
	// Нэг INSERT … SELECT unnest — олон газрыг нэг дор. Талбайг энд бодно.
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		WITH src AS (
			SELECT `+releaseGeomSQL+` AS g, t.f, t.i, t.n
			  FROM unnest($1::text[], $2::text[], $3::int[]) WITH ORDINALITY AS t(w, f, i, n)
		)
		INSERT INTO land_release (import_id, name, description, status, structure_type_id, geometry, area_m2,
		                          source_file, source_index, created_by, updated_by)
		SELECT $4::uuid, $5, $6, $7, $9, g, COALESCE(public.calculate_area_utm(g), 0), f, i, $8, $8
		  FROM src ORDER BY n
		RETURNING id::text`, "t.w"),
		wkts, files, idx, imp.ID, imp.Name, imp.Description, imp.InitialStatus, imp.CreatedBy, imp.StructureTypeID)
	if err != nil {
		return 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	// Сум/дүүрэг — байршлаар тодорхойлж хадгална.
	if _, err := tx.Exec(ctx, landreleasesql.AssignDistrictsSQL, ids); err != nil {
		return 0, err
	}

	if decreeID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO land_release_decree_link (release_id, decree_id, linked_by)
			SELECT unnest($1::uuid[]), $2::uuid, $3`, ids, *decreeID, imp.CreatedBy); err != nil {
			return 0, err
		}
	}
	if imp.Photo != nil {
		if err := insertPhoto(ctx, tx, imp.Photo, ids); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO land_release_status_history (release_id, from_status, to_status, decree_id, note, changed_by)
		SELECT unnest($1::uuid[]), NULL, $2, $3::uuid, 'Оруулсан', $4`,
		ids, imp.InitialStatus, decreeID, imp.CreatedBy); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// resolveStructureType — сонгосон төрөл байгаа эсэхийг шалгах, эсвэл шинэ
// нэрийг (гүйлгээ дотор) нэмнэ. Ижил нэр (том/жижиг үсэг, зай ялгахгүй) байвал
// түүнийг ашиглана — давхар үүсгэхгүй.
func resolveStructureType(ctx context.Context, tx pgx.Tx, imp *domain.LandReleaseImport) error {
	if imp.StructureTypeID != nil {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM land_release_structure_type WHERE id = $1)`,
			*imp.StructureTypeID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return domain.ErrLandReleaseStructureType
		}
		return nil
	}
	name := strings.TrimSpace(imp.StructureTypeName)
	if name == "" {
		return domain.ErrLandReleaseStructureType
	}
	var id int
	if err := tx.QueryRow(ctx, `
		INSERT INTO land_release_structure_type (name, created_by) VALUES ($1, $2)
		ON CONFLICT ((lower(btrim(name)))) DO UPDATE SET name = land_release_structure_type.name
		RETURNING id`, name, imp.CreatedBy).Scan(&id); err != nil {
		return err
	}
	imp.StructureTypeID = &id
	return nil
}

func (r *LandReleaseRepo) ListStructureTypes(ctx context.Context) ([]*domain.LandReleaseStructureType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, created_at, created_by FROM land_release_structure_type ORDER BY lower(name)`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*domain.LandReleaseStructureType, error) {
		t := &domain.LandReleaseStructureType{}
		err := row.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.CreatedBy)
		return t, err
	})
}

// insertPhoto — зургийн PDF мөр ба газруудтай холбоос (гүйлгээ дотор).
func insertPhoto(ctx context.Context, tx pgx.Tx, p *domain.LandReleasePhoto, releaseIDs []string) error {
	if err := tx.QueryRow(ctx, `
		INSERT INTO land_release_photo (file_name, file_url, size_bytes, page_count, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text, created_at`,
		p.FileName, p.FileURL, p.SizeBytes, p.PageCount, p.CreatedBy,
	).Scan(&p.ID, &p.CreatedAt); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO land_release_photo_link (release_id, photo_id, linked_by)
		SELECT unnest($1::uuid[]), $2::uuid, $3`, releaseIDs, p.ID, p.CreatedBy)
	return err
}

// AddPhoto — нэг газарт зураг (PDF) нэмнэ.
func (r *LandReleaseRepo) AddPhoto(ctx context.Context, releaseID string, p *domain.LandReleasePhoto) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM land_release WHERE id = $1)`, releaseID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	if err := insertPhoto(ctx, tx, p, []string{releaseID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *LandReleaseRepo) ListPhotos(ctx context.Context, releaseID string) ([]*domain.LandReleasePhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id::text, p.file_name, p.file_url, p.size_bytes, p.page_count, p.created_at, p.created_by,
		       (SELECT count(*) FROM land_release_photo_link x WHERE x.photo_id = p.id)::int
		  FROM land_release_photo_link l
		  JOIN land_release_photo p ON p.id = l.photo_id
		 WHERE l.release_id = $1
		 ORDER BY p.created_at DESC`, releaseID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*domain.LandReleasePhoto, error) {
		p := &domain.LandReleasePhoto{}
		err := row.Scan(&p.ID, &p.FileName, &p.FileURL, &p.SizeBytes, &p.PageCount, &p.CreatedAt, &p.CreatedBy, &p.ReleaseCount)
		return p, err
	})
}

// ChangeStatus — from → to. decree != nil бол шинэ захирамж үүсч, газарт
// холбогдоно (нэг нэгээр солиход захирамж ТУСДАА хадгалагдана).
func (r *LandReleaseRepo) ChangeStatus(
	ctx context.Context, id string, from, to int, decree *domain.LandReleaseDecree, note, actor string,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE land_release SET status = $1, updated_at = NOW(), updated_by = $2
		 WHERE id = $3 AND status = $4`, to, actor, id, from)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrLandReleaseTransition
	}
	var decreeID *string
	if decree != nil {
		if err := insertDecree(ctx, tx, decree); err != nil {
			return err
		}
		decreeID = &decree.ID
		// Газар бүр НЭГ л захирамжтай — байвал солино.
		if _, err := tx.Exec(ctx, `
			INSERT INTO land_release_decree_link (release_id, decree_id, linked_by)
			VALUES ($1, $2, $3)
			ON CONFLICT (release_id) DO UPDATE
			   SET decree_id = EXCLUDED.decree_id, linked_at = NOW(), linked_by = EXCLUDED.linked_by`,
			id, decree.ID, actor); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO land_release_status_history (release_id, from_status, to_status, decree_id, note, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6)`, id, from, to, decreeID, note, actor); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// listWhere — нэр ЭСВЭЛ захирамжийн дугаараар хайх, төлөв, он.
func listWhere(f domain.LandReleaseFilter) (string, []any) {
	where := []string{"TRUE"}
	var args []any
	if s := strings.TrimSpace(f.Search); s != "" {
		args = append(args, "%"+s+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(lr.name ILIKE $%d OR d.decree_number ILIKE $%d OR lr.release_no::text = $%d)", n, n, n+1))
		// Системийн дугаар «Б-00012» → 12.
		no := strings.TrimPrefix(strings.TrimPrefix(s, "Б-"), "б-")
		args = append(args, strings.TrimLeft(no, "0#№ "))
	}
	if f.Status != 0 {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("lr.status = $%d", len(args)))
	}
	if f.Year != 0 {
		args = append(args, f.Year)
		where = append(where, fmt.Sprintf(""+releaseYearSQL+" = $%d", len(args)))
	}
	if f.StructureTypeID != 0 {
		args = append(args, f.StructureTypeID)
		where = append(where, fmt.Sprintf("lr.structure_type_id = $%d", len(args)))
	}
	return strings.Join(where, " AND "), args
}

// releaseYearSQL — газрын он: захирамжтай бол захирамжийн он, эс бөгөөс оруулсан он.
const releaseYearSQL = `EXTRACT(YEAR FROM COALESCE(d.decree_date, lr.created_at::date))::int`

const releaseFromSQL = `
	FROM land_release lr
	LEFT JOIN land_release_decree_link l ON l.release_id = lr.id
	LEFT JOIN land_release_decree d ON d.id = l.decree_id
	LEFT JOIN land_release_structure_type st ON st.id = lr.structure_type_id
	LEFT JOIN au2 ON au2.code = lr.au2_code
	LEFT JOIN au1 ON au1.code = lr.au1_code`

const releaseColumnsSQL = `
	lr.id::text, lr.release_no, lr.import_id::text, lr.name, lr.description, lr.status,
	lr.structure_type_id, COALESCE(st.name, ''),
	COALESCE(lr.au1_code, ''), COALESCE(au1.name, ''), COALESCE(lr.au2_code, ''), COALESCE(au2.name, ''),
	lr.area_m2::float8, lr.source_file, lr.source_index,
	lr.created_at, lr.created_by, lr.updated_at, lr.updated_by,
	d.id::text, COALESCE(d.decree_number, ''), d.decree_date, COALESCE(d.file_name, ''),
	COALESCE(d.file_type, ''), COALESCE(d.file_url, ''), COALESCE(d.size_bytes, 0),
	d.created_at, COALESCE(d.created_by, '')`

func scanRelease(row pgx.Row, extra ...any) (*domain.LandRelease, error) {
	lr := &domain.LandRelease{}
	var decreeID *string
	d := &domain.LandReleaseDecree{}
	var decreeCreated *time.Time
	dest := []any{
		&lr.ID, &lr.ReleaseNo, &lr.ImportID, &lr.Name, &lr.Description, &lr.Status,
		&lr.StructureTypeID, &lr.StructureTypeName,
		&lr.AU1Code, &lr.AU1Name, &lr.AU2Code, &lr.AU2Name,
		&lr.AreaM2, &lr.SourceFile, &lr.SourceIndex,
		&lr.CreatedAt, &lr.CreatedBy, &lr.UpdatedAt, &lr.UpdatedBy,
		&decreeID, &d.DecreeNumber, &d.DecreeDate, &d.FileName,
		&d.FileType, &d.FileURL, &d.SizeBytes,
		&decreeCreated, &d.CreatedBy,
	}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return nil, err
	}
	lr.StatusName = domain.LandReleaseStatusLabels[lr.Status]
	if decreeID != nil {
		d.ID = *decreeID
		if decreeCreated != nil {
			d.CreatedAt = *decreeCreated
		}
		lr.Decree = d
	}
	return lr, nil
}

func (r *LandReleaseRepo) List(ctx context.Context, f domain.LandReleaseFilter) ([]*domain.LandRelease, int64, error) {
	where, args := listWhere(f)
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) `+releaseFromSQL+` WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	args = append(args, size, (page-1)*size)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s %s WHERE %s ORDER BY lr.release_no DESC LIMIT $%d OFFSET $%d`,
		releaseColumnsSQL, releaseFromSQL, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*domain.LandRelease
	for rows.Next() {
		lr, err := scanRelease(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, lr)
	}
	return out, total, rows.Err()
}

func (r *LandReleaseRepo) GetByID(ctx context.Context, id string) (*domain.LandRelease, error) {
	var wkt string
	lr, err := scanRelease(r.db.QueryRow(ctx, `SELECT `+releaseColumnsSQL+`, ST_AsText(lr.geometry) `+releaseFromSQL+` WHERE lr.id = $1`, id), &wkt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	lr.GeometryWKT = wkt
	if lr.Decree != nil {
		if err := r.db.QueryRow(ctx, `SELECT count(*) FROM land_release_decree_link WHERE decree_id = $1`,
			lr.Decree.ID).Scan(&lr.Decree.ReleaseCount); err != nil {
			return nil, err
		}
	}
	return lr, nil
}

func (r *LandReleaseRepo) ListHistory(ctx context.Context, id string) ([]*domain.LandReleaseHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT h.id::text, h.from_status::int, h.to_status::int, COALESCE(d.decree_number, ''),
		       h.note, h.changed_at, h.changed_by
		  FROM land_release_status_history h
		  LEFT JOIN land_release_decree d ON d.id = h.decree_id
		 WHERE h.release_id = $1
		 ORDER BY h.changed_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.LandReleaseHistory
	for rows.Next() {
		h := &domain.LandReleaseHistory{}
		if err := rows.Scan(&h.ID, &h.FromStatus, &h.ToStatus, &h.DecreeNumber, &h.Note, &h.ChangedAt, &h.ChangedBy); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// MapItems — газрын зурагт (жагсаалтын шүүлттэй ижил, хуудаслалтгүй).
func (r *LandReleaseRepo) MapItems(ctx context.Context, f domain.LandReleaseFilter) ([]domain.LandReleaseMapItem, error) {
	where, args := listWhere(f)
	rows, err := r.db.Query(ctx, `
		SELECT lr.id::text, lr.release_no, lr.name, lr.status, ST_AsText(lr.geometry) `+releaseFromSQL+`
		 WHERE `+where+` ORDER BY lr.release_no`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.LandReleaseMapItem{}
	for rows.Next() {
		var m domain.LandReleaseMapItem
		if err := rows.Scan(&m.ID, &m.ReleaseNo, &m.Name, &m.Status, &m.GeometryWKT); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Stats — дээд хэсгийн статистик. Жагсаалт/газрын зурагтай ИЖИЛ шүүлтээр
// (listWhere). Years нь шүүлтгүй — оны сонголтыг шүүлтээр хумихгүй.
func (r *LandReleaseRepo) Stats(ctx context.Context, f domain.LandReleaseFilter) (*domain.LandReleaseStats, error) {
	st := &domain.LandReleaseStats{
		ByStatus: []domain.LandReleaseStatusRow{}, ByYear: []domain.LandReleaseYearRow{},
		ByDistrict: []domain.LandReleaseDistrictRow{}, ByDistrictYear: []domain.LandReleaseDistrictYearRow{},
		Districts: []domain.LandReleaseDistrictRef{}, Years: []int{},
	}
	where, args := listWhere(f)
	if err := r.db.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(lr.area_m2), 0)::float8, count(DISTINCT d.id)
		  `+releaseFromSQL+` WHERE `+where, args...).Scan(&st.Total, &st.TotalAreaM2, &st.DecreeCount); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT lr.status::int, count(*), COALESCE(sum(lr.area_m2), 0)::float8
		  `+releaseFromSQL+` WHERE `+where+` GROUP BY 1 ORDER BY 1`, args...)
	if err != nil {
		return nil, err
	}
	st.ByStatus, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LandReleaseStatusRow, error) {
		var s domain.LandReleaseStatusRow
		err := row.Scan(&s.Status, &s.Count, &s.AreaM2)
		return s, err
	})
	if err != nil {
		return nil, err
	}
	rows, err = r.db.Query(ctx, `
		SELECT `+releaseYearSQL+` AS y,
		       count(*) FILTER (WHERE lr.status = 1),
		       count(*) FILTER (WHERE lr.status = 2),
		       count(*) FILTER (WHERE lr.status = 3),
		       COALESCE(sum(lr.area_m2), 0)::float8,
		       count(DISTINCT d.id)
		  `+releaseFromSQL+` WHERE `+where+`
		 GROUP BY 1 ORDER BY 1`, args...)
	if err != nil {
		return nil, err
	}
	st.ByYear, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LandReleaseYearRow, error) {
		var y domain.LandReleaseYearRow
		err := row.Scan(&y.Year, &y.Planning, &y.Released, &y.Cancelled, &y.AreaM2, &y.Decrees)
		return y, err
	})
	if err != nil {
		return nil, err
	}
	// Сум/дүүрэг — оруулахад хадгалсан au2_code-оор (landreleasesql.AssignDistrictsSQL).
	rows, err = r.db.Query(ctx, `
		SELECT COALESCE(lr.au2_code, ''), COALESCE(au2.name, ''), COALESCE(au1.name, ''),
		       count(*), count(*) FILTER (WHERE lr.status = 1),
		       count(*) FILTER (WHERE lr.status = 2), count(*) FILTER (WHERE lr.status = 3),
		       COALESCE(sum(lr.area_m2), 0)::float8
		  `+releaseFromSQL+` WHERE `+where+`
		 GROUP BY 1, 2, 3
		 ORDER BY 4 DESC, 2`, args...)
	if err != nil {
		return nil, err
	}
	st.ByDistrict, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LandReleaseDistrictRow, error) {
		var d domain.LandReleaseDistrictRow
		err := row.Scan(&d.Code, &d.Name, &d.AU1Name, &d.Count, &d.Planning, &d.Released, &d.Cancelled, &d.AreaM2)
		return d, err
	})
	if err != nil {
		return nil, err
	}
	rows, err = r.db.Query(ctx, `
		SELECT COALESCE(lr.au2_code, ''), `+releaseYearSQL+`, count(*), COALESCE(sum(lr.area_m2), 0)::float8
		  `+releaseFromSQL+` WHERE `+where+`
		 GROUP BY 1, 2 ORDER BY 2, 1`, args...)
	if err != nil {
		return nil, err
	}
	st.ByDistrictYear, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LandReleaseDistrictYearRow, error) {
		var d domain.LandReleaseDistrictYearRow
		err := row.Scan(&d.Code, &d.Year, &d.Count, &d.AreaM2)
		return d, err
	})
	if err != nil {
		return nil, err
	}
	// Шүүлтгүй — өнгийг дүүрэгт тогтмол оноохын тулд.
	rows, err = r.db.Query(ctx, `
		SELECT DISTINCT COALESCE(lr.au2_code, ''), COALESCE(au2.name, ''), COALESCE(au1.name, '')
		  FROM land_release lr
		  LEFT JOIN au2 ON au2.code = lr.au2_code
		  LEFT JOIN au1 ON au1.code = lr.au1_code
		 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	st.Districts, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.LandReleaseDistrictRef, error) {
		var d domain.LandReleaseDistrictRef
		err := row.Scan(&d.Code, &d.Name, &d.AU1Name)
		return d, err
	})
	if err != nil {
		return nil, err
	}
	rows, err = r.db.Query(ctx, `SELECT DISTINCT `+releaseYearSQL+` `+releaseFromSQL+` ORDER BY 1 DESC`)
	if err != nil {
		return nil, err
	}
	if st.Years, err = pgx.CollectRows(rows, pgx.RowTo[int]); err != nil {
		return nil, err
	}
	return st, nil
}
