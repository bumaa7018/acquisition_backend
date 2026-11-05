package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/repository"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body)) //nolint:errcheck
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCollectSources_ZipAndLoose(t *testing.T) {
	z := zipOf(t, map[string]string{
		"dir/a.shp":        "SHP-A",
		"dir/a.prj":        "PRJ-A",
		"dir/a.dbf":        "ignored",
		"__MACOSX/._a.shp": "junk",
	})
	srcs, err := collectSources([]UploadedFile{
		{Filename: "bundle.zip", Data: z},
		{Filename: "b.shp", Data: []byte("SHP-B")},
		{Filename: "B.PRJ", Data: []byte("PRJ-B")},
		{Filename: "c.shp", Data: []byte("SHP-C")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(srcs) != 3 {
		t.Fatalf("want 3 sources, got %d", len(srcs))
	}
	want := map[string]string{"bundle.zip/dir/a.shp": "PRJ-A", "b.shp": "PRJ-B", "c.shp": ""}
	for _, s := range srcs {
		prj, ok := want[s.name]
		if !ok {
			t.Fatalf("unexpected source %q", s.name)
		}
		got := ""
		if s.prj != nil {
			got = *s.prj
		}
		if got != prj {
			t.Errorf("%s: prj = %q, want %q", s.name, got, prj)
		}
	}
}

func TestCollectSources_Rejects(t *testing.T) {
	cases := map[string][]UploadedFile{
		"wrong ext":  {{Filename: "a.kml", Data: []byte("x")}},
		"zip no shp": {{Filename: "a.zip", Data: zipOf(t, map[string]string{"a.dbf": "x"})}},
		"broken zip": {{Filename: "a.zip", Data: []byte("not a zip")}},
	}
	for name, files := range cases {
		_, err := collectSources(files)
		var fe *LandReleaseFileError
		if !errors.As(err, &fe) {
			t.Errorf("%s: want LandReleaseFileError, got %v", name, err)
		}
	}
}

func TestLandReleaseCreate_Validation(t *testing.T) {
	svc := NewLandReleaseService(nil, nil)
	ctx := t.Context()
	if _, err := svc.Create(ctx, CreateLandReleaseRequest{Name: "x", Status: domain.LandReleaseStatusReleased,
		Files: []UploadedFile{{Filename: "a.shp"}}}, "u"); !errors.Is(err, domain.ErrLandReleaseDecreeRequired) {
		t.Errorf("released without decree: %v", err)
	}
	if _, err := svc.Create(ctx, CreateLandReleaseRequest{Name: "x", Status: domain.LandReleaseStatusCancelled}, "u"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("cancelled on create: %v", err)
	}
	if _, err := svc.Create(ctx, CreateLandReleaseRequest{Name: "x", Status: domain.LandReleaseStatusPlanning,
		Files: []UploadedFile{{Filename: "a.shp"}}}, "u"); !errors.Is(err, domain.ErrLandReleaseStructureType) {
		t.Errorf("no structure type: %v", err)
	}
	if _, err := svc.Create(ctx, CreateLandReleaseRequest{Name: "x", Status: domain.LandReleaseStatusPlanning,
		StructureTypeName: "  "}, "u"); !errors.Is(err, domain.ErrLandReleaseStructureType) {
		t.Errorf("blank structure type: %v", err)
	}
	if _, err := svc.Create(ctx, CreateLandReleaseRequest{Name: "x", Status: domain.LandReleaseStatusPlanning,
		StructureTypeName: "Гэр"}, "u"); !errors.Is(err, domain.ErrLandReleaseNoFiles) {
		t.Errorf("no files: %v", err)
	}
}

func TestLandReleaseTransitionAllowed(t *testing.T) {
	ok := [][2]int{{1, 2}, {1, 3}}
	bad := [][2]int{{2, 1}, {2, 3}, {3, 1}, {3, 2}, {1, 1}}
	for _, p := range ok {
		if !domain.LandReleaseTransitionAllowed(p[0], p[1]) {
			t.Errorf("%v should be allowed", p)
		}
	}
	for _, p := range bad {
		if domain.LandReleaseTransitionAllowed(p[0], p[1]) {
			t.Errorf("%v should be rejected", p)
		}
	}
}

// ── Бүтэн урсгал: shapefile байт → zip/.prj → координатын шалгалт → repo ──

func testShp(records ...[][2]float64) []byte {
	var body []byte
	for n, ring := range records {
		contentLen := 4 + 32 + 4 + 4 + 4 + len(ring)*16
		rec := make([]byte, 8+contentLen)
		binary.BigEndian.PutUint32(rec[0:], uint32(n+1))
		binary.BigEndian.PutUint32(rec[4:], uint32(contentLen/2))
		off := 8
		binary.LittleEndian.PutUint32(rec[off:], 5)
		off += 4 + 32
		binary.LittleEndian.PutUint32(rec[off:], 1)
		binary.LittleEndian.PutUint32(rec[off+4:], uint32(len(ring)))
		off += 8 + 4
		for _, p := range ring {
			binary.LittleEndian.PutUint64(rec[off:], math.Float64bits(p[0]))
			binary.LittleEndian.PutUint64(rec[off+8:], math.Float64bits(p[1]))
			off += 16
		}
		body = append(body, rec...)
	}
	buf := make([]byte, 100+len(body))
	binary.BigEndian.PutUint32(buf[0:], 9994)
	binary.BigEndian.PutUint32(buf[24:], uint32(len(buf)/2))
	binary.LittleEndian.PutUint32(buf[28:], 1000)
	binary.LittleEndian.PutUint32(buf[32:], 5)
	copy(buf[100:], body)
	return buf
}

func sq(x, y, d float64) [][2]float64 {
	return [][2]float64{{x, y}, {x + d, y}, {x + d, y + d}, {x, y + d}, {x, y}}
}

type fakeLandReleaseRepo struct {
	repository.LandReleaseRepository
	features []domain.LandReleaseFeature
	decree   *domain.LandReleaseDecree
	imp      *domain.LandReleaseImport
	failSave bool
}

func (f *fakeLandReleaseRepo) AreasOf(_ context.Context, w []string) ([]float64, error) {
	out := make([]float64, len(w))
	for i := range out {
		out[i] = 100
	}
	return out, nil
}

func (f *fakeLandReleaseRepo) CreateImport(_ context.Context, imp *domain.LandReleaseImport, fs []domain.LandReleaseFeature, d *domain.LandReleaseDecree) (int, error) {
	if f.failSave {
		return 0, errors.New("db down")
	}
	f.imp, f.features, f.decree = imp, fs, d
	return len(fs), nil
}

type fakeStorage struct {
	filestorage.DisabledStorage
	saved, deleted []string
}

func (s *fakeStorage) Save(_ context.Context, ns, owner, name, ct string, data []byte, _ bool) (*filestorage.StoredFile, error) {
	u := "/api/files/" + ns + "/" + owner + "/" + name
	s.saved = append(s.saved, u)
	return &filestorage.StoredFile{URL: u, Name: name, Type: ct, SizeBytes: int64(len(data))}, nil
}

func (s *fakeStorage) Delete(_ context.Context, u string) error {
	s.deleted = append(s.deleted, u)
	return nil
}

const wgs84PRJ = `GEOGCS["GCS_WGS_1984",DATUM["D_WGS_1984",SPHEROID["WGS_1984",6378137.0,298.257223563]],PRIMEM["Greenwich",0.0],UNIT["Degree",0.0174532925199433]]`
const utmPRJ = `PROJCS["WGS_1984_UTM_Zone_48N",GEOGCS["GCS_WGS_1984",DATUM["D_WGS_1984",SPHEROID["WGS_1984",6378137.0,298.257223563]]],PROJECTION["Transverse_Mercator"]]`

func TestLandReleaseCreate_EndToEnd(t *testing.T) {
	repo := &fakeLandReleaseRepo{}
	st := &fakeStorage{}
	svc := NewLandReleaseService(repo, st)
	z := zipOf(t, map[string]string{
		"a.shp": string(testShp(sq(106.9, 47.9, 0.01), sq(106.95, 47.9, 0.01))),
		"a.prj": wgs84PRJ,
	})
	n, err := svc.Create(t.Context(), CreateLandReleaseRequest{
		Name: " Буулгалт 1 ", Status: domain.LandReleaseStatusReleased,
		StructureTypeName: " Хашаа ",
		Files: []UploadedFile{
			{Filename: "batch.zip", Data: z},
			{Filename: "b.shp", Data: testShp(sq(107, 48, 0.01))},
		},
		Decree: &LandReleaseDecreeInput{Number: "А/12", Date: "2026-05-01",
			File: &UploadedFile{Filename: "d.pdf", ContentType: "application/pdf", Data: []byte("%PDF")}},
	}, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 || len(repo.features) != 3 {
		t.Fatalf("want 3 features, got %d", n)
	}
	if repo.imp.Name != "Буулгалт 1" || repo.imp.InitialStatus != domain.LandReleaseStatusReleased ||
		repo.imp.StructureTypeName != "Хашаа" || repo.imp.StructureTypeID != nil {
		t.Errorf("import = %+v", repo.imp)
	}
	if repo.decree == nil || repo.decree.DecreeNumber != "А/12" || repo.decree.DecreeDate == nil {
		t.Fatalf("decree = %+v", repo.decree)
	}
	if len(st.saved) != 1 || !strings.HasPrefix(st.saved[0], "/api/files/land-release/") {
		t.Errorf("saved = %v", st.saved)
	}
	if repo.features[0].SourceFile != "batch.zip/a.shp" || repo.features[1].SourceIndex != 2 || repo.features[0].AreaM2 != 100 {
		t.Errorf("features = %+v", repo.features[:2])
	}
}

func TestLandReleaseCreate_RejectsProjectedPRJ(t *testing.T) {
	repo := &fakeLandReleaseRepo{}
	svc := NewLandReleaseService(repo, &fakeStorage{})
	_, err := svc.Preview(t.Context(), []UploadedFile{
		{Filename: "a.shp", Data: testShp(sq(106.9, 47.9, 0.01))},
		{Filename: "a.prj", Data: []byte(utmPRJ)},
	})
	var fe *LandReleaseFileError
	if !errors.As(err, &fe) || fe.File != "a.shp" {
		t.Fatalf("want file error for a.shp, got %v", err)
	}
	// Метрээр (UTM) координат — .prj-гүй ч координатын хүрээгээр барина.
	_, err = svc.Preview(t.Context(), []UploadedFile{{Filename: "m.shp", Data: testShp(sq(650000, 5300000, 100))}})
	if !errors.As(err, &fe) {
		t.Fatalf("want file error for meter coords, got %v", err)
	}
}

func TestLandReleaseCreate_DropsFileOnDBError(t *testing.T) {
	repo := &fakeLandReleaseRepo{failSave: true}
	st := &fakeStorage{}
	svc := NewLandReleaseService(repo, st)
	_, err := svc.Create(t.Context(), CreateLandReleaseRequest{
		Name: "x", Status: domain.LandReleaseStatusReleased, StructureTypeID: new(int),
		Files:  []UploadedFile{{Filename: "a.shp", Data: testShp(sq(106.9, 47.9, 0.01))}},
		Decree: &LandReleaseDecreeInput{Number: "А/1", File: &UploadedFile{Filename: "d.pdf", Data: []byte("%PDF")}},
	}, "u")
	if err == nil {
		t.Fatal("want error")
	}
	if len(st.deleted) != 1 || st.deleted[0] != st.saved[0] {
		t.Errorf("file not cleaned up: saved=%v deleted=%v", st.saved, st.deleted)
	}
}

func TestLandReleaseCreate_WithPhoto(t *testing.T) {
	repo := &fakeLandReleaseRepo{}
	st := &fakeStorage{}
	svc := NewLandReleaseService(repo, st)
	_, err := svc.Create(t.Context(), CreateLandReleaseRequest{
		Name: "x", Status: domain.LandReleaseStatusPlanning, StructureTypeName: "Гэр",
		Files: []UploadedFile{{Filename: "a.shp", Data: testShp(sq(106.9, 47.9, 0.01), sq(106.95, 47.9, 0.01))}},
		Photo: &LandReleasePhotoInput{File: UploadedFile{Filename: "z.pdf", ContentType: "application/pdf", Data: []byte("%PDF")}, PageCount: 3},
	}, "u1")
	if err != nil {
		t.Fatal(err)
	}
	p := repo.imp.Photo
	if p == nil || p.PageCount != 3 || p.CreatedBy != "u1" || !strings.HasPrefix(p.FileURL, "/api/files/land-release/") {
		t.Fatalf("photo = %+v", p)
	}
	if len(repo.features) != 2 {
		t.Fatalf("features = %d", len(repo.features))
	}
}

func TestLandReleaseCreate_DropsPhotoOnDBError(t *testing.T) {
	repo := &fakeLandReleaseRepo{failSave: true}
	st := &fakeStorage{}
	svc := NewLandReleaseService(repo, st)
	_, err := svc.Create(t.Context(), CreateLandReleaseRequest{
		Name: "x", Status: domain.LandReleaseStatusReleased, StructureTypeID: new(int),
		Files:  []UploadedFile{{Filename: "a.shp", Data: testShp(sq(106.9, 47.9, 0.01))}},
		Decree: &LandReleaseDecreeInput{Number: "А/1", File: &UploadedFile{Filename: "d.pdf", Data: []byte("%PDF")}},
		Photo:  &LandReleasePhotoInput{File: UploadedFile{Filename: "z.pdf", Data: []byte("%PDF")}},
	}, "u")
	if err == nil {
		t.Fatal("want error")
	}
	// Захирамж ба зураг хоёулаа хадгалагдаад, DB унахад хоёулаа устгагдана.
	if len(st.saved) != 2 || len(st.deleted) != 2 {
		t.Errorf("saved=%v deleted=%v", st.saved, st.deleted)
	}
}
