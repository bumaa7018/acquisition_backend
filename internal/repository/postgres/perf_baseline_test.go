//go:build perf

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
	"acquisition_backend/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Гүйцэтгэлийн СУУРЬ хэмжилт (docs/performance-optimization-plan.md §2.4).
//
// Энэ нь тест БИШ — оновчлолын өмнөх/дараах хугацааг ИЖИЛ орчинд хэмжиж
// харьцуулах багаж. Хариу шалгахгүй, зөвхөн хугацаа бүртгэнэ.
//
// Ажиллуулах (ЗӨВХӨН тусгаарласан хэмжилтийн сан дээр):
//
//	PERF_DATABASE_URL=postgres://postgres:postgres@localhost:5432/perfdb?sslmode=disable \
//	  go test -tags perf -run TestPerfBaseline -v -timeout 30m ./internal/repository/postgres/
//
// Үндсэн appdb дээр ХЭЗЭЭ Ч бүү ажиллуул.

func perfDSN(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("PERF_DATABASE_URL"); v != "" {
		return v
	}
	t.Skip("PERF_DATABASE_URL тохируулаагүй — хэмжилт алгаслаа")
	return ""
}

// scenario — нэг хэмжих зүйл. fn нь бүрэн repository дуудлагыг гүйцэтгэнэ.
type scenario struct {
	name string
	fn   func(ctx context.Context, repo repository.LandAcquisitionRepository) error
}

// stats — нэг хувилбарын хугацааны нэгтгэл.
type stats struct {
	avg, p50, p95, max time.Duration
	runs, failed       int
	err                error
}

