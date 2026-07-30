package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/pkg/geoserver"

	"github.com/google/uuid"
)

// DroneMosaic — GeoServer-ийн ImageMosaic давхаргатай харилцах гэрээ.
// Тест дээр орлуулж болохын тулд интерфэйсээр авна.
type DroneMosaic interface {
	// LayerName — WMS-д явуулах давхаргын бүтэн нэр (жишээ: "land:drone")
	LayerName() string
	// Refresh — шинэ .tif-үүдийг мозайкийн индекст нэмж, кэшийг цэвэрлэнэ
	Refresh(ctx context.Context) error
	// Rebuild — мозайкийг нурааж дахин үүсгэнэ. Хуучин тохиргоотой (нэг л CRS
	// хүлээдэг) мозайк шинэ зургийг хүлээж авахгүй байвал хэрэглэнэ.
	Rebuild(ctx context.Context) error
	// GranuleBoundsWait — harvest нь асинхрон (202) тул дахин оролдох хувилбар
	GranuleBoundsWait(ctx context.Context, fileName string, attempts int) (*geoserver.Bounds, error)
	DeleteGranule(ctx context.Context, fileName string) error
}

// DroneStorage — landService-д extras-аар дамжуулагдах дроны зургийн тохиргоо.
type DroneStorage struct {
	// Dir — API .tif-ийг бичих хавтас (GeoServer мөн ЭНЭ хавтсыг уншина)
	Dir    string
	Mosaic DroneMosaic
}

// droneNamespace — хадгалалтын объектын угтвар: drone/<acquisitionID>/<нэр>
const droneNamespace = "drone"

var (
	ErrDroneNotConfigured = errors.New("drone image storage is not configured")
	ErrDroneFileType      = errors.New("only GeoTIFF (.tif/.tiff) files are allowed")
	// ErrDroneDirectUploadUnsupported — хадгалалт нь шууд байршуулалтыг
	// дэмжихгүй (жишээ: FILE_SERVER_MODE=mock).
	ErrDroneDirectUploadUnsupported = errors.New("file storage does not support direct upload")
	// ErrDroneObjectMissing — зөвшөөрөл гаргасан объект хадгалалтад ороогүй
	// (байршуулалт тасалдсан эсвэл хийгдээгүй).
	ErrDroneObjectMissing     = errors.New("uploaded object not found in file storage")
	ErrDroneAlreadyRegistered = errors.New("drone image already registered")

	// Файлын нэрийг GeoServer-ийн CQL-д ашиглах тул зөвхөн аюулгүй тэмдэгт
	// үлдээнэ (хоосон зай, кирилл, хашилт г.м. mosaic индекст асуудал үүсгэнэ).
	droneNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	// droneStoredName — бүртгэхэд клиентээс ирэх нэрийн хатуу хэлбэр.
	droneStoredName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// openDroneCache — GeoServer уншдаг мозайкийн хавтсанд бичих файлыг нээнэ.
// Энэ нь ХАДГАЛАЛТ биш, зөвхөн рендерийн кэш (эх сурвалж нь файлын систем).
// 0644 — GeoServer өөр процесс/container-аас уншина.
func (s *landService) openDroneCache(fileName string) (*os.File, error) {
	if err := os.MkdirAll(s.drone.Dir, 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(s.drone.Dir, fileName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
}

// removeDroneCache — хагас бичигдсэн/шаардлагагүй болсон кэшийг зайлуулна.
func (s *landService) removeDroneCache(fileName string) {
	if err := os.Remove(filepath.Join(s.drone.Dir, fileName)); err != nil && !os.IsNotExist(err) {
		slog.Warn("drone cache cleanup failed", "file", fileName, "error", err)
	}
}

// ensureDroneCache — кэшэд байхгүй зургийг файлын системээс сэргээнэ.
// Кэш (volume) гээгдсэн эсвэл өөр хувилбар дээр асаасан үед хэрэгтэй.
//
// Ортофото хэдэн зуун MB байдаг тул урсгалаар хуулна — санах ойд бүтнээр
// уншвал сэргээлт өөрөө процессыг унагаж болно.
func (s *landService) ensureDroneCache(ctx context.Context, img *domain.DroneImage) error {
	path := filepath.Join(s.drone.Dir, img.FileName)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if img.FileURL == "" {
		return nil // хуучин бичлэг — файлын системд ороогүй
	}
	src, err := s.files.OpenStream(ctx, img.FileURL)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := s.openDroneCache(img.FileName)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		s.removeDroneCache(img.FileName)
		return err
	}
	slog.Info("drone cache restored from file storage", "file", img.FileName)
	return nil
}

// DroneLayerName — frontend WMS дуудахад хэрэглэх давхаргын нэр.
func (s *landService) DroneLayerName() string {
	if s.drone.Mosaic == nil {
		return ""
	}
	return s.drone.Mosaic.LayerName()
}

func (s *landService) ListDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, err
	}
	return s.landRepo.ListDroneImages(ctx, acquisitionID)
}

