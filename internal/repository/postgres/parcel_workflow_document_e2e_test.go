//go:build e2e

// Ажлын урсгалын ШИЛЖИЛТЭД заавал байх хавсралт — БОДИТ PostgreSQL дээр
// (seed + repo + service). Зөвхөн ӨӨРИЙН үүсгэсэн мөрийг цэвэрлэнэ.
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run ParcelWorkflowRequired -v
package postgres_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/repository/postgres"
	"acquisition_backend/internal/seed"
	"acquisition_backend/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// psrdFiles — санах ойн файл хадгалалт (MinIO-гүй).
type psrdFiles struct{ n int }

func (f *psrdFiles) Save(_ context.Context, _, _, filename, contentType string, data []byte, _ bool) (*filestorage.StoredFile, error) {
	f.n++
	return &filestorage.StoredFile{URL: "/e2e-psrd/" + uuid.NewString() + "/" + filename, Name: filename, Type: contentType, SizeBytes: int64(len(data))}, nil
}
func (f *psrdFiles) SaveStream(context.Context, string, string, string, string, io.Reader, int64, bool) (*filestorage.StoredFile, error) {
	return nil, errors.New("not used")
}
func (f *psrdFiles) Read(context.Context, string) ([]byte, error) { return nil, errors.New("not used") }
func (f *psrdFiles) OpenStream(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}
func (f *psrdFiles) Delete(context.Context, string) error { return nil }

