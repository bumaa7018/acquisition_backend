package http

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/i18n"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
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

// ListParcelStatuses returns all parcel status lookup records.
func (h *LandHandler) ListParcelStatuses(c *gin.Context) {
	list, err := h.svc.ListParcelStatuses(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
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
		PlanCode: c.Query("plan_code"),
		Status:   queryInt(c, "status", 0),
		AU3Code:  c.Query("au3_code"),
		Page:     page,
		PageSize: pageSize,
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
// @Description  plan_code-аар төлөвлөгөөг secondary DB-аас хайж, shapefile шалгаад чөлөөлөлт үүсгэнэ
// @Tags         land-acquisitions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        plan_code  formData  string  true  "Газар зохион байгуулалтын төлөвлөгөөний дугаар"
// @Param        start_date formData  string  true  "Эхлэх огноо (YYYY-MM-DD)"
// @Param        end_date   formData  string  true  "Дуусах огноо (YYYY-MM-DD)"
// @Param        shapefile  formData  file    true  "Чөлөөлөх бүсийн хил (.shp файл)"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      422  {object}  response.Response
// @Router       /land-acquisitions [post]
func (h *LandHandler) Create(c *gin.Context) {
	planCode := c.PostForm("plan_code")
	startStr := c.PostForm("start_date")
	endStr := c.PostForm("end_date")
	projectName := c.PostForm("acquisition_name")
	implementingOrg := c.PostForm("implementing_org")
	reason := c.PostForm("reason")
	responsibleOrg := c.PostForm("responsible_org")
	fundingSource := c.PostForm("funding_source")

	if planCode == "" || startStr == "" || endStr == "" || projectName == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrRequiredFields)
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrStartDateFormat)
		return
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrEndDateFormat)
		return
	}

	fh, err := c.FormFile("shapefile")
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileRequired)
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileOpen)
		return
	}
	defer f.Close()

	zipData, err := io.ReadAll(f)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileRead)
		return
	}

	createdByStr := ""
	if uid, ok := c.Get("userID"); ok {
		createdByStr = uid.(uuid.UUID).String()
	}

	var constructionTypeID *int
	if s := c.PostForm("construction_type_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			constructionTypeID = &n
		}
	}

	acq, err := h.svc.Create(c.Request.Context(), service.CreateLandAcquisitionRequest{
		PlanCode:           planCode,
		StartDate:          startDate,
		EndDate:            endDate,
		ShapefileData:      zipData,
		CreatedBy:          createdByStr,
		AcquisitionName:    projectName,
		ImplementingOrg:    implementingOrg,
		Reason:             reason,
		ResponsibleOrg:     responsibleOrg,
		FundingSource:      fundingSource,
		ConstructionTypeID: constructionTypeID,
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
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)

	filter := domain.ParcelFilter{
		ParcelID:  c.Query("parcel_id"),
		AU1Code:   c.Query("au1_code"),
		AU2Code:   c.Query("au2_code"),
		AU3Code:   c.Query("au3_code"),
		RightType: queryInt(c, "right_type", 0),
		Landuse:   c.Query("landuse"),
		Page:      page,
		PageSize:  pageSize,
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

// Update godoc
// @Summary      Газар чөлөөлөлтийн мэдээлэл засах
// @Description  Геометр өөрчлөх бол shapefile заавал шаардлагатай. Геометр өөрчлөгдвөл хуучин нэгж талбарын мэдээлэл бүгд устана.
// @Tags         land-acquisitions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      string  true   "LandAcquisition UUID"
// @Param        plan_code  formData  string  false  "Шинэ төлөвлөгөөний дугаар (солих бол shapefile заавал)"
// @Param        start_date formData  string  false  "Эхлэх огноо (YYYY-MM-DD)"
// @Param        end_date   formData  string  false  "Дуусах огноо (YYYY-MM-DD)"
// @Param        shapefile  formData  file    false  "Шинэ хилийн .shp файл"
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

	req := service.UpdateLandAcquisitionRequest{}

	req.PlanCode = c.PostForm("plan_code")

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

	if fh, err := c.FormFile("shapefile"); err == nil {
		f, err := fh.Open()
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileOpen)
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileRead)
			return
		}
		req.ShapefileData = data
	}

	if req.PlanCode != "" && len(req.ShapefileData) == 0 {
		response.Abort(c, http.StatusBadRequest, i18n.ErrShapefileRequiredForPlan)
		return
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
	req.ClearEndDate = c.PostForm("clear_end_date") == "true"
	if s := c.PostForm("construction_type_id"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			req.ConstructionTypeID = &n
		}
	}

	if uid, ok := c.Get("userID"); ok {
		req.UpdatedBy = uid.(uuid.UUID).String()
	}

	acq, geomChanged, err := h.svc.Update(c.Request.Context(), id, req)
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
	}
	r := updateResp{landAcquisitionResponse: toLandAcqResponse(acq)}
	if geomChanged {
		r.Warning = i18n.T(l, i18n.WarnGeomChanged)
	}
	response.OK(c, r)
}

// Delete godoc
// @Summary      Газар чөлөөлөлт устгах
// @Description  Зөвхөн статус нь NEW үед устгах боломжтой
// @Tags         land-acquisitions
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "LandAcquisition UUID"
// @Success      204
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /land-acquisitions/{id} [delete]
func (h *LandHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
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
	response.OK(c, toLandAcqResponse(acq))
}