// isGeoTIFF — TIFF-ийн magic bytes: II*\0 / MM\0* (BigTIFF: II+\0 / MM\0+).
// Өргөтгөлд найдахгүй — өөр төрлийн файлыг .tif нэрээр оруулахаас сэргийлнэ.
func isGeoTIFF(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	head := data[:4]
	return bytes.Equal(head, []byte{'I', 'I', 42, 0}) ||
		bytes.Equal(head, []byte{'M', 'M', 0, 42}) ||
		bytes.Equal(head, []byte{'I', 'I', 43, 0}) ||
		bytes.Equal(head, []byte{'M', 'M', 0, 43})
}

// droneFileName — мозайкийн `location` болох давхардахгүй, аюулгүй файлын нэр.
func droneFileName(original string) string {
	base := filepath.Base(original)
	ext := strings.ToLower(filepath.Ext(base))
	if ext != ".tif" && ext != ".tiff" {
		ext = ".tif"
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = droneNameUnsafe.ReplaceAllString(stem, "_")
	stem = strings.Trim(stem, "_")
	if stem == "" {
		stem = "drone"
	}
	if len(stem) > 60 {
		stem = stem[:60]
	}
	// uuid угтвар — нэр давхардахгүй байхыг хангана (мозайкийн location unique)
	return fmt.Sprintf("%s_%s%s", stem, uuid.NewString()[:8], ext)
}

// droneUploadTTL — байршуулах зөвшөөрлийн хүчинтэй хугацаа. Хэдэн GB ортофото
// сүлжээгээр удаан явж болох тул уужуу авна.
const droneUploadTTL = 6 * time.Hour

// CreateDroneUploadURL — browser файлыг ШУУД файлын систем руу тавих зөвшөөрөл
// гаргана.
//
// Байт API-аар ДАМЖИХГҮЙ: том ортофото (хэдэн GB) API-ийн санах ой, хугацаа,
// хүсэлтийн хэмжээний хязгаарт үл хамаарна. API нь зөвхөн зөвшөөрөл гаргаж,
// дараа нь объектыг шалгаж бүртгэнэ (RegisterDroneImage).
//
// Объектын зам нь "drone/<acquisitionID>/<нэр>" — сервер өөрөө бүтээх тул
// дуудагч өөр чөлөөлөлтийн хавтас руу бичиж чадахгүй.
func (s *landService) CreateDroneUploadURL(
	ctx context.Context,
	acquisitionID uuid.UUID,
	originalName string,
) (*filestorage.PresignedUpload, error) {
	if s.drone.Dir == "" {
		return nil, ErrDroneNotConfigured
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext != ".tif" && ext != ".tiff" {
		return nil, ErrDroneFileType
	}
	direct, ok := s.files.(filestorage.DirectUploader)
	if !ok {
		return nil, ErrDroneDirectUploadUnsupported
	}
	return direct.PresignUpload(ctx, droneNamespace, acquisitionID.String(),
		droneFileName(originalName), "image/tiff", droneUploadTTL)
}

// RegisterDroneImage — шууд байршуулсан объектыг ШАЛГАЖ чөлөөлөлтөд холбоно.
//
// Клиентээс зөвхөн НЭР авна; зам, хэмжээ, төрлийг хадгалалт дээрх бодит
// объектоос уншина:
//   - зам нь drone/<acquisitionID>/<нэр> — өөр эзэмшигчийн объект бүртгэх боломжгүй
//   - хэмжээ нь Stat-аас — клиент хуурамч хэмжээ мэдүүлэх боломжгүй
//   - төрөл нь эхний 4 байтаас (range GET) — .tif нэртэй өөр файл орох боломжгүй
//
// GeoServer-ийн рендерийн кэш нь ЭНД бичигдэхгүй: RefreshDroneImages нь
// объектыг файлын системээс урсгалаар татаж кэш бүтээнэ (ensureDroneCache).
func (s *landService) RegisterDroneImage(
	ctx context.Context,
	acquisitionID uuid.UUID,
	storedName string,
	originalName string,
	uploadedBy *uuid.UUID,
) (*domain.DroneImage, error) {
	if s.drone.Dir == "" {
		return nil, ErrDroneNotConfigured
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, err
	}
	direct, ok := s.files.(filestorage.DirectUploader)
	if !ok {
		return nil, ErrDroneDirectUploadUnsupported
	}

	// Нэрийг хатуу шалгана — зөвхөн presign гаргахад хэрэглэсэн тэмдэгтүүд.
	// (Зам дамжих `..`, `/` бүрэн хаагдана.)
	storedName = filepath.Base(storedName)
	if storedName == "" || len(storedName) > 128 || !droneStoredName.MatchString(storedName) {
		return nil, ErrDroneFileType
	}
	if ext := strings.ToLower(filepath.Ext(storedName)); ext != ".tif" && ext != ".tiff" {
		return nil, ErrDroneFileType
	}

	// Давхардлыг эндээс шалгана — DB-ийн unique зөрчил дээр объектыг устгах
	// логик руу орвол АЛЬ ХЭДИЙН бүртгэлтэй зургийн файлыг устгах болно.
	existing, err := s.landRepo.ListDroneImages(ctx, acquisitionID)
	if err != nil {
		return nil, err
	}
	for _, img := range existing {
		if img.FileName == storedName {
			return nil, ErrDroneAlreadyRegistered
		}
	}

	fileURL := direct.URLFor(droneNamespace, acquisitionID.String(), storedName)
	stat, err := direct.Stat(ctx, fileURL)
	if err != nil {
		// Байршуулалт бүтээгүй/тасалдсан — объект хадгалалтад байхгүй.
		return nil, fmt.Errorf("%w: %v", ErrDroneObjectMissing, err)
	}
	if stat.SizeBytes <= 0 {
		return nil, ErrDroneObjectMissing
	}

	head, err := direct.ReadRange(ctx, fileURL, 0, 4)
	if err != nil {
		return nil, err
	}
	if !isGeoTIFF(head) {
		// Бид зөвшөөрөл гаргасан объект — GeoTIFF биш бол хог болгож үлдээхгүй.
		if delErr := s.files.Delete(ctx, fileURL); delErr != nil {
			slog.Warn("drone object cleanup failed", "file", storedName, "error", delErr)
		}
		return nil, ErrDroneFileType
	}

	img := &domain.DroneImage{
		ID:            uuid.New(),
		AcquisitionID: acquisitionID,
		FileName:      storedName,
		FileURL:       fileURL,
		OriginalName:  droneOriginalName(originalName, storedName),
		ContentType:   "image/tiff",
		SizeBytes:     stat.SizeBytes,
		UploadedAt:    time.Now(),
		UploadedBy:    uploadedBy,
	}
	if err := s.landRepo.CreateDroneImage(ctx, img); err != nil {
		// Объектыг УСТГАХГҮЙ: хэрэглэгчийн файл аль хэдийн хадгалалтад бүтэн
		// орсон, DB-ийн доголдлын улмаас түүнийг үгүй хийхгүй. Дахин бүртгэх
		// боломжтой; хоцорсон объектыг лог дээрээс хянана.
		slog.Warn("drone image registration failed; object kept in file storage",
			"file", storedName, "url", fileURL, "error", err)
		return nil, err
	}

	s.auditLog(ctx, "uploaded", domain.AuditResourceDocument, img.ID.String(),
		uuidPtr(acquisitionID), nil, map[string]any{
			"kind":          "drone_image",
			"file_name":     img.FileName,
			"original_name": img.OriginalName,
			"size_bytes":    img.SizeBytes,
			"direct_upload": true,
		})
	return img, nil
}

// droneOriginalName — харуулах нэр. Клиентээс ирдэг тул цэвэрлэж, хэтэрхий
// урт бол хадгалсан нэр рүү унана.
func droneOriginalName(original, fallback string) string {
	name := strings.TrimSpace(filepath.Base(original))
	if name == "" || name == "." || name == ".." || len([]rune(name)) > 200 {
		return fallback
	}
	return name
}

// RefreshDroneImages — GeoServer-ийн мозайкийг шинэчилж (harvest + reset),
// бүртгэгдээгүй зургуудын хүрээг уншиж тэмдэглэнэ.
//
// Frontend зураг байршуулсны ДАРАА автоматаар дуудна — ингэснээр шинэ зураг
// давхаргад шууд харагдана.
func (s *landService) RefreshDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error) {
	if s.drone.Mosaic == nil {
		return nil, ErrDroneNotConfigured
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, err
	}
	images, err := s.landRepo.ListDroneImages(ctx, acquisitionID)
	if err != nil {
		return nil, err
	}
	// Мозайкийг индексжүүлэхээс ӨМНӨ кэшэд байхгүй зургийг файлын системээс
	// сэргээнэ — эс тэгвээс GeoServer файлыг олохгүй.
	for _, img := range images {
		if err := s.ensureDroneCache(ctx, img); err != nil {
			slog.Warn("drone cache restore failed", "file", img.FileName, "error", err)
		}
	}

	if err := s.drone.Mosaic.Refresh(ctx); err != nil {
		return nil, err
	}

	missing, err := s.syncGranuleBounds(ctx, images)
	if err != nil {
		return nil, err
	}

	// Granule бүртгэгдээгүй зураг байвал мозайк нь хуучин тохиргоотой (жишээ:
	// зөвхөн нэг CRS хүлээдэг) байж мэдэх тул НЭГ удаа нурааж дахин барина.
	// Индекс нь .tif-үүдээс дахин бүтээгддэг тул зураг гээгдэхгүй.
	if missing > 0 {
		slog.Info("drone mosaic rebuild — granule нэмэгдээгүй зураг байна", "count", missing)
		if err := s.drone.Mosaic.Rebuild(ctx); err != nil {
			slog.Warn("drone mosaic rebuild failed", "error", err)
			return images, nil
		}
		if _, err := s.syncGranuleBounds(ctx, images); err != nil {
			return nil, err
		}
	}
	return images, nil
}

// syncGranuleBounds — хүрээ нь тодорхойгүй зургуудын хүрээг GeoServer-ээс уншиж
// тэмдэглэнэ. Буцаах тоо нь granule олдоогүй (мозайкт ороогүй) зургийн тоо.
func (s *landService) syncGranuleBounds(ctx context.Context, images []*domain.DroneImage) (int, error) {
	missing := 0
	for _, img := range images {
		// Хадгалагдсан хүрээ ХҮЧИНГҮЙ бол (жишээ: тэнхлэг солигдсон хуучин
		// бичлэг) дахин тооцно — эс тэгвээс frontend дээр газрын зураг
		// эвдэрсэн хэвээр үлдэнэ.
		if img.Published && img.MinX != nil && img.MinY != nil &&
			img.MaxX != nil && img.MaxY != nil &&
			geoserver.ValidateWGS84Bounds(*img.MinX, *img.MinY, *img.MaxX, *img.MaxY) == nil {
			continue
		}
		// 3 удаа хүртэл оролдоно — harvest асинхрон тул granule хараахан
		// бүртгэгдээгүй байж болно.
		bounds, err := s.drone.Mosaic.GranuleBoundsWait(ctx, img.FileName, 3)
		if err != nil {
			// Хүрээ уншиж чадаагүй нь давхарга харуулахад саад биш —
			// зөвхөн "зураг дээр очих" боломжгүй болно.
			slog.Warn("drone granule bounds unavailable", "file", img.FileName, "error", err)
			missing++
			continue
		}
		if err := s.landRepo.UpdateDroneImageBounds(ctx, img.ID,
			bounds.MinX, bounds.MinY, bounds.MaxX, bounds.MaxY); err != nil {
			return missing, err
		}
		img.MinX, img.MinY = &bounds.MinX, &bounds.MinY
		img.MaxX, img.MaxY = &bounds.MaxX, &bounds.MaxY
		img.Published = true
	}
	return missing, nil
}

// DeleteDroneImage — granule, файл, бүртгэлийг устгана.
func (s *landService) DeleteDroneImage(ctx context.Context, acquisitionID, imageID uuid.UUID) error {
	img, err := s.landRepo.GetDroneImage(ctx, imageID)
	if err != nil {
		return err
	}
	if img.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}

	if s.drone.Mosaic != nil {
		if err := s.drone.Mosaic.DeleteGranule(ctx, img.FileName); err != nil {
			// Мозайкаас хасаж чадаагүй ч бүртгэл/файлыг устгана — эс тэгвээс
			// хэрэглэгч зургийг хэзээ ч устгаж чадахгүй болно.
			slog.Warn("drone granule delete failed", "file", img.FileName, "error", err)
		}
	}
	// Рендерийн кэш
	if s.drone.Dir != "" {
		if err := os.Remove(filepath.Join(s.drone.Dir, img.FileName)); err != nil && !os.IsNotExist(err) {
			slog.Warn("drone cache delete failed", "file", img.FileName, "error", err)
		}
	}
	// Файлын систем (эх сурвалж)
	if img.FileURL != "" {
		if err := s.files.Delete(ctx, img.FileURL); err != nil {
			slog.Warn("drone object delete failed", "url", img.FileURL, "error", err)
		}
	}
	if err := s.landRepo.DeleteDroneImage(ctx, imageID); err != nil {
		return err
	}

	s.auditLog(ctx, "deleted", domain.AuditResourceDocument, imageID.String(),
		uuidPtr(acquisitionID), nil, map[string]any{
			"kind":          "drone_image",
			"file_name":     img.FileName,
			"original_name": img.OriginalName,
		})
	return nil
}
