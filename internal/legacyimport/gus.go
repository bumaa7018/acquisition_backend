package legacyimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── ГУС (goverment-middleware) клиент ───────────────────────────────────────
//
// Эх систем ачааллаас хамгаалагдсан тул БҮХ хүсэлт НЭГ дараалалд орж,
// хоорондоо Delay (анхдагч 1 секунд) завсарлагатай явна. Зэрэгцээ хүсэлт
// явуулахгүй. Хариуг санах ойд кэшлэнэ (олдоогүй хариуг ч) — нэг дугаарыг
// давтан асуухгүй.

// GusConfig — backend-ийн MIDDLEWARE_* тохиргоо.
type GusConfig struct {
	BaseURL  string
	Username string
	Password string
	// Delay — хүсэлт хоорондын завсарлага. 0 бол 1 секунд.
	Delay time.Duration
}

// ErrGusAuth — нэвтрэх эрх татгалзсан; дахин оролдох нь утгагүй.
var ErrGusAuth = errors.New("ГУС нэвтрэх эрх татгалзсан — MIDDLEWARE_USERNAME/MIDDLEWARE_PASSWORD-ийг шалгана уу")

// SourceParcel — ГУС-ын нэгж талбарын мэдээлэл (/parcel/info эсвэл /parcel/ub).
type SourceParcel struct {
	ParcelID            string   `json:"parcelID"`
	OldParcelID         string   `json:"oldParcelID"`
	GeometryWKT         string   `json:"geometryWKT"`
	AU1Code             string   `json:"au1Code"`
	AU2Code             string   `json:"au2Code"`
	AU3Code             string   `json:"au3Code"`
	RightType           *int     `json:"rightType"`
	Landuse             string   `json:"landuse"`
	LanduseName         string   `json:"landuseName"`
	AreaM2              *float64 `json:"areaM2"`
	AddressKhashaa      string   `json:"addressKhashaa"`
	AddressStreet       string   `json:"addressStreet"`
	PropertyNo          string   `json:"propertyNo"`
	ValidFrom           string   `json:"validFrom"`
	ValidTill           string   `json:"validTill"`
	AppNo               string   `json:"appNo"`
	AppTimestamp        string   `json:"appTimestamp"`
	AppType             string   `json:"appType"`
	AppStatus           string   `json:"appStatus"`
	DecisionNo          string   `json:"decisionNo"`
	DecisionDate        string   `json:"decisionDate"`
	ContractNo          string   `json:"contractNo"`
	ContractDate        string   `json:"contractDate"`
	ContractBegin       string   `json:"contractBegin"`
	ContractEnd         string   `json:"contractEnd"`
	ContractPropertyNo  string   `json:"contractPropertyNo"`
	ContractStatus      string   `json:"contractStatus"`
	CertificateNo       string   `json:"certificateNo"`
	CertificateDate     string   `json:"certificateDate"`
	RecordNo            string   `json:"recordNo"`
	RecordDate          string   `json:"recordDate"`
	RecordCertificateNo string   `json:"recordCertificateNo"`
	RecordStatus        string   `json:"recordStatus"`
	// Origin — info (/parcel/info) | ub (data_ub.ca_ub_parcel)
	Origin string `json:"origin"`
}

