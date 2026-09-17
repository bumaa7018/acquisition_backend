//go:build perf

package postgres_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
	"acquisition_backend/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ТЭНЦҮҮ БАЙДЛЫН ХАРЬЦУУЛАГЧ (docs/performance-optimization-plan.md §2.4).
//
// Зорилго: SQL оновчлол хийхэд ҮР ДҮН өөрчлөгдөөгүйг НОТЛОХ. Төлөвлөгөөний
// хатуу нөхцөл — "Оновчлолоос ӨМНӨ хуучин хэрэгжүүлэлтийг өөрчлөгдөхгүй
// reference болгон хадгална".
//
// Ажиллах журам:
//
//  1. Оновчлол хийхийн ӨМНӨ эталоныг бичиж авна:
//     PERF_RECORD=1 PERF_DATABASE_URL=... go test -tags perf -run TestEquivalence ./internal/repository/postgres/
//  2. Асуулгыг дахин бичнэ.
//  3. Дахин ажиллуулна (PERF_RECORD-гүйгээр). Ялгаа гарвал тест УНАНА.
//
// Эталон файлууд: internal/repository/postgres/testdata/equivalence/*.json
// (git-д хийнэ — code review дээр "юу өөрчлөгдсөн"-ийг шууд харуулна).
//
// АНХААР: эталон нь fixture-ийн ТУХАЙН хувилбартай уяатай. `perf_fixture.sql`
// нь `now()` ашигладаг тул fixture-ыг ДАХИН үүсгэвэл цагийн талбарууд өөрчлөгдөж
// бүх эталон хүчингүй болно — тэр тохиолдолд оновчлол хийхийн өмнө PERF_RECORD=1-ээр
// дахин бичнэ (хуучин код дээр!), дараа нь л асуулгад гар хүрнэ.

const equivDir = "testdata/equivalence"

// equivCase — нэг харьцуулах дуудлага. fn нь repository-гийн хариуг буцаана.
type equivCase struct {
	name string
	fn   func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error)
}

