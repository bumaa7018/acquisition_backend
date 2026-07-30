// Package geoserver — GeoServer-ийн REST API-тай харилцах багц.
//
// Дроны ортофото зургуудыг НЭГ ImageMosaic давхаргад (coveragestore + coverage)
// granule болгон бүртгэнэ. Ингэснээр:
//   - GeoServer дээр тохиргоо НЭГ л удаа хийгдэнэ (давхарга нэмэгдэхгүй),
//   - WMS дуудахад CQL_FILTER=location='<файлын нэр>' гэж явуулж тухайн
//     зургийг л харуулна.
package geoserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL   string // .../geoserver
	user      string
	pass      string
	workspace string
	store     string // coveragestore ба coverage (layer)-ийн нэр
	mosaicDir string // GeoServer-ийн ДОТРООС харагдах mosaic хавтас
	localDir  string // ЯГ ТЭР хавтас, API-ийн зүгээс (indexer.properties бичихэд)
	http      *http.Client
}

func NewClient(baseURL, user, pass, workspace, store, mosaicDir, localDir string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		user:      user,
		pass:      pass,
		workspace: workspace,
		store:     store,
		mosaicDir: strings.TrimRight(mosaicDir, "/"),
		localDir:  localDir,
		// Мозайк индексжүүлэх (granule harvest) нь том файл дээр удаж болно.
		http: &http.Client{Timeout: 120 * time.Second},
	}
}

// indexerProperties — мозайкийг ӨӨР ӨӨР CRS-тэй granule хүлээж авдаг болгоно.
//
// Яагаад: дроны ортофото бодит байдал дээр UTM эсвэл Web Mercator (EPSG:3857)
// зэрэг ялгаатай CRS-тэй ирдэг. Анхдагчаар ImageMosaic нь ЭХНИЙ granule-ийн
// CRS-ээр "цоожлогдож" (HeterogeneousCRS=false), өөр CRS-тэй зургийг ЧИМЭЭГҮЙ
// хүлээж авахгүй — улмаас зураг давхаргад харагдахгүй болно.
//
// HeterogeneousCRS=true + ReprojectingGranuleHandler нь granule бүрийг
// MosaicCRS руу шилжүүлж уншдаг болгоно.
//
// MosaicCRS нь ЗОРИУД EPSG:3857 (Web Mercator):
//
//  1. EPSG:4326-ийн тэнхлэгийн дараалал ХОЁРДМОЛ. GeoServer түүнийг
//     `urn:ogc:def:crs:EPSG::4326` болгон хувиргадаг бөгөөд URN хэлбэр нь
//     OGC-ийн дүрмээр ӨРГӨРӨГ ТЭРГҮҮЛЭХ. Улмаас granule-ийн индексийн
//     геометр [lat,lon] болж, хүрээ солигдон газрын зураг эвдэрч байсан.
//     3857-д X=зүүн, Y=хойд — хоёрдмол утга БАЙХГҮЙ.
//  2. Frontend-ийн газрын зураг мөн EPSG:3857 — WMS дуудахад мозайк дахин
//     хөрвүүлэлт хийхгүй болж, реndер хурдан бөгөөд цэвэр болно.
const mosaicCRS = "EPSG:3857"

func (c *Client) indexerProperties() string {
	return "Name=" + c.store + "\n" +
		"TypeName=" + c.store + "\n" +
		"Caching=false\n" +
		"AbsolutePath=false\n" +
		"CheckAuxiliaryMetadata=false\n" +
		"LocationAttribute=location\n" +
		"HeterogeneousCRS=true\n" +
		"MosaicCRS=" + mosaicCRS + "\n" +
		"GranuleAcceptors=org.geotools.gce.imagemosaic.acceptors.HeterogeneousCRSAcceptorFactory\n" +
		"GranuleHandler=org.geotools.gce.imagemosaic.granulehandler.ReprojectingGranuleHandlerFactory\n" +
		"Schema=*the_geom:Polygon,location:String,crs:String\n" +
		"PropertyCollectors=CRSExtractorSPI[crs]\n"
}

