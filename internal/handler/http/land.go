package http

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LandHandler struct {
	svc service.LandService
}

func NewLandHandler(svc service.LandService) *LandHandler {
	return &LandHandler{svc: svc}
}

// ListConstructionTypes returns all construction type lookup records.
func (h *LandHandler) ListConstructionTypes(c *gin.Context) {
	list, err := h.svc.ListConstructionTypes(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// ListAcquisitionCategories returns acquisition categories (general or sub).
func (h *LandHandler) ListAcquisitionCategories(c *gin.Context) {
	var parentID *int
	if s := c.Query("parent_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			parentID = &n
		}
	}
	list, err := h.svc.ListAcquisitionCategories(c.Request.Context(), parentID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// ListParcelStatuses returns all parcel status lookup records.
func (h *LandHandler) ListParcelStatuses(c *gin.Context) {
	list, err := h.svc.ListParcelStatuses(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// Suggest godoc
// @Summary      Газар чөлөөлөлт нэр/код-оор autocomplete хайлт
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        q  query  string  true  "Хайх үг (нэр эсвэл план код)"
// @Success      200  {object}  response.Response
// @Router       /land-acquisitions/suggest [get]
func (h *LandHandler) Suggest(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		response.OK(c, []gin.H{})
		return
	}
	// Жагсаалттай ижил дүрэм: санхүү/МИКА зөвхөн "Хээрийн судалгаа" төлөвийн нэрсийг харна
	status := 0
	if ctxHasRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "mika", "МИКА") &&
		!ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн") {
		status = domain.LandAcquisitionStatusField
	}
	list, err := h.svc.Suggest(c.Request.Context(), q, status)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]gin.H, 0, len(list))
	for _, a := range list {
		result = append(result, gin.H{
			"id":               a.ID,
			"acquisition_name": a.AcquisitionName,
			"plan_code":        a.PlanCode,
		})
	}
	response.OK(c, result)
}

// FilterOptions godoc
// @Summary      Шүүлтүүрийн dropdown-д зориулсан хөнгөн жагсаалт
// @Description  Тайлан/нэгж талбарын хуудасны "Төлөвлөгөө" ба "Газар чөлөөлөлт"
// @Description  сонголтуудыг дүүргэхэд зориулав. Үндсэн жагсаалт (GET
// @Description  /land-acquisitions) нь мөр тутамд нэгж талбарын COUNT(*) болон
// @Description  sdplatform join хийдэг тул dropdown-д хэрэглэвэл өгөгдөл өсөх
// @Description  тутам удааширч timeout болдог. Энд зөвхөн id, нэр, план код,
// @Description  план нэр буцна — хуудаслалтгүй.
// @Description
// @Description  Эрх: нэвтэрсэн БҮХ хэрэглэгчид нээлттэй (роль/permission шалгахгүй).
// @Description  Жагсаалтын шүүлтүүрийг хэн ч харж болно; бодит өгөгдлийн эрх нь
// @Description  жагсаалтын endpoint-ууд дээрээ шалгагдана.
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /land-acquisitions/filter-options [get]
// Districts godoc
// @Summary      Шүүлтүүрийн сум/дүүргийн жагсаалт
// @Description  Чөлөөлөлт бүртгэгдсэн сум/дүүргүүд (land_acquisition_au-аас
// @Description  давхардалгүй). Дашбоард болон чөлөөлөлтийн жагсаалтын
// @Description  "Дүүрэг" сонголтыг дүүргэнэ.
// @Description
// @Description  Эрх: нэвтэрсэн БҮХ хэрэглэгчид нээлттэй — FilterOptions-той
// @Description  ижил шалтгаанаар (шүүлтүүрийн сонголт хязгаарлагдвал хэрэглэгч
// @Description  өөрт харагдах ёстой мөрүүдээ шүүж чадахгүй болно).
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /land-acquisitions/districts [get]
func (h *LandHandler) Districts(c *gin.Context) {
	list, err := h.svc.ListAcquisitionDistricts(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

func (h *LandHandler) FilterOptions(c *gin.Context) {
	list, err := h.svc.ListFilterOptions(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]gin.H, 0, len(list))
	for _, o := range list {
		result = append(result, gin.H{
			"id":               o.ID,
			"acquisition_name": o.AcquisitionName,
			"plan_code":        o.PlanCode,
			"plan_name":        o.PlanName,
		})
	}
	response.OK(c, result)
}

// List godoc
// @Summary      Газар чөлөөлөлтийн жагсаалт
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        page             query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size        query  int     false  "Хуудасны хэмжээ (default: 20)"
// @Param        plan_code        query  string  false  "Төлөвлөгөөний дугаар (ILIKE)"
// @Param        status           query  int     false  "Статус (1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED)"
// @Param        au3_code         query  string  false  "Багийн код"
// @Param        start_date_from  query  string  false  "Эхлэх огноо дээд хязгаар (YYYY-MM-DD)"
// @Param        start_date_to    query  string  false  "Эхлэх огноо доод хязгаар (YYYY-MM-DD)"
// @Success      200  {object}  response.PaginatedResponse
// @Router       /land-acquisitions [get]
func (h *LandHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)

	filter := domain.LandAcquisitionFilter{
		PlanCode:          c.Query("plan_code"),
		AcquisitionName:   c.Query("acquisition_name"),
		ParcelID:          strings.TrimSpace(c.Query("parcel_id")),
		Status:            queryInt(c, "status", 0),
		AU2Code:           c.Query("au2_code"),
		AU3Code:           c.Query("au3_code"),
		GeneralCategoryID: queryInt(c, "general_category_id", 0),
		SubCategoryID:     queryInt(c, "sub_category_id", 0),
		AssignedUserID:    c.Query("assigned_user_id"),
		SortBy:            c.Query("sort"),
		SortDir:           c.Query("order"),
		Page:              page,
		PageSize:          pageSize,
	}
	for _, y := range c.QueryArray("year") {
		if n, err := strconv.Atoi(y); err == nil {
			filter.Years = append(filter.Years, n)
		}
	}

	if s := c.Query("start_date_from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDateFrom = &t
		}
	}
	if s := c.Query("start_date_to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDateTo = &t
		}
	}

	// МИКА (хяналтын гадаад роль) зөвхөн "Хээрийн судалгаа" төлөвтэй
	// чөлөөлөлтийг харна — query параметрээс үл хамааран сервер талд хатуу шүүнэ.
	//
	// САНХҮҮГИЙН мэргэжилтэн ЖАГСААЛТАД бүх чөлөөлөлтийг харна: нөхөх олговрын
	// хяналт, санхүүжилтийн тайлан нь дууссан/баталгаажсан чөлөөлөлтийг ч
	// хамардаг. Үйлдлийн эрх (баталгаажуулах, цуцлах, засах) нь ӨӨРЧЛӨГДӨӨГҮЙ —
	// тэдгээр нь тус тусын маршрут/шилжилтийн дүрмээр шалгагдана.
	if ctxHasRole(c, "mika", "МИКА") &&
		!ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн") &&
		!ctxHasRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү") {
		filter.Status = domain.LandAcquisitionStatusField
	}

	list, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]landAcquisitionResponse, 0, len(list))
	for _, acq := range list {
		result = append(result, toLandAcqResponse(acq))
	}
	response.Paginated(c, result, total, page, pageSize)
}

// Create godoc
// @Summary      Газар чөлөөлөлтийн мэдээлэл үүсгэх
// @Description  plan_parcel_id-аар төлөвлөгөөг middleware-ээс татаж appdb-ийн plan хүснэгтэд хадгалаад чөлөөлөлт үүсгэнэ.
// @Description  Чөлөөлөлтийн ХИЛ нь олдсон ТӨЛӨВЛӨГӨӨНИЙ хилээс ХУУЛАГДАНА — гараас shapefile оруулахгүй.
// @Description  Төлөвлөгөөнд хил байхгүй бол 422-оор бүртгэгдэхгүй.
// @Description  Талбайн хэмжээ (area_m2) нь тэр хилээс АВТОМАТААР тооцогдоно — гараас оруулахгүй.
// @Description  Нэгж талбарыг үүсгэх үед автоматаар ТАТАХГҮЙ (хил шинэчлэх эсвэл sync-ээр татагдана).
// @Description  Хилийн шалгалт (биелэхгүй бол 422-оор бүртгэгдэхгүй):
// @Description  Чөлөөлөх хил нь ЗӨВХӨН нэг сум/дүүрэгт (au2) багтсан байх — дамнасан бол "Сум/дүүрэг дамнах боломжгүй".
// @Tags         land-acquisitions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        plan_parcel_id formData string true  "Төлөвлөгөөний нэгж талбарын дугаар"
// @Param        start_date formData  string  true  "Эхлэх огноо (YYYY-MM-DD)"
// @Param        end_date   formData  string  false  "Дуусах огноо (YYYY-MM-DD)"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      422  {object}  response.Response
// @Router       /land-acquisitions [post]
func (h *LandHandler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxShapefileFormSize)
	if c.ContentType() == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(maxShapefileFormSize); isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
	}

	// Төлөвлөгөөг ТӨЛӨВЛӨГӨӨНИЙ НЭГЖ ТАЛБАРЫН ДУГААРААР хайна — дундын
	// сервисийн /plan/project нь pl_project_parcel.parcel_id-гаар хайдаг.
	// Хуучин `plan_code` талбарыг зөвхөн нийцлийн үүднээс хүлээж авна.
	planParcelID := strings.TrimSpace(c.PostForm("plan_parcel_id"))
	if planParcelID == "" {
		planParcelID = strings.TrimSpace(c.PostForm("plan_code"))
	}
	planCode := planParcelID
	startStr := c.PostForm("start_date")
	endStr := c.PostForm("end_date")
	projectName := c.PostForm("acquisition_name")
	implementingOrg := c.PostForm("implementing_org")
	reason := c.PostForm("reason")
	responsibleOrg := c.PostForm("responsible_org")
	fundingSource := c.PostForm("funding_source")
	nithDecreeNumber := strings.TrimSpace(c.PostForm("nith_decree_number"))
	groupListNumber := strings.TrimSpace(c.PostForm("group_list_number"))

	if planCode == "" || startStr == "" || projectName == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrStartDateFormat)
		return
	}
	var endDate *time.Time
	if endStr != "" {
		t, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrEndDateFormat)
			return
		}
		endDate = &t
	}

	// Хилийн файл ХҮЛЭЭЖ АВАХГҮЙ: чөлөөлөлтийн хил нь олдсон төлөвлөгөөний
	// хилээс хуулагдана (service.Create → planBoundaryWKT).

	createdByStr := contextUserIDString(c)

	var generalCategoryID *int
	if s := c.PostForm("general_category_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			generalCategoryID = &n
		}
	}
	var subCategoryID *int
	if s := c.PostForm("sub_category_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			subCategoryID = &n
		}
	}

	acq, err := h.svc.Create(c.Request.Context(), service.CreateLandAcquisitionRequest{
		PlanCode:          planCode,
		StartDate:         startDate,
		EndDate:           endDate, // nil if not provided
		CreatedBy:         createdByStr,
		AcquisitionName:   projectName,
		ImplementingOrg:   implementingOrg,
		Reason:            reason,
		ResponsibleOrg:    responsibleOrg,
		FundingSource:     fundingSource,
		GeneralCategoryID: generalCategoryID,
		SubCategoryID:     subCategoryID,
		NithDecreeNumber:  nithDecreeNumber,
		GroupListNumber:   groupListNumber,
	})
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			response.Abort(c, http.StatusNotFound, i18n.ErrPlanNotFound)
		case domain.ErrInvalidInput:
			response.Abort(c, http.StatusUnprocessableEntity, i18n.ErrBoundaryExceeds)
		default:
			response.Error(c, err)
		}
		return
	}

	response.Created(c, toLandAcqResponse(acq))
}

// ListParcels godoc
// @Summary      Чөлөөлөлтийн нэгж талбарын жагсаалт
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id         path   string  true   "LandAcquisition UUID"
// @Param        page       query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size  query  int     false  "Хуудасны хэмжээ (default: 20)"
// @Param        parcel_id  query  string  false  "Нэгж талбарын дугаар (ILIKE)"
// @Param        au1_code   query  string  false  "Аймгийн код"
// @Param        au2_code   query  string  false  "Сумын код"
// @Param        au3_code   query  string  false  "Багийн код"
// @Param        right_type query  string  false  "Эрхийн төрөл (OWNERSHIP, LEASE, ...)"
// @Param        landuse    query  string  false  "Газрын зориулалт"
// @Success      200  {object}  response.PaginatedResponse
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels [get]
func (h *LandHandler) ListParcels(c *gin.Context) {
	h.listParcels(c, false)
}

// ListEvaluationParcels — мэргэжлийн байгууллагад зөвхөн «Үнэлгээний шат»
// (parcel_status.is_valuation_stage) төлөвтэй нэгж талбар.
func (h *LandHandler) ListEvaluationParcels(c *gin.Context) {
	h.listParcels(c, true)
}

func (h *LandHandler) listParcels(c *gin.Context, valuationStageOnly bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	statusID := queryInt(c, "status_id", 0)
	if valuationStageOnly {
		// Клиентийн төлөвийн шүүлтийг үл хэрэгсэнэ — өмнөх (2-оор албадах) шиг.
		statusID = 0
	}

	filter := domain.ParcelFilter{
		ParcelID:           c.Query("parcel_id"),
		AU1Code:            c.Query("au1_code"),
		AU2Code:            c.Query("au2_code"),
		AU3Code:            c.Query("au3_code"),
		RightType:          queryInt(c, "right_type", 0),
		Landuse:            c.Query("landuse"),
		StatusID:           statusID,
		ValuationStageOnly: valuationStageOnly,
		SortBy:             c.Query("sort"),
		SortDir:            c.Query("order"),
		Page:               page,
		PageSize:           pageSize,
	}

	// БАЙРШЛЫН давхардлын шүүлтүүр: "1"/"true" → зөвхөн давхардалтай,
	// "0"/"false" → зөвхөн давхардалгүй, заагаагүй → бүгд.
	if raw := strings.TrimSpace(c.Query("has_overlap")); raw != "" {
		switch strings.ToLower(raw) {
		case "1", "true":
			v := true
			filter.HasOverlap = &v
		case "0", "false":
			v := false
			filter.HasOverlap = &v
		}
	}

	// Үнэлгээний байгууллага: чөлөөлөлтийн үндсэн байгууллага бол бүх нэгж
	// талбар, зөвхөн хөндлөнгийн үнэлгээчин бол ӨӨРТ холбогдсон нэгж талбар л
	// харагдана. Гишүүнчлэлийн middleware нь чөлөөлөлтөд хандахыг зөвшөөрдөг ч
	// ЖАГСААЛТЫГ хязгаарладаггүй тул шүүлтийг энд тавина.
	if middleware.IsValuationOrgCaller(c) {
		orgID, ok := middleware.GetCallerOrgID(c)
		if !ok {
			response.Abort(c, http.StatusForbidden, i18n.ErrForbidden)
			return
		}
		filter.ProfessionalOrgID = strconv.FormatInt(int64(orgID), 10)
	}

	parcels, total, err := h.svc.GetParcels(c.Request.Context(), id, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]parcelResponse, 0, len(parcels))
	for _, p := range parcels {
		result = append(result, toParcelResponse(p))
	}
	response.Paginated(c, result, total, page, pageSize)
}

