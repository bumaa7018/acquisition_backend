// Package geoserver — дроны ортофотог GeoServer-ээр давхарга болгон нийтлэх
// REST client.
//
// АРХИТЕКТУР: GeoServer растрыг ФАЙЛЫН СИСТЕМЭЭС (MinIO) ШУУД уншина —
// COG (Cloud Optimized GeoTIFF) + HTTP Range хүсэлтээр. Backend растрыг
// өөр дээрээ ХАДГАЛАХГҮЙ, GeoServer-т ч дамжуулахгүй: зөвхөн объектын URL-ийг
// каталогт бүртгэнэ.
//
// Зураг тус бүр өөрийн coverage store + давхаргатай (`land:<store>`):
//   - хуваалцсан хавтас, мозайкийн индекс, indexer.properties шаардлагагүй
//   - зураг бүр өөрийн CRS-тэй тул heterogeneous CRS-ийн тохиргоо ч хэрэггүй
//   - хүрээ (bbox) нь coverage-ийн latLonBoundingBox-оос шууд гарна
package geoserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// geoserverTimeout — GeoServer-ийн REST дуудлагын дээд хугацаа.
//
// Хамгийн урт дуудлага нь растрыг нээх (coverages?list=all / coverage үүсгэх)
// — доорх NewClient-ийн тайлбарыг үзнэ үү.
const geoserverTimeout = 10 * time.Minute

type Client struct {
	baseURL   string // .../geoserver
	user      string
	pass      string
	workspace string
	http      *http.Client
}

// NewClient — baseURL нь GeoServer-ийн үндэс (жишээ:
// http://geoserver:8080/geoserver). workspace нь аль хэдийн байх ёстой
// (government-geoserver: make config үүсгэдэг).
func NewClient(baseURL, user, pass, workspace string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		user:      user,
		pass:      pass,
		workspace: workspace,
		// Алсын растрыг ПЕРВЫЙ УДАА нээх нь хамгийн үнэтэй үйлдэл: GeoServer
		// толгойг (IFD) HTTP Range-ээр уншина.
		//
		// ЯАГААД ТАК УУЖУУ: жинхэнэ COG дээр энэ нь 1-2 секунд. Харин COG БИШ
		// GeoTIFF-ийн толгой файлын төгсгөлд байдаг тул уншигч файлыг эхнээс
		// дуустал мөлхөж унших ба энэ нь файлын хэмжээ, сүлжээнээс хамаарч
		// минут хүрч болно (138 MB зураг дээр анхдагч тохиргоотой 230-350 сек
		// хэмжигдсэн — тэр үед 120 сек-ийн timeout нь ЗУРГИЙГ БҮРМӨСӨН
		// НИЙТЛЭГДЭХГҮЙ болгож байсан).
		//
		// geoserver-ийн блокийн хэмжээг томсгосноор (docker-compose дахь
		// it.geosolutions.cog.default.header.length) тэр 4-6 сек болсон ч
		// timeout нь ХАМГААЛАЛТ хэвээр байх ёстой: хэрэглэгч 500 MB хүртэл
		// зураг оруулж болно.
		http: &http.Client{Timeout: geoserverTimeout},
	}
}

// Configured — GeoServer-ийн тохиргоо бүрэн эсэх.
func (c *Client) Configured() bool { return c.baseURL != "" && c.workspace != "" }

// LayerName — WMS-д явуулах давхаргын бүтэн нэр.
func (c *Client) LayerName(store string) string {
	if !c.Configured() || store == "" {
		return ""
	}
	return c.workspace + ":" + store
}

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
	data, err := io.ReadAll(res.Body)
	return res.StatusCode, data, err
}

func (c *Client) exists(ctx context.Context, path string) (bool, error) {
	status, _, err := c.do(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return false, err
	}
	return status == http.StatusOK, nil
}

func (c *Client) storePath(store string) string {
	return fmt.Sprintf("/rest/workspaces/%s/coveragestores/%s", c.workspace, url.PathEscape(store))
}