// SyncParcel godoc
// @Summary      Нэгж талбарын дэлгэрэнгүй мэдээллийг secondary DB-аас татах
// @Description  Эзэмшигч болон эрх зүйн мэдээллийг auth DB-аас татаж main DB-д хадгална
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
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidParcelUUID)
		return
	}

	if err := h.svc.SyncParcel(c.Request.Context(), acqID, parcelID); err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, nil, i18n.MsgSynced)
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
	_ = acqID

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

	changedBy := ""
	if uid, ok := c.Get("userID"); ok {
		changedBy = uid.(uuid.UUID).String()
	}

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

func (h *LandHandler) UpdateParcelMeta(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	var body struct {
		DbChanged       bool   `json:"db_changed"`
		ChangedParcelID string `json:"changed_parcel_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Abort(c, http.StatusBadRequest, "invalid body")
		return
	}

	if err := h.svc.UpdateParcelMeta(c.Request.Context(), parcelID, body.DbChanged, body.ChangedParcelID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
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
	const maxSize = 10 * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	fh, err := c.FormFile("file")
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRead)
		return
	}
	uploadedBy := ""
	if uid, ok := c.Get("userID"); ok {
		uploadedBy = uid.(uuid.UUID).String()
	}
	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	doc, err := h.svc.UploadAcquisitionDocument(c.Request.Context(), id, fh.Filename, contentType, data, uploadedBy)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toDocumentResponse(doc))
}

func (h *LandHandler) DeleteAcquisitionDocument(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteAcquisitionDocument(c.Request.Context(), docID); err != nil {
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
		Years:           years,
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
func (h *LandHandler) ListReportParcels(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if pageSize > 200 {
		pageSize = 200
	}
	var years []int
	for _, s := range c.QueryArray("year") {
		if y, err := strconv.Atoi(s); err == nil && y > 0 {
			years = append(years, y)
		}
	}

	filter := domain.ReportParcelFilter{
		AcquisitionID:    c.Query("acquisition_id"),
		PlanCode:         c.Query("plan_code"),
		AcquisitionName:  c.Query("acquisition_name"),
		AU3Code:          c.Query("au3_code"),
		RightType:        queryInt(c, "right_type", 0),
		Landuse:          c.Query("landuse"),
		Years:            years,
		CompensationType: c.Query("compensation_type"),
		Page:             page,
		PageSize:         pageSize,
	}

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
		ParcelID: c.Query("parcel_id"),
		Page:     page,
		PageSize: pageSize,
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

	b := &domain.Asset{
		AcquisitionID: acqID,
		ParcelID:      body.ParcelID,
		AssetNumber:   body.AssetNumber,
		AssetType:     normalizeAssetRecordType(body.AssetType),
		AssetName:     body.AssetName,
		FloorCount:    body.FloorCount,
		AreaM2:        body.AreaM2,
		OwnerName:     body.OwnerName,
		Address:       body.Address,
		Notes:         body.Notes,
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

	var body struct {
		ParcelID             string  `json:"parcel_id"`
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

	b := &domain.Asset{
		ID:            assetID,
		AcquisitionID: acqID,
		ParcelID:      body.ParcelID,
		AssetNumber:   body.AssetNumber,
		AssetType:     normalizeAssetRecordType(body.AssetType),
		AssetName:     body.AssetName,
		FloorCount:    body.FloorCount,
		AreaM2:        body.AreaM2,
		OwnerName:     body.OwnerName,
		Address:       body.Address,
		Notes:         body.Notes,
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
	assetID, err := uuid.Parse(c.Param("asset_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}

	if err := h.svc.DeleteAsset(c.Request.Context(), assetID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ListCompensations lists compensations for an acquisition
func (h *LandHandler) ListCompensations(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	parcelID := c.Query("parcel_id")
	list, err := h.svc.ListCompensations(c.Request.Context(), acqID, parcelID)
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

	createdBy := ""
	if uid, ok := c.Get("userID"); ok {
		createdBy = uid.(uuid.UUID).String()
	}

	comp := &domain.Compensation{
		AcquisitionID:    acqID,
		TargetType:       normalizeTargetType(body.TargetType),
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
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteCompensation(c.Request.Context(), compID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
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

	createdBy := ""
	if uid, ok := c.Get("userID"); ok {
		createdBy = uid.(uuid.UUID).String()
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
	compID, err := uuid.Parse(c.Param("comp_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteCompensationGrantByCompensationID(c.Request.Context(), compID); err != nil {
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

	createdBy := ""
	if uid, ok := c.Get("userID"); ok {
		createdBy = uid.(uuid.UUID).String()
	}

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

func (h *LandHandler) UploadParcelDocument(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	const maxSize = 10 * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	fh, err := c.FormFile("file")
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRequired)
		return
	}
	if fh.Size > maxSize {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileOpen)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileRead)
		return
	}
	uploadedBy := ""
	if uid, ok := c.Get("userID"); ok {
		uploadedBy = uid.(uuid.UUID).String()
	}
	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	doc, err := h.svc.UploadParcelDocument(c.Request.Context(), parcelID, fh.Filename, contentType, data, uploadedBy)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toDocumentResponse(doc))
}

func (h *LandHandler) DeleteParcelDocument(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("doc_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteParcelDocument(c.Request.Context(), docID); err != nil {
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
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
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
