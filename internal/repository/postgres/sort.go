package postgres

import "strings"

// orderClause — ЖАГСААЛТЫН ORDER BY өгүүлбэрийг хэрэглэгчийн сонгосон
// баганаас байгуулна.
//
// Хэрэглэгчийн өгсөн түлхүүрийг SQL-д ХЭЗЭЭ Ч ШУУД оруулахгүй: зөвхөн
// `allowed` жагсаалтад урьдчилан бичигдсэн илэрхийлэл рүү хөрвөнө (SQL
// injection-оос хамгаална). Танигдаагүй эсвэл хоосон түлхүүр ирвэл жагсаалтын
// ҮНДСЭН эрэмбэ (`fallback`) хэвээр үлдэнэ.
//
// `tiebreak` нь ижил утгатай мөрүүд хуудас хооронд үсэрч давхардахаас
// сэргийлэх тогтвортой хоёрдогч эрэмбэ (ихэвчлэн ПК) — хуудаслалттай
// жагсаалтад ЗААВАЛ өгнө.
func orderClause(allowed map[string]string, key, dir, fallback, tiebreak string) string {
	clause := fallback
	if expr, ok := allowed[strings.ToLower(strings.TrimSpace(key))]; ok {
		direction := "ASC"
		if strings.EqualFold(strings.TrimSpace(dir), "desc") {
			direction = "DESC"
		}
		// NULLS LAST — хоосон утгатай мөр аль ч чиглэлд сүүлд эрэмбэлэгдэнэ
		// (эс бөгөөс буурахаар эрэмбэлэхэд хоосон мөрүүд эхэнд цугларна).
		clause = expr + " " + direction + " NULLS LAST"
	}
	// Тогтвортой эрэмбэ нь ҮНДСЭН эрэмбэнд ч хэрэгтэй: ижил огноотой мөрүүд
	// хуудас бүрд өөр дарааллаар ирвэл зарим мөр давхардаж, зарим нь огт
	// харагдахгүй өнгөрнө.
	if tiebreak != "" {
		clause += ", " + tiebreak
	}
	return clause
}

// acquisitionSortColumns — газар чөлөөлөлтийн жагсаалтын эрэмбэлэх боломжтой
// баганууд (frontend-ийн хүснэгтийн толгойтой 1:1 тохирно).
var acquisitionSortColumns = map[string]string{
	"plan_code":        "la.plan_code",
	"acquisition_name": "la.acquisition_name",
	"general_category": "gc.name",
	"sub_category":     "sc.name",
	"status":           "la.status",
	"area_m2":          "la.area_m2",
	"start_date":       "la.start_date",
	"parcel_count":     "(SELECT COUNT(*) FROM parcel WHERE acquisition_id = la.id)",
	"created_at":       "la.created_at",
}

// globalParcelSortColumns — бүх чөлөөлөлтийн нэгж талбарын нэгдсэн жагсаалт.
var globalParcelSortColumns = map[string]string{
	"parcel_id":          "p.parcel_id",
	"acquisition_name":   "la.acquisition_name",
	"plan_code":          "la.plan_code",
	"start_date":         "la.start_date",
	"acquisition_status": "la.status",
	"right_type":         "p.right_type",
	"landuse":            "p.landuse",
	"area_m2":            "p.area_m2",
	"status_id":          "p.status",
	"created_at":         "p.created_at",
	// Нөхөх олговрын БАГАНА нь мөнгөн ба газраар олгосон дүнг хамт харуулдаг
	// тул эрэмбэ нь ҮНДСЭН (сонгогдсон) урсгалын нийт дүнгээр тооцогдоно.
	"compensation": `(SELECT COALESCE(SUM(c.amount), 0) FROM compensation c
		WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
		  AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset'))`,
}

// parcelSortColumns — чөлөөлөлтийн дэлгэрэнгүй доторх нэгж талбарын жагсаалт.
var parcelSortColumns = map[string]string{
	"parcel_id":  "p.parcel_id",
	"au3_code":   "p.au3_code",
	"right_type": "p.right_type",
	"landuse":    "p.landuse",
	"area_m2":    "p.area_m2",
	// "Давхцал" багана — чөлөөлөлтийн хилтэй давхцаж буй талбай.
	"acquisition_area_m2": "COALESCE(p.acquisition_area_m2, 0)",
	"status_id":           "p.status",
	"created_at":          "p.created_at",
	"holder": `(SELECT TRIM(CONCAT_WS(' ', NULLIF(rh.last_name,''), rh.name))
		FROM parcel_holder ph
		JOIN right_holder rh ON rh.id = ph.holder_id
		WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder'
		ORDER BY ph.main_applicant DESC, ph.created_at
		LIMIT 1)`,
	"compensation": `(SELECT COALESCE(SUM(c.amount), 0) FROM compensation c
		WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
		  AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset'))`,
}