// readBoundaryShapefile — хүсэлтээс хилийн .shp файлыг уншина.
//
// Буцаах: (файлын байт, үргэлжлүүлэх эсэх). Файл огт өгөөгүй бол (nil, true) —
// хил хөндөгдөхгүй ердийн засвар. ok=false үед хариу аль хэдийн бичигдсэн тул
// дуудагч ШУУД буцах ёстой.
func readBoundaryShapefile(c *gin.Context) ([]byte, bool) {
	fh, err := c.FormFile("shapefile")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return nil, false
		}
		return nil, true // файл өгөөгүй
	}
	if fh.Size > maxShapefileSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return nil, false
	}
	data, err := readMultipartFile(fh, maxShapefileSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return nil, false
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileRead)
		return nil, false
	}
	return data, true
}

// PreviewBoundary godoc
// @Summary      Гараас оруулах хилийг ХАДГАЛАЛГҮЙ шалгах
// @Description  .shp файлыг уншиж, хилийн WKT ба талбайн зөрүүг буцаана — хадгалахаас өмнө газрын зураг дээр харах, зөрүүг тоогоор мэдэхэд.
// @Description  Талбай нь нэгж талбартай ИЖИЛ аргачлалаар (public.calculate_area_utm) бодогдоно. Зөрүү хязгаараас хэтэрсэн ч энд алдаа буцахгүй — accepted=false болно.
// @Tags         land-acquisitions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      string  true  "LandAcquisition UUID"
// @Param        shapefile  formData  file    true  "Хилийн .shp файл"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      422  {object}  response.Response
// @Router       /land-acquisitions/{id}/boundary-preview [post]
func (h *LandHandler) PreviewBoundary(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxShapefileFormSize)
	if c.ContentType() == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(maxShapefileFormSize); isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
	}

	data, ok := readBoundaryShapefile(c)
	if !ok {
		return
	}
	if len(data) == 0 {
		response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileRequired)
		return
	}

	check, err := h.svc.PreviewBoundaryShapefile(c.Request.Context(), id, data)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, boundaryPreviewResponse{
		GeometryWKT:         check.GeometryWKT,
		AreaM2:              check.AreaM2,
		ReferenceAreaM2:     check.ReferenceAreaM2,
		DeviationPercent:    check.DeviationRatio * 100,
		MaxDeviationPercent: check.MaxDeviationRatio * 100,
		Accepted:            check.Accepted,
	})
}

// Update godoc
// @Summary      Газар чөлөөлөлтийн мэдээлэл засах
// @Description  ХИЛ солих ХОЁР зам (хоёуланг зэрэг өгвөл 400):
// @Description  1) plan_parcel_id — тэр дугаараар төлөвлөгөөг дахин хайж, олдсон ТӨЛӨВЛӨГӨӨНИЙ хилийг шинэ хил болгоно.
// @Description  2) shapefile — гараас (.shp) оруулсан хилээр солино. Талбайн зөрүү нь төлөвлөгөөний хилийн талбайгаас 30%-иас БАГА байх ёстой (үгүй бол 422). Төлөвлөгөөний мэдээлэл хөндөгдөхгүй.
// @Description  Хил өөрчлөгдвөл шинэ хилтэй давхцах нэгж талбарууд нэмэгдэж, давхцахаа больсон нь устана.
// @Description  Шинэ хил дээр Create-тай ИЖИЛ шалгалт хийгдэнэ: нэг сум/дүүрэгт багтах.
// @Tags         land-acquisitions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      string  true   "LandAcquisition UUID"
// @Param        plan_parcel_id formData string false "Төлөвлөгөөний нэгж талбарын дугаар — өгвөл хил тэр төлөвлөгөөнөөс дахин хуулагдана"
// @Param        shapefile  formData  file    false  "Шинэ хилийн .shp файл (гараас хил засах)"
// @Param        start_date formData  string  false  "Эхлэх огноо (YYYY-MM-DD)"
// @Param        end_date   formData  string  false  "Дуусах огноо (YYYY-MM-DD)"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      422  {object}  response.Response
// @Router       /land-acquisitions/{id} [put]
func (h *LandHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxShapefileFormSize)
	if c.ContentType() == "multipart/form-data" {
		if err := c.Request.ParseMultipartForm(maxShapefileFormSize); isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
	}

	req := service.UpdateLandAcquisitionRequest{}

	// Хил солих 1-р зам: төлөвлөгөөний НЭГЖ ТАЛБАРЫН ДУГААР. Утга өгөгдвөл
	// service нь тэр дугаараар төлөвлөгөөг дахин хайж, олдсон төлөвлөгөөний
	// хилээр чөлөөлөлтийн хилийг солино. Хуучин `plan_code` талбарыг зөвхөн
	// нийцлийн үүднээс хүлээж авна.
	req.PlanCode = strings.TrimSpace(c.PostForm("plan_parcel_id"))
	if req.PlanCode == "" {
		req.PlanCode = strings.TrimSpace(c.PostForm("plan_code"))
	}

	// Хил солих 2-р зам: ГАРААС оруулсан .shp файл. Файлыг задлах/шалгах ажил
	// service дээр (shapefile.Parse + талбайн зөрүү) — handler зөвхөн байтыг
	// уншина. Хоёр замыг зэрэг өгсөн эсэхийг service шийднэ
	// (domain.ErrBoundarySourceConflict).
	if data, ok := readBoundaryShapefile(c); ok {
		req.ShapefileData = data
	} else {
		return // алдааны хариу readBoundaryShapefile дотор бичигдсэн
	}

	if s := c.PostForm("start_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrStartDateFormat)
			return
		}
		req.StartDate = &t
	}
	if s := c.PostForm("end_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrEndDateFormat)
			return
		}
		req.EndDate = &t
	}

	req.AcquisitionName = c.PostForm("acquisition_name")
	if req.AcquisitionName == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	req.ImplementingOrg = c.PostForm("implementing_org")
	req.Reason = c.PostForm("reason")
	req.ResponsibleOrg = c.PostForm("responsible_org")
	req.FundingSource = c.PostForm("funding_source")
	req.NithDecreeNumber = strings.TrimSpace(c.PostForm("nith_decree_number"))
	req.GroupListNumber = strings.TrimSpace(c.PostForm("group_list_number"))
	req.ClearEndDate = c.PostForm("clear_end_date") == "true"
	if s := c.PostForm("general_category_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			req.GeneralCategoryID = &n
		}
	}
	if s := c.PostForm("sub_category_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			req.SubCategoryID = &n
		}
	}
	if s := c.PostForm("area_m2"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			req.AreaM2 = &v
		}
	}

	req.UpdatedBy = contextUserIDString(c)

	acq, boundaryChange, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			response.Abort(c, http.StatusNotFound, i18n.ErrAcqNotFound)
		case domain.ErrInvalidInput:
			response.Abort(c, http.StatusUnprocessableEntity, i18n.ErrNewBoundaryExceeds)
		default:
			response.Error(c, err)
		}
		return
	}

	l := i18n.Detect(c.GetHeader("Accept-Language"))
	type updateResp struct {
		landAcquisitionResponse
		Warning string `json:"warning,omitempty"`
		// Хил солиход нэгж талбарт гарсан өөрчлөлт. RemovedParcels нь
		// БҮРМӨСӨН устгагдсан (шинэ хилд ороогүй) талбарын тоо.
		AddedParcels   int `json:"added_parcels,omitempty"`
		RemovedParcels int `json:"removed_parcels,omitempty"`
	}
	r := updateResp{landAcquisitionResponse: toLandAcqResponse(acq)}
	if boundaryChange != nil {
		r.Warning = i18n.T(l, i18n.WarnGeomChanged)
		r.AddedParcels = boundaryChange.Added
		r.RemovedParcels = boundaryChange.Removed
	}
	response.OK(c, r)
}

// Delete godoc
// @Summary      Газар чөлөөлөлт устгах
// @Description  Устгалт нь SOFT delete — мөр устахгүй, "устгасан" гэж тэмдэглэгдэж
// @Description  бүх уншилтаас хасагдана (CASCADE-ээр холбоотой 23 хүснэгт цэвэрлэгдэхгүй).
// @Description  Хязгаарлалт: 1) зөвхөн "Шинэ" төлөвтэй (403), 2) "Чөлөөлсөн" төлөвтэй
// @Description  нэгж талбартай бол устгахгүй (409), 3) баталгаажсан захирамжтай
// @Description  холбогдсон бол устгахгүй (409). Хавсралт/дроны файлууд фонд цэвэрлэгдэнэ.
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      204
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /land-acquisitions/{id} [delete]
func (h *LandHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, contextUserIDString(c)); err != nil {
		switch err {
		case domain.ErrNotFound:
			response.Abort(c, http.StatusNotFound, i18n.ErrAcqNotFound)
		case domain.ErrForbidden:
			response.Abort(c, http.StatusForbidden, i18n.ErrDeleteNotNew)
		default:
			response.Error(c, err)
		}
		return
	}
	response.NoContent(c)
}

// GetByID godoc
// @Summary      Газар чөлөөлөлтийн дэлгэрэнгүй
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "UUID"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id} [get]
func (h *LandHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	acq, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	detail := *acq
	detail.Parcels = nil
	r := toLandAcqResponse(&detail)
	// Санхүүжилтийн эх үүсвэрийг дэлгэрэнгүйд хамт буцаана — тусдаа GET API байхгүй
	if sources, err := h.svc.ListFundingSources(c.Request.Context(), id); err == nil {
		r.FundingSources = make([]fundingSourceResponse, 0, len(sources))
		for _, fs := range sources {
			r.FundingSources = append(r.FundingSources, toFundingSourceResponse(fs))
		}
	}
	response.OK(c, r)
}

// DiscoverParcels godoc
// @Summary      Чөлөөлөх хилээр нэгж талбарыг тодорхойлж бүртгэх
// @Description  Чөлөөлөлтийн хилээр дундын сервисийн /parcels/by/acquisition-ыг 100-ийн
// @Description  БАГЦААР дуудаж (250 талбар = 3 хүсэлт), давхцах нэгж талбаруудыг
// @Description  бүртгэлийн баганатайгаа (геометр, талбай, зориулалт, хаяг, эрхийн
// @Description  хугацаа) хамт бүртгэнэ. ЭРХ ЗҮЙН дэлгэрэнгүй (хүсэлт, шийдвэр, гэрээ,
// @Description  улсын бүртгэл, эзэмшигч) ЭНД татагдахгүй — дараа нь дугаар тус бүрээр
// @Description  /land-acquisitions/{id}/parcels/{parcel_id}/sync дуудна.
// @Description  Дугаар бүрийг бүртгэхийн өмнө байгаа эсэхийг шалгах тул давхардал үүсэхгүй.
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      200  {object}  response.Response{data=domain.ParcelDiscoveryResult}
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/by-acquisition [post]
func (h *LandHandler) DiscoverParcels(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}

	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.DiscoverParcels(ctx, acqID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// SyncParcel godoc
// @Summary      Нэгж талбарын дэлгэрэнгүй мэдээллийг дундын сервисээс татах
// @Description  1) /parcel/info/{parcel_id}-аар нэгж талбарын дэлгэрэнгүйг татаж хадгална.
// @Description  2) Дэлгэрэнгүй дээрх app_no-гоор /application/persons дуудаж, ЭЗЭМШИГЧДИЙГ
// @Description  (нэг хүсэлт дээр төрөл өөр олон хүн байж болно) бүгдийг хадгална.
// @Description  Үндсэн хүсэлт гаргагч нь parcel_detail.holder_id-д холбогдоно; бүх эзэмшигч
// @Description  нэгж талбарын дэлгэрэнгүй хариултын holders дотор харагдана.
// @Description  Sync нь ОРЛУУЛАХ үйлдэл: хуучин дэлгэрэнгүй болон эзэмшигчид устгагдаж шинээр
// @Description  үүснэ. Хэрэглэгчийн оруулсан үнэлгээ/олговор/хөрөнгө/төлөв хөндөгдөхгүй.
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын UUID"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/sync [post]
func (h *LandHandler) SyncParcel(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelCode := c.Param("parcel_id")

	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	if err := h.svc.SyncParcel(ctx, acqID, parcelCode); err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, nil, i18n.MsgSynced)
}

func (h *LandHandler) SyncValuation(c *gin.Context) {
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	response.OK(c, gin.H{
		"valuation_zone":    "3Б",
		"base_price_per_ha": 850000,
		"auction_coeff":     1.2,
		"auction_price":     1020000,
		"valuation_date":    "2024-01-20",
	})
}

// GetAssignees godoc
func (h *LandHandler) GetAssignees(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	assignees, err := h.svc.GetAssignees(c.Request.Context(), acqID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, assignees)
}

// SetAssignees godoc
func (h *LandHandler) SetAssignees(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}

	var req struct {
		Users []struct {
			UserID       string `json:"user_id"`
			UserName     string `json:"user_name"`
			UserPosition string `json:"user_position"`
		} `json:"users"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}

	callerID, _ := middleware.GetCallerID(c)
	callerName := middleware.GetCallerName(c)

	assignees := make([]domain.AcquisitionAssignee, 0, len(req.Users))
	for _, u := range req.Users {
		uid, ok := parseID(u.UserID)
		if !ok {
			continue
		}
		assignees = append(assignees, domain.AcquisitionAssignee{
			UserID:         uid,
			UserName:       u.UserName,
			UserPosition:   u.UserPosition,
			AssignedBy:     callerID,
			AssignedByName: callerName,
		})
	}

	if err := h.svc.SetAssignees(c.Request.Context(), acqID, assignees); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, assignees)
}

// GetParcelByID godoc
// @Summary      Нэгж талбарын дэлгэрэнгүй мэдээлэл
// @Description  Парцелийн геометр, чөлөөлөлтийн давхцал, эрх эзэмшигчийн мэдээлэл
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын UUID"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id} [get]
func (h *LandHandler) GetParcelByID(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}

	pf, err := h.svc.GetParcelDetail(c.Request.Context(), acqID, parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toParcelFullResponse(pf))
}

// UpdateParcelValuation godoc
// @Summary      Нэгж талбарын үнэлгээний мэдээлэл шинэчлэх
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/valuation [patch]
func (h *LandHandler) UpdateParcelValuation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}
	// parcel тухайн чөлөөлөлтөд харьяалагдаж буйг шалгана (IDOR).
	if err := h.svc.EnsureParcelInAcquisition(c.Request.Context(), acqID, parcelID); err != nil {
		response.Error(c, err)
		return
	}

	var body struct {
		ValuationZone  string   `json:"valuation_zone"`
		BasePricePerHa *float64 `json:"base_price_per_ha"`
		AuctionCoeff   *float64 `json:"auction_coeff"`
		AuctionPrice   *float64 `json:"auction_price"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}

	if err := h.svc.UpdateParcelValuation(c.Request.Context(), parcelID, body.ValuationZone, body.BasePricePerHa, body.AuctionCoeff, body.AuctionPrice); err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, nil, "Амжилттай шинэчлэгдлээ")
}