// cogStoreXML — COG coverage store-ийн тодорхойлолт.
//
// Хоёр зүйл ХОЁУЛАА шаардлагатай (аль нэг нь дутвал GeoServer файлыг локал зам
// гэж үзэн нээж чадахгүй):
//   - url нь `cog://` угтвартай (CoverageReaderCogInputObjectConverter үүнээс
//     алсын COG гэдгийг таньдаг)
//   - `CogSettings.Key` метадата дахь rangeReaderSettings=HTTP
//
// MinIO-ийн bucket нь зөвхөн УНШИХ нийтийн бодлоготой тул GeoServer-т нэр/нууц
// үг тохируулах шаардлагагүй — HTTP Range хүсэлт хангалттай.
func cogStoreXML(workspace, store, cogURL string) []byte {
	return []byte(`<coverageStore>` +
		`<name>` + xmlEscape(store) + `</name>` +
		`<type>GeoTIFF</type><enabled>true</enabled>` +
		`<workspace><name>` + xmlEscape(workspace) + `</name></workspace>` +
		`<url>cog://` + xmlEscape(cogURL) + `</url>` +
		`<metadata><entry key="CogSettings.Key"><cogSettingsStore>` +
		`<rangeReaderSettings>HTTP</rangeReaderSettings>` +
		`<useCachingStream>false</useCachingStream>` +
		`</cogSettingsStore></entry></metadata>` +
		`</coverageStore>`)
}

var xmlEscaper = strings.NewReplacer(
	"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;",
)

func xmlEscape(s string) string { return xmlEscaper.Replace(s) }

// Publish — объектын URL-ийг GeoServer-т давхарга болгон нийтэлж, WGS84 хүрээг
// буцаана. Идемпотент: аль хэдийн байгаа store/coverage-ийг дахин үүсгэхгүй.
func (c *Client) Publish(ctx context.Context, store, cogURL string, hasAlpha bool) (*Bounds, error) {
	if !c.Configured() {
		return nil, errors.New("geoserver is not configured")
	}
	if err := c.ensureStore(ctx, store, cogURL); err != nil {
		return nil, err
	}
	if err := c.ensureCoverage(ctx, store, hasAlpha); err != nil {
		return nil, err
	}
	return c.CoverageBounds(ctx, store)
}

func (c *Client) ensureStore(ctx context.Context, store, cogURL string) error {
	ok, err := c.exists(ctx, c.storePath(store)+".json")
	if err != nil {
		return fmt.Errorf("geoserver unreachable: %w", err)
	}
	if ok {
		return nil
	}
	status, body, err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/rest/workspaces/%s/coveragestores", c.workspace),
		"text/xml", cogStoreXML(c.workspace, store, cogURL))
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("cog store create failed (%d): %s", status, snippet(body))
	}
	slog.Info("geoserver cog store created", "workspace", c.workspace, "store", store, "url", cogURL)
	return nil
}

