package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/pkg/geoserver"

	"github.com/google/uuid"
)

// DronePublisher — GeoServer-т давхарга нийтлэх гэрээ.
//
// Зураг тус бүр өөрийн COG store/давхаргатай бөгөөд GeoServer растрыг ФАЙЛЫН
// СИСТЕМЭЭС ШУУД (HTTP Range) уншина. Тест дээр орлуулж болохын тулд
// интерфэйсээр авна.
type DronePublisher interface {
	Configured() bool
	// LayerName — WMS-д явуулах давхаргын бүтэн нэр (жишээ: "land:orto_ab12cd34")
	LayerName(store string) string
	// Publish — объектын URL-ийг давхарга болгон нийтэлж WGS84 хүрээг буцаана.
	// hasAlpha нь тунгалаг байдлын тохиргоог шийднэ (Configure-ийг үзнэ үү).
	Publish(ctx context.Context, store, objectURL string, hasAlpha bool) (*geoserver.Bounds, error)
	// Configure — давхаргын харуулах тохиргоог (тунгалаг байдал г.м.) дахин
	// хэрэглэнэ. Идемпотент.
	Configure(ctx context.Context, store string, hasAlpha bool) error
	// Delete — давхаргыг каталогоос хасна (объект хөндөгдөхгүй)
	Delete(ctx context.Context, store string) error
	Reset(ctx context.Context) error
}