// SetParcelEstimatedValue godoc
//
//	@Summary		Нэгж талбарын ТӨСӨӨЛЛИЙН үнэлгээ хадгалах
//	@Description	value өгөөгүй бол хадгалсан суурь төлбөр, өртсөн талбайгаар сервер тооцно (confidence_percent — итгэлцүүр; ӨМЧЛӨХ эрхийн газарт хэрэглэгдэхгүй). value=null бол үнэлгээг арилгана.
//	@Tags			land-acquisitions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"LandAcquisition UUID"
//	@Param			parcel_id	path		string	true	"Нэгж талбарын UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response
//	@Router			/land-acquisitions/{id}/parcels/{parcel_id}/estimated-value [patch]
func (h *LandHandler) SetParcelEstimatedValue(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}

	var body struct {
		Value             json.RawMessage `json:"value"`
		ConfidencePercent *float64        `json:"confidence_percent"`
		BaseFeePerM2      *float64        `json:"base_fee_per_m2"`
	}
	// ХОЁР хэлбэрийг дэмжинэ:
	//   application/json    — өмнөх шигээ (хавсралтгүй)
	//   multipart/form-data — confidence_percent/base_fee_per_m2 + ЗААВАЛ БИШ `file`
	//
	// Хавсралт нь тооцооллыг гэрчилдэг (жишиг үнийн лавлагаа) тул үнэлгээтэй
	// НЭГ хүсэлтэд явна: тусад нь байвал үнэлгээ хадгалагдаад файл унах
	// (эсвэл эсрэгээр) болно.
	var attachment *service.EstimatedValueAttachment
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)
		if err := c.Request.ParseMultipartForm(maxDocumentUploadSize); err != nil {
			if isBodyTooLarge(err) {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
				return
			}
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
			return
		}
		// Хоосон талбар = "огт өгөөгүй" (JSON дээрх nil-тэй ижил).
		if v, ok := formFloat(c, "confidence_percent"); ok {
			body.ConfidencePercent = &v
		} else if c.PostForm("confidence_percent") != "" {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
			return
		}
		if v, ok := formFloat(c, "base_fee_per_m2"); ok {
			body.BaseFeePerM2 = &v
		} else if c.PostForm("base_fee_per_m2") != "" {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
			return
		}
		if fh, err := c.FormFile("file"); err == nil {
			if fh.Size > maxDocumentUploadSize {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
				return
			}
			data, err := readMultipartFile(fh, maxDocumentUploadSize)
			if err == errUploadTooLarge {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
				return
			}
			if err != nil {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
				return
			}
			contentType, ok := allowedEstimatedValueContentType(data)
			if !ok {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
				return
			}
			attachment = &service.EstimatedValueAttachment{
				Filename:    fh.Filename,
				ContentType: contentType,
				Data:        data,
			}
		}
	} else if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}

	userID := contextUserIDString(c)
	// `value` талбар ОГТ ирээгүй бол ТООЦООЛОХ салаа. Итгэлцүүр өгөөгүй байж
	// болно: ӨМЧЛӨХ эрхийн газарт итгэлцүүр хэрэглэдэггүй (service нь эрхийн
	// төрлөөр шийднэ). Арилгах/шууд бичих нь `{"value": ...}`-аар ирдэг тул
	// хоёр салаа хоорондоо хутгалдахгүй.
	if len(body.Value) == 0 {
		value, err := h.svc.CalculateParcelEstimatedValue(c.Request.Context(), acqID, parcelID, body.ConfidencePercent, body.BaseFeePerM2, attachment, userID)
		if err != nil {
			response.Error(c, err)
			return
		}
		response.OKMsg(c, gin.H{"estimated_value": value, "estimated_confidence_percent": body.ConfidencePercent}, "Амжилттай хадгаллаа")
		return
	}
	var value *float64
	// Арилгах/шууд бичихэд хавсралт утгагүй (файл нь тооцооллын гэрчилгээ).
	if attachment != nil || body.ConfidencePercent != nil || body.BaseFeePerM2 != nil || json.Unmarshal(body.Value, &value) != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}
	if err := h.svc.SetParcelEstimatedValue(c.Request.Context(), acqID, parcelID, value, userID); err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, gin.H{"estimated_value": value, "estimated_confidence_percent": nil}, "Амжилттай хадгаллаа")
}

// formFloat — multipart талбарыг тоо болгож уншина. ok=false бол талбар
// хоосон/байхгүй (эсвэл тоо биш) — дуудагч тэр хоёрыг ялган шалгана.
func formFloat(c *gin.Context, name string) (float64, bool) {
	raw := strings.TrimSpace(c.PostForm(name))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// ListParcelEstimatedValueHistory godoc
//
//	@Summary		Нэгж талбарын ТӨСӨӨЛЛИЙН үнэлгээний өөрчлөлтийн түүх
//	@Description	Өөрчлөлт бүр (тооцоолол ба арилгалт) хавсралттайгаа, шинэ мөрөөс хуучин руу
//	@Tags			land-acquisitions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"LandAcquisition UUID"
//	@Param			parcel_id	path		string	true	"Нэгж талбарын UUID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response
//	@Router			/land-acquisitions/{id}/parcels/{parcel_id}/estimated-value/history [get]
func (h *LandHandler) ListParcelEstimatedValueHistory(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}
	list, err := h.svc.ListParcelEstimatedValueHistory(c.Request.Context(), acqID, parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// ListProgress godoc
// @Summary      Статус өөрчлөлтийн түүх
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id}/progress [get]
func (h *LandHandler) ListProgress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	list, err := h.svc.GetProgress(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]progressResponse, 0, len(list))
	for _, p := range list {
		result = append(result, progressResponse{
			ID:         p.ID.String(),
			FromStatus: p.FromStatus,
			ToStatus:   p.ToStatus,
			Note:       p.Note,
			ChangedBy:  p.ChangedBy,
			ChangedAt:  p.ChangedAt.Format(time.RFC3339),
		})
	}
	response.OK(c, result)
}

// ListBoundaryHistory godoc
// @Summary      Чөлөөлөлтийн хилийн өөрчлөлтийн түүх
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/boundary-history [get]
func (h *LandHandler) ListBoundaryHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	list, err := h.svc.ListBoundaryHistory(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]boundaryHistoryResponse, 0, len(list))
	for _, item := range list {
		result = append(result, boundaryHistoryResponse{
			ID:                item.ID.String(),
			LandAcquisitionID: item.LandAcquisitionID.String(),
			OldGeometryWKT:    item.OldGeometryWKT,
			NewGeometryWKT:    item.NewGeometryWKT,
			OldAreaM2:         item.OldAreaM2,
			NewAreaM2:         item.NewAreaM2,
			ChangedBy:         item.ChangedBy,
			ChangedAt:         item.ChangedAt.Format(time.RFC3339),
		})
	}
	response.OK(c, result)
}

// GetAvailableStatuses godoc
// @Summary      Шилжих боломжтой статусуудын жагсаалт
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id}/available-statuses [get]
func (h *LandHandler) GetAvailableStatuses(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	opts, err := h.svc.GetAvailableStatuses(c.Request.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			response.Abort(c, http.StatusNotFound, i18n.ErrAcqNotFound)
		default:
			response.Error(c, err)
		}
		return
	}
	response.OK(c, opts)
}

// AdvanceStatus godoc
// @Summary      Сонгосон статус руу шилжүүлэх
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string  true  "LandAcquisition UUID"
// @Param        body  body  object  true  "to_status, note"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      422  {object}  response.Response
// @Router       /land-acquisitions/{id}/advance [post]
func (h *LandHandler) AdvanceStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		ToStatus     int     `json:"to_status" binding:"required"`
		Note         string  `json:"note"`
		DecreeNumber string  `json:"decree_number"`
		DecreeDate   *string `json:"decree_date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrStatusRequired)
		return
	}

	changedBy := contextUserDisplayName(c)

	var decreeDate *time.Time
	if body.DecreeDate != nil && *body.DecreeDate != "" {
		t, err := time.Parse("2006-01-02", *body.DecreeDate)
		if err == nil {
			decreeDate = &t
		}
	}

	acq, err := h.svc.AdvanceStatus(c.Request.Context(), id, body.ToStatus, body.Note, changedBy, body.DecreeNumber, decreeDate)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			response.Abort(c, http.StatusNotFound, i18n.ErrAcqNotFound)
		case domain.ErrInvalidInput:
			response.Abort(c, http.StatusUnprocessableEntity, i18n.ErrInvalidStatus)
		case domain.ErrDecreeRequired:
			response.Abort(c, http.StatusUnprocessableEntity, "НЗД-ын захирамжийн огноо, дугаарыг оруулна уу")
		default:
			response.Error(c, err)
		}
		return
	}
	response.OK(c, toLandAcqResponse(acq))
}

// UpdateParcelMeta — нэгж талбарын чөлөөлөгдөх талбай/хил болон эх системийн
// өөрчлөлтийн тэмдэглэгээг шинэчилнэ.
//
// acquisition_geom_wkt / acquisition_area_m2 оруулсан тохиолдолд шинэ хил нь
// нэгж талбарын ҮНДСЭН хил дотор байх ёстой — гарсан бол 422-оор
// татгалзаж, өөрчлөлт хадгалагдахгүй.
func (h *LandHandler) UpdateParcelMeta(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	// parcel тухайн чөлөөлөлтөд харьяалагдаж буйг шалгана (IDOR).
	if err := h.svc.EnsureParcelInAcquisition(c.Request.Context(), acqID, parcelID); err != nil {
		response.Error(c, err)
		return
	}

	var body struct {
		DbChanged          string   `json:"db_changed"`
		ChangedParcelID    string   `json:"changed_parcel_id"`
		AcquisitionAreaM2  *float64 `json:"acquisition_area_m2"`
		AcquisitionGeomWKT *string  `json:"acquisition_geom_wkt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, "invalid body")
		return
	}

	if err := h.svc.UpdateParcelMeta(c.Request.Context(), parcelID, body.DbChanged, body.ChangedParcelID, body.AcquisitionAreaM2, body.AcquisitionGeomWKT); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ComputeGeometryArea — WKT хилийн талбайг тооцоод буцаана.