// writeIndexer — indexer.properties-ийг мозайк хавтсанд бичнэ (байхгүй бол).
// GeoServer store үүсгэхдээ энэ файлыг уншдаг тул store үүсгэхээс ӨМНӨ бичнэ.
func (c *Client) writeIndexer() error {
	if c.localDir == "" {
		return nil // хавтас мэдэгдэхгүй — GeoServer өөрөө homogeneous үүсгэнэ
	}
	if err := os.MkdirAll(c.localDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(c.localDir, "indexer.properties")
	if _, err := os.Stat(path); err == nil {
		return nil // аль хэдийн байна
	}
	return os.WriteFile(path, []byte(c.indexerProperties()), 0o644)
}

// LayerName — WMS-д явуулах давхаргын бүтэн нэр (жишээ: "land:drone").
func (c *Client) LayerName() string { return c.workspace + ":" + c.store }

func (c *Client) do(ctx context.Context, method, path, contentType string, body []byte) (int, []byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.SetBasicAuth(c.user, c.pass)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	// Хариу их урт байж болох тул (GeoServer HTML алдаа) 32KB-ээр хязгаарлана.
	payload, err := io.ReadAll(io.LimitReader(res.Body, 32<<10))
	return res.StatusCode, payload, err
}

func (c *Client) exists(ctx context.Context, path string) (bool, error) {
	status, _, err := c.do(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return false, err
	}
	return status == http.StatusOK, nil
}

// EnsureMosaic — ImageMosaic coveragestore/coverage байхгүй бол үүсгэнэ
// (идемпотент). Мозайк нь хавтсанд ДОР ДУНД нэг .tif байхыг шаарддаг тул
// эхний зураг байршуулсны дараа л амжилттай үүснэ.
func (c *Client) EnsureMosaic(ctx context.Context) error {
	storePath := fmt.Sprintf("/rest/workspaces/%s/coveragestores/%s.json", c.workspace, c.store)
	ok, err := c.exists(ctx, storePath)
	if err != nil {
		return fmt.Errorf("geoserver unreachable: %w", err)
	}
	if ok {
		return nil
	}

	// indexer.properties-ийг store үүсгэхээс ӨМНӨ бичнэ — эс тэгвээс GeoServer
	// нь зөвхөн нэг CRS хүлээх мозайк үүсгэнэ.
	if err := c.writeIndexer(); err != nil {
		return fmt.Errorf("indexer.properties write failed: %w", err)
	}

	// Хавтсыг бүхэлд нь мозайк болгон үүсгэнэ. GeoServer нь хавтас дотор байгаа
	// .tif-үүдийг өөрөө индексжүүлж, `location` атрибут бүхий granule болгоно.
	dirURL := "file://" + c.mosaicDir
	status, body, err := c.do(ctx,
		http.MethodPut,
		fmt.Sprintf("/rest/workspaces/%s/coveragestores/%s/external.imagemosaic", c.workspace, c.store),
		"text/plain",
		[]byte(dirURL),
	)
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("mosaic create failed (%d): %s", status, strings.TrimSpace(string(body)))
	}
	slog.Info("geoserver mosaic created", "workspace", c.workspace, "store", c.store, "dir", c.mosaicDir)
	// configureCoverage-ыг ЭНД дуудахгүй — Refresh/Rebuild хоёулаа дуудна.
	// (Эс тэгвээс эхний refresh дээр ижил PUT хоёр удаа явна.)
	return nil
}

// configureCoverage — давхаргыг ДАВХЦУУЛЖ харуулахад тохирсон болгоно.
//
// Хоёр тохиргоо ХОЁУЛАА шаардлагатай:
//
//   - BackgroundValues=-1 — granule байхгүй хэсгийг "nodata" болгоно.
//
//   - OutputTransparentColor=#000000 — ортофотогийн ХАР ЗАХ болон өөр CRS-ээс
//     reproject хийхэд булангуудад үүсэх хар дүүргэлтийг тунгалаг болгоно.
//     Үүнгүйгээр зураг "хар фонтой" харагдана (бодит зураг дээр 40% хар пиксел
//     → 4% болж буурсан). Гэрэлтүүлгийн бодит ХАР пиксел ч тунгалаг болох нь
//     ортофотод хүлээн зөвшөөрөгдсөн зардал (хар нь бараг үргэлж nodata).
//
// InputTransparentColor-ыг ХЭРЭГЛЭХГҮЙ — тэр нь хар захыг тунгалаг болгосны
// дараа мозайкийн ЦАГААН фон дээр давхарлаж, булангуудыг цагаан болгоно.
func (c *Client) configureCoverage(ctx context.Context) error {
	body := []byte(`{"coverage":{"parameters":{"entry":[` +
		`{"string":["BackgroundValues","-1"]},` +
		`{"string":["InputTransparentColor",""]},` +
		`{"string":["OutputTransparentColor","#000000"]}` +
		`]}}}`)
	status, res, err := c.do(ctx, http.MethodPut,
		fmt.Sprintf("/rest/workspaces/%s/coveragestores/%s/coverages/%s", c.workspace, c.store, c.store),
		"application/json", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("coverage config failed (%d): %s", status, strings.TrimSpace(string(res)))
	}
	return nil
}

