package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/pkg/shapefile"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

// landReleaseFileNamespace — захирамжийн файлын namespace
// (`land-release/<захирамжийн id>/<нэр>`). Эрхийг /files/authorize шалгана.
const landReleaseFileNamespace = "land-release"

// Zip задлахад нэг файлын дээд хэмжээ (zip bomb-оос хамгаална).
const maxLandReleaseEntrySize = 50 * 1024 * 1024

type LandReleaseService interface {
	// Preview — файлуудыг уншиж, координатын системийг шалгаад газруудыг
	// (талбайтай) буцаана. Юу ч хадгалахгүй.
	Preview(ctx context.Context, files []UploadedFile) ([]domain.LandReleaseFeature, error)
	Create(ctx context.Context, req CreateLandReleaseRequest, actor string) (int, error)
	List(ctx context.Context, f domain.LandReleaseFilter) ([]*domain.LandRelease, int64, error)
	Map(ctx context.Context, f domain.LandReleaseFilter) ([]domain.LandReleaseMapItem, error)
	Stats(ctx context.Context, f domain.LandReleaseFilter) (*domain.LandReleaseStats, error)
	ListStructureTypes(ctx context.Context) ([]*domain.LandReleaseStructureType, error)
	// AddPhoto — газарт зураг (PDF) нэмнэ.
	AddPhoto(ctx context.Context, id string, in *LandReleasePhotoInput, actor string) (*domain.LandRelease, error)
	Get(ctx context.Context, id string) (*domain.LandRelease, error)
	ChangeStatus(ctx context.Context, id string, req ChangeLandReleaseStatusRequest, actor string) (*domain.LandRelease, error)
}

// UploadedFile — multipart-аар ирсэн файл (handler хэмжээг шалгасан).
type UploadedFile struct {
	Filename    string
	ContentType string
	Data        []byte
}

// LandReleaseDecreeInput — захирамж: дугаар, огноо, PDF.
type LandReleaseDecreeInput struct {
	Number string
	Date   string // YYYY-MM-DD, заавал биш
	File   *UploadedFile
}

type CreateLandReleaseRequest struct {
	Name        string
	Description string
	Status      int
	// Буулгасан байгууламжийн төрөл — StructureTypeID (байгаа) эсвэл
	// StructureTypeName (шинэ — хадгалахад нэмэгдэнэ), аль нэг нь заавал.
	StructureTypeID   *int
	StructureTypeName string
	Files             []UploadedFile
	// Decree — Status = Захирамж гарч чөлөөлсөн үед заавал.
	Decree *LandReleaseDecreeInput
	// Photo — заавал биш: зургуудыг (frontend) нэгтгэсэн PDF, бүх газарт холбогдоно.
	Photo *LandReleasePhotoInput
}

// LandReleasePhotoInput — зургуудаас үүсгэсэн PDF (handler PDF эсэхийг шалгасан).
type LandReleasePhotoInput struct {
	File      UploadedFile
	PageCount int
}

type ChangeLandReleaseStatusRequest struct {
	Status int
	Note   string
	Decree *LandReleaseDecreeInput
}

// LandReleaseFileError — тухайн файлын уншилтын алдаа (хэрэглэгчид шууд).
type LandReleaseFileError struct {
	File string
	Err  error
}

func (e *LandReleaseFileError) Error() string { return e.File + ": " + e.Err.Error() }
func (e *LandReleaseFileError) Unwrap() error { return e.Err }

type landReleaseService struct {
	repo  repository.LandReleaseRepository
	files filestorage.Storage
}

func NewLandReleaseService(repo repository.LandReleaseRepository, files filestorage.Storage) LandReleaseService {
	if files == nil {
		files = filestorage.DisabledStorage{}
	}
	return &landReleaseService{repo: repo, files: files}
}

// shpSource — нэг .shp ба (байвал) түүний .prj.
type shpSource struct {
	name string
	shp  []byte
	prj  *string
}