//
// ЯАГААД СЕРВЕР ДЭЭР: талбайг ГУС-тай ИЖИЛ аргаар (UTM проекц, бүхэл м²)
// бодох ёстой. Browser дээр ойролцоо тооцвол ГУС-ийн болон нэгж талбар
// татахад бодогдсон тооноос зөрдөг байсан.
//
// Хил өөрөө нууц мэдээлэл биш бөгөөд дуудагч нь хилээ өөрөө явуулж байгаа тул
// нэмэлт эрх шалгах зүйлгүй — зөвхөн нэвтэрсэн байхыг router шаардана.
func (h *LandHandler) ComputeGeometryArea(c *gin.Context) {
	var body struct {
		GeomWKT string `json:"geom_wkt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, "invalid body")
		return
	}

	areaM2, err := h.svc.ComputeAreaM2FromWKT(c.Request.Context(), body.GeomWKT)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"area_m2": areaM2})
}

func (h *LandHandler) ListAcquisitionDocuments(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	list, err := h.svc.ListAcquisitionDocuments(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]documentResponse, 0, len(list))
	for _, d := range list {
		result = append(result, toDocumentResponse(d))
	}
	response.OK(c, result)
}

func (h *LandHandler) UploadAcquisitionDocument(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)
	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxDocumentUploadSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	uploadedBy := contextUserIDString(c)
	documentTypeID, err := documentTypeIDFromForm(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	// Хавсралт ерөнхийдөө зөвхөн PDF өргөтгөлтэй байна; "Хурлын тэмдэглэл" төрөлд
	// Гэрээтэй нэгтгэж хэвлэдэг тул DOCX-ийг нэмж зөвшөөрнө.
	dt, err := documentTypeForUpload(c, h.svc, documentTypeID)
	if err != nil {
		response.Error(c, err)
		return
	}
	contentType, ok := allowedDocumentContentType(fh.Filename, data, dt)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
		return
	}
	// filename нь өргөтгөл тодорхойлоход, name нь дэлгэцийн нэрэнд ашиглагдана
	doc, err := h.svc.UploadAcquisitionDocument(c.Request.Context(), id, fh.Filename, c.PostForm("name"), contentType, data, uploadedBy, documentTypeID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toDocumentResponse(doc))
}

// GetAcquisitionSocioSurvey godoc
//
//	@Summary		Нийгэм, эдийн засгийн судалгааны бүртгэл
//	@Description	Чөлөөлөлтөд бүртгэсэн судалгааны дүн (зөвшөөрсөн/татгалзсан тоо, тайлбар, PDF). Бүртгэл ороогүй бол data=null.
//	@Tags			land-acquisitions
//	@Produce		json
//	@Param			id	path		string	true	"Газар чөлөөлөлтийн ID"
//	@Success		200	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/land-acquisitions/{id}/socio-survey [get]
func (h *LandHandler) GetAcquisitionSocioSurvey(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	survey, err := h.svc.GetAcquisitionSocioSurvey(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	// Бүртгэл ороогүй нь алдаа биш — data=null.
	if survey == nil {
		response.OK(c, nil)
		return
	}
	response.OK(c, toSocioSurveyResponse(survey))
}

// SaveAcquisitionSocioSurvey godoc
//
//	@Summary		Нийгэм, эдийн засгийн судалгааны мэдээлэл оруулах
//	@Description	Зөвшөөрсөн/татгалзсан тоо, тайлбар ба судалгааны PDF-ийг хадгална. Чөлөөлөлтөд НЭГ бүртгэл байх ба дахин оруулахад орлогдоно. Файл нь ЗӨВХӨН PDF; олон файлыг frontend нэг PDF болгож нэгтгэж илгээнэ.
//	@Tags			land-acquisitions
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id				path		string	true	"Газар чөлөөлөлтийн ID"
//	@Param			agreed_count	formData	int		true	"Зөвшөөрсөн тоо"
//	@Param			rejected_count	formData	int		true	"Татгалзсан тоо"
//	@Param			note			formData	string	false	"Тайлбар"
//	@Param			file			formData	file	true	"Судалгааны файл (PDF)"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response
//	@Failure		404				{object}	response.Response
//	@Security		BearerAuth
//	@Router			/land-acquisitions/{id}/socio-survey [post]
func (h *LandHandler) SaveAcquisitionSocioSurvey(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)

	in := service.SocioSurveyInput{
		Note:    c.PostForm("note"),
		SavedBy: contextUserIDString(c),
	}
	if in.AgreedCount, err = nonNegativeFormInt(c, "agreed_count"); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}
	if in.RejectedCount, err = nonNegativeFormInt(c, "rejected_count"); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}

	// Файл нь ЗААВАЛ — зөвхөн ХАДГАЛАГДСАН бүртгэл дээр тоо/тайлбар засахад
	// файлгүй илгээж болно (тэр шалгалт service дээр).
	fh, fileErr := c.FormFile("file")
	if fileErr != nil && isBodyTooLarge(fileErr) {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if fileErr == nil {
		if fh.Size > maxDocumentUploadSize {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		data, readErr := readMultipartFile(fh, maxDocumentUploadSize)
		if readErr == errUploadTooLarge {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		if readErr != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
			return
		}
		contentType, ok := allowedPDFContentType(data)
		if !ok {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
			return
		}
		in.FileName = fh.Filename
		in.ContentType = contentType
		in.FileData = data
	}

	survey, err := h.svc.SaveAcquisitionSocioSurvey(c.Request.Context(), id, in)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, toSocioSurveyResponse(survey), i18n.MsgUpdated)
}

// ListParcelNoticeEmails godoc
//
//	@Summary		Мэдэгдэх хуудас илгээсэн түүх
//	@Description	Нэгж талбар дээр имэйлээр илгээсэн урьдчилан мэдэгдэх хуудсуудын түүх (шинэ нь эхэнд, огноотой).
//	@Tags			parcels
//	@Produce		json
//	@Param			id	path		string	true	"Нэгж талбарын ID (UUID)"
//	@Success		200	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/notice-emails [get]
func (h *LandHandler) ListParcelNoticeEmails(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}
	list, err := h.svc.ListParcelNoticeEmails(c.Request.Context(), parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]noticeEmailResponse, 0, len(list))
	for _, m := range list {
		result = append(result, toNoticeEmailResponse(m))
	}
	response.OK(c, result)
}

// SendParcelNoticeEmail godoc
//
//	@Summary		Мэдэгдэх хуудсыг имэйлээр илгээх
//	@Description	Урьдчилан мэдэгдэх хуудсын PDF-ийг хавсаргаж иргэний имэйл руу илгээнэ. Илгээлт бүр (амжилтгүй нь ч) түүхэд бүртгэгдэнэ.
//	@Tags			parcels
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id		path		string	true	"Нэгж талбарын ID (UUID)"
//	@Param			to		formData	string	true	"Хүлээн авагчийн имэйл"
//	@Param			file	formData	file	true	"Мэдэгдэх хуудас (PDF)"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		502		{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/notice-emails [post]
func (h *LandHandler) SendParcelNoticeEmail(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)

	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxDocumentUploadSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	// Мэдэгдэл нь ЗӨВХӨН PDF-ээр хүргэгдэнэ (загвараас үүсгэсэн файл).
	if _, ok := allowedPDFContentType(data); !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
		return
	}

	res, err := h.svc.SendParcelNoticeEmail(c.Request.Context(), parcelID, service.NoticeEmailInput{
		ToEmail:  c.PostForm("to"),
		FileName: fh.Filename,
		FileData: data,
		SentBy:   contextUserIDString(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toNoticeEmailResponse(*res))
}

// ListParcelEMongoliaNotices godoc
//
//	@Summary		E-Mongolia-гаар хүргүүлсэн мэдэгдлийн түүх
//	@Tags			parcels
//	@Produce		json
//	@Param			id	path		string	true	"Нэгж талбарын ID (UUID)"
//	@Success		200	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/emongolia-notices [get]
func (h *LandHandler) ListParcelEMongoliaNotices(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}
	list, err := h.svc.ListParcelEMongoliaNotices(c.Request.Context(), parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]emongoliaNoticeResponse, 0, len(list))
	for _, m := range list {
		result = append(result, toEMongoliaNoticeResponse(m))
	}
	response.OK(c, result)
}

// SendParcelEMongoliaNotice godoc
//
//	@Summary		Мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлэх
//	@Description	Мэдэгдэх хуудсын PDF-ийг иргэний регистрийн дугаараар E-Mongolia-руу илгээнэ. Гадаад холболт одоогоор МОК тул түүхэд status="mocked" гэж бүртгэгдэнэ.
//	@Tags			parcels
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id				path		string	true	"Нэгж талбарын ID (UUID)"
//	@Param			register_no		formData	string	true	"Хүлээн авагчийн регистрийн дугаар"
//	@Param			recipient_name	formData	string	false	"Хүлээн авагчийн нэр"
//	@Param			file			formData	file	true	"Мэдэгдэх хуудас (PDF)"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response
//	@Failure		502				{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/emongolia-notices [post]
func (h *LandHandler) SendParcelEMongoliaNotice(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)

	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxDocumentUploadSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	// Мэдэгдэл нь ЗӨВХӨН PDF-ээр хүргэгдэнэ (загвараас үүсгэсэн файл).
	if _, ok := allowedPDFContentType(data); !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
		return
	}

	res, err := h.svc.SendParcelEMongoliaNotice(c.Request.Context(), parcelID, service.EMongoliaNoticeInput{
		RegisterNo:    c.PostForm("register_no"),
		RecipientName: c.PostForm("recipient_name"),
		FileName:      fh.Filename,
		FileData:      data,
		SentBy:        contextUserIDString(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toEMongoliaNoticeResponse(*res))
}

// nonNegativeFormInt — multipart формын бүхэл тоо. Хоосон бол 0.
func nonNegativeFormInt(c *gin.Context, field string) (int, error) {
	raw := strings.TrimSpace(c.PostForm(field))
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 0, domain.ErrInvalidInput
	}
	return v, nil
}

func (h *LandHandler) DeleteAcquisitionDocument(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteAcquisitionDocument(c.Request.Context(), acqID, docID); err != nil {
		if err == domain.ErrNotFound {
			response.Abort(c, http.StatusNotFound, i18n.ErrDocNotFound)
			return
		}
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListGlobalParcels godoc
// @Summary      Бүх нэгж талбарын нэгдсэн жагсаалт
// @Tags         parcels
// @Produce      json
// @Security     BearerAuth
// @Param        page              query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size         query  int     false  "Хуудасны хэмжээ (default: 20)"
// @Param        parcel_id         query  string  false  "Нэгж талбарын дугаар (ILIKE)"
// @Param        right_type        query  int     false  "Эрхийн төрөл"
// @Param        landuse           query  string  false  "Газрын зориулалт"
// @Param        au3_code          query  string  false  "Багийн код"
// @Param        acquisition_name  query  string  false  "Чөлөөлөлтийн нэр (ILIKE)"
// @Param        status            query  int     false  "Чөлөөлөлтийн статус"
// @Param        status_id         query  int     false  "Нэгж талбарын төлөв"
// @Param        unlinked_only     query  bool    false  "Идэвхтэй захирамжид холбогдоогүй нэгж талбар"
// @Param        sort              query  string  false  "Эрэмбэлэх багана (parcel_id, acquisition_name, area_m2, compensation…)"
// @Param        order             query  string  false  "Эрэмбийн чиглэл: asc | desc"
// @Success      200  {object}  response.PaginatedResponse
// @Router       /parcels [get]
func (h *LandHandler) ListGlobalParcels(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	var years []int
	for _, s := range c.QueryArray("year") {
		if y, err := strconv.Atoi(s); err == nil && y > 0 {
			years = append(years, y)
		}
	}
	filter := domain.GlobalParcelFilter{
		ParcelID:        c.Query("parcel_id"),
		RightType:       queryInt(c, "right_type", 0),
		Landuse:         c.Query("landuse"),
		AU3Code:         c.Query("au3_code"),
		AcquisitionName: c.Query("acquisition_name"),
		AcquisitionID:   c.Query("acquisition_id"),
		PlanCode:        c.Query("plan_code"),
		Status:          queryInt(c, "status", 0),
		StatusID:        queryInt(c, "status_id", 0),
		UnlinkedOnly:    strings.EqualFold(c.Query("unlinked_only"), "true") || c.Query("unlinked_only") == "1",
		ReleasedOnly:    strings.EqualFold(c.Query("released_only"), "true") || c.Query("released_only") == "1",
		Years:           years,
		SortBy:          c.Query("sort"),
		SortDir:         c.Query("order"),
		Page:            page,
		PageSize:        pageSize,
	}
	parcels, total, err := h.svc.ListGlobalParcels(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]globalParcelResponse, 0, len(parcels))
	for _, p := range parcels {
		result = append(result, toGlobalParcelResponse(p))
	}
	response.Paginated(c, result, total, page, pageSize)
}

// ListReportParcels godoc
// @Summary      Тайлангийн нэгж талбарын жагсаалт
// @Description  Нэгж талбар бүрийн эзэмшигч, нөхөн төлбөр нэгтгэсэн мэдээллийг буцаана
// @Tags         report
// @Produce      json
// @Security     BearerAuth
// @Param        acquisition_id     query  string  false  "Чөлөөлөлтийн ID"
// @Param        plan_code          query  string  false  "Төлөвлөгөөний код (ILIKE)"
// @Param        acquisition_name   query  string  false  "Чөлөөлөлтийн нэр (ILIKE)"
// @Param        au3_code           query  string  false  "Хорооны код"
// @Param        right_type         query  int     false  "Эрхийн төрөл"
// @Param        landuse            query  string  false  "Газрын зориулалт (ILIKE)"
// @Param        year               query  int     false  "Захирамжийн он (decree_date эсвэл start_date)"
// @Param        compensation_type  query  string  false  "Нөхөн төлбөрийн төрөл: cash | land_grant"
// @Param        page               query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size          query  int     false  "Хуудасны хэмжээ (default: 20, max: 200)"
// @Success      200  {object}  response.PaginatedResponse
// @Failure      500  {object}  response.Response
// @Router       /report/download [get]
// reportParcelFilterFromQuery — ListReportParcels/ReportSummary хоёулаа адилхан
// query параметрээр шүүлт бүтээдэг тул нэг дор задална.
func reportParcelFilterFromQuery(c *gin.Context) domain.ReportParcelFilter {
	var years []int
	for _, s := range c.QueryArray("year") {
		if y, err := strconv.Atoi(s); err == nil && y > 0 {
			years = append(years, y)
		}
	}

	return domain.ReportParcelFilter{
		AcquisitionID:     c.Query("acquisition_id"),
		PlanCode:          c.Query("plan_code"),
		AcquisitionName:   c.Query("acquisition_name"),
		AU3Code:           c.Query("au3_code"),
		RightType:         queryInt(c, "right_type", 0),
		Landuse:           c.Query("landuse"),
		Years:             years,
		CompensationType:  c.Query("compensation_type"),
		GeneralCategoryID: queryInt(c, "general_category_id", 0),
		SubCategoryID:     queryInt(c, "sub_category_id", 0),
	}
}

func (h *LandHandler) ListReportParcels(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 200 {
		pageSize = 200
	}

	filter := reportParcelFilterFromQuery(c)
	filter.Page = page
	filter.PageSize = pageSize

	rows, total, err := h.svc.ListReportParcels(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]reportParcelRowResponse, 0, len(rows))
	for _, r := range rows {
		result = append(result, toReportParcelRowResponse(r))
	}
	response.Paginated(c, result, total, page, pageSize)
}

// ReportSummary godoc
// @Summary      Тайлангийн жагсаалтын статистикийн карт (нийт чөлөөлөлт/талбар/талбай/олговор)
// @Tags         report
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /report/summary [get]
func (h *LandHandler) ReportSummary(c *gin.Context) {
	filter := reportParcelFilterFromQuery(c)

	summary, err := h.svc.GetReportSummary(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, toReportSummaryResponse(summary))
}

// ListAssets godoc
// @Summary      Чөлөөлөлтийн хөрөнгийн мэдээллийн жагсаалт
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id         path   string  true   "LandAcquisition UUID"
// @Param        page       query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size  query  int     false  "Хуудасны хэмжээ (default: 20)"
// @Param        parcel_id  query  string  false  "Нэгж талбарын дугаар (ILIKE)"
// @Success      200  {object}  response.PaginatedResponse
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/assets [get]
func (h *LandHandler) ListAssets(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	filter := domain.AssetFilter{
		ParcelID:      c.Query("parcel_id"),
		ValuationType: c.Query("valuation_type"),
		Page:          page,
		PageSize:      pageSize,
	}

	assets, total, err := h.svc.ListAssets(c.Request.Context(), id, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]assetResponse, 0, len(assets))
	for _, a := range assets {
		result = append(result, toAssetResponse(a))
	}
	response.Paginated(c, result, total, page, pageSize)
}

// CreateAsset godoc
// @Summary      Хөрөнгө нэмэх
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string  true  "LandAcquisition UUID"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/assets [post]
// mikaOnlyCaller — дуудагч МИКА роль-той бөгөөд дотоод (ахлах) эрхгүй бол true.
// Ийм дуудагчийн үнэлгээний бичих үйлдэл ЗӨВХӨН "mika" урсгалд хязгаарлагдана —
// valuation_type-ийг хүсэлтээс үл хамааран хатуу тогтооно.
func mikaOnlyCaller(c *gin.Context) bool {
	return ctxHasRole(c, "mika", "МИКА") && !ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн")
}

// profOnlyCaller — дуудагч үнэлгээний байгууллагын ажилтан бөгөөд дотоод
// (ахлах) эрхгүй.
//
// Байгууллагын ажилтныг РОЛЬ-оор БИШ, токены `valuation_org` тэмдгээр
// (appdb.valuation_org_employee) таньдаг: тэдгээр хэрэглэгчид системийн роль
// огт эзэмшихгүй.
func profOnlyCaller(c *gin.Context) bool {
	return middleware.IsValuationOrgCaller(c) &&
		!ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн")
}

// callerValuationStreams — дуудагч УСТГАЖ болох үнэлгээний урсгалууд.
// nil = хязгаарлалтгүй (дотоод ахлах мэргэжилтэн/админ).
//
// Бичих талд valuation_type нь mikaOnlyCaller-ээр хатуу тогтоогддог байсан ч
// УСТГАХ талд ямар ч шалгалт байгаагүй тул МИКА (compensation:update эрхтэй ба
// valuationEditor нь МИКА-г хуваарилалтаас чөлөөлдөг) аль ч чөлөөлөлтөөс
// мэргэжлийн байгууллагын үнэлгээг устгаж чаддаг байв.
func callerValuationStreams(c *gin.Context) []string {
	if mikaOnlyCaller(c) {
		return []string{domain.ValuationTypeMika}
	}
	if profOnlyCaller(c) {
		// Мэргэжлийн байгууллага үндсэн (asset) ба хөндлөнгийн (independent)
		// урсгалд ажиллана — МИКА-гийн урсгалд ХЭЗЭЭ Ч биш.
		return []string{domain.ValuationTypeAsset, domain.ValuationTypeIndependent}
	}
	return nil
}

func (h *LandHandler) CreateAsset(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		ParcelID             string  `json:"parcel_id" binding:"required"`
		AssetNumber          string  `json:"asset_number"`
		AssetType            string  `json:"asset_type"`
		AssetName            string  `json:"asset_name"`
		LegacyBuildingNumber string  `json:"building_number"`
		LegacyBuildingType   string  `json:"building_type"`
		FloorCount           int     `json:"floor_count"`
		AreaM2               float64 `json:"area_m2"`
		OwnerName            string  `json:"owner_name"`
		Address              string  `json:"address"`
		Notes                string  `json:"notes"`
		Unit                 string  `json:"unit"`
		Capacity             string  `json:"capacity"`
		Description          string  `json:"description"`
		UnitPrice            float64 `json:"unit_price"`
		ValuationType        string  `json:"valuation_type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	if body.AssetNumber == "" {
		body.AssetNumber = body.LegacyBuildingNumber
	}
	if body.AssetName == "" {
		body.AssetName = body.LegacyBuildingType
	}
	if mikaOnlyCaller(c) {
		body.ValuationType = domain.ValuationTypeMika
	}

	b := &domain.Asset{
		AcquisitionID: acqID,
		ParcelID:      body.ParcelID,
		ValuationType: body.ValuationType,
		AssetNumber:   body.AssetNumber,
		AssetType:     normalizeAssetRecordType(body.AssetType),
		AssetName:     body.AssetName,
		FloorCount:    body.FloorCount,
		AreaM2:        body.AreaM2,
		OwnerName:     body.OwnerName,
		Address:       body.Address,
		Notes:         body.Notes,
		Unit:          body.Unit,
		Capacity:      body.Capacity,
		Description:   body.Description,
		UnitPrice:     body.UnitPrice,
	}

	created, err := h.svc.CreateAsset(c.Request.Context(), b)
	if err != nil {
		response.Error(c, err)
		return
	}
	r := toAssetResponse(created)
	response.Created(c, r)
}

