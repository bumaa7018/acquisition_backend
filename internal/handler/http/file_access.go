package http

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FileAccessHandler — файлын системийн объектод хандах эрхийг шалгана.
//
// Бүх файл MinIO дээр байх ба frontend-ийн `/api/files/<key>` proxy-гоор
// тараагдана. Тэр proxy өмнө нь ЗӨВХӨН нэвтэрсэн эсэхийг шалгадаг байсан тул
// нэвтэрсэн дурын хэрэглэгч (гадаад роль ч гэсэн) өөр байгууллагын баримтыг
// татаж чаддаг байв. Түлхүүр нь `acquisition/<acq_id>/<файлын нэр>` хэлбэртэй
// бөгөөд баримтын төрөл зааж өгсөн тохиолдолд файлын нэр нь ЭХ нэр хэвээр
// үлддэг (санамсаргүй бүрэлдэхүүнгүй) тул таамаглах боломжтой байсан.
//
// Энэ маршрут түлхүүрийг задалж, харьяалагдах чөлөөлөлт/нэгж талбарыг олоод,
// харгалзах API маршрутуудад мөрдөгддөг ЯГ ижил дүрмийг хэрэглэнэ. ШИНЭ эрх
// нэмээгүй — одоо байгаа land:read / compensation:read болон хуваарилалт /
// байгууллагын гишүүнчлэлийн шалгалтуудыг л ашиглана.
type FileAccessHandler struct {
	svc service.LandService
}

func NewFileAccessHandler(svc service.LandService) *FileAccessHandler {
	return &FileAccessHandler{svc: svc}
}