func TestE2EParcelWorkflowRequiredDocuments(t *testing.T) {
	dsn := os.Getenv("E2E_DATABASE_URL")
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var dbName string
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName))
	if dbName == "appdb" {
		t.Skip("ажлын appdb дээр ажиллуулахгүй — хуулбар DB (appdb_e2e) ашиглана уу")
	}

	repo := postgres.NewLandAcquisitionRepo(pool)
	svc := service.NewLandService(repo, &psrdFiles{}, nil)
	seeder := seed.NewMainSeeder(pool)
	processing, evaluating, rejected, released := domain.ParcelStatusProcessing, domain.ParcelStatusEvaluating, domain.ParcelStatusRejected, domain.ParcelStatusReleased
	// Seed-ийн (000053) хавсралтын тохиргоог түр цэвэрлэж, 000047-ийн өгөгдмөлийг
	// дангаар нь шалгана; тестийн дараа БУЦААНА.
	var savedReqs [][2]int
	reqRows, err := pool.Query(ctx, `SELECT workflow_id, document_type_id FROM parcel_workflow_required_document`)
	require.NoError(t, err)
	for reqRows.Next() {
		var r [2]int
		require.NoError(t, reqRows.Scan(&r[0], &r[1]))
		savedReqs = append(savedReqs, r)
	}
	reqRows.Close()
	_, err = pool.Exec(ctx, `DELETE FROM parcel_workflow_required_document`)
	require.NoError(t, err)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM parcel_workflow_required_document`)
		for _, r := range savedReqs {
			_, _ = pool.Exec(c, `INSERT INTO parcel_workflow_required_document (workflow_id, document_type_id) VALUES ($1, $2)`, r[0], r[1])
		}
	})
	typeID := func(code string) int {
		var id int
		require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM document_type WHERE type = $1`, code).Scan(&id))
		return id
	}

	// Seed-ийн өмнө байх шилжилтүүд (байхгүй бол үүсгэж, дараа нь устгана).
	var created []int
	t.Cleanup(func() {
		for _, id := range created {
			_ = repo.DeleteParcelWorkflow(context.Background(), id)
		}
	})
	ensureWorkflow := func(from, to int) {
		var id int
		if err := pool.QueryRow(ctx, `SELECT id FROM parcel_workflow WHERE from_status_id = $1 AND to_status_id = $2`, from, to).Scan(&id); err == nil {
			return
		}
		w, err := repo.CreateParcelWorkflow(ctx, &from, to, 99)
		require.NoError(t, err)
		created = append(created, w.ID)
	}
	ensureWorkflow(processing, rejected)
	ensureWorkflow(processing, released)
	var directID int
	if err := pool.QueryRow(ctx, `SELECT id FROM parcel_workflow WHERE from_status_id = $1 AND to_status_id = $2`, evaluating, released).Scan(&directID); err == nil {
		t.Skip("Үнэлгээ хийх → Чөлөөлсөн шилжилт аль хэдийн байна — тестийн тохиргоог хөндөхгүй")
	}

	// ── Хуучин ТӨРӨЛГҮЙ татгалзлын хавсралт (seed-ийн нөхөлтөд) ─────────────
	var acqID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
		VALUES ('E2E-PSRD', 'E2E явцын хавсралт', 2, 'e2e-psrd') RETURNING id`).Scan(&acqID))
	newParcel := func(code string, status int) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by)
			VALUES ($1, $2, $3, 'e2e-psrd') RETURNING id`, acqID, code, status).Scan(&id))
		return id
	}
	addDoc := func(parcel uuid.UUID, name string, typeCode string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO document (name, file_type, file_url, uploaded_by, document_type_id)
			VALUES ($1, 'application/pdf', '/e2e-psrd/' || $1, 'e2e-psrd', (SELECT id FROM document_type WHERE type = NULLIF($2, '')))
			RETURNING id`, name, typeCode).Scan(&id))
		_, err := pool.Exec(ctx, `INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1, $2)`, parcel, id)
		require.NoError(t, err)
		return id
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, q := range []string{
			`DELETE FROM parcel_status_history WHERE acquisition_id = $1`,
			`DELETE FROM compensation WHERE acquisition_id = $1`,
			`DELETE FROM document WHERE id IN (SELECT document_id FROM parcel_document pd JOIN parcel p ON p.id = pd.parcel_id WHERE p.acquisition_id = $1)`,
			`DELETE FROM parcel WHERE acquisition_id = $1`,
			`DELETE FROM land_acquisition WHERE id = $1`,
		} {
			_, _ = pool.Exec(c, q, acqID)
		}
	})
	parcelB := newParcel("E2EPSRD00002", processing)
	legacy := addDoc(parcelB, "huuchin-tatgalzal.pdf", "")
	_, err = pool.Exec(ctx, `INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, created_by, reason, attachment_url, attachment_name)
		VALUES ('E2EPSRD00002', $1, $2, 'e2e-psrd', 'хуучин', '/e2e-psrd/huuchin-tatgalzal.pdf', 'huuchin-tatgalzal.pdf')`, acqID, rejected)
	require.NoError(t, err)

	// ── Seed: очих төлөвөөр БҮХ шилжилтэд (2 удаа — давхардахгүй) ─────────
	require.NoError(t, seeder.SeedParcelWorkflowRequiredDocuments(ctx))
	require.NoError(t, seeder.SeedParcelWorkflowRequiredDocuments(ctx))
	workflows, err := repo.ListParcelWorkflows(ctx)
	require.NoError(t, err)
	for _, w := range workflows {
		codes := []string{}
		for _, d := range w.RequiredDocuments {
			codes = append(codes, d.DocumentTypeCode)
		}
		switch w.ToStatusID {
		case released:
			require.Equal(t, []string{"valuation_report"}, codes, "%s → %s", w.FromStatusName, w.ToStatusName)
		case rejected:
			require.Equal(t, []string{"rejection_document"}, codes, "%s → %s", w.FromStatusName, w.ToStatusName)
		}
		require.NotNil(t, w.RequiredDocuments, "JSON-д null биш [] байна")
	}
	var legacyType string
	require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(dt.type, '') FROM document d LEFT JOIN document_type dt ON dt.id = d.document_type_id WHERE d.id = $1`, legacy).Scan(&legacyType))
	require.Equal(t, "rejection_document", legacyType, "хуучин татгалзлын хавсралт төрөлтэй болно")

	reqs := func(parcel uuid.UUID, to int) []domain.ParcelStatusRequirement {
		list, err := repo.ParcelStatusRequirements(ctx, acqID, parcel, to)
		require.NoError(t, err)
		return list
	}
	codesOf := func(list []domain.ParcelStatusRequirement) []string {
		out := []string{}
		for _, r := range list {
			out = append(out, r.DocumentTypeCode)
		}
		return out
	}
	status := func(parcel uuid.UUID) int {
		var s int
		require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM parcel WHERE id = $1`, parcel).Scan(&s))
		return s
	}
	change := func(parcel uuid.UUID, to int, att *service.ParcelStatusAttachment) error {
		return svc.UpdateParcelStatus(ctx, domain.ParcelStatusChange{ParcelID: parcel, AcquisitionID: acqID, StatusID: to, CreatedBy: "e2e-psrd", Reason: "шалтгаан"}, att)
	}

	// Хуучин баримттай нэгж талбар — «аль хэдийн орсон».
	r := reqs(parcelB, rejected)
	require.Len(t, r, 1)
	require.True(t, r[0].Present)
	require.Equal(t, "huuchin-tatgalzal.pdf", r[0].DocumentName)

	// ── Зөвшилцөх → Татгалзсан: хавсралтгүй → шилжихгүй ───────────────────
	parcelA := newParcel("E2EPSRD00001", processing)
	r = reqs(parcelA, rejected)
	require.Len(t, r, 1)
	require.False(t, r[0].Present)
	var missing *domain.RequiredDocumentsMissingError
	require.ErrorAs(t, change(parcelA, rejected, nil), &missing)
	require.Equal(t, []string{"Татгалзсан тухай баримт"}, missing.Names)
	require.Equal(t, processing, status(parcelA), "төлөв хэвээр")

	// Төрөлгүй файл (өмнөх клиент) → дутуу ганц төрлөөр бүртгэгдэж шилжинэ.
	require.NoError(t, change(parcelA, rejected,
		&service.ParcelStatusAttachment{Filename: "shiidver.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.7")}))
	require.Equal(t, rejected, status(parcelA))
	var histURL, histName string
	require.NoError(t, pool.QueryRow(ctx, `SELECT attachment_url FROM parcel_status_history
		WHERE parcel_id = 'E2EPSRD00001' AND status_id = $1 ORDER BY status_date DESC LIMIT 1`, rejected).Scan(&histURL))
	require.True(t, strings.HasPrefix(histURL, "/e2e-psrd/"), "түүхэнд хавсралтын холбоос үлдэнэ: %q", histURL)

	// Шинэ цонхны урсгал: хавсралтыг УРЬДЧИЛАН оруулаад файлгүй шилжинэ —
	// түүхэнд тэр баримтын холбоос үлдэнэ.
	parcelD := newParcel("E2EPSRD00004", processing)
	addDoc(parcelD, "tatgalzal-urid.pdf", "rejection_document")
	require.NoError(t, change(parcelD, rejected, nil))
	require.NoError(t, pool.QueryRow(ctx, `SELECT attachment_url, attachment_name FROM parcel_status_history
		WHERE parcel_id = 'E2EPSRD00004' AND status_id = $1`, rejected).Scan(&histURL, &histName))
	require.Equal(t, "/e2e-psrd/tatgalzal-urid.pdf", histURL)
	require.Equal(t, "tatgalzal-urid.pdf", histName)

	// ── Зөвшилцөх → Чөлөөлсөн: тайлан; олговор батлагдаагүй бол өмнөх алдаа ──
	r = reqs(parcelB, released)
	require.Equal(t, []string{"valuation_report"}, codesOf(r))
	require.False(t, r[0].Present)
	require.ErrorAs(t, change(parcelB, released, nil), &missing)
	addDoc(parcelB, "unelgee.pdf", "valuation_report")
	require.True(t, reqs(parcelB, released)[0].Present)
	require.ErrorIs(t, change(parcelB, released, nil), domain.ErrValuationReportRequired)
	require.Equal(t, processing, status(parcelB))

	// Хуучин флоу: зөвхөн олговрын valuation_report_url — тайлан орсонд тооцно.
	parcelC := newParcel("E2EPSRD00003", processing)
	_, err = pool.Exec(ctx, `INSERT INTO compensation (acquisition_id, parcel_id, status, valuation_type, valuation_report_url)
		VALUES ($1, 'E2EPSRD00003', 'approved', 'asset', '/e2e-psrd/huuchin-tailan.pdf')`, acqID)
	require.NoError(t, err)
	require.True(t, reqs(parcelC, released)[0].Present, "хуучин олговрын тайлан тооцогдоно")
	require.NoError(t, change(parcelC, released, nil))
	require.Equal(t, released, status(parcelC))

	// ── ШИЛЖИЛТ БҮРД ӨӨР: «Үнэлгээ хийх → Чөлөөлсөн» — гэрээ ─────────────
	_, err = svc.CreateParcelWorkflow(ctx, &evaluating, released, 99, []int{-1})
	require.Error(t, err, "буруу хавсралтын төрөл")
	require.Error(t, pool.QueryRow(ctx, `SELECT id FROM parcel_workflow WHERE from_status_id = $1 AND to_status_id = $2`, evaluating, released).Scan(&directID),
		"алдаатай бол шилжилт ч үүсэхгүй")
	direct, err := svc.CreateParcelWorkflow(ctx, &evaluating, released, 99, []int{typeID("contract")})
	require.NoError(t, err)
	created = append(created, direct.ID)
	require.Len(t, direct.RequiredDocuments, 1)
	require.Equal(t, "contract", direct.RequiredDocuments[0].DocumentTypeCode)

	parcelE := newParcel("E2EPSRD00005", evaluating)
	require.Equal(t, []string{"contract"}, codesOf(reqs(parcelE, released)), "Үнэлгээ хийх-ээс — гэрээ")
	require.Equal(t, []string{"valuation_report"}, codesOf(reqs(parcelB, released)), "Зөвшилцөх-өөс — тайлан хэвээр")
	require.Empty(t, reqs(parcelE, rejected), "тохируулаагүй шилжилт — шаардлагагүй")

	// ── Шилжилтийн хавсралт солих ──────────────────────────────────────────
	var acqOnlyType int
	if err := pool.QueryRow(ctx, `SELECT id FROM document_type WHERE target = 'acquisition' LIMIT 1`).Scan(&acqOnlyType); err == nil {
		require.ErrorIs(t, repo.SetParcelWorkflowRequiredDocuments(ctx, direct.ID, []int{acqOnlyType}), domain.ErrInvalidInput,
			"чөлөөлөлтийн л хавсралтын төрөл нэгж талбарын шилжилтэд тохирохгүй")
	}
	require.ErrorIs(t, repo.SetParcelWorkflowRequiredDocuments(ctx, -999, nil), domain.ErrNotFound)
	contract, report := typeID("contract"), typeID("valuation_report")
	require.NoError(t, repo.SetParcelWorkflowRequiredDocuments(ctx, direct.ID, []int{contract, report, contract}))
	require.ElementsMatch(t, []string{"contract", "valuation_report"}, codesOf(reqs(parcelE, released)))
	require.NoError(t, repo.SetParcelWorkflowRequiredDocuments(ctx, direct.ID, nil))
	require.Empty(t, reqs(parcelE, released), "шаардлагагүй болгож болно")

	// Шилжилт устгахад хавсралтын тохиргоо нь хамт устна.
	require.NoError(t, repo.SetParcelWorkflowRequiredDocuments(ctx, direct.ID, []int{contract}))
	require.NoError(t, repo.DeleteParcelWorkflow(ctx, direct.ID))
	var left int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM parcel_workflow_required_document WHERE workflow_id = $1`, direct.ID).Scan(&left))
	require.Zero(t, left)
}