// UpdateAsset godoc
// @Summary      Хөрөнгийн мэдээлэл засах
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id           path  string  true  "LandAcquisition UUID"
// @Param        asset_id     path  string  true  "Asset UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/assets/{asset_id} [put]
func (h *LandHandler) UpdateAsset(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	// ЗААГЧ (pointer) — талбар ИРСЭН эсэхийг ялгана. Дэлгэц бүр өөрийн хэсгийг
	// л илгээдэг тул ирээгүй талбарыг ХЭВЭЭР үлдээнэ; эс бөгөөс нэр, ангилал,
	// нэгж талбарын дугаар зэрэг нь хоосон утгаар дарагдана (parcel_id хоосон
	// болоход UPDATE нь мөр олохгүй → 404).
	var body struct {
		ParcelID             *string  `json:"parcel_id"`
		AssetNumber          *string  `json:"asset_number"`
		AssetType            *string  `json:"asset_type"`
		AssetName            *string  `json:"asset_name"`
		LegacyBuildingNumber *string  `json:"building_number"`
		LegacyBuildingType   *string  `json:"building_type"`
		FloorCount           *int     `json:"floor_count"`
		AreaM2               *float64 `json:"area_m2"`
		OwnerName            *string  `json:"owner_name"`
		Address              *string  `json:"address"`
		Notes                *string  `json:"notes"`
		Unit                 *string  `json:"unit"`
		Capacity             *string  `json:"capacity"`
		Description          *string  `json:"description"`
		UnitPrice            *float64 `json:"unit_price"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	if body.AssetNumber == nil {
		body.AssetNumber = body.LegacyBuildingNumber
	}
	if body.AssetName == nil {
		body.AssetName = body.LegacyBuildingType
	}
	if body.AssetType != nil {
		normalized := normalizeAssetRecordType(*body.AssetType)
		body.AssetType = &normalized
	}

	b := &domain.AssetUpdate{
		ID:            assetID,
		AcquisitionID: acqID,
		ParcelID:      body.ParcelID,
		AssetNumber:   body.AssetNumber,
		AssetType:     body.AssetType,
		AssetName:     body.AssetName,
		FloorCount:    body.FloorCount,
		AreaM2:        body.AreaM2,
		OwnerName:     body.OwnerName,
		Address:       body.Address,
		Notes:         body.Notes,
		Unit:          body.Unit,
		Capacity:      body.Capacity,
		Description:   body.Description,
		UnitPrice:     body.UnitPrice,
	}

	updated, err := h.svc.UpdateAsset(c.Request.Context(), b)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toAssetResponse(updated))
}

// DeleteAsset godoc
// @Summary      Хөрөнгө устгах
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id           path  string  true  "LandAcquisition UUID"
// @Param        asset_id     path  string  true  "Asset UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/assets/{asset_id} [delete]
func (h *LandHandler) DeleteAsset(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	if err := h.svc.DeleteAsset(c.Request.Context(), acqID, assetID, callerValuationStreams(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListAssetSpecs returns building spec values for an asset
func (h *LandHandler) ListAssetSpecs(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	specs, err := h.svc.ListAssetSpecs(c.Request.Context(), acqID, assetID)
	if err != nil {
		response.OK(c, []assetSpecResponse{})
		return
	}
	out := make([]assetSpecResponse, len(specs))
	for i, s := range specs {
		out[i] = toAssetSpecResponse(s)
	}
	response.OK(c, out)
}

// UpsertAssetSpecs saves building spec values for an asset
func (h *LandHandler) UpsertAssetSpecs(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		Specs []struct {
			SpecTypeID int    `json:"spec_type_id"`
			Value      string `json:"value"`
		} `json:"specs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	specs := make([]domain.AssetSpec, len(body.Specs))
	for i, s := range body.Specs {
		specs[i] = domain.AssetSpec{AssetID: assetID, SpecTypeID: s.SpecTypeID, Value: s.Value}
	}
	if err := h.svc.UpsertAssetSpecs(c.Request.Context(), acqID, assetID, specs); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListAssetCalculations returns calculation values for an asset
func (h *LandHandler) ListAssetCalculations(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	calcs, err := h.svc.ListAssetCalculations(c.Request.Context(), acqID, assetID)
	if err != nil {
		response.OK(c, []assetCalcResponse{})
		return
	}
	out := make([]assetCalcResponse, len(calcs))
	for i, c := range calcs {
		out[i] = toAssetCalcResponse(c)
	}
	response.OK(c, out)
}

// UpsertAssetCalculations saves calculation values for an asset
func (h *LandHandler) UpsertAssetCalculations(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		Calculations []struct {
			CalcTypeID int     `json:"calc_type_id"`
			Unit       string  `json:"unit"`
			Value      float64 `json:"value"`
			// Бүлэг (Итгэлцүүр г.м) — мөр өөрийн бүлгээ авч явна. Хоосон ирвэл
			// хадгалагдсан бүлэг хэвээр үлдэнэ (зөвхөн тоо засахад алдагдахгүй).
			Group string `json:"group"`
		} `json:"calculations"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	calcs := make([]domain.AssetCalculation, len(body.Calculations))
	for i, c := range body.Calculations {
		calcs[i] = domain.AssetCalculation{
			AssetID: assetID, CalcTypeID: c.CalcTypeID, Unit: c.Unit, Value: c.Value, CalcGroup: c.Group,
		}
	}
	if err := h.svc.UpsertAssetCalculations(c.Request.Context(), acqID, assetID, calcs); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// UpsertLandValuation creates or updates the land valuation for a parcel
func (h *LandHandler) UpsertLandValuation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		ParcelID       string  `json:"parcel_id" binding:"required"`
		ValuationType  string  `json:"valuation_type"`
		LandAreaM2     float64 `json:"land_area_m2"`
		BasePricePerM2 float64 `json:"base_price_per_m2"`
		// Excel импортоор ирдэг нэмэлт талбарууд (заавал биш)
		OwnershipCertNo     string `json:"ownership_cert_no"`
		AppraiserOrgName    string `json:"appraiser_org_name"`
		AppraiserRegNo      string `json:"appraiser_reg_no"`
		AppraiserStateRegNo string `json:"appraiser_state_reg_no"`
		AppraiserDirector   string `json:"appraiser_director"`
		AppraiserLicense    string `json:"appraiser_license"`
		AppraiserAddress    string `json:"appraiser_address"`
		AppraiserContact    string `json:"appraiser_contact"`
		SourceFileName      string `json:"source_file_name"`
		SourceFileHash      string `json:"source_file_hash"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	if mikaOnlyCaller(c) {
		body.ValuationType = domain.ValuationTypeMika
	}
	v := &domain.LandValuation{
		AcquisitionID:       acqID,
		ParcelID:            body.ParcelID,
		ValuationType:       body.ValuationType,
		LandAreaM2:          body.LandAreaM2,
		BasePricePerM2:      body.BasePricePerM2,
		OwnershipCertNo:     body.OwnershipCertNo,
		AppraiserOrgName:    body.AppraiserOrgName,
		AppraiserRegNo:      body.AppraiserRegNo,
		AppraiserStateRegNo: body.AppraiserStateRegNo,
		AppraiserDirector:   body.AppraiserDirector,
		AppraiserLicense:    body.AppraiserLicense,
		AppraiserAddress:    body.AppraiserAddress,
		AppraiserContact:    body.AppraiserContact,
		SourceFileName:      body.SourceFileName,
		SourceFileHash:      body.SourceFileHash,
	}
	saved, err := h.svc.UpsertLandValuation(c.Request.Context(), v)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toLandValuationResponse(saved))
}

// GetLandValuation returns the land valuation for a parcel within an acquisition
func (h *LandHandler) GetLandValuation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Query("parcel_id")
	if parcelID == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	v, err := h.svc.GetLandValuation(c.Request.Context(), acqID, parcelID, c.Query("valuation_type"))
	if err != nil {
		response.OK(c, nil)
		return
	}
	response.OK(c, toLandValuationResponse(v))
}

// DeleteLandValuation — нэгж талбарын газрын үнэлгээг устгана.
func (h *LandHandler) DeleteLandValuation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Query("parcel_id")
	if parcelID == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	valuationType := c.Query("valuation_type")
	if mikaOnlyCaller(c) {
		valuationType = domain.ValuationTypeMika
	}
	// Гадаад дуудагч зөвхөн ӨӨРИЙН урсгалыг устгана. valuation_type нь query-ээс
	// ирдэг тул энэ шалгалтгүй бол мэрг. байгууллага `?valuation_type=mika`-аар
	// МИКА-гийн газрын үнэлгээг устгах боломжтой.
	if streams := callerValuationStreams(c); len(streams) > 0 {
		if valuationType == "" {
			// Хоосон = БҮХ урсгал устана — хязгаарлагдсан дуудагчид зөвшөөрөхгүй.
			response.Error(c, domain.ErrForbidden)
			return
		}
		if !domain.ValuationStreamAllowed(valuationType, streams) {
			response.Error(c, domain.ErrForbidden)
			return
		}
	}
	if err := h.svc.DeleteLandValuation(c.Request.Context(), acqID, parcelID, valuationType); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListValuationNotes — нэгж талбарын үнэлгээний хүснэгт бүрийн ТАЙЛБАР.