// splitObjectKey — "acquisition/<uuid>/нэр.pdf" → ("acquisition", "<uuid>").
// Гурван хэсэггүй түлхүүр (bucket/prefix жагсаах оролдлого) хүчингүй.
func splitObjectKey(key string) (namespace, ownerID string, ok bool) {
	parts := strings.Split(strings.Trim(key, "/"), "/")
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// supervisoryReader — байгууллага даяар УНШИХ үүрэгтэй ролиуд
// (middleware.RequireAssignedOrSenior-ийн supervisoryReadRoles-тэй ижил жагсаалт).
func supervisoryReader(c *gin.Context) bool {
	return ctxHasRole(c,
		"finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү",
		"mika", "МИКА",
		"decision_registration", "Захирамж бүртгэл")
}

// Authorize godoc
// @Summary      Файлын объектод хандах эрхийг шалгах
// @Description  frontend-ийн /api/files proxy энэ маршрутаар объект тус бүрийн эрхийг шалгана
// @Tags         files
// @Produce      json
// @Security     BearerAuth
// @Param        key  query  string  true  "Объектын түлхүүр (namespace/owner_id/нэр)"
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /files/authorize [get]
func (h *FileAccessHandler) Authorize(c *gin.Context) {
	// Татгалзсан шалтгааныг ТОДОРХОЙ бичнэ. Урьд нь бүх татгалзал ялгаагүй
	// 403 байсан тул "яагаад харагдахгүй байна вэ" гэдгийг лог, дэмжлэгээс
	// тогтоох боломжгүй байв (эрх, гишүүнчлэл, хуваарилалт, түлхүүрийн
	// формат — аль нь ч байж болно).
	deny := func(stage string) {
		userID, _ := middleware.GetCallerID(c)
		orgID, hasOrg := middleware.GetCallerOrgID(c)
		slog.Warn("file access denied",
			"stage", stage,
			"key", c.Query("key"),
			"user_id", userID,
			"org_id", orgID,
			"has_org", hasOrg,
			"valuation_org", middleware.IsValuationOrgCaller(c),
			"senior", middleware.IsSenior(c),
		)
		response.Error(c, domain.ErrForbidden)
	}

	namespace, ownerID, ok := splitObjectKey(c.Query("key"))
	if !ok {
		deny("bad_key")
		return
	}

	// 1-р давхарга: уншиж болох эрх огт байгаа эсэх.
	//
	// Үнэлгээний байгууллагын ажилтан системийн эрх (permission) ЭЗЭМШИХГҮЙ —
	// түүний хандалт нь 2-р давхаргын БАЙГУУЛЛАГЫН гишүүнчлэлээр шалгагдана.
	// decision — захирамжийн хавсралт: decision:read эрх ч уншиж болох эрхэд тооцогдоно.
	if !middleware.IsValuationOrgCaller(c) &&
		!callerHasPermission(c, "land:read") && !callerHasPermission(c, "compensation:read") &&
		(namespace != "decision" || !callerHasPermission(c, "decision:read")) {
		deny("no_read_permission")
		return
	}

	// 1б давхарга: санхүү/МИКА — зөвхөн ҮНЭЛГЭЭНИЙ файл. Тэдний API маршрутууд
	// (нийгмийн судалгаа, чөлөөлөлтийн баримт, мэдэгдэл, захирамж) хаалттай
	// бөгөөд файлын замаар тэр хаалтыг тойрох ёсгүй.
	if reviewerOnlyCaller(c) && !reviewerFileNamespaces[namespace] {
		deny("reviewer_namespace:" + namespace)
		return
	}

	// 2-р давхарга: тухайн объектын ЭЗЭН рүү буулгаж шалгана.
	var err error
	switch namespace {
	// socio-survey — нийгэм, эдийн засгийн судалгааны PDF. Түлхүүрийн эзэн нь
	// ЧӨЛӨӨЛӨЛТИЙН id тул чөлөөлөлтийн ижил дүрмээр шалгагдана.
	case "acquisition", "drone", "socio-survey":
		err = h.authorizeAcquisition(c, ownerID)
	// notice / emongolia — иргэнд илгээсэн мэдэгдэх хуудсын PDF (имэйл ба
	// E-Mongolia). Эзэн нь НЭГЖ ТАЛБАР.
	// valuation-return — үнэлгээг БУЦААХАД хавсаргасан файл. Түлхүүр нь
	// `valuation-return/<нэгж талбарын uuid>/<нэр>` тул нэгж талбарын ИЖИЛ
	// дүрмээр шалгагдана: буцаалтыг санхүү хийдэг ч хавсралтыг үнэлгээ
	// бичсэн байгууллага УНШИХ шаардлагатай.
	// valuation-cancel — баталгаажсан үнэлгээг ЦУЦЛАХАД заавал хавсаргах
	// үндэслэлийн PDF (`valuation-cancel/<нэгж талбарын uuid>/<нэр>`).
	// valuation-return-тэй ижил дүрэм: цуцлалтыг санхүү/ахлах хийдэг ч
	// үндэслэлийг үнэлгээ бичсэн байгууллага УНШИХ шаардлагатай.
	// estimated-value — ТӨСӨӨЛЛИЙН үнэлгээг гэрчлэх хавсралт. Түлхүүр нь
	// `estimated-value/<нэгж талбарын uuid>/<нэр>` тул нэгж талбарын ИЖИЛ
	// дүрмээр шалгагдана.
	// valuation-draft — БАТАЛГААЖААГҮЙ үнэлгээний тайлан (илгээх үед хавсаргана).
	// Санхүүгийн мэргэжилтэн хянахдаа, үнэлгээ хийсэн байгууллага буцаж шалгахдаа
	// уншина — тиймээс нэгж талбарын ИЖИЛ дүрмээр.
	case "parcel", "notice", "emongolia", "valuation-return", "valuation-cancel", "estimated-value", "valuation-draft":
		err = h.authorizeParcel(c, ownerID)
	case "compensation-report":
		err = h.authorizeCompensation(c, ownerID)
	case "asset-photo":
		err = h.authorizeAsset(c, ownerID)
	// decision — баталгаажсан захирамжийн хавсралт (`decision/<захирамжийн id>/<нэр>`).
	// Захирамжийн цэсээр (decision:read) эсвэл холбогдсон нэгж талбарын
	// хавсралтаар (тэр нэгж талбарын дүрмээр) уншина.
	case "decision":
		err = h.authorizeDecision(c, ownerID)
	// land-release — «Буулгалтын мэдээлэл»-ийн захирамж. Эзэн нь тусдаа (нэгж
	// талбар/чөлөөлөлттэй холбоогүй) тул цэсний эрхээр (land:read) л уншина.
	case "land-release":
		if middleware.IsValuationOrgCaller(c) || !callerHasPermission(c, "land:read") {
			err = domain.ErrForbidden
		}
	default:
		// Танихгүй namespace — зөвшөөрөхгүй (шинэ төрөл нэмэгдвэл энд бүртгэнэ).
		err = domain.ErrForbidden
	}
	// Санхүү/МИКА — нэгж талбарын хавсралтаас зөвхөн үнэлгээний төрлийнх
	// (/parcels/:id/documents жагсаалттай ижил дүрэм).
	if err == nil && namespace == "parcel" && reviewerOnlyCaller(c) {
		err = h.authorizeReviewerParcelFile(c, ownerID, c.Query("key"))
	}
	if err != nil {
		deny("owner_check:" + namespace)
		return
	}
	response.OK(c, gin.H{"allowed": true})
}

func (h *FileAccessHandler) authorizeAcquisition(c *gin.Context, id string) error {
	acqID, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrForbidden
	}
	// Мэргэжлийн байгууллага — БАЙГУУЛЛАГЫН гишүүнчлэлээр (/prof маршрутын адил).
	if profOnlyCaller(c) {
		orgID, ok := middleware.GetCallerOrgID(c)
		if !ok {
			return domain.ErrForbidden
		}
		member, err := h.svc.IsProfMemberOfAcquisition(c.Request.Context(), acqID, orgID)
		if err != nil || !member {
			return domain.ErrForbidden
		}
		return nil
	}
	// Хэлтсийн хүрээ — /land-acquisitions/:id маршрутын RequireAcquisitionDepartment-тэй ижил.
	if !h.inCallerDepartment(c, func(ctx context.Context) (*int, error) {
		return h.svc.AcquisitionDepartmentID(ctx, acqID)
	}) {
		return domain.ErrForbidden
	}
	// Ахлах/админ, мөн хяналтын унших ролиуд — хуваарилалтгүйгээр.
	if middleware.IsSenior(c) || supervisoryReader(c) {
		return nil
	}
	userID, ok := middleware.GetCallerID(c)
	if !ok {
		return domain.ErrForbidden
	}
	assigned, err := h.svc.IsAssigned(c.Request.Context(), acqID, userID)
	if err != nil || !assigned {
		return domain.ErrForbidden
	}
	return nil
}