// Harvest — хавтас дахь шинэ .tif-үүдийг мозайкийн индекст нэмнэ.
//
// Мозайк аль хэдийн байгаа үед POST external.imagemosaic нь ЗӨВХӨН шинэ
// granule-уудыг индексжүүлдэг (одоо байгааг давхардуулахгүй). Тиймээс зураг
// байршуулсан бүрд дуудаж болно.
func (c *Client) Harvest(ctx context.Context) error {
	if err := c.EnsureMosaic(ctx); err != nil {
		return err
	}
	status, body, err := c.do(ctx,
		http.MethodPost,
		fmt.Sprintf("/rest/workspaces/%s/coveragestores/%s/external.imagemosaic", c.workspace, c.store),
		"text/plain",
		[]byte("file://"+c.mosaicDir),
	)
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK && status != http.StatusAccepted {
		return fmt.Errorf("mosaic harvest failed (%d): %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// Reset — GeoServer-ийн resource/caталог кэшийг цэвэрлэнэ. Шинэ granule
// хуучин кэшээс болж харагдахгүй байхаас сэргийлнэ.
func (c *Client) Reset(ctx context.Context) error {
	status, body, err := c.do(ctx, http.MethodPost, "/rest/reset", "", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("geoserver reset failed (%d): %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

// Refresh — байршуулсны дараа дуудна: мозайкийг бэлдэж, шинэ granule-ийг
// индексжүүлж, кэшийг цэвэрлэнэ.
func (c *Client) Refresh(ctx context.Context) error {
	if err := c.Harvest(ctx); err != nil {
		// Store нь GeoServer-ийн каталогт байгаа ч дискэн дээрх индекс
		// (drone.shp/.properties) гээгдсэн/эвдэрсэн байж болно — тэр үед
		// harvest "Failed to create reader" гэж унадаг. EnsureMosaic нь store
		// байгааг хараад indexer.properties-ыг дахин бичихгүй тул өөрөө
		// сэргэхгүй. Иймд НЭГ удаа нурааж дахин барина (.tif-үүд хөндөгдөхгүй).
		slog.Warn("geoserver harvest failed — mosaic rebuild хийж байна", "error", err)
		return c.Rebuild(ctx)
	}
	// Store нь өмнө (жишээ: make config-оор) үүссэн байж мэдэх тул давхцуулах
	// тохиргоог үргэлж дахин баталгаажуулна — идемпотент.
	if err := c.configureCoverage(ctx); err != nil {
		slog.Warn("geoserver coverage config failed", "error", err)
	}
	return c.Reset(ctx)
}

// Rebuild — мозайкийг НУРААЖ дахин үүсгэнэ.
//
// Хэзээ хэрэгтэй: мозайк нь хуучин тохиргоотой (жишээ: HeterogeneousCRS=false)
// үүссэн бол өөр CRS-тэй зургийг ЧИМЭЭГҮЙ хүлээж авахгүй. Индекс нь дүрсээс
// дахин бүтээгддэг (derived) өгөгдөл тул нурааж дахин барихад зураг гээгдэхгүй —
// .tif файлууд хавтсанд байгаа хэвээр.
func (c *Client) Rebuild(ctx context.Context) error {
	// 1. Store-ийг бүх coverage/layer-тэй нь хамт устгана
	status, body, err := c.do(ctx, http.MethodDelete,
		fmt.Sprintf("/rest/workspaces/%s/coveragestores/%s?recurse=true", c.workspace, c.store),
		"", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNotFound {
		return fmt.Errorf("mosaic delete failed (%d): %s", status, strings.TrimSpace(string(body)))
	}

	// 2. Устгасны ДАРАА кэшийг цэвэрлэнэ. Үүнгүйгээр GeoServer устгагдсан
	//    store-ийн reader-ийг кэшдээ хадгалж, дахин үүсгэхэд
	//    "Could not acquire reader for coverage" (500) буцаана.
	if err := c.Reset(ctx); err != nil {
		return err
	}

	// 3. Хуучин индексийн файлуудыг цэвэрлэнэ (.tif-үүдэд ХҮРЭХГҮЙ)
	if err := c.removeIndexFiles(); err != nil {
		return err
	}

	// 4. Шинэ тохиргоотой дахин үүсгээд granule-уудыг индексжүүлнэ
	if err := c.Harvest(ctx); err != nil {
		return err
	}
	if err := c.configureCoverage(ctx); err != nil {
		return err
	}
	slog.Info("geoserver mosaic rebuilt", "workspace", c.workspace, "store", c.store)
	return c.Reset(ctx)
}

// removeIndexFiles — мозайкийн индекс/тохиргооны файлуудыг устгана.
// Зургийн файл (.tif/.tiff) хэзээ ч устахгүй.
func (c *Client) removeIndexFiles() error {
	if c.localDir == "" {
		return nil
	}
	entries, err := os.ReadDir(c.localDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".tif" || ext == ".tiff" {
			continue // ортофотог хэзээ ч устгахгүй
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		// drone.* (индексийн shapefile/properties), sample_image*, indexer.properties
		isIndex := stem == c.store ||
			strings.HasPrefix(name, "sample_image") ||
			name == "indexer.properties" ||
			name == "datastore.properties"
		if !isIndex {
			continue
		}
		if err := os.Remove(filepath.Join(c.localDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Bounds — WGS84 хүрээ.
type Bounds struct {
	MinX, MinY, MaxX, MaxY float64
}

type granuleResponse struct {
	Features []struct {
		Properties map[string]any `json:"properties"`
		Geometry   struct {
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
		BBox []float64 `json:"bbox"`
	} `json:"features"`
}

var ErrGranuleNotFound = errors.New("granule not found")

// GranuleBoundsWait — granule-ийн хүрээг уншина, олдохгүй бол хэд хэдэн удаа
// дахин оролдоно.
//
// Яагаад шаардлагатай: harvest (POST external.imagemosaic) нь 202 Accepted
// буцаадаг — индексжүүлэлт АСИНХРОН. Шууд асуувал granule хараахан бүртгэгдээгүй
// байж "олдсонгүй" гэж гарна. Тэрийг "CRS-ээс болж хүлээж авсангүй" гэж андуурч
// мозайкийг нурааж дахин барих (Rebuild) нь мозайкийг эвдэж байсан.
func (c *Client) GranuleBoundsWait(ctx context.Context, fileName string, attempts int) (*Bounds, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		bounds, err := c.GranuleBounds(ctx, fileName)
		if err == nil {
			return bounds, nil
		}
		lastErr = err
		// Зөвхөн "олдсонгүй" тохиолдолд хүлээнэ — бусад алдаа (сүлжээ,
		// эрх) дахин оролдоод сайжрахгүй.
		if !errors.Is(err, ErrGranuleNotFound) {
			return nil, err
		}
		if i+1 < attempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i+1) * 500 * time.Millisecond):
			}
		}
	}
	return nil, lastErr
}

// GranuleBounds — тухайн файлын granule-ийн хүрээг мозайкийн индексээс уншина.
// "Зураг дээр очих" (zoom) товчинд хэрэглэнэ; уншиж чадаагүй нь мохоо алдаа
// биш тул дуудагч зүгээр өнгөрөөж болно.
func (c *Client) GranuleBounds(ctx context.Context, fileName string) (*Bounds, error) {
	filter := url.QueryEscape(fmt.Sprintf("location LIKE '%%%s'", fileName))
	path := fmt.Sprintf(
		"/rest/workspaces/%s/coveragestores/%s/coverages/%s/index/granules.json?filter=%s",
		c.workspace, c.store, c.store, filter,
	)
	status, body, err := c.do(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("granule query failed (%d)", status)
	}

	var parsed granuleResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Features) == 0 {
		return nil, ErrGranuleNotFound
	}

	f := parsed.Features[0]
	var b *Bounds
	if len(f.BBox) >= 4 {
		b = &Bounds{MinX: f.BBox[0], MinY: f.BBox[1], MaxX: f.BBox[2], MaxY: f.BBox[3]}
	} else {
		// bbox байхгүй бол геометрийн координатуудаас өөрөө бодно.
		var ok bool
		b, ok = boundsFromCoordinates(f.Geometry.Coordinates)
		if !ok {
			return nil, ErrGranuleNotFound
		}
	}

	// Мозайк EPSG:3857 тул координат МЕТРЭЭР ирнэ — WGS84 болгоно.
	b = webMercatorToWGS84(b)
	if err := validateWGS84(b); err != nil {
		return nil, fmt.Errorf("granule %s: %w", fileName, err)
	}
	return b, nil
}

// webMercatorToWGS84 — координат метрээр (|x| > 180) байвал EPSG:3857-оос
// градус болгоно. Аль хэдийн градус бол хөндөхгүй (хуучин 4326 мозайктай
// нийцтэй байхын тулд).
func webMercatorToWGS84(b *Bounds) *Bounds {
	if math.Abs(b.MinX) <= 180 && math.Abs(b.MaxX) <= 180 &&
		math.Abs(b.MinY) <= 90 && math.Abs(b.MaxY) <= 90 {
		return b
	}
	const r = 6378137.0
	lon := func(x float64) float64 { return x / r * 180 / math.Pi }
	lat := func(y float64) float64 {
		return (2*math.Atan(math.Exp(y/r)) - math.Pi/2) * 180 / math.Pi
	}
	return &Bounds{
		MinX: lon(b.MinX), MinY: lat(b.MinY),
		MaxX: lon(b.MaxX), MaxY: lat(b.MaxY),
	}
}

// ValidateWGS84Bounds — хүрээ бодит WGS84 талбай эзэлж байгааг шалгана.
// Экспортлосон нь дуудагч (service) хадгалагдсан хүрээг ч шалгаж чадахын тулд.
func ValidateWGS84Bounds(minX, minY, maxX, maxY float64) error {
	return validateWGS84(&Bounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY})
}

// validateWGS84 — буруу хүрээ DB-д хадгалагдвал frontend дээр газрын зураг
// цагаан болж, OpenLayers хязгааргүй хүсэлтийн эргэлтэд ордог. Тиймээс энд
// хатуу шалгаж, буруу бол ХАДГАЛАХГҮЙ.
func validateWGS84(b *Bounds) error {
	for _, v := range []float64{b.MinX, b.MinY, b.MaxX, b.MaxY} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("bounds contain NaN/Inf")
		}
	}
	if math.Abs(b.MinX) > 180 || math.Abs(b.MaxX) > 180 {
		return fmt.Errorf("longitude out of range: %.6f..%.6f", b.MinX, b.MaxX)
	}
	if math.Abs(b.MinY) > 90 || math.Abs(b.MaxY) > 90 {
		return fmt.Errorf("latitude out of range: %.6f..%.6f", b.MinY, b.MaxY)
	}
	if b.MinX >= b.MaxX || b.MinY >= b.MaxY {
		return fmt.Errorf("degenerate bounds: %.6f,%.6f..%.6f,%.6f", b.MinX, b.MinY, b.MaxX, b.MaxY)
	}
	return nil
}

// boundsFromCoordinates — GeoJSON-ийн үүрлэсэн координатын массивыг гүйж хүрээ бодно.
func boundsFromCoordinates(raw json.RawMessage) (*Bounds, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, false
	}
	minX, minY := 0.0, 0.0
	maxX, maxY := 0.0, 0.0
	found := false

	var walk func(any)
	walk = func(n any) {
		arr, ok := n.([]any)
		if !ok {
			return
		}
		if len(arr) >= 2 {
			x, xOK := arr[0].(float64)
			y, yOK := arr[1].(float64)
			if xOK && yOK {
				if !found {
					minX, minY, maxX, maxY = x, y, x, y
					found = true
					return
				}
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
				return
			}
		}
		for _, item := range arr {
			walk(item)
		}
	}
	walk(node)

	if !found {
		return nil, false
	}
	return &Bounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}, true
}

// DeleteGranule — мозайкийн индексээс тухайн файлын granule-ийг хасна.
func (c *Client) DeleteGranule(ctx context.Context, fileName string) error {
	filter := url.QueryEscape(fmt.Sprintf("location LIKE '%%%s'", fileName))
	path := fmt.Sprintf(
		"/rest/workspaces/%s/coveragestores/%s/coverages/%s/index/granules?filter=%s",
		c.workspace, c.store, c.store, filter,
	)
	status, body, err := c.do(ctx, http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}
	// Мозайк/granule байхгүй байсан ч устгах хүсэлт бүтсэн гэж үзнэ.
	if status == http.StatusNotFound {
		return nil
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("granule delete failed (%d): %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}