func (h *LandHandler) ListValuationNotes(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Query("parcel_id")
	if parcelID == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	vType := c.Query("valuation_type")
	if mikaOnlyCaller(c) {
		vType = domain.ValuationTypeMika
	}
	list, err := h.svc.ListValuationNotes(c.Request.Context(), acqID, parcelID, vType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// SaveValuationNotes — хүснэгтийн тайлбаруудыг хадгална (section_key → бичвэр).
// Мапад ороогүй хэсгийн тайлбар хэвээр үлдэнэ, хоосон бичвэр нь тухайн тайлбарыг устгана.
func (h *LandHandler) SaveValuationNotes(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		ParcelID      string            `json:"parcel_id" binding:"required"`
		ValuationType string            `json:"valuation_type"`
		Notes         map[string]string `json:"notes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	if mikaOnlyCaller(c) {
		body.ValuationType = domain.ValuationTypeMika
	}
	// Гадаад дуудагч (мэрг. байгууллага / МИКА) зөвхөн ӨӨРИЙН урсгалд бичнэ —
	// эс бөгөөс `valuation_type`-аар бусдын урсгалын тайлбарыг дарж бичих боломжтой.
	if streams := callerValuationStreams(c); len(streams) > 0 {
		if body.ValuationType == "" {
			body.ValuationType = streams[0]
		}
		if !domain.ValuationStreamAllowed(body.ValuationType, streams) {
			response.Error(c, domain.ErrForbidden)
			return
		}
	}
	// Мэргэжлийн байгууллагын дуудагч зөвхөн ХАРЬЯАЛАГДАХ нэгж талбартаа бичнэ
	// (parcel_id нь body-гоор ирдэг тул middleware шалгаж чадахгүй — IDOR хамгаалалт).
	if middleware.IsValuationOrgCaller(c) {
		orgID, ok := middleware.GetCallerOrgID(c)
		if !ok {
			response.Error(c, domain.ErrForbidden)
			return
		}
		member, err := h.svc.IsProfMemberOfParcelByCode(c.Request.Context(), acqID, body.ParcelID, orgID)
		if err != nil || !member {
			response.Error(c, domain.ErrForbidden)
			return
		}
	}
	updatedBy := ""
	if uid, ok := c.Get("userID"); ok {
		if id, ok2 := uid.(domain.ID); ok2 {
			updatedBy = strconv.FormatInt(int64(id), 10)
		}
	}
	list, err := h.svc.SaveValuationNotes(c.Request.Context(), acqID, body.ParcelID, body.ValuationType, body.Notes, updatedBy)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// ImportValuation — Excel-ээс задлан авсан бүхэл үнэлгээг НЭГ хүсэлтээр (нэг транзакц) оруулна.
func (h *LandHandler) ImportValuation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		ParcelID      string `json:"parcel_id" binding:"required"`
		ValuationType string `json:"valuation_type"`
		Replace       bool   `json:"replace"`
		Land          struct {
			LandAreaM2          float64 `json:"land_area_m2"`
			BasePricePerM2      float64 `json:"base_price_per_m2"`
			OwnershipCertNo     string  `json:"ownership_cert_no"`
			AppraiserOrgName    string  `json:"appraiser_org_name"`
			AppraiserRegNo      string  `json:"appraiser_reg_no"`
			AppraiserStateRegNo string  `json:"appraiser_state_reg_no"`
			AppraiserDirector   string  `json:"appraiser_director"`
			AppraiserLicense    string  `json:"appraiser_license"`
			AppraiserAddress    string  `json:"appraiser_address"`
			AppraiserContact    string  `json:"appraiser_contact"`
			SourceFileName      string  `json:"source_file_name"`
			SourceFileHash      string  `json:"source_file_hash"`
		} `json:"land"`
		// Хүснэгт бүрийн ТАЙЛБАР (section_key → бичвэр). Excel-д байвал ирнэ.
		SectionNotes map[string]string `json:"section_notes"`
		// Хүснэгтийн тайлбар + таних мэдээлэл (дугаар, "Хүснэгт-N", гарчиг, дараалал).
		// Ирвэл section_notes-оос давамгайлна.
		Sections []struct {
			Key        string `json:"key"`
			Note       string `json:"note"`
			SectionNo  string `json:"section_no"`
			TableLabel string `json:"table_label"`
			Title      string `json:"title"`
			SortOrder  int    `json:"sort_order"`
		} `json:"sections"`
		Assets []struct {
			AssetNumber        string  `json:"asset_number"`
			AssetType          string  `json:"asset_type"`
			AssetName          string  `json:"asset_name"`
			AreaM2             float64 `json:"area_m2"`
			FloorCount         int     `json:"floor_count"`
			Unit               string  `json:"unit"`
			Capacity           string  `json:"capacity"`
			Description        string  `json:"description"`
			OwnerName          string  `json:"owner_name"`
			Notes              string  `json:"notes"`
			UnitPrice          float64 `json:"unit_price"`
			CompensationAmount float64 `json:"compensation_amount"`
			Specs              []struct {
				SpecTypeID int    `json:"spec_type_id"`
				Value      string `json:"value"`
			} `json:"specs"`
			Calculations []struct {
				CalcTypeID int     `json:"calc_type_id"`
				Name       string  `json:"name"`  // calc_type_id байхгүй бол нэрээр нь олж/үүсгэнэ
				Group      string  `json:"group"` // бүлгийн нэр (Итгэлцүүр г.м)
				Unit       string  `json:"unit"`
				Value      float64 `json:"value"`
			} `json:"calculations"`
		} `json:"assets"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	createdBy := ""
	var callerID domain.ID
	if uid, ok := c.Get("userID"); ok {
		callerID, _ = uid.(domain.ID)
		createdBy = strconv.FormatInt(int64(callerID), 10)
	}
	if mikaOnlyCaller(c) {
		body.ValuationType = domain.ValuationTypeMika
	}

	// professional_org (үндсэн болон хөндлөнгийн) хэрэглэгч зөвхөн ХАРЬЯАЛАГДАХ
	// нэгж талбарт л үнэлгээ импортлох ёстой. /prof/... маршрут дээр body.ParcelID
	// JSON body-гоор ирдэг (URL param биш) тул profParcelMemberByParam middleware-ээр
	// шалгагдахгүй — энэ шалгалт байхгүй бол acquisition-ийн гишүүн ямар ч parcel_id-д
	// (өөрт нь харьяалагдахгүй байсан ч) бичих боломжтой байсан (IDOR).
	if middleware.IsValuationOrgCaller(c) {
		orgID, ok := middleware.GetCallerOrgID(c)
		if !ok {
			response.Error(c, domain.ErrForbidden)
			return
		}
		member, err := h.svc.IsProfMemberOfParcelByCode(c.Request.Context(), acqID, body.ParcelID, orgID)
		if err != nil || !member {
			response.Error(c, domain.ErrForbidden)
			return
		}
	}

	imp := &domain.ValuationImport{
		ParcelID:      body.ParcelID,
		ValuationType: body.ValuationType,
		Replace:       body.Replace,
		SectionNotes:  body.SectionNotes,
		CreatedBy:     createdBy,
		Land: domain.LandValuation{
			ValuationType:       body.ValuationType,
			LandAreaM2:          body.Land.LandAreaM2,
			BasePricePerM2:      body.Land.BasePricePerM2,
			OwnershipCertNo:     body.Land.OwnershipCertNo,
			AppraiserOrgName:    body.Land.AppraiserOrgName,
			AppraiserRegNo:      body.Land.AppraiserRegNo,
			AppraiserStateRegNo: body.Land.AppraiserStateRegNo,
			AppraiserDirector:   body.Land.AppraiserDirector,
			AppraiserLicense:    body.Land.AppraiserLicense,
			AppraiserAddress:    body.Land.AppraiserAddress,
			AppraiserContact:    body.Land.AppraiserContact,
			SourceFileName:      body.Land.SourceFileName,
			SourceFileHash:      body.Land.SourceFileHash,
		},
	}
	for _, sec := range body.Sections {
		imp.Sections = append(imp.Sections, domain.ValuationSectionInput{
			Key:        sec.Key,
			Note:       sec.Note,
			SectionNo:  sec.SectionNo,
			TableLabel: sec.TableLabel,
			Title:      sec.Title,
			SortOrder:  sec.SortOrder,
		})
	}
	for _, a := range body.Assets {
		ia := domain.ValuationImportAsset{
			AssetNumber:        a.AssetNumber,
			AssetType:          normalizeAssetRecordType(a.AssetType),
			AssetName:          a.AssetName,
			AreaM2:             a.AreaM2,
			FloorCount:         a.FloorCount,
			Unit:               a.Unit,
			Capacity:           a.Capacity,
			Description:        a.Description,
			OwnerName:          a.OwnerName,
			Notes:              a.Notes,
			UnitPrice:          a.UnitPrice,
			CompensationAmount: a.CompensationAmount,
		}
		for _, s := range a.Specs {
			ia.Specs = append(ia.Specs, domain.AssetSpec{SpecTypeID: s.SpecTypeID, Value: s.Value})
		}
		for _, cc := range a.Calculations {
			ia.Calculations = append(ia.Calculations, domain.AssetCalculation{CalcTypeID: cc.CalcTypeID, CalcName: cc.Name, CalcGroup: cc.Group, Unit: cc.Unit, Value: cc.Value})
		}
		imp.Assets = append(imp.Assets, ia)
	}

	res, err := h.svc.ImportValuation(c.Request.Context(), acqID, imp)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res)
}

// ListCompensations lists compensations for an acquisition
func (h *LandHandler) ListCompensations(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Query("parcel_id")
	list, err := h.svc.ListCompensations(c.Request.Context(), acqID, parcelID, c.Query("valuation_type"))
	if err != nil {
		response.Error(c, err)
		return
	}
	resp := make([]compensationResponse, 0, len(list))
	for _, c2 := range list {
		resp = append(resp, toCompensationResponse(c2))
	}
	response.OK(c, resp)
}

// CreateCompensation creates a new compensation record
func (h *LandHandler) CreateCompensation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		TargetType       string  `json:"target_type"`
		ValuationType    string  `json:"valuation_type"`
		ParcelID         string  `json:"parcel_id"`
		AssetID          *string `json:"asset_id"`
		LegacyBuildingID *string `json:"building_id"`
		CompensationType string  `json:"compensation_type" binding:"required"`
		CoveragePercent  float64 `json:"coverage_percent"`
		Amount           float64 `json:"amount"`
		CompensationDate *string `json:"compensation_date"`
		Note             string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	createdBy := contextUserIDString(c)
	if mikaOnlyCaller(c) {
		body.ValuationType = domain.ValuationTypeMika
	}

	comp := &domain.Compensation{
		AcquisitionID:    acqID,
		TargetType:       normalizeTargetType(body.TargetType),
		ValuationType:    body.ValuationType,
		ParcelID:         body.ParcelID,
		CompensationType: body.CompensationType,
		CoveragePercent:  body.CoveragePercent,
		Amount:           body.Amount,
		Note:             body.Note,
		CreatedBy:        createdBy,
	}
	if body.CoveragePercent == 0 {
		comp.CoveragePercent = 100
	}
	assetID := body.AssetID
	if assetID == nil {
		assetID = body.LegacyBuildingID
	}
	if assetID != nil {
		if id, err := uuid.Parse(*assetID); err == nil {
			comp.AssetID = &id
		}
	}
	if body.CompensationDate != nil {
		if t, err := time.Parse("2006-01-02", *body.CompensationDate); err == nil {
			comp.CompensationDate = &t
		}
	}

	created, err := h.svc.CreateCompensation(c.Request.Context(), comp)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toCompensationResponse(created))
}

// UpdateCompensation updates a compensation record
func (h *LandHandler) UpdateCompensation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		TargetType       string  `json:"target_type"`
		ParcelID         string  `json:"parcel_id"`
		AssetID          *string `json:"asset_id"`
		LegacyBuildingID *string `json:"building_id"`
		CompensationType string  `json:"compensation_type"`
		CoveragePercent  float64 `json:"coverage_percent"`
		Amount           float64 `json:"amount"`
		CompensationDate *string `json:"compensation_date"`
		Note             string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	comp := &domain.Compensation{
		ID:               compID,
		AcquisitionID:    acqID,
		TargetType:       normalizeTargetType(body.TargetType),
		ParcelID:         body.ParcelID,
		CompensationType: body.CompensationType,
		CoveragePercent:  body.CoveragePercent,
		Amount:           body.Amount,
		Note:             body.Note,
	}
	assetID := body.AssetID
	if assetID == nil {
		assetID = body.LegacyBuildingID
	}
	if assetID != nil {
		if id, err := uuid.Parse(*assetID); err == nil {
			comp.AssetID = &id
		}
	}
	if body.CompensationDate != nil {
		if t, err := time.Parse("2006-01-02", *body.CompensationDate); err == nil {
			comp.CompensationDate = &t
		}
	}

	updated, err := h.svc.UpdateCompensation(c.Request.Context(), comp)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toCompensationResponse(updated))
}

func normalizeTargetType(t string) string {
	switch t {
	case "parcel", "asset":
		return t
	default:
		return "asset"
	}
}

func normalizeAssetRecordType(assetType string) string {
	switch assetType {
	case "", "building":
		return "real_state"
	case "parcel":
		return "property"
	default:
		return assetType
	}
}

// DeleteCompensation deletes a compensation record
func (h *LandHandler) DeleteCompensation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteCompensation(c.Request.Context(), acqID, compID, callerValuationStreams(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ApproveCompensation godoc
// @Summary      Үнэлгээ зөвшөөрөх
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "LandAcquisition UUID"
// @Param        comp_id  path  string  true  "Compensation UUID"
// @Router       /land-acquisitions/{id}/compensations/{comp_id}/approve [post]
func (h *LandHandler) ApproveCompensation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	reviewedBy := contextUserIDString(c)
	if err := h.svc.ApproveCompensation(c.Request.Context(), acqID, compID, body.Note, reviewedBy); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// RejectCompensation godoc
// @Summary      Үнэлгээ татгалзах
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "LandAcquisition UUID"
// @Param        comp_id  path  string  true  "Compensation UUID"
// @Router       /land-acquisitions/{id}/compensations/{comp_id}/reject [post]
func (h *LandHandler) RejectCompensation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	reviewedBy := contextUserIDString(c)
	if err := h.svc.RejectCompensation(c.Request.Context(), acqID, compID, body.Note, reviewedBy); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *LandHandler) UploadCompensationReport(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)
	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxDocumentUploadSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	uploadedBy := contextUserIDString(c)
	contentType, ok := allowedPDFContentType(data)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
		return
	}
	comp, err := h.svc.UploadCompensationReport(c.Request.Context(), acqID, compID, fh.Filename, contentType, data, uploadedBy)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toCompensationResponse(comp))
}

// UploadAssetPhoto — хөрөнгийн зургуудыг нэгтгэсэн PDF-ийг хадгална.
func (h *LandHandler) UploadAssetPhoto(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAssetPhotoSize)
	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxAssetPhotoSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	data, err := readMultipartFile(fh, maxAssetPhotoSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	contentType, ok := allowedPDFContentType(data)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
		return
	}
	url, name, err := h.svc.UploadAssetPhoto(c.Request.Context(), acqID, assetID, fh.Filename, contentType, data)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"photo_pdf_url": url, "photo_pdf_name": name})
}

// ListCompensationHistory godoc
// @Summary      Үнэлгээний татгалзсан түүх
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "LandAcquisition UUID"
// @Param        comp_id  path  string  true  "Compensation UUID"
// @Router       /land-acquisitions/{id}/compensations/{comp_id}/history [get]
func (h *LandHandler) ListCompensationHistory(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	list, err := h.svc.ListCompensationHistory(c.Request.Context(), acqID, compID)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]compensationHistoryResponse, 0, len(list))
	for _, h := range list {
		result = append(result, toCompensationHistoryResponse(h))
	}
	response.OK(c, result)
}

// CreateCompensationGrant creates a new compensation grant record
func (h *LandHandler) CreateCompensationGrant(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		Amount       float64 `json:"amount"`
		GrantDate    *string `json:"grant_date"`
		Note         string  `json:"note"`
		LandAreaM2   float64 `json:"land_area_m2"`
		LandPrice    float64 `json:"land_price"`
		LandLocation string  `json:"land_location"`
		LandPurpose  string  `json:"land_purpose"`
		LandUseType  string  `json:"land_use_type"`
		ParcelNumber string  `json:"parcel_number"`
		DecreeNumber string  `json:"decree_number"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	createdBy := contextUserIDString(c)

	g := &domain.CompensationGrant{
		AcquisitionID:  acqID,
		CompensationID: compID,
		Amount:         body.Amount,
		Note:           body.Note,
		LandAreaM2:     body.LandAreaM2,
		LandPrice:      body.LandPrice,
		LandLocation:   body.LandLocation,
		LandPurpose:    body.LandPurpose,
		LandUseType:    body.LandUseType,
		ParcelNumber:   body.ParcelNumber,
		DecreeNumber:   body.DecreeNumber,
		CreatedBy:      createdBy,
	}
	if body.GrantDate != nil {
		if t, err := time.Parse("2006-01-02", *body.GrantDate); err == nil {
			g.GrantDate = &t
		}
	}

	created, err := h.svc.CreateCompensationGrant(c.Request.Context(), g)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toCompensationGrantResponse(created))
}

// UpdateCompensationGrant updates a compensation grant record
func (h *LandHandler) UpdateCompensationGrant(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		Amount       float64 `json:"amount"`
		GrantDate    *string `json:"grant_date"`
		Note         string  `json:"note"`
		LandAreaM2   float64 `json:"land_area_m2"`
		LandPrice    float64 `json:"land_price"`
		LandLocation string  `json:"land_location"`
		LandPurpose  string  `json:"land_purpose"`
		LandUseType  string  `json:"land_use_type"`
		ParcelNumber string  `json:"parcel_number"`
		DecreeNumber string  `json:"decree_number"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	g := &domain.CompensationGrant{
		AcquisitionID:  acqID,
		CompensationID: compID,
		Amount:         body.Amount,
		Note:           body.Note,
		LandAreaM2:     body.LandAreaM2,
		LandPrice:      body.LandPrice,
		LandLocation:   body.LandLocation,
		LandPurpose:    body.LandPurpose,
		LandUseType:    body.LandUseType,
		ParcelNumber:   body.ParcelNumber,
		DecreeNumber:   body.DecreeNumber,
	}
	if body.GrantDate != nil {
		if t, err := time.Parse("2006-01-02", *body.GrantDate); err == nil {
			g.GrantDate = &t
		}
	}

	updated, err := h.svc.UpdateCompensationGrant(c.Request.Context(), g)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toCompensationGrantResponse(updated))
}