// DroneStorage — landService-д extras-аар дамжуулагдах дроны тохиргоо.
//
// Локал хавтас БАЙХГҮЙ: backend растрыг өөр дээрээ хадгалахгүй.
type DroneStorage struct {
	Publisher DronePublisher
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
	// ErrDroneTooLarge — зураг зөвшөөрөгдсөн хэмжээнээс давсан.
	ErrDroneTooLarge = errors.New("drone image exceeds the maximum allowed size")

	// Файлын нэрийг GeoServer-ийн CQL-д ашиглах тул зөвхөн аюулгүй тэмдэгт
	// үлдээнэ (хоосон зай, кирилл, хашилт г.м. давхаргын нэрэнд асуудал үүсгэнэ).
	droneNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	// droneStoredName — бүртгэхэд клиентээс ирэх нэрийн хатуу хэлбэр.
	droneStoredName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// droneStoreName — зургийн GeoServer store/давхаргын нэр. Файлын нэр нь аль
// хэдийн давхардахгүй, аюулгүй тэмдэгттэй тул өргөтгөлийг хасаад хэрэглэнэ
// (цэгийг доогуур зураасаар сольж давхаргын нэрийг цэвэр болгоно).
// RemoveLayer — дроны зургийн GeoServer давхаргыг устгана (хуучин мэдээллийн
// импортыг «Хасах» үед). Тохируулаагүй бол юу ч хийхгүй.
func (d DroneStorage) RemoveLayer(ctx context.Context, fileName string) error {
	if d.Publisher == nil || !d.Publisher.Configured() {
		return nil
	}
	return d.Publisher.Delete(ctx, droneStoreName(fileName))
}

func droneStoreName(fileName string) string {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	name = strings.ReplaceAll(name, ".", "_")
	if name == "" {
		return ""
	}
	return "drone_" + name
}

// DroneLayerName — frontend WMS дуудахад хэрэглэх давхаргын нэр. Зураг тус бүр
// өөрийн давхаргатай тул файлын нэрээс тооцно.
func (s *landService) DroneLayerName(fileName string) string {
	if s.drone.Publisher == nil {
		return ""
	}
	return s.drone.Publisher.LayerName(droneStoreName(fileName))
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

// droneUploadTTL — байршуулах зөвшөөрлийн хүчинтэй хугацаа. Том ортофото
// сүлжээгээр удаан явж болох тул уужуу авна.
const droneUploadTTL = 6 * time.Hour

// maxDroneSizeBytes — дроны зургийн дээд хэмжээ (500 MB).
//
// Frontend мөн адил шалгадаг ч энэ нь ЖИНХЭНЭ хаалт: файл нь presigned URL-ээр
// ШУУД хадгалалт руу явдаг тул зөвхөн браузерын шалгалтад найдаж болохгүй.
// Хэмжээг бүртгэх үед хадгалалтаас (Stat) уншиж шалгана — клиентийн мэдүүлсэн
// утга биш.
const maxDroneSizeBytes int64 = 500 * 1024 * 1024

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
	if s.drone.Publisher == nil || !s.drone.Publisher.Configured() {
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
// GeoServer-т нийтлэх нь ЭНД хийгдэхгүй — RefreshDroneImages тусад нь хийнэ.
func (s *landService) RegisterDroneImage(
	ctx context.Context,
	acquisitionID uuid.UUID,
	storedName string,
	originalName string,
	uploadedBy *domain.ID,
) (*domain.DroneImage, error, error) {
	if s.drone.Publisher == nil || !s.drone.Publisher.Configured() {
		return nil, nil, ErrDroneNotConfigured
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, nil, err
	}
	direct, ok := s.files.(filestorage.DirectUploader)
	if !ok {
		return nil, nil, ErrDroneDirectUploadUnsupported
	}

	// Нэрийг хатуу шалгана — зөвхөн presign гаргахад хэрэглэсэн тэмдэгтүүд.
	// (Зам дамжих `..`, `/` бүрэн хаагдана.)
	storedName = filepath.Base(storedName)
	if storedName == "" || len(storedName) > 128 || !droneStoredName.MatchString(storedName) {
		return nil, nil, ErrDroneFileType
	}
	if ext := strings.ToLower(filepath.Ext(storedName)); ext != ".tif" && ext != ".tiff" {
		return nil, nil, ErrDroneFileType
	}

	// Давхардлыг эндээс шалгана — DB-ийн unique зөрчил дээр объектыг устгах
	// логик руу орвол АЛЬ ХЭДИЙН бүртгэлтэй зургийн файлыг устгах болно.
	existing, err := s.landRepo.ListDroneImages(ctx, acquisitionID)
	if err != nil {
		return nil, nil, err
	}
	for _, img := range existing {
		if img.FileName == storedName {
			return nil, nil, ErrDroneAlreadyRegistered
		}
	}

	fileURL := direct.URLFor(droneNamespace, acquisitionID.String(), storedName)
	stat, err := direct.Stat(ctx, fileURL)
	if err != nil {
		// Байршуулалт бүтээгүй/тасалдсан — объект хадгалалтад байхгүй.
		return nil, nil, fmt.Errorf("%w: %v", ErrDroneObjectMissing, err)
	}
	if stat.SizeBytes <= 0 {
		return nil, nil, ErrDroneObjectMissing
	}
	if stat.SizeBytes > maxDroneSizeBytes {
		// Хэт том объектыг хадгалалтад хог болгож үлдээхгүй — бүртгэгдэхгүй тул
		// хэн ч түүнийг устгах боломжгүй болно.
		if delErr := s.files.Delete(ctx, fileURL); delErr != nil {
			slog.Warn("oversized drone object cleanup failed", "file", storedName, "error", delErr)
		}
		return nil, nil, fmt.Errorf("%w: %d > %d bytes", ErrDroneTooLarge, stat.SizeBytes, maxDroneSizeBytes)
	}

	head, err := direct.ReadRange(ctx, fileURL, 0, 4)
	if err != nil {
		return nil, nil, err
	}
	if !isGeoTIFF(head) {
		// Бид зөвшөөрөл гаргасан объект — GeoTIFF биш бол хог болгож үлдээхгүй.
		if delErr := s.files.Delete(ctx, fileURL); delErr != nil {
			slog.Warn("drone object cleanup failed", "file", storedName, "error", delErr)
		}
		return nil, nil, ErrDroneFileType
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
		return nil, nil, err
	}

	// GeoServer-т давхарга болгон НЭН ДАРУЙ нийтэлнэ. Ингэснээр frontend
	// нэмэлт refresh дуудах шаардлагагүй болно.
	//
	// Бүтэлгүйтвэл АЛДАА БУЦААХГҮЙ: хэрэглэгчийн файл хадгалалтад бүтэн орж,
	// бүртгэгдсэн байна. Зөвхөн давхарга үүсээгүй (published=false) тул
	// "GeoServer шинэчлэх" товчоор дахин оролдоно.
	var publishErr error
	if err := s.publishDroneImage(ctx, img); err != nil {
		slog.Warn("drone publish failed on register", "file", img.FileName, "error", err)
		publishErr = err
	}

	s.auditLog(ctx, "uploaded", domain.AuditResourceDocument, img.ID.String(),
		uuidPtr(acquisitionID), nil, map[string]any{
			"kind":          "drone_image",
			"file_name":     img.FileName,
			"original_name": img.OriginalName,
			"size_bytes":    img.SizeBytes,
			"published":     img.Published,
			"direct_upload": true,
		})
	return img, publishErr, nil
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

// droneHasAlpha — растрт alpha зурвас байгаа эсэхийг объектын толгойгоос уншина.
//
// Тунгалаг байдлын тохиргоо ҮҮНЭЭС шалтгаална (geoserver.Client.Configure).
// Толгойг уншиж чадаагүй бол ALPHA-ГҮЙ гэж үзнэ: тэр үед InputTransparentColor
// хэрэглэгдэж хар фон арилна — alpha-тай файлыг андуурч тэгж тохируулбал фон
// цагаан болох ч, alpha-гүй файлыг андуурвал фон ХАР болно. Хоёр талын эрсдэлээс
// хар нь илүү элбэг тохиолддог тул анхдагчаар түүнийг шийднэ.
func (s *landService) droneHasAlpha(ctx context.Context, direct filestorage.DirectUploader, fileURL string) bool {
	info, err := readDroneRasterInfo(ctx, func(ctx context.Context, offset, length int64) ([]byte, error) {
		return direct.ReadRange(ctx, fileURL, offset, length)
	})
	if err != nil {
		slog.Warn("drone raster header read failed; assuming no alpha band",
			"url", fileURL, "error", err)
		return false
	}
	return info.HasAlpha
}

// publishDroneImage — объектыг GeoServer-т давхарга болгон нийтэлж, хүрээг
// DB-д тэмдэглэнэ. Идемпотент.
//
// Register ба Refresh ХОЁУЛАА үүнийг дуудна — ингэснээр нийтлэх логик нэг
// газар байна.
func (s *landService) publishDroneImage(ctx context.Context, img *domain.DroneImage) error {
	if s.drone.Publisher == nil || !s.drone.Publisher.Configured() {
		return ErrDroneNotConfigured
	}
	if img.FileURL == "" {
		return ErrDroneObjectMissing
	}
	direct, ok := s.files.(filestorage.DirectUploader)
	if !ok {
		return ErrDroneDirectUploadUnsupported
	}
	// GeoServer объектыг ЭНЭ URL-ээр Range хүсэлтээр уншина.
	objectURL, err := direct.ObjectURL(img.FileURL)
	if err != nil {
		return err
	}
	bounds, err := s.drone.Publisher.Publish(ctx, droneStoreName(img.FileName), objectURL,
		s.droneHasAlpha(ctx, direct, img.FileURL))
	if err != nil {
		return err
	}
	if err := s.landRepo.UpdateDroneImageBounds(ctx, img.ID,
		bounds.MinX, bounds.MinY, bounds.MaxX, bounds.MaxY); err != nil {
		return err
	}
	img.MinX, img.MinY = &bounds.MinX, &bounds.MinY
	img.MaxX, img.MaxY = &bounds.MaxX, &bounds.MaxY
	img.Published = true
	// Шинэ давхарга WMS-д шууд харагдахын тулд.
	if err := s.drone.Publisher.Reset(ctx); err != nil {
		slog.Warn("geoserver reset failed", "error", err)
	}
	return nil
}

// RefreshDroneImages — ЗАСВАРЫН үйлдэл: нийтлэгдээгүй эсвэл хүрээ нь хүчингүй
// зургуудыг дахин нийтэлж, нийтлэгдсэн давхаргуудын харуулах тохиргоог
// шинэчилнэ.
//
// Байршуулах ҮНДСЭН урсгалд ОРОХГҮЙ: RegisterDroneImage өөрөө нийтэлдэг тул
// frontend дараа нь дахин дуудах шаардлагагүй (давхар дуудлага байсныг хасав).
// Хэрэглэгч "GeoServer шинэчлэх" товч дарах, эсвэл нийтлэлт бүтэлгүйтсэн
// зургийг дахин оролдоход хэрэглэгдэнэ.
func (s *landService) RefreshDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error) {
	if s.drone.Publisher == nil || !s.drone.Publisher.Configured() {
		return nil, ErrDroneNotConfigured
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, err
	}
	images, err := s.landRepo.ListDroneImages(ctx, acquisitionID)
	if err != nil {
		return nil, err
	}
	direct, _ := s.files.(filestorage.DirectUploader)

	for _, img := range images {
		// Хадгалагдсан хүрээ ХҮЧИНТЭЙ бол дахин нийтлэхгүй — зөвхөн ХАРУУЛАХ
		// ТОХИРГООГ дахин хэрэглэнэ (код дээр тохиргоо зассан үед хуучин
		// давхаргууд хоцорч хар/цагаан фонтой үлдэхгүй).
		if img.Published && img.MinX != nil && img.MinY != nil &&
			img.MaxX != nil && img.MaxY != nil &&
			geoserver.ValidateWGS84Bounds(*img.MinX, *img.MinY, *img.MaxX, *img.MaxY) == nil {
			hasAlpha := false
			if direct != nil {
				hasAlpha = s.droneHasAlpha(ctx, direct, img.FileURL)
			}
			if err := s.drone.Publisher.Configure(ctx, droneStoreName(img.FileName), hasAlpha); err != nil {
				slog.Warn("drone layer reconfigure failed", "file", img.FileName, "error", err)
			}
			continue
		}
		// Нэг зураг нийтлэгдэхгүй нь бусдыг хөндөх ёсгүй.
		if err := s.publishDroneImage(ctx, img); err != nil {
			slog.Warn("drone publish failed", "file", img.FileName, "error", err)
		}
	}
	return images, nil
}

// DeleteDroneImage — давхарга, объект, бүртгэлийг устгана.
func (s *landService) DeleteDroneImage(ctx context.Context, acquisitionID, imageID uuid.UUID) error {
	img, err := s.landRepo.GetDroneImage(ctx, imageID)
	if err != nil {
		return err
	}
	if img.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}

	if s.drone.Publisher != nil {
		if err := s.drone.Publisher.Delete(ctx, droneStoreName(img.FileName)); err != nil {
			// Каталогоос хасаж чадаагүй ч бүртгэл/объектыг устгана — эс тэгвээс
			// хэрэглэгч зургийг хэзээ ч устгаж чадахгүй болно.
			slog.Warn("drone layer delete failed", "file", img.FileName, "error", err)
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