// collectSources — .shp / .prj / .zip файлуудаас .shp-үүдийг цуглуулна. .prj
// нь ижил нэртэй .shp-д харгалзана (zip дотор ч, тусад нь оруулсан ч).
func collectSources(files []UploadedFile) ([]shpSource, error) {
	var srcs []shpSource
	prjs := map[string]string{}
	add := func(name string, data []byte) {
		base := strings.TrimSuffix(name, path.Ext(name))
		switch strings.ToLower(path.Ext(name)) {
		case ".shp":
			srcs = append(srcs, shpSource{name: name, shp: data})
		case ".prj":
			prjs[strings.ToLower(base)] = string(data)
		}
	}
	for _, f := range files {
		name := path.Base(strings.ReplaceAll(f.Filename, "\\", "/"))
		switch strings.ToLower(path.Ext(name)) {
		case ".zip":
			zr, err := zip.NewReader(bytes.NewReader(f.Data), int64(len(f.Data)))
			if err != nil {
				return nil, &LandReleaseFileError{File: name, Err: fmt.Errorf("zip файл уншигдсангүй")}
			}
			found := false
			for _, zf := range zr.File {
				if zf.FileInfo().IsDir() || strings.HasPrefix(path.Base(zf.Name), "._") {
					continue
				}
				ext := strings.ToLower(path.Ext(zf.Name))
				if ext != ".shp" && ext != ".prj" {
					continue
				}
				rc, err := zf.Open()
				if err != nil {
					return nil, &LandReleaseFileError{File: name, Err: err}
				}
				data, err := io.ReadAll(io.LimitReader(rc, maxLandReleaseEntrySize+1))
				rc.Close()
				if err != nil || len(data) > maxLandReleaseEntrySize {
					return nil, &LandReleaseFileError{File: name + "/" + zf.Name, Err: fmt.Errorf("файл хэт том эсвэл эвдэрсэн")}
				}
				if ext == ".shp" {
					found = true
				}
				add(name+"/"+zf.Name, data)
			}
			if !found {
				return nil, &LandReleaseFileError{File: name, Err: fmt.Errorf("zip дотор .shp файл алга")}
			}
		case ".shp", ".prj":
			add(name, f.Data)
		default:
			return nil, &LandReleaseFileError{File: name, Err: fmt.Errorf("зөвхөн .shp, .prj эсвэл .zip файл оруулна")}
		}
	}
	for i := range srcs {
		base := strings.ToLower(strings.TrimSuffix(srcs[i].name, path.Ext(srcs[i].name)))
		if p, ok := prjs[base]; ok {
			srcs[i].prj = &p
		}
	}
	return srcs, nil
}

// readFeatures — координатын систем (.prj + координатын хүрээ) шалгаж газруудыг уншина.
func (s *landReleaseService) readFeatures(ctx context.Context, files []UploadedFile) ([]domain.LandReleaseFeature, []string, error) {
	if len(files) == 0 {
		return nil, nil, domain.ErrLandReleaseNoFiles
	}
	srcs, err := collectSources(files)
	if err != nil {
		return nil, nil, err
	}
	if len(srcs) == 0 {
		return nil, nil, domain.ErrLandReleaseNoFiles
	}
	var features []domain.LandReleaseFeature
	names := make([]string, 0, len(srcs))
	for _, src := range srcs {
		names = append(names, src.name)
		if src.prj != nil {
			if err := shapefile.CheckPRJ(*src.prj); err != nil {
				return nil, nil, &LandReleaseFileError{File: src.name, Err: err}
			}
		}
		wkts, err := shapefile.ParseAll(src.shp)
		if err != nil {
			return nil, nil, &LandReleaseFileError{File: src.name, Err: err}
		}
		for i, w := range wkts {
			features = append(features, domain.LandReleaseFeature{SourceFile: src.name, SourceIndex: i + 1, GeometryWKT: w})
		}
	}
	wkts := make([]string, len(features))
	for i, f := range features {
		wkts[i] = f.GeometryWKT
	}
	areas, err := s.repo.AreasOf(ctx, wkts)
	if err != nil {
		return nil, nil, err
	}
	for i := range features {
		if i < len(areas) {
			features[i].AreaM2 = areas[i]
		}
	}
	return features, names, nil
}

func (s *landReleaseService) Preview(ctx context.Context, files []UploadedFile) ([]domain.LandReleaseFeature, error) {
	features, _, err := s.readFeatures(ctx, files)
	return features, err
}