// DeleteCompensationGrant deletes a compensation grant record by compensation ID
func (h *LandHandler) DeleteCompensationGrant(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteCompensationGrantByCompensationID(c.Request.Context(), acqID, compID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListParcelPayments godoc
// @Summary      Нэгж талбарын төлбөрийн бичлэгүүд
// @Tags         parcels
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Parcel UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /parcels/{id}/payments [get]
func (h *LandHandler) ListParcelPayments(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	list, err := h.svc.ListParcelPayments(c.Request.Context(), parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}

	result := make([]parcelPaymentResponse, 0, len(list))
	for _, p := range list {
		r := parcelPaymentResponse{
			ID:        p.ID.String(),
			ParcelID:  p.ParcelID.String(),
			Amount:    p.Amount,
			Currency:  p.Currency,
			Note:      p.Note,
			CreatedAt: p.CreatedAt.Format(time.RFC3339),
			CreatedBy: p.CreatedBy,
		}
		if p.PaidAt != nil {
			s := p.PaidAt.Format("2006-01-02")
			r.PaidAt = &s
		}
		result = append(result, r)
	}
	response.OK(c, result)
}

// AddParcelPayment godoc
// @Summary      Нэгж талбарт төлбөрийн бичлэг нэмэх
// @Tags         parcels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string  true  "Parcel UUID"
// @Param        body  body  object  true  "amount, currency, paid_at, note"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /parcels/{id}/payments [post]
func (h *LandHandler) AddParcelPayment(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		Amount   float64 `json:"amount" binding:"required"`
		Currency string  `json:"currency"`
		PaidAt   string  `json:"paid_at"`
		Note     string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	currency := body.Currency
	if currency == "" {
		currency = "MNT"
	}

	createdBy := contextUserIDString(c)

	p := &domain.ParcelPayment{
		ParcelID:  parcelID,
		Amount:    body.Amount,
		Currency:  currency,
		Note:      body.Note,
		CreatedBy: createdBy,
	}
	if body.PaidAt != "" {
		if t, err := time.Parse("2006-01-02", body.PaidAt); err == nil {
			p.PaidAt = &t
		}
	}

	if err := h.svc.AddParcelPayment(c.Request.Context(), p); err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, nil)
}

func (h *LandHandler) ListParcelDocuments(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	list, err := h.svc.ListParcelDocuments(c.Request.Context(), parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]documentResponse, 0, len(list))
	for _, d := range list {
		result = append(result, toDocumentResponse(d))
	}
	response.OK(c, result)
}

// SyncParcelHolders godoc
//
//	@Summary		Нэгж талбарын эзэмшигчдийг татах
//	@Description	Хадгалагдсан app_no-гоор иргэн/хуулийн этгээдийн мэдээллийг ГУС-аас татна.
//	@Tags			land-acquisitions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path	string	true	"LandAcquisition UUID"
//	@Param			parcel_id	path	string	true	"Нэгж талбарын дугаар"
//	@Success		200	{object}	response.Response
//	@Failure		422	{object}	response.Response	"app_no байхгүй — эхлээд /sync"
//	@Router			/land-acquisitions/{id}/parcels/{parcel_id}/sync/holders [post]
func (h *LandHandler) SyncParcelHolders(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelCode := c.Param("parcel_id")

	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelHolders(ctx, acqID, parcelCode)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, gin.H{
		"found":   res.Found,
		"saved":   res.Saved,
		"skipped": res.Skipped,
	}, i18n.MsgSynced)
}

// SyncParcelBasePrice — газрын суурь үнэ (ГУС data_cama)
func (h *LandHandler) SyncParcelBasePrice(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelBasePrice(ctx, acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, res, i18n.MsgSynced)
}

// SyncParcelInvoices — газрын төлбөр. Өмчлөлийн газарт 422 буцна.
func (h *LandHandler) SyncParcelInvoices(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	items, err := h.svc.SyncParcelInvoices(ctx, acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	if items == nil {
		items = []*domain.ParcelInvoice{}
	}
	response.OKMsg(c, gin.H{"found": len(items), "invoices": items}, i18n.MsgSynced)
}

// SyncParcelFee — газрын ТӨЛБӨРИЙН БОДОЛТ. Өмчлөлийн газарт 422 буцна.
//
// status=false нь HTTP алдаа БИШ: ГУС төлбөрийг бодож чадаагүй шалтгаан msg-д
// монголоор ирнэ (жишээ нь үнэлгээний зэрэглэл тодорхойгүй).
func (h *LandHandler) SyncParcelFee(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelFee(ctx, acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	items := res.Items
	if items == nil {
		items = []domain.ParcelFeeItem{}
	}
	response.OKMsg(c, gin.H{
		"found":         len(items),
		"status":        res.Status,
		"msg":           res.Msg,
		"area_m2":       res.AreaM2,
		"total_payment": res.TotalPayment,
		"items":         items,
	}, i18n.MsgSynced)
}

// SyncParcelMortgages — барьцааны бүртгэл (давхар барьцаатай)
func (h *LandHandler) SyncParcelMortgages(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelMortgages(ctx, acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, res, i18n.MsgSynced)
}

// SyncParcelCourtDecisions — шүүхийн шийдвэрийн бүртгэл
func (h *LandHandler) SyncParcelCourtDecisions(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelCourtDecisions(ctx, acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, res, i18n.MsgSynced)
}

// SyncParcelMonitorings — хянан баталгааны мэдээлэл
func (h *LandHandler) SyncParcelMonitorings(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelMonitorings(ctx, acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, res, i18n.MsgSynced)
}

// SyncParcelOwnerships godoc
//
//	@Summary		УБЕГ-ийн газар өмчлөлийн бүртгэл татах
//	@Description	ХУР (XYP) гарцаас нэгж талбарын дугаараар өмчлөгчид ба өмчлөлийн түүхийг татаж хадгална. Нэгж талбарын мөрүүд БҮХЭЛДЭЭ орлуулагдана.
//	@Tags			land-acquisitions
//	@Produce		json
//	@Param			id			path		string	true	"Газар чөлөөлөлтийн ID"
//	@Param			parcel_id	path		string	true	"Нэгж талбарын дугаар (код)"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response
//	@Failure		404			{object}	response.Response
//	@Failure		503			{object}	response.Response
//	@Security		BearerAuth
//	@Router			/land-acquisitions/{id}/parcels/{parcel_id}/sync/ownership [post]
func (h *LandHandler) SyncParcelOwnerships(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	// ХУР гарц нь ГУС-ийн дундын сервис БИШ тул хэрэглэгчийн bearer token-ыг
	// дамжуулахгүй (тэр гарц түүнийг шалгадаггүй).
	res, err := h.svc.SyncParcelOwnerships(c.Request.Context(), acqID, c.Param("parcel_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, res, i18n.MsgSynced)
}

// SyncParcelDocuments godoc
//
//	@Summary		Нэгж талбарын хавсралтыг эх системээс татах
//	@Description	Хүсэлтийн дугаараар (app_no) хавсралт татна. ЗӨВХӨН ШИНЭ хавсралт нэмэгдэнэ — хадгалагдсан хавсралт устгагдахгүй, дарагдахгүй. role параметрээр баримтын үүргээр шүүж болно (давтаж эсвэл таслалаар зааглаж).
//	@Tags			land-acquisitions
//	@Produce		json
//	@Param			id			path		string	true	"Газар чөлөөлөлтийн ID"
//	@Param			parcel_id	path		string	true	"Нэгж талбарын дугаар (код)"
//	@Param			role		query		[]string	false	"Баримтын үүргийн код"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response
//	@Failure		404			{object}	response.Response
//	@Security		BearerAuth
//	@Router			/land-acquisitions/{id}/parcels/{parcel_id}/sync/documents [post]
func (h *LandHandler) SyncParcelDocuments(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelCode := c.Param("parcel_id")

	ctx := service.ContextWithAuthorization(c.Request.Context(), c.GetHeader("Authorization"))
	res, err := h.svc.SyncParcelDocuments(ctx, acqID, parcelCode, documentRoleQuery(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	docs := make([]documentResponse, 0, len(res.Documents))
	for _, d := range res.Documents {
		docs = append(docs, toDocumentResponse(d))
	}
	response.OKMsg(c, gin.H{
		"found":     res.Found,
		"saved":     res.Saved,
		"skipped":   res.Skipped,
		"documents": docs,
	}, i18n.MsgSynced)
}

// GetParcelDocumentFile godoc
//
//	@Summary		Эх системээс татсан хавсралтын файлыг харах/татах
//	@Description	Файл нь эх системийн файл сервер дээр байдаг — API тэндээс уншиж дамжуулна. Анхдагчаар браузерт нээгдэхээр (inline) буцна; ?download=1 үед татагдана. Зөвхөн эх системээс татсан хавсралтад ажиллана.
//	@Tags			parcels
//	@Produce		octet-stream
//	@Param			id			path	string	true	"Нэгж талбарын UUID"
//	@Param			doc_id		path	string	true	"Баримтын UUID"
//	@Param			download	query	bool	false	"1 бол татна, эс бөгөөс браузерт нээнэ"
//	@Success		200
//	@Failure		404	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/documents/{doc_id}/file [get]
func (h *LandHandler) GetParcelDocumentFile(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	file, err := h.svc.OpenParcelDocumentFile(c.Request.Context(), parcelID, docID)
	if err != nil {
		response.Error(c, err)
		return
	}
	defer file.Content.Close()

	contentType := file.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	disposition := "inline"
	if isTruthyQuery(c.Query("download")) {
		disposition = "attachment"
	}
	// Нэр нь кирилл байж болох тул RFC 5987 (filename*) хэлбэрээр илгээнэ.
	headers := map[string]string{
		"Content-Disposition": disposition +
			`; filename="` + sanitizeASCIIFilename(file.Name) + `"` +
			`; filename*=UTF-8''` + url.PathEscape(file.Name),
	}

	// Хэмжээ мэдэгдэхгүй үед (-1) Content-Length тавихгүй — chunked дамжина.
	c.DataFromReader(http.StatusOK, file.SizeBytes, contentType, file.Content, headers)
}

// isTruthyQuery — "1", "true", "yes" зэргийг үнэн гэж үзнэ.
func isTruthyQuery(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y":
		return true
	}
	return false
}

// sanitizeASCIIFilename — Content-Disposition-ий ASCII нөөц нэр. Кирилл болон
// тусгай тэмдэгтийг зайгаар солино (filename* нь бүтэн нэрийг агуулна).
func sanitizeASCIIFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "document"
	}
	return b.String()
}

// documentRoleQuery — ?role=10&role=20 болон ?role=10,20 хоёуланг задална.
func documentRoleQuery(c *gin.Context) []string {
	values := c.QueryArray("role")
	roles := make([]string, 0, len(values))
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				roles = append(roles, part)
			}
		}
	}
	return roles
}

func (h *LandHandler) UploadParcelDocument(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)
	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxDocumentUploadSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	uploadedBy := contextUserIDString(c)
	documentTypeID, err := documentTypeIDFromForm(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	// Хавсралт ерөнхийдөө зөвхөн PDF өргөтгөлтэй байна; "Хурлын тэмдэглэл" төрөлд
	// Гэрээтэй нэгтгэж хэвлэдэг тул DOCX-ийг нэмж зөвшөөрнө.
	dt, err := documentTypeForUpload(c, h.svc, documentTypeID)
	if err != nil {
		response.Error(c, err)
		return
	}
	contentType, ok := allowedDocumentContentType(fh.Filename, data, dt)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
		return
	}
	// filename нь өргөтгөл тодорхойлоход, name нь дэлгэцийн нэрэнд ашиглагдана
	payment, err := documentPaymentFromForm(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	doc, err := h.svc.UploadParcelDocumentWithPayment(c.Request.Context(), parcelID, fh.Filename, c.PostForm("name"), contentType, data, uploadedBy, documentTypeID, payment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toDocumentResponse(doc))
}

// documentPaymentFromForm — «Нөхөх олговор олгосон баримт»-ын олголтын шат
// (payment_stage: 60 | 40 | full) ба огноо (payment_date: YYYY-MM-DD). Шатгүй
// бол nil. Төрөлтэй нийцэх эсэхийг service шалгана.
func documentPaymentFromForm(c *gin.Context) (*service.DocumentPayment, error) {
	raw := strings.TrimSpace(c.PostForm("payment_stage"))
	if raw == "" {
		return nil, nil
	}
	var stage int
	switch raw {
	case "60":
		stage = domain.PaymentStage60
	case "40":
		stage = domain.PaymentStage40
	case "100", "full":
		stage = domain.PaymentStageFull
	default:
		return nil, domain.ErrInvalidInput
	}
	payment := &service.DocumentPayment{Stage: stage}
	if d := strings.TrimSpace(c.PostForm("payment_date")); d != "" {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		payment.Date = &t
	}
	return payment, nil
}

func documentTypeIDFromForm(c *gin.Context) (int, error) {
	raw := c.PostForm("document_type_id")
	if raw == "" {
		return 0, nil
	}
	documentTypeID, err := strconv.Atoi(raw)
	if err != nil || documentTypeID < 0 {
		return 0, domain.ErrInvalidInput
	}
	return documentTypeID, nil
}

// documentTypeForUpload — байршуулж буй хавсралтын төрлийг өгөгдсөн ID-гаар
// татаж авчирна (агуулгын төрлийн шалгалтад ашиглана). ID заагаагүй бол nil буцаана.
func documentTypeForUpload(c *gin.Context, svc service.LandService, documentTypeID int) (*domain.DocumentType, error) {
	if documentTypeID <= 0 {
		return nil, nil
	}
	dt, err := svc.GetDocumentType(c.Request.Context(), documentTypeID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrInvalidInput
		}
		return nil, err
	}
	return dt, nil
}

