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
	"progress":         acquisitionProgressExpr,
	"created_at":       "la.created_at",
}

// acquisitionProgressExpr — чөлөөлөлтийн ГҮЙЦЭТГЭЛИЙН ХУВЬ (0-100) SQL дээр.
// domain.LandAcquisition.ProgressPercent-тэй ИЖИЛ: эцсийн төлөвт
// (parcel_status.is_final) шилжсэн талбар / нийт талбар, бүхэл рүү тоймлоно;
// нэгж талбаргүй бол 0.
const acquisitionProgressExpr = `(SELECT CASE WHEN COUNT(*) = 0 THEN 0 ELSE LEAST(100, ROUND(
	SUM(CASE WHEN COALESCE(pp.status,0) IN (SELECT id FROM parcel_status WHERE is_final) THEN 1 ELSE 0 END)
	* 100.0 / COUNT(*))) END
	FROM parcel pp WHERE pp.acquisition_id = la.id)`

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

// globalCompensationSortColumns — нөхөн төлбөрийн нэгдсэн жагсаалт (/compensations).
var globalCompensationSortColumns = map[string]string{
	"parcel_id":         "c.parcel_id",
	"holder":            "TRIM(CONCAT_WS(' ', NULLIF(rh.last_name,''), rh.name))",
	"register_no":       "rh.register_no",
	"acquisition_name":  "a.acquisition_name",
	"compensation_type": "c.compensation_type",
	"amount":            "COALESCE(c.amount, 0)",
	"compensation_date": "c.compensation_date",
	"status":            "c.status",
	"created_at":        "c.created_at",
}

// reportParcelSortColumns — тайлангийн нэгж талбарын жагсаалт (/report/download).
// Query нь GROUP BY-тай тул зөвхөн бүлэглэсэн багана эсвэл нэгтгэх функц орно.
var reportParcelSortColumns = map[string]string{
	"construction_type":   "CONCAT_WS(' / ', NULLIF(gc.name,''), NULLIF(sc.name,''))",
	"acquisition_name":    "la.acquisition_name",
	"decree_date":         "la.decree_date",
	"holder":              "TRIM(CONCAT_WS(' ', NULLIF(rh.last_name,''), rh.name))",
	"parcel_id":           "p.parcel_id",
	"area_m2":             "COALESCE(p.area_m2, 0)",
	"right_type":          "p.right_type",
	"acquisition_area_m2": "COALESCE(p.acquisition_area_m2, 0)",
	"land_comp":           "COALESCE(SUM(CASE WHEN c.target_type = 'parcel' THEN c.amount ELSE 0 END), 0)",
	"real_state_comp":     "COALESCE(SUM(CASE WHEN c.target_type = 'asset' AND a.asset_type = 'real_state' THEN c.amount ELSE 0 END), 0)",
	"property_comp":       "COALESCE(SUM(CASE WHEN c.target_type = 'asset' AND a.asset_type = 'property' THEN c.amount ELSE 0 END), 0)",
	"total_comp": `COALESCE(SUM(CASE WHEN c.target_type = 'parcel'
		OR (c.target_type = 'asset' AND a.asset_type IN ('real_state', 'property'))
		THEN c.amount ELSE 0 END), 0)`,
	"remaining_area_m2": "COALESCE(p.remaining_area_m2, p.area_m2 - COALESCE(p.acquisition_area_m2, 0), 0)",
	"db_changed":        "p.db_changed",
	"changed_parcel_id": "p.changed_parcel_id",
}

// decisionDraftSortColumns — захирамжийн төсөл / захирамжийн жагсаалт.
var decisionDraftSortColumns = map[string]string{
	"proposal_no":      "d.proposal_no",
	"decree_number":    "d.decree_number",
	"decision_date":    "d.decision_date",
	"location":         "d.location",
	"duration_year":    "d.duration_year",
	"status":           "d.status",
	"current_progress": "ph.progress_type",
	"funding_source": `(SELECT string_agg(fs.organization_name, ', ' ORDER BY fs.organization_name)
		FROM decision_draft_funding_source ddfs
		JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
		WHERE ddfs.decision_draft_id = d.id)`,
	"work_type": "wt.name",
	"budget":    "bg.name",
	"parcel_count": `(SELECT COUNT(*) FROM decision_draft_parcel ddp
		WHERE ddp.decision_draft_id = d.id AND ddp.removed_at IS NULL)`,
	"created_at": "d.created_at",
}

// userSortColumns — хэрэглэгчийн жагсаалт (userFromSQL-ийн u/e/pos alias-тай).
var userSortColumns = map[string]string{
	"name":     "TRIM(CONCAT_WS(' ', NULLIF(u.lastname,''), u.firstname))",
	"username": "u.username",
	"employee": `(SELECT ` + personDisplayNameExpr + `
		FROM base.bs_person p WHERE p.person_id = e.person_id)`,
	"department": `(SELECT d.name FROM sdplatform.hr_department d
		WHERE d.department_id = e.department_id)`,
	"position":   "pos.name",
	"email":      "u.email",
	"is_active":  "COALESCE(u.is_active, false)",
	"created_at": "u.created_at",
}

// auditLogSortColumns — үйлдлийн бүртгэл (audit_log).
var auditLogSortColumns = map[string]string{
	"created_at":    "created_at",
	"actor_name":    "actor_name",
	"action":        "action",
	"resource_type": "resource_type",
	"ip_address":    "ip_address",
}

// parcelSurveySortColumns — чөлөөлөгдсөн газрын судалгааны жагсаалт.
var parcelSurveySortColumns = map[string]string{
	"parcel_id":        "p.parcel_id",
	"acquisition_name": "la.acquisition_name",
	"chimney_count":    "s.chimney_count",
	"toilet_count":     "s.toilet_count",
	"cohabitant_count": "s.cohabitant_count",
	"relocation_type":  "s.relocation_type",
	"updated_at":       "s.updated_at",
}
