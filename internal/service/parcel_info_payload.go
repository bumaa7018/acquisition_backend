package service

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
)

// parcelInfoPayload — дундын сервисийн /parcel/info/{parcel_id} хариулт.
//
// Хариулт нь эх системийн SQL-ийн БАГАНУУДЫГ шууд дамжуулдаг тул дараах
// зөрүүг тэвчих ёстой:
//
//   - SQL-д БАЙХГҮЙ талбар ирэхгүй: id, plan_code, au1_code, au3_code,
//     holder_* (эзэмшигчийг /application/persons-оос авна), created_at.
//   - Геометр нь "geometry_wkt" ЭСВЭЛ "geometry" нэрээр ирж болно.
//   - Хэсэгчилсэн байршил (au2) нь "au2_code" эсвэл "au2" байж болно.
//   - Кодууд нь текст ч, тоо ч байж болно (landuse=2201, property_no,
//     old_parcel_id...) — эх баганууд numeric байдаг.
//   - "0" нь "хоосон" гэсэн утга (old_parcel_id=0, address_khashaa=0).
//   - Огноо нь "2026-05-13", "2026-03-30 17:00:00.000", RFC3339 гэх мэт
//     хэлбэртэй. 'infinity' (DBeaver-т 292278994-08-17) болон 4 оронгоос
//     хэтэрсэн он нь "хугацаа хязгааргүй" гэсэн утгатай тул NULL болно.
type parcelInfoPayload struct {
	ParcelID    flexString `json:"parcel_id"`
	OldParcelID flexString `json:"old_parcel_id"`

	GeometryWKT flexString `json:"geometry_wkt"`
	Geometry    flexString `json:"geometry"`

	AU1Code  flexString `json:"au1_code"`
	AU2Code  flexString `json:"au2_code"`
	AU2Alt   flexString `json:"au2"`
	AU3Code  flexString `json:"au3_code"`
	PlanCode flexString `json:"plan_code"`

	RightType   flexInt    `json:"right_type"`
	Landuse     flexString `json:"landuse"`
	LanduseName flexString `json:"landuse_name"`
	AreaM2      flexFloat  `json:"area_m2"`

	AddressKhashaa flexString `json:"address_khashaa"`
	AddressStreet  flexString `json:"address_streetname"`
	PropertyNo     flexString `json:"property_no"`

	ValidFrom flexTime `json:"valid_from"`
	ValidTill flexTime `json:"valid_till"`

	AppNo        flexString `json:"app_no"`
	AppTimestamp flexTime   `json:"app_timestamp"`
	AppType      flexString `json:"app_type"`
	AppStatus    flexString `json:"app_status"`

	DecisionNo   flexString `json:"decision_no"`
	DecisionDate flexTime   `json:"decision_date"`

	// Гэрээ. SQL дээр certificate_no / property_no нэрс ХОЁР УДАА (гэрээ ба
	// бүртгэл) гарч ирдэг тул дундын сервис тэднийг contract_* / record_*
	// болгож нэрлэдэг — JSON-д ижил түлхүүр хоёр удаа байж болохгүй.
	ContractNo         flexString `json:"contract_no"`
	ContractDate       flexTime   `json:"contract_date"`
	ContractBegin      flexTime   `json:"contract_begin"`
	ContractEnd        flexTime   `json:"contract_end"`
	ContractPropertyNo flexString `json:"contract_property_no"`
	ContractStatus     flexString `json:"contract_status"`
	CertificateNo      flexString `json:"certificate_no"`
	CertificateDate    flexTime   `json:"certificate_date"`

	// Улсын бүртгэл
	RecordNo            flexString `json:"record_no"`
	RecordDate          flexTime   `json:"record_date"`
	RecordCertificateNo flexString `json:"record_certificate_no"`
	RecordStatus        flexString `json:"record_status"`
}