func (h *LandHandler) DeleteParcelDocument(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteParcelDocument(c.Request.Context(), parcelID, docID); err != nil {
		if err == domain.ErrNotFound {
			response.Abort(c, http.StatusNotFound, i18n.ErrDocNotFound)
			return
		}
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// SetParcelCompensation godoc
// @Summary      Нэгж талбарын нөхөн төлбөрийн төлөв шинэчлэх
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Parcel UUID"
// @Param        body       body  object  true  "paid"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/compensation [put]
func (h *LandHandler) SetParcelCompensation(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	// parcel тухайн чөлөөлөлтөд харьяалагдаж буйг шалгана (IDOR).
	if err := h.svc.EnsureParcelInAcquisition(c.Request.Context(), acqID, parcelID); err != nil {
		response.Error(c, err)
		return
	}
	var body struct {
		Paid bool `json:"paid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	if err := h.svc.SetParcelCompensation(c.Request.Context(), parcelID, body.Paid); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// Dashboard godoc
// @Summary      Дашбоардын нийлмэл мэдээлэл
// @Description  Чөлөөлөлт, нэгж талбарын төлөв, тайлан, бүх нэгж талбарыг нэг хүсэлтэд буцаана
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /dashboard [get]
// FinanceDashboard godoc
// @Summary      Санхүүгийн хяналтын самбарын нэгтгэл
// @Description  Он / дүүрэг / хороогоор нөхөх олговрын дүн, олгосон дүн, тоо
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        year      query  []int   false  "Он (олон удаа)"
// @Param        au2_code  query  string  false  "Дүүрэг/сум"
// @Param        au3_code  query  string  false  "Хороо/баг"
// @Success      200  {object}  response.Response
// @Router       /dashboard/finance [get]
func (h *LandHandler) FinanceDashboard(c *gin.Context) {
	filter := domain.FinanceDashboardFilter{
		AU2Code:       strings.TrimSpace(c.Query("au2_code")),
		AU3Code:       strings.TrimSpace(c.Query("au3_code")),
		AcquisitionID: strings.TrimSpace(c.Query("acquisition_id")),
	}
	for _, y := range c.QueryArray("year") {
		if n, err := strconv.Atoi(y); err == nil {
			filter.Years = append(filter.Years, n)
		}
	}
	summary, err := h.svc.GetFinanceDashboard(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, summary)
}

// Chronos — «Хронос» цаг хугацааны изометр зургийн өгөгдөл. acquisition_id
// олон удаа өгч болно; өгөөгүй бол бүх чөлөөлөлт. Гадаад (үнэлгээний
// байгууллага, санхүү, МИКА) хэрэглэгчийн хамрах хүрээ нарийн тул ЗӨВХӨН дотоод ажилтанд.
func (h *LandHandler) Chronos(c *gin.Context) {
	isSenior := ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн")
	isReviewer := ctxHasRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "mika", "МИКА")
	if !isSenior && (middleware.IsValuationOrgCaller(c) || isReviewer) {
		response.Abort(c, http.StatusForbidden, i18n.ErrForbidden)
		return
	}
	var ids []uuid.UUID
	for _, raw := range c.QueryArray("acquisition_id") {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
			return
		}
		ids = append(ids, id)
	}
	data, err := h.svc.GetChronos(c.Request.Context(), ids)
	if err != nil {
		response.Error(c, err)
		return
	}
	// Геометр бүхий хариу хэдэн MB — gzip-ээр шахна.
	response.OKCompressed(c, data)
}

func (h *LandHandler) Dashboard(c *gin.Context) {
	filter := domain.DashboardFilter{
		AcquisitionID:     c.Query("acquisition_id"),
		PlanCode:          c.Query("plan_code"),
		AU2Code:           c.Query("au2_code"),
		Status:            queryInt(c, "status", 0),
		GeneralCategoryID: queryInt(c, "general_category_id", 0),
		SubCategoryID:     queryInt(c, "sub_category_id", 0),
		AssignedUserID:    c.Query("assigned_user_id"),
	}
	for _, y := range c.QueryArray("year") {
		if n, err := strconv.Atoi(y); err == nil {
			filter.Years = append(filter.Years, n)
		}
	}

	isSenior := ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн")
	// Үнэлгээний байгууллага — роль БИШ, аппын өөрийн бүртгэлээр танигдана.
	isProf := middleware.IsValuationOrgCaller(c)
	isReviewer := ctxHasRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "mika", "МИКА")
	isExternalDashboard := !isSenior && (isProf || isReviewer)
	if isExternalDashboard {
		// «Үнэлгээний шат» (is_valuation_stage) төлөвтэй нэгж талбар.
		filter.ValuationStageOnly = true
		if isReviewer {
			filter.Status = domain.LandAcquisitionStatusField
		}
		if isProf {
			// Байгууллагын харьяалалаар шүүнэ — тухайн байгууллагын аль ч
			// ажилтан ижил хяналтын самбар харна. Харьяалалгүй (ажилтны
			// бүртгэлгүй) хэрэглэгчид өгөгдөл харуулах ёсгүй.
			if orgID, ok := middleware.GetCallerOrgID(c); ok {
				filter.ProfessionalOrgID = strconv.FormatInt(int64(orgID), 10)
			} else {
				response.Abort(c, http.StatusForbidden, i18n.ErrForbidden)
				return
			}
		}
	}

	summary, err := h.svc.GetDashboard(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	acqs := make([]dashboardAcqResponse, 0, len(summary.Acquisitions))
	for _, a := range summary.Acquisitions {
		acqs = append(acqs, dashboardAcqResponse{
			ID:               a.ID.String(),
			AcquisitionName:  a.AcquisitionName,
			PlanCode:         a.PlanCode,
			Status:           a.Status,
			ParcelCount:      a.ParcelCount,
			FinalParcelCount: a.FinalParcelCount,
			ProgressPercent:  a.ProgressPercent(),
		})
	}

	res := dashboardResponse{
		Acquisitions:      acqs,
		ParcelStatuses:    summary.ParcelStatuses,
		TotalParcels:      summary.TotalParcels,
		FreedParcels:      summary.FreedParcels,
		FreedAreaM2:       summary.FreedAreaM2,
		PlanAreaM2:        summary.PlanAreaM2,
		TotalOrders:       summary.TotalOrders,
		TotalCompensation: summary.TotalCompensation,
		Decisions:         summary.Decisions,
		Compensation:      summary.Compensation,
		StatusBreakdown:   summary.StatusBreakdown,
		Timeline:          summary.Timeline,
		FilteredAU1Codes:  summary.FilteredAU1Codes,
		FilteredAU2Codes:  summary.FilteredAU2Codes,
		FilteredAU3Codes:  summary.FilteredAU3Codes,
		Locations:         summary.Locations,
	}
	if isExternalDashboard {
		res.ValuationStatuses = summary.ValuationStatuses
		res.ValuationTypes = summary.ValuationTypes
	}

	response.OK(c, res)
}

// CreateFundingSource godoc
// @Summary      Санхүүжилтийн эх үүсвэр нэмэх
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/funding-sources [post]
func (h *LandHandler) CreateFundingSource(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		OrganizationName string   `json:"organization_name" binding:"required"`
		SourceType       string   `json:"source_type" binding:"required"`
		Amount           *float64 `json:"amount"`
		Currency         string   `json:"currency"`
		Note             string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	currency := body.Currency
	if currency == "" {
		currency = "MNT"
	}
	fs := &domain.FundingSource{
		AcquisitionID:    acqID,
		OrganizationName: body.OrganizationName,
		SourceType:       body.SourceType,
		Amount:           body.Amount,
		Currency:         currency,
		Note:             body.Note,
		CreatedBy:        contextUserIDString(c),
	}
	created, err := h.svc.CreateFundingSource(c.Request.Context(), fs)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 200, "data": toFundingSourceResponse(created), "message": "success"})
}

// UpdateFundingSource godoc
// @Summary      Санхүүжилтийн эх үүсвэр засах
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  string  true  "LandAcquisition UUID"
// @Param        src_id  path  string  true  "FundingSource UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/funding-sources/{src_id} [put]
func (h *LandHandler) UpdateFundingSource(c *gin.Context) {
	srcID, err := uuid.Parse(c.Param("src_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var body struct {
		OrganizationName string   `json:"organization_name" binding:"required"`
		SourceType       string   `json:"source_type" binding:"required"`
		Amount           *float64 `json:"amount"`
		Currency         string   `json:"currency"`
		Note             string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	currency := body.Currency
	if currency == "" {
		currency = "MNT"
	}
	fs := &domain.FundingSource{
		ID:               srcID,
		OrganizationName: body.OrganizationName,
		SourceType:       body.SourceType,
		Amount:           body.Amount,
		Currency:         currency,
		Note:             body.Note,
	}
	updated, err := h.svc.UpdateFundingSource(c.Request.Context(), fs)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toFundingSourceResponse(updated))
}

// DeleteFundingSource godoc
// @Summary      Санхүүжилтийн эх үүсвэр устгах
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  string  true  "LandAcquisition UUID"
// @Param        src_id  path  string  true  "FundingSource UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/funding-sources/{src_id} [delete]
func (h *LandHandler) DeleteFundingSource(c *gin.Context) {
	srcID, err := uuid.Parse(c.Param("src_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteFundingSource(c.Request.Context(), srcID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListAuthorizedRepresentatives godoc
// @Summary      Итгэмжлэгдсэн төлөөлөгчийн жагсаалт
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын дугаар"
// @Success      200  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/representatives [get]
func (h *LandHandler) ListAuthorizedRepresentatives(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Param("parcel_id")
	reps, err := h.svc.ListParcelRepresentatives(c.Request.Context(), acqID, parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]parcelHolderResponse, 0, len(reps))
	for _, rep := range reps {
		result = append(result, toParcelHolderResponse(rep))
	}
	response.OK(c, result)
}

// SetParcelPaymentRecipient godoc
// @Summary      Нөхөн төлбөр хүлээн авах эзэмшигчийг сонгох
// @Description  Үндсэн эзэмшигч олон байх үед төлбөрийг хэнд олгохыг сонгоно.
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын дугаар"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/payment-recipient [put]
func (h *LandHandler) SetParcelPaymentRecipient(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Param("parcel_id")
	var body struct {
		HolderID string `json:"holder_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	holderRowID, err := uuid.Parse(body.HolderID)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.SetParcelPaymentRecipient(c.Request.Context(), acqID, parcelID, holderRowID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// CreateAuthorizedRepresentative godoc
// @Summary      Итгэмжлэгдсэн төлөөлөгч нэмэх
// @Tags         land-acquisitions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын дугаар"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/representatives [post]
func (h *LandHandler) CreateAuthorizedRepresentative(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Param("parcel_id")
	var body struct {
		LastName   string `json:"last_name" binding:"required"`
		FirstName  string `json:"first_name" binding:"required"`
		RegisterNo string `json:"register_no"`
		Phone      string `json:"phone"`
		Email      string `json:"email"`
		Address    string `json:"address"`
		Note       string `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}
	// Итгэмжлэгдсэн төлөөлөгч нь эзэмшигчийн нэг төрөл болж хадгалагдана —
	// FirstName нь right_holder.name-д тохирно.
	rep := &domain.ParcelHolder{
		LastName:   body.LastName,
		Name:       body.FirstName,
		RegisterNo: body.RegisterNo,
		Phone:      body.Phone,
		Email:      body.Email,
		Address:    body.Address,
		Note:       body.Note,
		PersonType: domain.ParcelHolderPersonTypeRepresentative,
	}
	created, err := h.svc.CreateParcelRepresentative(c.Request.Context(), acqID, parcelID, rep)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 200, "data": toParcelHolderResponse(*created), "message": "success"})
}

// ListAllCompensations godoc
// @Summary      Глобал нөхөн төлбөрийн жагсаалт
// @Tags         compensations
// @Produce      json
// @Security     BearerAuth
// @Param        status          query  string  false  "Статус (pending|approved|rejected)"
// @Param        search          query  string  false  "Хайлт (parcel_id эсвэл эзэмшигчийн нэр)"
// @Param        acquisition_id  query  string  false  "Чөлөөлөлтийн ID"
// @Param        page            query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size       query  int     false  "Хуудасны хэмжээ (default: 20)"
// @Success      200  {object}  response.PaginatedResponse
// @Router       /compensations [get]
func (h *LandHandler) ListAllCompensations(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	filter := domain.GlobalCompensationFilter{
		Status:        c.Query("status"),
		Search:        c.Query("search"),
		AcquisitionID: c.Query("acquisition_id"),
		Page:          page,
		PageSize:      pageSize,
	}
	list, total, err := h.svc.ListAllCompensations(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]globalCompensationResponse, 0, len(list))
	for _, gc := range list {
		result = append(result, toGlobalCompensationResponse(gc))
	}
	response.Paginated(c, result, total, page, pageSize)
}

// GetGlobalCompensation godoc
// @Summary      Глобал нөхөн төлбөр дэлгэрэнгүй
// @Tags         compensations
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Нөхөн төлбөрийн UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /compensations/{id} [get]
func (h *LandHandler) GetGlobalCompensation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	gc, err := h.svc.GetGlobalCompensation(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toGlobalCompensationResponse(gc))
}

// DeleteAuthorizedRepresentative godoc
// @Summary      Итгэмжлэгдсэн төлөөлөгч устгах
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string  true  "LandAcquisition UUID"
// @Param        parcel_id  path  string  true  "Нэгж талбарын дугаар"
// @Param        rep_id     path  string  true  "Representative UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /land-acquisitions/{id}/parcels/{parcel_id}/representatives/{rep_id} [delete]
func (h *LandHandler) DeleteAuthorizedRepresentative(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	repID, err := uuid.Parse(c.Param("rep_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteParcelRepresentative(c.Request.Context(), acqID, c.Param("parcel_id"), repID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListMyAcquisitions — үнэлгээний байгууллагын ажилтанд зориулсан endpoint.
// JWT-ийн org_id-г ProfessionalOrgID filter болгон ашиглана — иймд нэг
// байгууллагын БҮХ ажилтан ижил жагсаалттай ажиллана.
//
// Шүүлт нь чөлөөлөлтийн үндсэн байгууллага (professional_org_id) БОЛОН нэгж
// талбарын хөндлөнгийн байгууллага (independent_org_id) хоёуланг хамарна —
// байгууллага холбогдоогүй чөлөөлөлт ХАРАГДАХГҮЙ.
func (h *LandHandler) ListMyAcquisitions(c *gin.Context) {
	// Зөвхөн үнэлгээний байгууллагын ажилтан (роль БИШ — токены
	// `valuation_org` тэмдэг).
	if !middleware.IsValuationOrgCaller(c) {
		response.Abort(c, http.StatusForbidden, i18n.ErrForbidden)
		return
	}

	orgID, ok := middleware.GetCallerOrgID(c)
	if !ok {
		// Ажилтны бүртгэлгүй тул ямар ч байгууллагад харьяалагдахгүй —
		// байгууллагад оноогдсон ажлыг харах эрхгүй.
		response.Abort(c, http.StatusForbidden, i18n.ErrForbidden)
		return
	}
	orgIDStr := strconv.FormatInt(int64(orgID), 10)

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)

	filter := domain.LandAcquisitionFilter{
		PlanCode:        c.Query("plan_code"),
		AcquisitionName: c.Query("acquisition_name"),
		// Мэргэжлийн байгууллагад зөвхөн "Хээрийн судалгаа" төлөвтэй чөлөөлөлт
		// харагдана — query-гийн status параметрийг үл хэрэгсэнэ.
		Status:            domain.LandAcquisitionStatusField,
		AU3Code:           c.Query("au3_code"),
		ProfessionalOrgID: orgIDStr,
		SortBy:            c.Query("sort"),
		SortDir:           c.Query("order"),
		Page:              page,
		PageSize:          pageSize,
	}

	list, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	items := make([]landAcquisitionResponse, len(list))
	for i, a := range list {
		items[i] = toLandAcqResponse(a)
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *LandHandler) SetAcquisitionProfessionalOrg(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	orgID, ok := bindOrgAssignmentBody(c)
	if !ok {
		return
	}
	if err := h.svc.SetAcquisitionProfessionalOrg(c.Request.Context(), acqID, orgID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// bindOrgAssignmentBody — үнэлгээний байгууллага оноох биеийг уншина.
//
// Утга нь одоо ХЭРЭГЛЭГЧИЙН биш БАЙГУУЛЛАГЫН ID. Шинэ нэр нь `org_id`; хуучин
// `org_user_id` түлхүүрийг зэрэгцүүлж хүлээн авсаар байна (нэвтрүүлэлтийн үед
// хуучин frontend эвдрэхээс сэргийлнэ). Хоосон/байхгүй = оноолтыг цуцлах.
func bindOrgAssignmentBody(c *gin.Context) (*domain.ID, bool) {
	var body struct {
		OrgID *string `json:"org_id"`
		// Deprecated: org_id-г хэрэглэнэ үү.
		OrgUserID *string `json:"org_user_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.Key(err.Error()))
		return nil, false
	}
	raw := body.OrgID
	if raw == nil || *raw == "" {
		raw = body.OrgUserID
	}
	if raw == nil || *raw == "" {
		return nil, true
	}
	parsed, ok := parseID(*raw)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return nil, false
	}
	return &parsed, true
}

func (h *LandHandler) SetParcelIndependentOrg(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	orgID, ok := bindOrgAssignmentBody(c)
	if !ok {
		return
	}
	if err := h.svc.SetParcelIndependentOrg(c.Request.Context(), acqID, parcelID, orgID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}