// PlanProject — /plan/project?code=
type PlanProject struct {
	ParcelID     string `json:"parcelID"`
	ProjectID    string `json:"projectID"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	PlanTypeName string `json:"planTypeName"`
	Gazner       string `json:"gazner"`
	BoundaryWKT  string `json:"boundaryWKT"`
	StartData    string `json:"startData"`
	EndData      string `json:"endData"`
	ApprovedDate string `json:"approvedDate"`
}

// GusHit — хайлтын үр дүн: мэдээлэл ба ЯГ АЛЬ дугаараар олдсон нь.
type GusHit struct {
	Source SourceParcel `json:"source"`
	UsedID string       `json:"usedID"`
}

type gusCacheEntry struct {
	source *SourceParcel
	err    error
}

// GusClient — цуварсан, кэштэй ГУС клиент (процесс даяар нэг).
type GusClient struct {
	cfg  GusConfig
	http *http.Client

	mu       sync.Mutex // дараалал: нэг удаад нэг хүсэлт
	lastCall time.Time

	cacheMu sync.Mutex
	cache   map[string]gusCacheEntry
}

func NewGusClient(cfg GusConfig) *GusClient {
	if cfg.Delay <= 0 {
		cfg.Delay = time.Second
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &GusClient{cfg: cfg, http: &http.Client{Timeout: 30 * time.Second}, cache: map[string]gusCacheEntry{}}
}

// ClearFailures — өмнөх ажлын ТҮР алдааг мартана (амжилттай/олдоогүй хариу үлдэнэ).
func (c *GusClient) ClearFailures() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	for k, v := range c.cache {
		if v.err != nil {
			delete(c.cache, k)
		}
	}
}

const gusMaxRetries = 2

func (c *GusClient) request(ctx context.Context, path string) (map[string]any, error) {
	if c.cfg.BaseURL == "" {
		return nil, errors.New("MIDDLEWARE_BASE_URL тохируулаагүй")
	}
	if c.cfg.Username == "" || c.cfg.Password == "" {
		return nil, errors.New("ГУС-ын нэвтрэх эрх (MIDDLEWARE_USERNAME, MIDDLEWARE_PASSWORD) backend-д тохируулаагүй")
	}
	var lastErr error
	for attempt := 0; attempt <= gusMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.cfg.Delay * time.Duration(attempt)):
			}
		}
		data, status, err := c.once(ctx, path)
		switch {
		case err == nil && status == http.StatusNotFound:
			return nil, nil
		case err == nil && (status == http.StatusUnauthorized || status == http.StatusForbidden):
			return nil, ErrGusAuth
		case err == nil && status >= 200 && status < 300:
			return data, nil
		case err == nil:
			err = fmt.Errorf("ГУС %d: %s", status, path)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		lastErr = err
		slog.Warn("ГУС дуудлага амжилтгүй", "path", path, "attempt", attempt+1, "error", err)
	}
	return nil, lastErr
}

// once — дараалалд орж, өмнөх хүсэлтээс хойш Delay хүлээгээд НЭГ хүсэлт.
func (c *GusClient) once(ctx context.Context, path string) (map[string]any, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.cfg.Delay - time.Since(c.lastCall); wait > 0 {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-time.After(wait):
		}
	}
	defer func() { c.lastCall = time.Now() }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Middleware-Username", c.cfg.Username)
	req.Header.Set("X-Middleware-Password", c.cfg.Password)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, nil
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("ГУС хариу JSON биш: %w", err)
	}
	return envelope.Data, resp.StatusCode, nil
}

// ── Хариуг тэвчих туслахууд (утга текст ч, тоо ч байж болно) ────────────────

func flexString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	}
	return ""
}

func noZero(v any) string {
	if s := flexString(v); s != "0" {
		return s
	}
	return ""
}

func flexNumber(v any) *float64 {
	if n, err := strconv.ParseFloat(flexString(v), 64); err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
		return &n
	}
	return nil
}

func flexInt(v any) *int {
	if n := flexNumber(v); n != nil {
		i := int(*n)
		return &i
	}
	return nil
}

// flexDate — "infinity" ба 4 оронгоос урт он → хоосон (хугацаагүй).
func flexDate(v any) string {
	raw := flexString(v)
	if raw == "" || strings.EqualFold(raw, "infinity") || strings.EqualFold(raw, "-infinity") {
		return ""
	}
	if y, _, ok := strings.Cut(raw, "-"); ok && len(y) > 4 {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.000", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format("2006-01-02")
		}
	}
	if len(raw) >= 10 {
		if t, err := time.Parse("2006-01-02", raw[:10]); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

var wktPrefixes = []string{"POINT", "LINESTRING", "POLYGON", "MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION", "SRID="}

func sanitizeWKT(raw string) string {
	wkt := strings.TrimSpace(raw)
	upper := strings.ToUpper(wkt)
	for _, p := range wktPrefixes {
		if strings.HasPrefix(upper, p) {
			return wkt
		}
	}
	return ""
}

func toSourceParcel(d map[string]any, requested, origin string) *SourceParcel {
	id := flexString(d["parcel_id"])
	if id == "" {
		id = requested
	}
	if id == "" {
		return nil
	}
	geom := flexString(d["geometry_wkt"])
	if geom == "" {
		geom = flexString(d["geometry"])
	}
	au2 := flexString(d["au2_code"])
	if au2 == "" {
		au2 = flexString(d["au2"])
	}
	return &SourceParcel{
		ParcelID: id, OldParcelID: noZero(d["old_parcel_id"]), GeometryWKT: sanitizeWKT(geom),
		AU1Code: flexString(d["au1_code"]), AU2Code: au2, AU3Code: flexString(d["au3_code"]),
		RightType: flexInt(d["right_type"]), Landuse: flexString(d["landuse"]), LanduseName: flexString(d["landuse_name"]),
		AreaM2: flexNumber(d["area_m2"]), AddressKhashaa: noZero(d["address_khashaa"]),
		AddressStreet: noZero(d["address_streetname"]), PropertyNo: noZero(d["property_no"]),
		ValidFrom: flexDate(d["valid_from"]), ValidTill: flexDate(d["valid_till"]),
		AppNo: flexString(d["app_no"]), AppTimestamp: flexDate(d["app_timestamp"]),
		AppType: flexString(d["app_type"]), AppStatus: flexString(d["app_status"]),
		DecisionNo: flexString(d["decision_no"]), DecisionDate: flexDate(d["decision_date"]),
		ContractNo: flexString(d["contract_no"]), ContractDate: flexDate(d["contract_date"]),
		ContractBegin: flexDate(d["contract_begin"]), ContractEnd: flexDate(d["contract_end"]),
		ContractPropertyNo: noZero(d["contract_property_no"]), ContractStatus: flexString(d["contract_status"]),
		CertificateNo: flexString(d["certificate_no"]), CertificateDate: flexDate(d["certificate_date"]),
		RecordNo: flexString(d["record_no"]), RecordDate: flexDate(d["record_date"]),
		RecordCertificateNo: flexString(d["record_certificate_no"]), RecordStatus: flexString(d["record_status"]),
		Origin: origin,
	}
}

// PlanProject — төлөвлөгөөг ТӨЛӨВЛӨГӨӨНИЙ НЭГЖ ТАЛБАРЫН ДУГААРААР.
func (c *GusClient) PlanProject(ctx context.Context, planParcelID string) (*PlanProject, error) {
	d, err := c.request(ctx, "/plan/project?code="+url.QueryEscape(planParcelID))
	if err != nil || d == nil {
		return nil, err
	}
	code := flexString(d["code"])
	if code == "" {
		return nil, nil
	}
	pid := flexString(d["parcel_id"])
	if pid == "" {
		pid = planParcelID
	}
	first := func(keys ...string) string {
		for _, k := range keys {
			if v := flexString(d[k]); v != "" {
				return v
			}
		}
		return ""
	}
	return &PlanProject{
		ParcelID: pid, ProjectID: flexString(d["project_id"]), Code: code, Name: flexString(d["name"]),
		PlanTypeName: flexString(d["plan_type_name"]), Gazner: flexString(d["gazner"]),
		BoundaryWKT: sanitizeWKT(flexString(d["geometry_wkt"])),
		StartData:   first("start_data", "start_date"), EndData: first("end_data", "end_date"),
		ApprovedDate: flexString(d["approved_date"]),
	}, nil
}

// parcel — /parcel/info эсвэл /parcel/ub, шинэ эсвэл хуучин дугаараар (кэштэй).
func (c *GusClient) parcel(ctx context.Context, origin, id string, old bool) (*SourceParcel, error) {
	key := fmt.Sprintf("%s:%t:%s", origin, old, id)
	c.cacheMu.Lock()
	if hit, ok := c.cache[key]; ok {
		c.cacheMu.Unlock()
		return hit.source, hit.err
	}
	c.cacheMu.Unlock()

	path := "/parcel/" + origin + "/" + url.PathEscape(id)
	if old {
		path += "?id_type=old"
	}
	d, err := c.request(ctx, path)
	var src *SourceParcel
	if err == nil && d != nil {
		src = toSourceParcel(d, id, origin)
	}
	if ctx.Err() == nil { // цуцлагдсан хүсэлтийг кэшлэхгүй
		c.cacheMu.Lock()
		c.cache[key] = gusCacheEntry{source: src, err: err}
		c.cacheMu.Unlock()
	}
	return src, err
}

// ParcelByEitherID — эхний олдсоныг: info(шинэ) → info(хуучин) → ub(шинэ) → ub(хуучин).
func (c *GusClient) ParcelByEitherID(ctx context.Context, parcelID, oldParcelID string) (*GusHit, error) {
	hasOld := oldParcelID != "" && oldParcelID != parcelID
	for _, origin := range []string{"info", "ub"} {
		if parcelID != "" {
			src, err := c.parcel(ctx, origin, parcelID, false)
			if err != nil {
				return nil, err
			}
			if src != nil {
				return &GusHit{Source: *src, UsedID: parcelID}, nil
			}
		}
		if hasOld {
			src, err := c.parcel(ctx, origin, oldParcelID, true)
			if err != nil {
				return nil, err
			}
			if src != nil {
				return &GusHit{Source: *src, UsedID: oldParcelID}, nil
			}
		}
	}
	return nil, nil
}