// saveDecree — захирамжийн PDF-ийг хадгалж мөрийг бэлдэнэ (DB-д бичихгүй).
func (s *landReleaseService) saveDecree(ctx context.Context, in *LandReleaseDecreeInput, actor string) (*domain.LandReleaseDecree, error) {
	if in == nil || strings.TrimSpace(in.Number) == "" || in.File == nil || len(in.File.Data) == 0 {
		return nil, domain.ErrLandReleaseDecreeRequired
	}
	d := &domain.LandReleaseDecree{DecreeNumber: strings.TrimSpace(in.Number), CreatedBy: actor}
	if date := strings.TrimSpace(in.Date); date != "" {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		d.DecreeDate = &t
	}
	// Объектын хавтас — тусдаа uuid (DB-ийн id нь INSERT хийхэд үүснэ).
	stored, err := s.files.Save(ctx, landReleaseFileNamespace, uuid.NewString(), in.File.Filename, in.File.ContentType, in.File.Data, true)
	if err != nil {
		slog.Error("буулгалтын захирамж хадгалахад алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	d.FileName = in.File.Filename
	d.FileType = stored.Type
	d.FileURL = stored.URL
	d.SizeBytes = stored.SizeBytes
	return d, nil
}

func (s *landReleaseService) dropFile(d *domain.LandReleaseDecree) {
	if d == nil {
		return
	}
	if err := s.files.Delete(context.Background(), d.FileURL); err != nil {
		slog.Error("буулгалт амжилтгүй — захирамжийн файлыг устгаж чадсангүй", "file_url", d.FileURL, "error", err)
	}
}

func (s *landReleaseService) Create(ctx context.Context, req CreateLandReleaseRequest, actor string) (int, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, domain.ErrInvalidInput
	}
	if req.Status != domain.LandReleaseStatusPlanning && req.Status != domain.LandReleaseStatusReleased {
		return 0, domain.ErrInvalidInput
	}
	if req.Status == domain.LandReleaseStatusReleased &&
		(req.Decree == nil || strings.TrimSpace(req.Decree.Number) == "" || req.Decree.File == nil) {
		return 0, domain.ErrLandReleaseDecreeRequired
	}
	typeName := strings.TrimSpace(req.StructureTypeName)
	if req.StructureTypeID == nil && typeName == "" {
		return 0, domain.ErrLandReleaseStructureType
	}
	features, names, err := s.readFeatures(ctx, req.Files)
	if err != nil {
		return 0, err
	}
	var decree *domain.LandReleaseDecree
	if req.Status == domain.LandReleaseStatusReleased {
		if decree, err = s.saveDecree(ctx, req.Decree, actor); err != nil {
			return 0, err
		}
	}
	imp := &domain.LandReleaseImport{
		Name: name, Description: strings.TrimSpace(req.Description),
		InitialStatus: req.Status, FileNames: names, CreatedBy: actor,
		StructureTypeID: req.StructureTypeID, StructureTypeName: typeName,
	}
	if req.Photo != nil {
		if imp.Photo, err = s.savePhoto(ctx, req.Photo, actor); err != nil {
			s.dropFile(decree)
			return 0, err
		}
	}
	n, err := s.repo.CreateImport(ctx, imp, features, decree)
	if err != nil {
		s.dropFile(decree)
		s.dropPhoto(imp.Photo)
		return 0, err
	}
	return n, nil
}

// savePhoto — зургийн PDF-ийг хадгалж мөрийг бэлдэнэ (DB-д бичихгүй).
func (s *landReleaseService) savePhoto(ctx context.Context, in *LandReleasePhotoInput, actor string) (*domain.LandReleasePhoto, error) {
	if in == nil || len(in.File.Data) == 0 {
		return nil, domain.ErrLandReleasePhoto
	}
	stored, err := s.files.Save(ctx, landReleaseFileNamespace, uuid.NewString(), in.File.Filename, in.File.ContentType, in.File.Data, true)
	if err != nil {
		slog.Error("буулгалтын зураг хадгалахад алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	return &domain.LandReleasePhoto{
		FileName: in.File.Filename, FileURL: stored.URL, SizeBytes: stored.SizeBytes,
		PageCount: in.PageCount, CreatedBy: actor,
	}, nil
}

func (s *landReleaseService) dropPhoto(p *domain.LandReleasePhoto) {
	if p == nil {
		return
	}
	if err := s.files.Delete(context.Background(), p.FileURL); err != nil {
		slog.Error("буулгалт амжилтгүй — зургийн файлыг устгаж чадсангүй", "file_url", p.FileURL, "error", err)
	}
}

func (s *landReleaseService) AddPhoto(ctx context.Context, id string, in *LandReleasePhotoInput, actor string) (*domain.LandRelease, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.savePhoto(ctx, in, actor)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddPhoto(ctx, id, p); err != nil {
		s.dropPhoto(p)
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *landReleaseService) List(ctx context.Context, f domain.LandReleaseFilter) ([]*domain.LandRelease, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *landReleaseService) Map(ctx context.Context, f domain.LandReleaseFilter) ([]domain.LandReleaseMapItem, error) {
	return s.repo.MapItems(ctx, f)
}

func (s *landReleaseService) ListStructureTypes(ctx context.Context) ([]*domain.LandReleaseStructureType, error) {
	return s.repo.ListStructureTypes(ctx)
}

func (s *landReleaseService) Stats(ctx context.Context, f domain.LandReleaseFilter) (*domain.LandReleaseStats, error) {
	return s.repo.Stats(ctx, f)
}

func (s *landReleaseService) Get(ctx context.Context, id string) (*domain.LandRelease, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, domain.ErrNotFound
	}
	lr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lr.History, err = s.repo.ListHistory(ctx, id); err != nil {
		return nil, err
	}
	if lr.Photos, err = s.repo.ListPhotos(ctx, id); err != nil {
		return nil, err
	}
	return lr, nil
}

func (s *landReleaseService) ChangeStatus(ctx context.Context, id string, req ChangeLandReleaseStatusRequest, actor string) (*domain.LandRelease, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !domain.LandReleaseTransitionAllowed(cur.Status, req.Status) {
		return nil, domain.ErrLandReleaseTransition
	}
	var decree *domain.LandReleaseDecree
	if req.Status == domain.LandReleaseStatusReleased {
		// Нэг нэгээр солиход захирамж ТУСДАА мөр болж хадгалагдана.
		if decree, err = s.saveDecree(ctx, req.Decree, actor); err != nil {
			return nil, err
		}
	}
	if err := s.repo.ChangeStatus(ctx, id, cur.Status, req.Status, decree, strings.TrimSpace(req.Note), actor); err != nil {
		s.dropFile(decree)
		return nil, err
	}
	return s.Get(ctx, id)
}