// inCallerDepartment — файлын эзэн (чөлөөлөлт/нэгж талбар) дуудагчийн хэлтэст
// хамаарах эсэх. Системийн админ хязгааргүй; хэлтэсгүй дотоод дуудагч, эсвэл
// хэлтэс нь тодорхойгүй эзэн — хаалттай (маршрутын middleware-тэй ижил).
// Өмнө нь ахлах/санхүү/МИКА файлын түлхүүр мэдвэл БҮХ хэлтсийн файлыг нээдэг байв.
func (h *FileAccessHandler) inCallerDepartment(c *gin.Context, lookup func(context.Context) (*int, error)) bool {
	mine, restricted := middleware.DataDepartmentScope(c)
	if !restricted {
		return true
	}
	if mine == nil {
		return false
	}
	theirs, err := lookup(c.Request.Context())
	return err == nil && theirs != nil && *theirs == *mine
}

// reviewerFileNamespaces — санхүү/МИКА-д нээлттэй файлын төрлүүд: үнэлгээний
// тайлан, хөрөнгийн зураг, үнэлгээг буцаах/цуцлах/илгээх хавсралт, төсөөллийн
// үнэлгээ, нэгж талбарын хавсралт (доор төрлөөр нь дахин шүүнэ).
var reviewerFileNamespaces = map[string]bool{
	"compensation-report": true,
	"asset-photo":         true,
	"valuation-return":    true,
	"valuation-cancel":    true,
	"valuation-draft":     true,
	"estimated-value":     true,
	"parcel":              true,
}

// authorizeReviewerParcelFile — түлхүүрт харгалзах нэгж талбарын хавсралт нь
// хяналтын ролийн харах төрлийнх эсэх. Хавсралт олдохгүй бол хаалттай.
func (h *FileAccessHandler) authorizeReviewerParcelFile(c *gin.Context, parcelIDStr, key string) error {
	parcelID, err := uuid.Parse(parcelIDStr)
	if err != nil {
		return domain.ErrForbidden
	}
	docs, err := h.svc.ListParcelDocuments(c.Request.Context(), parcelID)
	if err != nil {
		return domain.ErrForbidden
	}
	visible := reviewerDocFilter(c.Request.Context(), h.svc)
	for _, d := range docs {
		if documentMatchesKey(d, key) {
			if visible(d) {
				return nil
			}
			return domain.ErrForbidden
		}
	}
	// Цуцлагдсан үнэлгээний архивын файл (тайлан, эх хүснэгт) — хавсралтаас
	// хасагдсан ч үнэлгээний түүхэнд нээгдэнэ.
	archived, err := h.svc.ArchivedValuationFileURLs(c.Request.Context(), parcelID)
	if err != nil {
		return domain.ErrForbidden
	}
	for _, u := range archived {
		if documentMatchesKey(&domain.Document{FileURL: u}, key) {
			return nil
		}
	}
	return domain.ErrForbidden
}

