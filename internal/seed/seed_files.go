package seed

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"acquisition_backend/internal/pkg/filestorage"
)

// unescapeSegment — URL сегментийг тайлна. Хуучин бичлэгүүд escape хийгдээгүй
// (түүхий '%', зай агуулсан) байж болох тул алдаа гарвал байгаагаар нь авна.
func unescapeSegment(seg string) string {
	if decoded, err := url.PathUnescape(seg); err == nil {
		return decoded
	}
	return seg
}

// fileURLColumns — DB-д файлын URL хадгалдаг бүх багана.
// Шинэ багана нэмэгдвэл ЭНД БАС нэмэх — эс тэгвээс шилжүүлэлт тэрийг өнгөрөөнө.
var fileURLColumns = []struct{ table, column string }{
	{"document", "file_url"},
	{"assets", "photo_pdf_url"},
	{"compensation", "valuation_report_url"},
}

// SeedFileStorageMigration — хуучин ЛОКАЛ файлуудыг файлын систем (MinIO) руу
// зөөж, DB дахь URL-уудыг шинэ хэлбэрт (BaseURL/<key>) шинэчилнэ.
//
// Идемпотент: аль хэдийн шинэ BaseURL-тэй болсон бичлэгийг өнгөрөөнө. Тиймээс
// дахин ажиллуулж болно (жишээ: хагас дуусаад тасалдсан бол).
//
// legacy нь хуучин локал хадгалалт (MockStorage) — түүгээр файлыг уншиж,
// target (S3) руу ЯГ ижил namespace/owner/нэрээр бичнэ.
func (s *MainSeeder) SeedFileStorageMigration(
	ctx context.Context,
	legacy filestorage.Storage,
	target filestorage.Storage,
	newBaseURL string,
) error {
	// Тохиргоо бэлэн болоогүй бол АЛДАА буцаана — nil буцаавал runner үүнийг
	// "хэрэгжсэн" гэж бүртгэж, файлын систем асаалттай болсон хойно ч дахин
	// ажиллахгүй болно (жинхэнэ шилжүүлэлт хийгдээгүй үлдэнэ).
	if legacy == nil || target == nil {
		return errors.New("file storage not configured yet — migration will retry on next start")
	}
	newPrefix := strings.TrimRight(newBaseURL, "/") + "/"

	var moved, skipped, failed int
	for _, col := range fileURLColumns {
		rows, err := s.db.Query(ctx,
			`SELECT id::text, `+col.column+` FROM `+col.table+
				` WHERE `+col.column+` IS NOT NULL AND `+col.column+` <> ''`)
		if err != nil {
			// Хүснэгт байхгүй байж болно (хөгжүүлэлтийн өөр төлөв) — өнгөрөөнө.
			slog.Warn("file migration: table unreadable", "table", col.table, "error", err)
			continue
		}
		type row struct{ id, url string }
		var pending []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.url); err != nil {
				rows.Close()
				return err
			}
			pending = append(pending, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		for _, r := range pending {
			if strings.HasPrefix(r.url, newPrefix) {
				skipped++
				continue // аль хэдийн шинэ хадгалалтад
			}
			namespace, ownerID, name, ok := splitLegacyURL(r.url)
			if !ok {
				slog.Warn("file migration: URL-ийг задлаж чадсангүй",
					"table", col.table, "id", r.id, "url", r.url)
				failed++
				continue
			}
			data, err := legacy.Read(ctx, r.url)
			if err != nil {
				// Файл диск дээр байхгүй (устсан/зөөгдсөн) — бичлэгийг хөндөхгүй.
				slog.Warn("file migration: файл уншиж чадсангүй",
					"table", col.table, "id", r.id, "url", r.url, "error", err)
				failed++
				continue
			}
			// unique=false — нэрийг ЯГ хэвээр хадгалж, URL-ийн бүтэц таарна.
			stored, err := target.Save(ctx, namespace, ownerID, name, "", data, false)
			if err != nil {
				slog.Warn("file migration: хадгалж чадсангүй",
					"table", col.table, "id", r.id, "error", err)
				failed++
				continue
			}
			if _, err := s.db.Exec(ctx,
				`UPDATE `+col.table+` SET `+col.column+` = $1 WHERE id::text = $2`,
				stored.URL, r.id,
			); err != nil {
				return err
			}
			moved++
		}
	}

	slog.Info("file storage migration done", "moved", moved, "skipped", skipped, "failed", failed)
	return nil
}

// splitLegacyURL — хуучин URL-ээс namespace/ownerID/файлын нэрийг гаргана.
//
// Хуучин бүтэц: <BaseURL>/<namespace>/<ownerID>/<storedName>
// (жишээ: http://localhost:8080/mock-files/acquisition/<uuid>/<uuid>_file.pdf)
func splitLegacyURL(fileURL string) (namespace, ownerID, name string, ok bool) {
	trimmed := strings.TrimSuffix(fileURL, "/")
	segments := strings.Split(trimmed, "/")
	if len(segments) < 3 {
		return "", "", "", false
	}
	// Хамгийн сүүлийн 3 сегмент нь namespace/owner/нэр.
	tail := segments[len(segments)-3:]
	for i, seg := range tail {
		if seg == "" {
			return "", "", "", false
		}
		tail[i] = unescapeSegment(seg)
	}
	return tail[0], tail[1], tail[2], true
}

// SeedDroneImagesToFileStorage — файлын систем нэвтрэхээс ӨМНӨ орсон дроны
// ортофотуудыг (file_url хоосон) файлын систем руу зөөнө.
//
// Файлыг GeoServer-ийн мозайк хавтсаас (рендерийн кэш) уншина — тэр нь эдгээр
// зургийн цорын ганц хуулбар байсан. Зөөсний дараа кэш нь дахин бүтээгдэх
// боломжтой болж, эх сурвалж нь файлын систем болно.
func (s *MainSeeder) SeedDroneImagesToFileStorage(
	ctx context.Context,
	target filestorage.Storage,
	droneDir string,
) error {
	if target == nil || droneDir == "" {
		return errors.New("file storage/drone dir not configured yet — will retry on next start")
	}

	rows, err := s.db.Query(ctx, `
		SELECT id::text, acquisition_id::text, file_name
		FROM drone_image
		WHERE file_url IS NULL OR file_url = ''`)
	if err != nil {
		slog.Warn("drone migration: хүснэгт уншиж чадсангүй", "error", err)
		return nil // drone_image хүснэгт байхгүй байж болно
	}
	type row struct{ id, acqID, fileName string }
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.acqID, &r.fileName); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var moved, failed int
	for _, r := range pending {
		data, err := os.ReadFile(filepath.Join(droneDir, r.fileName))
		if err != nil {
			slog.Warn("drone migration: файл уншиж чадсангүй",
				"file", r.fileName, "error", err)
			failed++
			continue
		}
		// unique=false — нэр нь мозайкийн `location`-той таарах ёстой.
		stored, err := target.Save(ctx, "drone", r.acqID, r.fileName, "image/tiff", data, false)
		if err != nil {
			slog.Warn("drone migration: хадгалж чадсангүй", "file", r.fileName, "error", err)
			failed++
			continue
		}
		if _, err := s.db.Exec(ctx,
			`UPDATE drone_image SET file_url = $1 WHERE id::text = $2`, stored.URL, r.id,
		); err != nil {
			return err
		}
		moved++
	}
	slog.Info("drone image file storage migration done", "moved", moved, "failed", failed)
	return nil
}