// nativeCoverageName — GeoServer-ээс растрын дотоод coverage нэрийг асууна.
//
// Энэ дуудлага нь COG-ийг АЛСААС нээж чадсан эсэхийн шалгуур ч болно: файлд
// хүрч чадаагүй бол хоосон буцна.
func (c *Client) nativeCoverageName(ctx context.Context, store string) (string, error) {
	status, body, err := c.do(ctx, http.MethodGet, c.storePath(store)+"/coverages.json?list=all", "", nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("coverage list failed (%d): %s", status, snippet(body))
	}
	var res struct {
		List struct {
			String []string `json:"string"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("coverage list parse failed: %w", err)
	}
	if len(res.List.String) == 0 {
		return "", errors.New("geoserver растрыг уншиж чадсангүй (объект хүртээмжтэй, COG эсэхийг шалгана уу)")
	}
	return res.List.String[0], nil
}

func (c *Client) ensureCoverage(ctx context.Context, store string, hasAlpha bool) error {
	coveragePath := c.storePath(store) + "/coverages/" + url.PathEscape(store)
	ok, err := c.exists(ctx, coveragePath+".json")
	if err != nil {
		return err
	}
	if !ok {
		native, err := c.nativeCoverageName(ctx, store)
		if err != nil {
			return err
		}
		body := []byte(`<coverage><name>` + xmlEscape(store) + `</name>` +
			`<nativeName>` + xmlEscape(native) + `</nativeName>` +
			`<nativeCoverageName>` + xmlEscape(native) + `</nativeCoverageName>` +
			`<enabled>true</enabled></coverage>`)
		status, res, err := c.do(ctx, http.MethodPost,
			c.storePath(store)+"/coverages", "text/xml", body)
		if err != nil {
			return err
		}
		if status != http.StatusCreated && status != http.StatusOK {
			return fmt.Errorf("coverage publish failed (%d): %s", status, snippet(res))
		}
		slog.Info("geoserver coverage published", "store", store, "native", native)
	}
	// Давхцуулж харуулах тохиргоо — идемпотент.
	return c.Configure(ctx, store, hasAlpha)
}

// Configure — давхаргыг ДАВХЦУУЛЖ харуулахад тохирсон болгоно. Идемпотент тул
// тохиргоо код дээр солигдоход refresh нь хуучин давхаргад ч дахин хэрэглэнэ.
//
// hasAlpha — растрт alpha зурвас байгаа эсэх. Тунгалаг байдлыг хэрхэн шийдэх нь
// ҮҮНЭЭС ШАЛТГААЛНА, хоёр тохиолдлыг ХОЛИХ БОЛОМЖГҮЙ:
//
//	alpha БАЙХГҮЙ (3 зурваст RGB, ихэвчлэн JPEG-шахалттай):
//	  nodata нь бүтэн ХАР дүүргэлт болж хадгалагддаг тул
//	  InputTransparentColor=#000000 хэрэгтэй. Хэмжсэн (190 MB ортофото, захын
//	  6% фрэйм дээр): үүнгүйгээр хар 59.4%, үүнтэй 5.5% болж буурна.
//
//	alpha БАЙГАА (4 зурваст RGBA):
//	  InputTransparentColor-ыг ТАВИХ БОЛОМЖГҮЙ. Тавьбал GeoServer тунгалаг
//	  байдлыг өнгө тааруулж ӨӨРӨӨ бүтээх ба файлын alpha зурвасыг ХАЯНА. Alpha
//	  доорх RGB нь ихэвчлэн ЦАГААН байдаг тул фон бүтэн цагаан болно. Хэмжсэн
//	  (138 MB, alpha-тай ортофото): ITC-тэй үед тунгалаг 4.7%/цагаан 33.0%,
//	  ITC-гүй үед тунгалаг 37.6%/цагаан 0.0%.
//
// ЯАГААД Output- БИШ Input-: OutputTransparentColor нь ImageMosaic-ийн
// параметр — ердийн GeoTIFF/COG store дээр GeoServer түүнийг ХАДГАЛДАГ ч
// УНШИГЧ нь мэдэхгүй тул нөлөө үзүүлэхгүй (хар фон хэвээр үлдэнэ).
//
// ҮЛДЭХ ХЯЗГААР: InputTransparentColor нь өнгийг ЯГ тааруулдаг (tolerance
// байхгүй). JPEG нь гэмтэлтэй шахалт тул хар дүүргэлтийн зах (1,1,2) г.м.
// болж хувирдаг ба тааруулалтад өртөхгүй — иймд нимгэн хар хүрээ ҮЛДЭНЭ
// (хэмжсэн 5.5%). Энэ нь overview-ээс биш ЭХ ФАЙЛД шигдсэн: OverviewPolicy
// QUALITY/IGNORE хоёр ижил хэмжигдсэн. Бүрэн арилгахын тулд эх файлд alpha /
// mask нэмэх шаардлагатай (COG болгон хөрвүүлэх үед).
func (c *Client) Configure(ctx context.Context, store string, hasAlpha bool) error {
	transparent := `{"string":["InputTransparentColor","#000000"]},`
	if hasAlpha {
		transparent = ""
	}
	body := []byte(`{"coverage":{"parameters":{"entry":[` +
		transparent +
		`{"string":["OverviewPolicy","QUALITY"]}` +
		`]}}}`)
	status, res, err := c.do(ctx, http.MethodPut,
		c.storePath(store)+"/coverages/"+url.PathEscape(store),
		"application/json", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("coverage config failed (%d): %s", status, snippet(res))
	}
	// Тохиргоо СОЛИГДСОН тул GeoWebCache-ийн хуучин тайлуудыг хаях ЗААВАЛ.
	// Frontend нь тайлаар (TILED=true) дуудаж GWC-ийн кэшийг ашигладаг ба
	// /rest/reset нь тайлын кэшийг ЦЭВЭРЛЭДЭГГҮЙ — иймд үүнгүйгээр шинэ
	// тохиргоо хэрэглэгчид ХАРАГДАХГҮЙ (хэмжсэн: тунгалаг фонгүй хуучин тайл
	// truncate хийх хүртэл үйлдсээр байв).
	if err := c.truncateCache(ctx, store); err != nil {
		// Кэш хаяж чадаагүй нь тохиргоог бүтэлгүйтсэн гэж үзэх шалтгаан биш —
		// тайл нь expire болоод шинэчлэгдэнэ.
		slog.Warn("gwc truncate failed", "layer", c.LayerName(store), "error", err)
	}
	return nil
}

// truncateCache — давхаргын кэшлэгдсэн тайлуудыг GeoWebCache-ээс хаяна.
func (c *Client) truncateCache(ctx context.Context, store string) error {
	body := []byte(`<truncateLayer><layerName>` +
		xmlEscape(c.LayerName(store)) + `</layerName></truncateLayer>`)
	status, res, err := c.do(ctx, http.MethodPost, "/gwc/rest/masstruncate", "text/xml", body)
	if err != nil {
		return err
	}
	// Хэзээ ч кэшлэгдээгүй давхарга дээр 400 буцаж болно — асуудал биш.
	if status != http.StatusOK && status != http.StatusBadRequest {
		return fmt.Errorf("gwc masstruncate failed (%d): %s", status, snippet(res))
	}
	return nil
}

// Delete — давхарга, coverage, store-ыг каталогоос устгана. MinIO дээрх объект
// хөндөгдөхгүй — түүнийг дуудагч тусад нь устгана.
func (c *Client) Delete(ctx context.Context, store string) error {
	if !c.Configured() || store == "" {
		return nil
	}
	status, body, err := c.do(ctx, http.MethodDelete, c.storePath(store)+"?recurse=true", "", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNotFound {
		return fmt.Errorf("cog store delete failed (%d): %s", status, snippet(body))
	}
	return nil
}

// Reset — GeoServer-ийн resource кэшийг цэвэрлэнэ.
func (c *Client) Reset(ctx context.Context) error {
	status, body, err := c.do(ctx, http.MethodPost, "/rest/reset", "", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("geoserver reset failed (%d): %s", status, snippet(body))
	}
	return nil
}

type Bounds struct {
	MinX, MinY, MaxX, MaxY float64
}

// CoverageBounds — давхаргын WGS84 хүрээ. GeoServer өөрөө latLonBoundingBox-ыг
// тооцоод хадгалдаг тул нэмэлт хөрвүүлэлт шаардахгүй.
func (c *Client) CoverageBounds(ctx context.Context, store string) (*Bounds, error) {
	status, body, err := c.do(ctx, http.MethodGet,
		c.storePath(store)+"/coverages/"+url.PathEscape(store)+".json", "", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("coverage read failed (%d): %s", status, snippet(body))
	}
	var res struct {
		Coverage struct {
			LatLonBoundingBox struct {
				MinX float64 `json:"minx"`
				MinY float64 `json:"miny"`
				MaxX float64 `json:"maxx"`
				MaxY float64 `json:"maxy"`
			} `json:"latLonBoundingBox"`
		} `json:"coverage"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("coverage parse failed: %w", err)
	}
	b := res.Coverage.LatLonBoundingBox
	bounds := &Bounds{MinX: b.MinX, MinY: b.MinY, MaxX: b.MaxX, MaxY: b.MaxY}
	if err := ValidateWGS84Bounds(bounds.MinX, bounds.MinY, bounds.MaxX, bounds.MaxY); err != nil {
		return nil, err
	}
	return bounds, nil
}

// ValidateWGS84Bounds — хүрээ нь бодит WGS84 (lon/lat) эсэхийг шалгана.
//
// Тэнхлэг солигдсон эсвэл хөрвүүлэлт бүтэлгүйтсэн хүрээ frontend дээр газрын
// зургийг эвдэдэг (NaN extent) тул DB-д бичихээс ӨМНӨ хатуу шалгана.
func ValidateWGS84Bounds(minX, minY, maxX, maxY float64) error {
	for _, v := range []float64{minX, minY, maxX, maxY} {
		if v != v { // NaN
			return errors.New("bounds contain NaN")
		}
	}
	if minX < -180 || maxX > 180 || minY < -90 || maxY > 90 {
		return fmt.Errorf("bounds outside WGS84 range: [%f %f %f %f]", minX, minY, maxX, maxY)
	}
	if minX >= maxX || minY >= maxY {
		return fmt.Errorf("bounds are empty or inverted: [%f %f %f %f]", minX, minY, maxX, maxY)
	}
	return nil
}

func snippet(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}