func summarize(ds []time.Duration) stats {
	if len(ds) == 0 {
		return stats{}
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	var total time.Duration
	for _, d := range ds {
		total += d
	}
	idx := func(q float64) time.Duration {
		i := int(float64(len(ds)-1) * q)
		return ds[i]
	}
	return stats{
		avg:  total / time.Duration(len(ds)),
		p50:  idx(0.50),
		p95:  idx(0.95),
		max:  ds[len(ds)-1],
		runs: len(ds),
	}
}

// run — тухайн хувилбарыг `concurrency` зэрэгцээ, нийт `total` удаа ажиллуулна.
func run(ctx context.Context, repo repository.LandAcquisitionRepository, s scenario, concurrency, total int) stats {
	var (
		mu     sync.Mutex
		ds     = make([]time.Duration, 0, total)
		failed int
		firs   error
		wg     sync.WaitGroup
		jobs   = make(chan struct{}, total)
	)
	for i := 0; i < total; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				start := time.Now()
				err := s.fn(ctx, repo)
				el := time.Since(start)
				mu.Lock()
				if err != nil {
					failed++
					if firs == nil {
						firs = err
					}
				} else {
					ds = append(ds, el) // ЗӨВХӨН амжилттай дуудлага
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	out := summarize(ds)
	out.failed = failed
	out.err = firs
	return out
}

func TestPerfBaseline(t *testing.T) {
	dsn := perfDSN(t)
	ctx := context.Background()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("DSN задарсангүй: %v", err)
	}
	// Production-той ижил хэмжээ (cmd/api/main.go дахь 30 холболт).
	cfg.MaxConns = 30
	// PERF_SEQUENTIAL=1 — холболтыг НЭГ болгож, дашбоардын 8 зэрэгцээ асуулгыг
	// ДАРААЛУУЛНА. Ингэснээр "ижил ажлыг дараалуулбал хэдэн хугацаа зарцуулах
	// вэ" гэдгийг ТААМАГЛАХГҮЙ, бодитоор хэмжинэ (PHP-FPM хүсэлт дотроо ажлаа
	// дараалуулдагтай ижил нөхцөл).
	if os.Getenv("PERF_SEQUENTIAL") == "1" {
		cfg.MaxConns = 1
	}
	// ТЕСТИЙН хамгаалалтын таслалт (төлөвлөгөө §8-д зөвшөөрсөн). Үүнгүйгээр
	// хүнд хувилбар санах ойг дүүргэж postgres-ийн backend процессыг
	// алуулж байсан ("server process exited with exit code 2"). Timeout
	// болсон дуудлагыг амжилттай хугацаанд ТООЦОХГҮЙ.
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "30000"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("DB холбогдсонгүй: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping бүтсэнгүй: %v", err)
	}

	var parcels int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM parcel`).Scan(&parcels); err != nil {
		t.Fatalf("fixture уншигдсангүй: %v", err)
	}
	if parcels < 1000 {
		t.Fatalf("fixture дутуу (%d нэгж талбар) — scripts/perf_fixture.sql ажиллуул", parcels)
	}

	repo := postgres.NewLandAcquisitionRepo(pool)

	// Хэмжилтэд ашиглах бодит чөлөөлөлтийн id (жагсаалтын эхнийх).
	var acqID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM land_acquisition ORDER BY plan_code LIMIT 1`).Scan(&acqID); err != nil {
		t.Fatalf("чөлөөлөлт олдсонгүй: %v", err)
	}
	year := time.Now().Year()

	scenarios := []scenario{
		{"dashboard", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, err := r.GetDashboard(ctx, domain.DashboardFilter{})
			return err
		}},
		{"dashboard+жил", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, err := r.GetDashboard(ctx, domain.DashboardFilter{Years: []int{year, year - 1}})
			return err
		}},
		{"report summary", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, err := r.GetReportSummary(ctx, domain.ReportParcelFilter{})
			return err
		}},
		{"report download (20)", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, _, err := r.ListReportParcels(ctx, domain.ReportParcelFilter{Page: 1, PageSize: 20})
			return err
		}},
		{"чөлөөлөлтийн жагсаалт", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, _, err := r.List(ctx, domain.LandAcquisitionFilter{Page: 1, PageSize: 15})
			return err
		}},
		{"нэгдсэн талбарын жагсаалт", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, _, err := r.ListGlobalParcels(ctx, domain.GlobalParcelFilter{Page: 1, PageSize: 20})
			return err
		}},
		{"чөлөөлөлтийн талбарууд", func(ctx context.Context, r repository.LandAcquisitionRepository) error {
			_, _, err := r.GetParcels(ctx, acqID, domain.ParcelFilter{Page: 1, PageSize: 20})
			return err
		}},
	}

	type row struct {
		name       string
		one        stats
		concur     stats
		failure    string
		skipConcur bool
	}
	results := make([]row, 0, len(scenarios))

	for _, s := range scenarios {
		// Дулаацуулах (plan cache, buffer) — хэмжилтэд оруулахгүй.
		_ = s.fn(ctx, repo)

		one := run(ctx, repo, s, 1, 10)
		r := row{name: s.name, one: one}
		if one.err != nil {
			r.failure = one.err.Error()
		}
		// 1x дээр аль хэдийн маш удаан бол 10x нь санах ойгоор DB-г
		// унагаадаг тул зэрэгцээ хэмжилтийг ХИЙХГҮЙ (тусад нь тэмдэглэнэ).
		if one.err == nil && one.max < 2*time.Second {
			r.concur = run(ctx, repo, s, 10, 30)
			if r.concur.err != nil {
				r.failure = r.concur.err.Error()
			}
		} else {
			r.skipConcur = true
		}
		results = append(results, r)
	}

	t.Log("")
	t.Log("=== СУУРЬ ХЭМЖИЛТ (perfdb: 100 чөлөөлөлт / 10k талбар / 30k олговор / 50k түүх) ===")
	t.Logf("%-28s | %10s %10s | %10s %10s %10s", "хувилбар", "1x дундаж", "1x max", "10x дундаж", "10x p95", "10x max")
	t.Log("-----------------------------+-----------------------+-------------------------------")
	for _, r := range results {
		if r.failure != "" {
			t.Logf("%-28s | АЛДАА: %s", r.name, r.failure)
			continue
		}
		if r.skipConcur {
			t.Logf("%-28s | %10s %10s | %s", r.name, fmtd(r.one.avg), fmtd(r.one.max),
				"10x АЛГАССАН — 1x аль хэдийн 2 сек+ (DB-г унагаах эрсдэлтэй)")
			continue
		}
		t.Logf("%-28s | %10s %10s | %10s %10s %10s",
			r.name,
			fmtd(r.one.avg), fmtd(r.one.max),
			fmtd(r.concur.avg), fmtd(r.concur.p95), fmtd(r.concur.max))
	}
	t.Log("")
}

func fmtd(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%.2fс", d.Seconds())
	}
	return fmt.Sprintf("%dмс", d.Milliseconds())
}