// toSourceParcel — хариултыг домэйн бүтэц болгоно. parcelID нь дуудсан дугаар
// (хариултад дугаар ирээгүй үед хэрэглэнэ).
func (p *parcelInfoPayload) toSourceParcel(parcelID string) *domain.SourceParcel {
	src := &domain.SourceParcel{
		ParcelID: firstNonEmpty(p.ParcelID.value(), strings.TrimSpace(parcelID)),
		// "0" = хоосон (эх системд "хуучин дугаар байхгүй"-г 0-оор тэмдэглэдэг)
		OldParcelID: p.OldParcelID.valueNoZero(),
		PlanCode:    p.PlanCode.value(),
		GeometryWKT: firstNonEmpty(p.GeometryWKT.value(), p.Geometry.value()),

		AU1Code: p.AU1Code.value(),
		AU2Code: firstNonEmpty(p.AU2Code.value(), p.AU2Alt.value()),
		AU3Code: p.AU3Code.value(),

		RightType:   p.RightType.value(),
		Landuse:     p.Landuse.value(),
		LanduseName: p.LanduseName.value(),
		AreaM2:      p.AreaM2.value(),

		AddressKhashaa: p.AddressKhashaa.valueNoZero(),
		AddressStreet:  p.AddressStreet.valueNoZero(),
		PropertyNo:     p.PropertyNo.valueNoZero(),

		ValidFrom: p.ValidFrom.value(),
		ValidTill: p.ValidTill.value(),

		AppNo:        p.AppNo.value(),
		AppTimestamp: p.AppTimestamp.value(),
		AppType:      p.AppType.value(),
		AppStatus:    p.AppStatus.value(),

		DecisionNo:   p.DecisionNo.value(),
		DecisionDate: p.DecisionDate.value(),

		ContractNo:         p.ContractNo.value(),
		ContractDate:       p.ContractDate.value(),
		ContractBegin:      p.ContractBegin.value(),
		ContractEnd:        p.ContractEnd.value(),
		ContractPropertyNo: p.ContractPropertyNo.valueNoZero(),
		ContractStatus:     p.ContractStatus.value(),
		CertificateNo:      p.CertificateNo.value(),
		CertificateDate:    p.CertificateDate.value(),

		RecordNo:            p.RecordNo.value(),
		RecordDate:          p.RecordDate.value(),
		RecordCertificateNo: p.RecordCertificateNo.value(),
		RecordStatus:        p.RecordStatus.value(),
	}

	// Танигдаагүй хэлбэртэй огноо чимээгүй гээгдэхээс сэргийлж логлоно.
	unparsed := map[string]string{}
	for name, f := range map[string]flexTime{
		"valid_from":       p.ValidFrom,
		"valid_till":       p.ValidTill,
		"app_timestamp":    p.AppTimestamp,
		"decision_date":    p.DecisionDate,
		"contract_date":    p.ContractDate,
		"contract_begin":   p.ContractBegin,
		"contract_end":     p.ContractEnd,
		"certificate_date": p.CertificateDate,
		"record_date":      p.RecordDate,
	} {
		if f.parsed == nil && f.raw != "" && !isUnboundedDate(f.raw) {
			unparsed[name] = f.raw
		}
	}
	if len(unparsed) > 0 {
		slog.Warn("нэгж талбарын огнооны хэлбэр танигдсангүй",
			"parcel_id", src.ParcelID, "fields", unparsed)
	}
	return src
}

// wktGeometryTypes — эх системээс ирж болох геометрийн төрлүүд.
var wktGeometryTypes = []string{
	"POINT", "LINESTRING", "POLYGON",
	"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON",
	"GEOMETRYCOLLECTION", "SRID=",
}