func (h *FileAccessHandler) authorizeParcel(c *gin.Context, id string) error {
	parcelID, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrForbidden
	}
	if profOnlyCaller(c) {
		orgID, ok := middleware.GetCallerOrgID(c)
		if !ok {
			return domain.ErrForbidden
		}
		member, err := h.svc.IsProfMemberOfParcel(c.Request.Context(), parcelID, orgID)
		if err != nil || !member {
			return domain.ErrForbidden
		}
		return nil
	}
	// Хэлтсийн хүрээ — /parcels/:id маршрутын RequireParcelDepartment-тэй ижил.
	if !h.inCallerDepartment(c, func(ctx context.Context) (*int, error) {
		return h.svc.ParcelDepartmentID(ctx, parcelID)
	}) {
		return domain.ErrForbidden
	}
	if middleware.IsSenior(c) || supervisoryReader(c) {
		return nil
	}
	userID, ok := middleware.GetCallerID(c)
	if !ok {
		return domain.ErrForbidden
	}
	assigned, err := h.svc.IsAssignedToParcel(c.Request.Context(), parcelID, userID)
	if err != nil || !assigned {
		return domain.ErrForbidden
	}
	return nil
}

// authorizeCompensation — үнэлгээний тайлан. Эзэн нь нөхөх олговрын мөр тул
// эхлээд харьяалагдах чөлөөлөлтийг олж, дараа нь ижил дүрмээр шалгана.
//
// Үнэлгээ ЦУЦЛАГДАХАД олговрын мөр устдаг ч файл нь архивын snapshot дотор
// харагдсаар байдаг тул эзнийг олохдоо snapshot-ыг ч хамруулна
// (GetCompensationAcquisition) — эс бөгөөс архивын тайлан бүр 403 болно.
func (h *FileAccessHandler) authorizeCompensation(c *gin.Context, id string) error {
	compID, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrForbidden
	}
	acqID, err := h.svc.GetCompensationAcquisition(c.Request.Context(), compID)
	if err != nil {
		return domain.ErrForbidden
	}
	// Үнэлгээний байгууллага зөвхөн ӨӨРИЙН урсгалын тайланг нээнэ (уншилтын
	// дүрэм бичилтийнхтэй ижил). Мөр устсан (цуцлагдсан үнэлгээний архив) бол
	// урсгалыг тодорхойлох боломжгүй тул доорх гишүүнчлэлийн шалгалтад үлдэнэ.
	if profOnlyCaller(c) {
		_, err := h.svc.AuthorizeCompensationWrite(c.Request.Context(), acqID, compID, valuationCallerFrom(c))
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
	}
	return h.authorizeAcquisition(c, acqID.String())
}

// authorizeAsset — байгууламжийн зураг. Эзэн нь хөрөнгийн мөр (устсан бол
// архивын snapshot).
func (h *FileAccessHandler) authorizeAsset(c *gin.Context, id string) error {
	assetID, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrForbidden
	}
	acqID, err := h.svc.GetAssetAcquisition(c.Request.Context(), assetID)
	if err != nil {
		return domain.ErrForbidden
	}
	// authorizeCompensation-тэй ижил: байгууллага зөвхөн ӨӨРИЙН урсгалын зургийг.
	if profOnlyCaller(c) {
		_, err := h.svc.AuthorizeAssetWrite(c.Request.Context(), acqID, assetID, valuationCallerFrom(c))
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
	}
	return h.authorizeAcquisition(c, acqID.String())
}

// authorizeDecision — захирамжийн хавсралт. decision:read эрхтэй бол шууд;
// эс бөгөөс захирамжид холбогдсон нэгж талбарын АЛЬ НЭГИЙГ уншиж чадвал
// (нэгж талбарын хавсралтаас нээж буй тохиолдол) зөвшөөрнө.
func (h *FileAccessHandler) authorizeDecision(c *gin.Context, id string) error {
	draftID, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrForbidden
	}
	if !profOnlyCaller(c) && callerHasPermission(c, "decision:read") {
		return nil
	}
	parcelIDs, err := h.svc.DecisionDraftParcelIDs(c.Request.Context(), draftID)
	if err != nil {
		return domain.ErrForbidden
	}
	for _, pid := range parcelIDs {
		if h.authorizeParcel(c, pid.String()) == nil {
			return nil
		}
	}
	return domain.ErrForbidden
}