func TestEquivalence(t *testing.T) {
	dsn := perfDSN(t)
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("DB холбогдсонгүй: %v", err)
	}
	defer pool.Close()

	var parcels int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM parcel`).Scan(&parcels); err != nil {
		t.Fatalf("fixture уншигдсангүй: %v", err)
	}
	if parcels < 1000 {
		t.Fatalf("fixture дутуу (%d нэгж талбар) — scripts/perf_fixture.sql ажиллуул", parcels)
	}

	repo := postgres.NewLandAcquisitionRepo(pool)

	// Бодит id-ууд — эрэмбэ тогтвортой байхаар сонгоно.
	var acqID uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT id FROM land_acquisition ORDER BY plan_code LIMIT 1`).Scan(&acqID); err != nil {
		t.Fatalf("чөлөөлөлт олдсонгүй: %v", err)
	}
	// ЖИЛИЙН шүүлт нь `la.start_date`-аар БИШ, нэгж талбарын СҮҮЛИЙН
	// статусын оноор явдаг (land_acquisition_repo.go:221, :4499 болон
	// dashboard). Fixture-д тэр он нь бүхэлдээ тухайн жил тул эндээс
	// динамикаар авна — эс бөгөөс шүүлт хоосон буцаж, эталон нь ямар ч
	// оновчлолыг барихаа болино.
	var latestYear int
	if err := pool.QueryRow(ctx, `
		SELECT EXTRACT(YEAR FROM MAX(psh.status_date))::int
		  FROM parcel_status_history psh`).Scan(&latestYear); err != nil {
		t.Fatalf("статусын он олдсонгүй: %v", err)
	}
	// Хоёр утга — IN (…) жагсаалтын салааг мөн хамруулна.
	years := []int{latestYear, latestYear - 1}

	// Хувилбарууд нь зөвхөн хурдан замыг биш, ӨӨР ӨӨР шүүлтийн САЛААНУУДЫГ
	// хамрах ёстой — оновчлол ихэвчлэн нэг салааг гэмтээж нөгөөг нь орхидог.
	cases := []equivCase{
		{"dashboard_all", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{})
		}},
		{"dashboard_years", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{Years: years})
		}},
		{"dashboard_one_acq", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{AcquisitionID: acqID.String()})
		}},
		{"dashboard_category", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{GeneralCategoryID: 1, SubCategoryID: 11})
		}},
		{"dashboard_parcel_status", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{
				FilterByParcelStatus: true,
				ParcelStatusID:       int(domain.ParcelStatusEvaluating),
			})
		}},
		{"dashboard_au2", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{AU2Code: "1101"})
		}},
		// Мэргэжлийн байгууллагын хүрээ — нэгж талбарын түвшний нэмэлт
		// параметртэй цорын ганц салаа. Жилийн шүүлттэй хосолсон хувилбар нь
		// параметрийн ДУГААРЛАЛТ зөв эсэхийг шалгана (он → байгууллага дараалал).
		{"dashboard_prof_org", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{ProfessionalOrgID: "2"})
		}},
		{"dashboard_prof_org_years", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{ProfessionalOrgID: "2", Years: years})
		}},
		{"dashboard_prof_org_parcel_status", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetDashboard(ctx, domain.DashboardFilter{
				ProfessionalOrgID:    "2",
				FilterByParcelStatus: true,
				ParcelStatusID:       int(domain.ParcelStatusEvaluating),
			})
		}},

		{"report_summary_all", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetReportSummary(ctx, domain.ReportParcelFilter{})
		}},
		{"report_summary_years", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetReportSummary(ctx, domain.ReportParcelFilter{Years: years})
		}},
		{"report_summary_cash", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetReportSummary(ctx, domain.ReportParcelFilter{CompensationType: "cash"})
		}},
		// ЗОРИУДААР хоосон. Задаргааны талбарууд `nil` хэвээр үлдэх ёстой —
		// хоосон массив болвол JSON хариу `null`-аас `[]` болж өөрчлөгдөнө.
		{"report_summary_none", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return r.GetReportSummary(ctx, domain.ReportParcelFilter{PlanCodeExact: "БАЙХГҮЙ-ПЛАН"})
		}},

		{"report_parcels_p1", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.ListReportParcels(ctx, domain.ReportParcelFilter{Page: 1, PageSize: 20}))
		}},
		{"report_parcels_p7", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.ListReportParcels(ctx, domain.ReportParcelFilter{Page: 7, PageSize: 20}))
		}},

		{"acq_list_p1", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.List(ctx, domain.LandAcquisitionFilter{Page: 1, PageSize: 15}))
		}},
		{"acq_list_sort_name_desc", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.List(ctx, domain.LandAcquisitionFilter{
				Page: 1, PageSize: 15, SortBy: "acquisition_name", SortDir: "desc"}))
		}},
		{"acq_list_years", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.List(ctx, domain.LandAcquisitionFilter{Page: 1, PageSize: 15, Years: years}))
		}},
		// ЗОРИУДААР хоосон: таарахгүй он. Оновчлол "юу ч олдохгүй" салааг
		// гэмтээгээд бүх мөрийг буцаах нь түгээмэл алдаа тул түүнийг барина.
		{"acq_list_years_none", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.List(ctx, domain.LandAcquisitionFilter{Page: 1, PageSize: 15, Years: []int{1999}}))
		}},

		{"global_parcels_p1", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.ListGlobalParcels(ctx, domain.GlobalParcelFilter{Page: 1, PageSize: 20}))
		}},
		{"global_parcels_status", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.ListGlobalParcels(ctx, domain.GlobalParcelFilter{
				Page: 1, PageSize: 20, StatusID: int(domain.ParcelStatusEvaluating)}))
		}},

		{"acq_parcels_p1", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.GetParcels(ctx, acqID, domain.ParcelFilter{Page: 1, PageSize: 20}))
		}},
		{"acq_parcels_sort_area_desc", func(ctx context.Context, r repository.LandAcquisitionRepository) (any, error) {
			return listResult(r.GetParcels(ctx, acqID, domain.ParcelFilter{
				Page: 1, PageSize: 20, SortBy: "area_m2", SortDir: "desc"}))
		}},
	}

	record := os.Getenv("PERF_RECORD") == "1"
	if record {
		if err := os.MkdirAll(equivDir, 0o755); err != nil {
			t.Fatalf("эталон хавтас үүсгэж чадсангүй: %v", err)
		}
		t.Log("PERF_RECORD=1 — ЭТАЛОН БИЧИЖ БАЙНА (оновчлолын ӨМНӨХ код дээр ажиллуулж байгаа эсэхээ шалга)")
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.fn(ctx, repo)
			if err != nil {
				t.Fatalf("дуудлага амжилтгүй: %v", err)
			}
			// Тогтвортой JSON: map-ийн түлхүүрийг Go нь цагаан толгойн
			// дарааллаар бичдэг тул ажиллуулалт бүрд ижил гарна.
			buf, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("JSON болгож чадсангүй: %v", err)
			}
			buf = append(buf, '\n')

			path := filepath.Join(equivDir, c.name+".json")
			if record {
				if err := os.WriteFile(path, buf, 0o644); err != nil {
					t.Fatalf("эталон бичигдсэнгүй: %v", err)
				}
				t.Logf("эталон бичлээ: %s (%d байт)", path, len(buf))
				return
			}

			want, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				t.Fatalf("эталон алга: %s — эхлээд PERF_RECORD=1-ээр ХУУЧИН код дээр бичиж ав", path)
			}
			if err != nil {
				t.Fatalf("эталон уншигдсангүй: %v", err)
			}
			if !bytes.Equal(want, buf) {
				t.Errorf("ҮР ДҮН ӨӨРЧЛӨГДСӨН — %s\n%s", c.name, firstDiff(want, buf))
			}
		})
	}
}

// listResult — (мөрүүд, нийт, алдаа) гурвалыг нэг харьцуулах утга болгоно.
// Нийт тоо нь ихэвчлэн ТУСДАА count асуулгаар гардаг тул мөрүүдтэй хамт
// хадгалах ёстой — оновчлол count-ыг гэмтээх нь түгээмэл алдаа.
func listResult[T any](rows []T, total int64, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return map[string]any{"total": total, "rows": rows}, nil
}

// firstDiff — эхний зөрсөн мөрийг эргэн тойрны контексттэй харуулна.
// Бүтэн diff хэвлэвэл лог хэдэн мянган мөр болох тул зөвхөн эхний зөрөө.
func firstDiff(want, got []byte) string {
	w := strings.Split(string(want), "\n")
	g := strings.Split(string(got), "\n")
	n := len(w)
	if len(g) < n {
		n = len(g)
	}
	for i := 0; i < n; i++ {
		if w[i] == g[i] {
			continue
		}
		var b strings.Builder
		lo := i - 3
		if lo < 0 {
			lo = 0
		}
		for j := lo; j < i; j++ {
			fmt.Fprintf(&b, "  %d: %s\n", j+1, w[j])
		}
		fmt.Fprintf(&b, "- %d: %s   (эталон)\n", i+1, w[i])
		fmt.Fprintf(&b, "+ %d: %s   (одоогийн)\n", i+1, g[i])
		return b.String()
	}
	return fmt.Sprintf("мөрийн ТОО зөрж байна: эталон %d, одоогийн %d", len(w), len(g))
}
