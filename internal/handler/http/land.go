package http

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
)


type LandHandler struct {
	svc service.LandService
}

func NewLandHandler(svc service.LandService) *LandHandler {
	return &LandHandler{svc: svc}
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
	planCode       := c.PostForm("plan_code")
	startStr       := c.PostForm("start_date")
	endStr         := c.PostForm("end_date")
	projectName    := c.PostForm("project_name")
	implementingOrg := c.PostForm("implementing_org")
	reason         := c.PostForm("reason")
	responsibleOrg := c.PostForm("responsible_org")
	fundingSource  := c.PostForm("funding_source")

	if planCode == "" || startStr == "" || endStr == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{
			Error: "plan_code, start_date, end_date заавал шаардлагатай",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "start_date формат буруу (YYYY-MM-DD)"})
		return
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "end_date формат буруу (YYYY-MM-DD)"})
		return
	}

	fh, err := c.FormFile("shapefile")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "shapefile файл шаардлагатай"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "shapefile нээхэд алдаа"})
		return
	}
	defer f.Close()

	zipData, err := io.ReadAll(f)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "shapefile уншихад алдаа"})
		return
	}

	createdByStr := ""
	if uid, ok := c.Get("userID"); ok {
		createdByStr = uid.(uuid.UUID).String()
	}

	acq, err := h.svc.Create(c.Request.Context(), service.CreateLandAcquisitionRequest{
		PlanCode:        planCode,
		StartDate:       startDate,
		EndDate:         endDate,
		ShapefileData:   zipData,
		CreatedBy:       createdByStr,
		ProjectName:     projectName,
		ImplementingOrg: implementingOrg,
		Reason:          reason,
		ResponsibleOrg:  responsibleOrg,
		FundingSource:   fundingSource,
	})
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			c.AbortWithStatusJSON(http.StatusNotFound, response.Response{Error: "Төлөвлөгөө олдсонгүй"})
		case domain.ErrInvalidInput:
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, response.Response{
				Error: "Чөлөөлөх хил нь төлөвлөгөөний хилээс гарсан эсвэл shapefile формат буруу",
			})
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу UUID формат"})
		return
	}

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)

	filter := domain.ParcelFilter{
		ParcelID:  c.Query("parcel_id"),
		AU1Code:   c.Query("au1_code"),
		AU2Code:   c.Query("au2_code"),
		AU3Code:   c.Query("au3_code"),
		RightType: c.Query("right_type"),
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
		result = append(result, parcelResponse{
			ID: p.ID.String(), ParcelID: p.ParcelID,
			AU1Code: p.AU1Code, AU2Code: p.AU2Code, AU3Code: p.AU3Code,
			RightType: p.RightType, Landuse: p.Landuse, AreaM2: p.AreaM2,
			AcquisitionAreaM2: p.AcquisitionAreaM2,
		})
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу UUID формат"})
		return
	}

	req := service.UpdateLandAcquisitionRequest{}

	req.PlanCode = c.PostForm("plan_code")

	if s := c.PostForm("start_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "start_date формат буруу (YYYY-MM-DD)"})
			return
		}
		req.StartDate = &t
	}
	if s := c.PostForm("end_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "end_date формат буруу (YYYY-MM-DD)"})
			return
		}
		req.EndDate = &t
	}

	if fh, err := c.FormFile("shapefile"); err == nil {
		f, err := fh.Open()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "shapefile нээхэд алдаа"})
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "shapefile уншихад алдаа"})
			return
		}
		req.ShapefileData = data
	}

	if req.PlanCode != "" && len(req.ShapefileData) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{
			Error: "Төлөвлөгөө солих бол шинэ shapefile заавал шаардлагатай",
		})
		return
	}

	if uid, ok := c.Get("userID"); ok {
		req.UpdatedBy = uid.(uuid.UUID).String()
	}

	acq, geomChanged, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			c.AbortWithStatusJSON(http.StatusNotFound, response.Response{Error: "Чөлөөлөлт олдсонгүй"})
		case domain.ErrInvalidInput:
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, response.Response{
				Error: "Шинэ хил нь төлөвлөгөөний хилээс гарсан эсвэл shapefile формат буруу",
			})
		default:
			response.Error(c, err)
		}
		return
	}

	type updateResp struct {
		landAcquisitionResponse
		Warning string `json:"warning,omitempty"`
	}
	r := updateResp{landAcquisitionResponse: toLandAcqResponse(acq)}
	if geomChanged {
		r.Warning = "Хуучин бүсийн хилтэй давхцах нэгж талбарын бүх мэдээлэл устгагдаж шинэчлэгдлээ"
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу UUID формат"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		switch err {
		case domain.ErrNotFound:
			c.AbortWithStatusJSON(http.StatusNotFound, response.Response{Error: "Чөлөөлөлт олдсонгүй"})
		case domain.ErrForbidden:
			c.AbortWithStatusJSON(http.StatusForbidden, response.Response{
				Error: "Зөвхөн NEW статустай чөлөөлөлтийг устгах боломжтой",
			})
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу UUID формат"})
		return
	}

	acq, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, domain.ErrNotFound)
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу acquisition UUID формат"})
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу parcel UUID формат"})
		return
	}

	if err := h.svc.SyncParcel(c.Request.Context(), acqID, parcelID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Нэгж талбарын мэдээлэл амжилттай татагдлаа"})
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу acquisition UUID формат"})
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "Буруу parcel UUID формат"})
		return
	}

	pf, err := h.svc.GetParcelDetail(c.Request.Context(), acqID, parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toParcelFullResponse(pf))
}

