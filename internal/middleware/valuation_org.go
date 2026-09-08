package middleware

import (
	"strings"

	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// Үнэлгээний (мэргэжлийн) байгууллагын хэрэглэгчийн хандалт.
//
// ЭДГЭЭР ХЭРЭГЛЭГЧ СИСТЕМИЙН РОЛЬ ХЭРЭГЛЭХГҮЙ. Байгууллагын ажилтан нь аппын
// өөрийн бүртгэл (appdb.valuation_org_employee) дээр амьдардаг ба нэвтрэхдээ
// токенд `valuation_org` тэмдэг + `org_id` (= valuation_org_profile.id) авдаг.
// sdplatform-ийн sd_role / sd_permission тэдэнд ямар ч эрх НЭМЭХГҮЙ, ямар ч
// эрх ХАСАХГҮЙ — доорх хоёр хаалт нь тэдний хандалтын ЦОРЫН ГАНЦ хэмжүүр:
//
//	ValuationOrgScope         — маршрутын allowlist (бусад бүх зам 403)
//	RequireValuationOrgCaller — /prof группын хаалт
//
// Өгөгдлийн хүрээ (зөвхөн өөрийн байгууллагад холбогдсон чөлөөлөлт / нэгж
// талбар) нь repo давхаргын org шүүлт ба RequireProfOrgMember /
// RequireProfParcelMember дээр хэвээр шалгагдана.

// IsValuationOrgCaller — хүсэлт гаргагч нь үнэлгээний байгууллагын ажилтан эсэх.
func IsValuationOrgCaller(c *gin.Context) bool {
	v, exists := c.Get(CtxValuationOrg)
	if !exists {
		return false
	}
	is, ok := v.(bool)
	return ok && is
}

// RequireValuationOrgCaller — зөвхөн үнэлгээний байгууллагын ажилтанд нээлттэй.
//
// Роль/эрх ШАЛГАХГҮЙ: чөлөөлөлт/нэгж талбарын гишүүнчлэл нь дараагийн
// хаалтууд (RequireProfOrgMember, RequireProfParcelMember) дээр шалгагдана.
func RequireValuationOrgCaller() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsValuationOrgCaller(c) {
			response.Error(c, ErrForbidden)
			return
		}
		c.Next()
	}
}

// AllowValuationOrgOr — үнэлгээний байгууллагын ажилтныг шууд нэвтрүүлж,
// бусад хэрэглэгчийг өгсөн хаалтаар шалгана.
//
// Дотоод хэрэглэгч эрхээр (land:read), үнэлгээний байгууллага гишүүнчлэлээр
// хандаж байгаа ЖИШИГ маршрутуудад (/dashboard) хэрэглэнэ.
func AllowValuationOrgOr(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsValuationOrgCaller(c) {
			c.Next()
			return
		}
		next(c)
	}
}

// valuationOrgRoute — allowlist-ийн нэг бичлэг. path нь gin-ийн маршрутын
// ХЭЛБЭР (`/prof/land-acquisitions/:id`) бөгөөд basePath-гүй бичигдэнэ.
type valuationOrgRoute struct {
	path   string
	prefix bool
	// methods — хоосон бол бүх метод. Лавлахуудад зөвхөн GET нээнэ.
	methods []string
}