// sanitizeWKT — эх системийн геометрийг хадгалахын өмнө хэлбэрийг нь шалгана.
//
// plan.boundary_wkt нь ST_GeomFromText-ээр бичигддэг: алдаатай текст ирвэл
// query УНАЖ, төлөвлөгөө хадгалагдахгүй → чөлөөлөлт огт үүсэхгүй болно.
// Танигдахгүй утгыг хаяж (логлож) хилгүй үргэлжлүүлэх нь аюулгүй.
func sanitizeWKT(raw string) string {
	wkt := strings.TrimSpace(raw)
	if wkt == "" {
		return ""
	}
	upper := strings.ToUpper(wkt)
	for _, prefix := range wktGeometryTypes {
		if strings.HasPrefix(upper, prefix) {
			return wkt
		}
	}
	slog.Warn("геометрийн хэлбэр танигдсангүй, хил хадгалагдахгүй",
		"value", truncateForLog(wkt))
	return ""
}

// truncateForLog — геометр нь мянган тэмдэгт байж болох тул логт таслана.
func truncateForLog(s string) string {
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// flexString — текст эсвэл тоогоор ирж болох талбар (эх багана numeric байж болно).
type flexString struct{ s string }

func (f *flexString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		f.s = strings.TrimSpace(s)
		return nil
	}
	// тоо, true/false — байгаа хэлбэрээр текст болгоно
	f.s = strings.TrimSpace(string(data))
	return nil
}

func (f flexString) value() string { return f.s }

// valueNoZero — эх системд "байхгүй"-г "0"-оор тэмдэглэдэг талбарууд
// (old_parcel_id, address_khashaa зэрэг) дээр хэрэглэнэ.
func (f flexString) valueNoZero() string {
	if f.s == "0" {
		return ""
	}
	return f.s
}

// flexInt — тоо эсвэл текстээр ирж болох бүхэл тоо.
type flexInt struct{ n int }

func (f *flexInt) UnmarshalJSON(data []byte) error {
	var s flexString
	if err := s.UnmarshalJSON(data); err != nil {
		return err
	}
	if s.s == "" {
		return nil
	}
	n, err := strconv.ParseFloat(s.s, 64)
	if err != nil {
		return nil // танигдаагүй утга → 0
	}
	f.n = int(n)
	return nil
}

func (f flexInt) value() int { return f.n }

// flexFloat — тоо эсвэл текстээр ирж болох бутархай тоо.
type flexFloat struct{ f float64 }

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	var s flexString
	if err := s.UnmarshalJSON(data); err != nil {
		return err
	}
	if s.s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s.s, 64)
	if err != nil {
		return nil
	}
	f.f = v
	return nil
}

func (f flexFloat) value() float64 { return f.f }

// parcelInfoDateLayouts — эх системээс ирж болох огнооны хэлбэрүүд.
var parcelInfoDateLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02.01.2006",
	"2006/01/02",
}

// flexTime — олон хэлбэрийн огноо. Хугацаа хязгааргүй ('infinity', 4 оронгоос
// хэтэрсэн он) болон хоосон утга nil болно.
type flexTime struct {
	parsed *time.Time
	raw    string
}

func (f *flexTime) UnmarshalJSON(data []byte) error {
	var s flexString
	if err := s.UnmarshalJSON(data); err != nil {
		return err
	}
	f.raw = s.s
	if f.raw == "" || isUnboundedDate(f.raw) {
		return nil
	}
	for _, layout := range parcelInfoDateLayouts {
		t, err := time.Parse(layout, f.raw)
		if err != nil {
			continue
		}
		if t.Year() < 1 || t.Year() > 9999 {
			return nil // main DB-ийн date хязгаараас гадна
		}
		parsed := t
		f.parsed = &parsed
		return nil
	}
	return nil
}

func (f flexTime) value() *time.Time { return f.parsed }

// isUnboundedDate — "хугацаа хязгааргүй"-г илэрхийлэх утгууд: Postgres-ийн
// infinity, эсвэл 4 оронгоос урт он (DBeaver 'infinity'-г 292278994-08-17
// гэж харуулдаг).
func isUnboundedDate(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "infinity" || lower == "-infinity" {
		return true
	}
	year, _, found := strings.Cut(lower, "-")
	if !found {
		return false
	}
	if len(year) <= 4 {
		return false
	}
	_, err := strconv.Atoi(year)
	return err == nil
}
