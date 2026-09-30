package legacyimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleDir — government-legacy-import-ийн жишээ файлууд (байхгүй бол алгасна).
func sampleDir(t *testing.T, sub string) string {
	t.Helper()
	dir := os.Getenv("LEGACY_IMPORT_SAMPLES")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "government-legacy-import", "input")
	}
	path := filepath.Join(dir, sub)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("жишээ файл алга: %s", path)
	}
	return path
}

func loadDir(t *testing.T, dir string) []UploadedFile {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []UploadedFile
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, UploadedFile{Name: e.Name(), Data: data})
	}
	return out
}

func TestReadSampleBoundary(t *testing.T) {
	sets := groupShapefiles(loadDir(t, sampleDir(t, "чөлөөлөлт")))
	if len(sets) != 1 || len(sets[0].issues()) != 0 {
		t.Fatalf("sets=%d issues=%v", len(sets), sets)
	}
	features, srid, err := readShapefileSet(sets[0])
	if err != nil {
		t.Fatal(err)
	}
	if srid != 32648 {
		t.Fatalf("WGS_1984_UTM_Zone_48N → 32648, got %d", srid)
	}
	if len(features) != 1 || attr(features[0].props, planParcelFields...) != "4932803" {
		t.Fatalf("features=%d props=%v", len(features), features[0].props)
	}
	if !strings.HasPrefix(features[0].wkt, "POLYGON((") {
		t.Fatalf("wkt=%.60s", features[0].wkt)
	}
	if !strings.Contains(attr(features[0].props, "gazner"), "Сонгинохайрхан") {
		t.Fatalf("UTF-8 атрибут буруу уншигдав: %q", attr(features[0].props, "gazner"))
	}
}

func TestReadSampleParcels(t *testing.T) {
	sets := groupShapefiles(loadDir(t, sampleDir(t, "нэгж-талбар")))
	features, srid, err := readShapefileSet(sets[0])
	if err != nil {
		t.Fatal(err)
	}
	if srid != 4326 || len(features) != 177 {
		t.Fatalf("srid=%d features=%d", srid, len(features))
	}
	first := features[0]
	if parcelKey(attr(first.props, parcelIDFields...)) != "1802301426" || *intAttr(first.props, statusCodeFields...) != 3 {
		t.Fatalf("props=%v", first.props)
	}
}

func TestReadSampleReport(t *testing.T) {
	files := loadDir(t, sampleDir(t, "үнэлгээ"))
	var batches [][]ReportRow
	for _, f := range files {
		if !strings.HasSuffix(f.Name, ".xlsx") {
			continue
		}
		rows, _, err := readReport(f.Name, f.Data)
		if err != nil {
			t.Fatal(err)
		}
		batches = append(batches, rows)
	}
	rows, _ := mergeReports(batches)
	if len(rows) != 176 {
		t.Fatalf("CLI-тэй ижил 176 мөр хүлээв, %d", len(rows))
	}
	total := 0.0
	for i := range rows {
		total += rows[i].amount()
	}
	if int64(total) != 35979437529 {
		t.Fatalf("нийт дүн CLI-тэй таарахгүй: %.0f", total)
	}
}

func TestTextHelpers(t *testing.T) {
	id, old, err := parseParcelIDs("1461500830 18642313828041")
	if err != nil || id != "1461500830" || old != "18642313828041" {
		t.Fatalf("%q %q %v", id, old, err)
	}
	if d, _ := parseDecree("2025.01.24-А70"); d == nil || d.number != "А/70" || d.date != "2025-01-24" {
		t.Fatalf("decree=%+v", d)
	}
	if n, ok := parseNumber("1 462 568 000₮"); !ok || n != 1462568000 {
		t.Fatalf("n=%v", n)
	}
	if got := syntheticParcelID("POLYGON((1 1,2 2))"); len(got) != 12 || !strings.HasPrefix(got, "ERR") {
		t.Fatalf("synthetic=%q", got)
	}
}