// valuationOrgAllowlist — үнэлгээний байгууллагын ажилтанд нээлттэй БҮХ маршрут.
//
// Цэсэнд зөвхөн Дашбоард ба Чөлөөлөлт харагддагтай НИЙЦНЭ (domain.
// ValuationOrgMenus). Шинэ маршрут нэмэхдээ: тухайн зам байгууллагын хүрээгээр
// шүүгддэг эсэхийг ЭХЛЭЭД шалга — allowlist нь зөвхөн "хаана хандаж болох"-ыг
// хэлдэг, "ямар мөр харагдах"-ыг repo давхаргын org шүүлт шийднэ.
var valuationOrgAllowlist = []valuationOrgRoute{
	// Үнэлгээний ажлын үндсэн орон зай — бүхэлдээ байгууллагын гишүүнчлэлээр
	// шүүгддэг (RequireProfOrgMember / RequireProfParcelMember).
	{path: "/prof", prefix: true},

	// Дашбоард — handler дотор org_id-гаар хатуу шүүгдэнэ.
	{path: "/dashboard", methods: []string{"GET"}},

	// Өөрийн профайл ба цэс.
	{path: "/users/me", methods: []string{"GET"}},
	{path: "/menus/me", methods: []string{"GET"}},

	// Мэдэгдэл — repo нь зөвхөн ӨӨРТ хаяглагдсан мөрийг буцаана.
	{path: "/notifications", prefix: true},

	// ── ЛАВЛАХ / ШҮҮЛТҮҮРИЙН маршрутууд (ЗӨВХӨН УНШИХ) ──────────────────────
	//
	// Эдгээр нь dropdown/шүүлтүүрийг тэжээх ХӨНГӨН жагсаалт бөгөөд тодорхой
	// чөлөөлөлтийн өгөгдөл агуулахгүй. Систем дээрээ ч роль/эрх шалгадаггүй
	// (нэвтэрсэн бол хүрнэ) тул үнэлгээний байгууллагад мөн нээлттэй.
	//
	// БИЧИХ (POST/PUT/DELETE) нь эдгээр бүлэг дээр admin:* эрх шаарддаг ба энд
	// ч зөвхөн GET зөвшөөрөгдсөн — хоёр давхарга хоёулаа хаана.
	{path: "/land-acquisitions/filter-options", methods: []string{"GET"}},
	// Шүүлтүүрийн сум/дүүрэг — au2 лавлахаас, чөлөөлөлтийн өгөгдөл агуулахгүй.
	{path: "/land-acquisitions/districts", methods: []string{"GET"}},
	{path: "/acquisition-categories", methods: []string{"GET"}},
	{path: "/parcel-statuses", prefix: true, methods: []string{"GET"}},
	{path: "/acquisition-progress-statuses", prefix: true, methods: []string{"GET"}},
	{path: "/document-types", prefix: true, methods: []string{"GET"}},
	{path: "/parcel-workflow", prefix: true, methods: []string{"GET"}},
	{path: "/acquisition-workflow", prefix: true, methods: []string{"GET"}},
	{path: "/asset-spec-types", prefix: true, methods: []string{"GET"}},
	{path: "/asset-calc-types", prefix: true, methods: []string{"GET"}},
	{path: "/decision-work-types", prefix: true, methods: []string{"GET"}},
	{path: "/decision-budgets", prefix: true, methods: []string{"GET"}},
	// Газрын зургийн СУУРЬ зургийн хаяг — үнэлгээний байгууллага ч газрын
	// зураг хардаг тул ижил суурь зурагтай байх ёстой. ЗӨВХӨН УНШИХ:
	// солих нь admin:update шаарддаг ба энд GET л зөвшөөрөгдсөн.
	{path: "/settings/basemap", methods: []string{"GET"}},

	// Төлөвлөгөөний хайлт/сануулга. Дээрхээс ЯЛГААТАЙ нь: маршрут дээрээ
	// land:read / compensation:read шаарддаг тул router-т AllowValuationOrgOr-
	// оор ороосон (үнэлгээний байгууллага эрхгүй ч токеноороо хүрнэ).
	{path: "/plans/search", methods: []string{"GET"}},
	{path: "/plans/suggest", methods: []string{"GET"}},
}

// ValuationOrgScope — үнэлгээний байгууллагын ажилтныг allowlist-аар хязгаарлана.
//
// JWTAuth-ийн ДАРАА, бүх хамгаалагдсан маршрутын ЭХЭНД тавина. Ингэснээр
// систем даяар шинэ endpoint нэмэгдэхэд тэр нь гадаад байгууллагад
// АНХНААСАА хаалттай байна (deny-by-default) — маршрут тус бүрд
// DenyAnyRole(...) нэмэх шаардлагагүй, мөн хэн нэгэн мартсанаас өгөгдөл
// урсахгүй.
//
// basePath — маршрутын ерөнхий залгуур (`/api/v1`).
func ValuationOrgScope(basePath string) gin.HandlerFunc {
	basePath = strings.TrimSuffix(basePath, "/")
	return func(c *gin.Context) {
		if !IsValuationOrgCaller(c) {
			c.Next()
			return
		}
		// CORS preflight — эрхийн шалгалтын объект биш.
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		route := strings.TrimPrefix(c.FullPath(), basePath)
		if route == "" || !valuationOrgRouteAllowed(route, c.Request.Method) {
			response.Error(c, ErrForbidden)
			return
		}
		c.Next()
	}
}

func valuationOrgRouteAllowed(route, method string) bool {
	for _, allowed := range valuationOrgAllowlist {
		if !allowed.matchesPath(route) {
			continue
		}
		if len(allowed.methods) == 0 {
			return true
		}
		for _, m := range allowed.methods {
			if m == method {
				return true
			}
		}
		// Зам таарсан ч метод таараагүй — өөр бичлэг мөн таарч болно
		// (жишээ нь prefix бичлэг + тодорхой метод).
	}
	return false
}

func (r valuationOrgRoute) matchesPath(route string) bool {
	if route == r.path {
		return true
	}
	// Prefix нь ЗӨВХӨН зам зүйн хилээр таарна: "/prof" нь "/prof/..."-той
	// таарах ч "/professional-orgs"-той таарахгүй.
	return r.prefix && strings.HasPrefix(route, r.path+"/")
}
