package http

import (
	"net/http"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DecisionDraftHandler struct {
	svc service.DecisionDraftService
}

func NewDecisionDraftHandler(svc service.DecisionDraftService) *DecisionDraftHandler {
	return &DecisionDraftHandler{svc: svc}
}

// List godoc
// @Summary      Захирамжийн төслийн жагсаалт
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        proposal_no        query  string  false  "Саналын хуудасны дугаар (ILIKE)"
// @Param        decree_number      query  string  false  "Захирамжийн дугаар (ILIKE)"
// @Param        location           query  string  false  "Байршил (ILIKE)"
// @Param        parcel_id          query  string  false  "Холбогдсон нэгж талбарын дугаар (ILIKE)"
// @Param        date_from          query  string  false  "Огноо (YYYY-MM-DD)-с эхлэн"
// @Param        date_to            query  string  false  "Огноо (YYYY-MM-DD) хүртэл"
// @Param        duration_year      query  int     false  "Хугацаа (он)"
// @Param        status             query  int     false  "Төлөв: 1=Төсөл, 2=Баталгаажсан"
// @Param        acquisition_id     query  string  false  "Чөлөөлөлтийн ID"
// @Param        funding_source_id  query  string  false  "Санхүүгийн эх үүсвэрийн ID"
// @Param        work_type_id       query  int     false  "Ажлын төрөл"
// @Param        budget_id          query  int     false  "Төсөв"
// @Param        page               query  int     false  "Хуудасны дугаар (default: 1)"
// @Param        page_size          query  int     false  "Хуудасны хэмжээ (default: 20)"
// @Success      200  {object}  response.PaginatedResponse
// @Router       /decision-drafts [get]
func (h *DecisionDraftHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	filter := domain.DecisionDraftFilter{
		ProposalNo:      c.Query("proposal_no"),
		DecreeNumber:    c.Query("decree_number"),
		Location:        c.Query("location"),
		ParcelID:        c.Query("parcel_id"),
		DateFrom:        c.Query("date_from"),
		DateTo:          c.Query("date_to"),
		DurationYear:    queryInt(c, "duration_year", 0),
		Status:          queryInt(c, "status", 0),
		AcquisitionID:   c.Query("acquisition_id"),
		FundingSourceID: c.Query("funding_source_id"),
		WorkTypeID:      queryInt(c, "work_type_id", 0),
		BudgetID:        queryInt(c, "budget_id", 0),
		Page:            page,
		PageSize:        pageSize,
	}
	// UUID шүүлтүүр буруу форматтай бол SQL-д хүргэлгүй чимээгүй алгасна —
	// эс бөгөөс 500 буцаана.
	if filter.AcquisitionID != "" {
		if _, err := uuid.Parse(filter.AcquisitionID); err != nil {
			filter.AcquisitionID = ""
		}
	}
	if filter.FundingSourceID != "" {
		if _, err := uuid.Parse(filter.FundingSourceID); err != nil {
			filter.FundingSourceID = ""
		}
	}

	list, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, page, pageSize)
}

// GetByID godoc
// @Summary      Захирамжийн төслийн дэлгэрэнгүй
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id} [get]
func (h *DecisionDraftHandler) GetByID(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	d, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, d)
}

// Create godoc
// @Summary      Захирамжийн төсөл үүсгэх
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      201  {object}  response.Response
// @Router       /decision-drafts [post]
func (h *DecisionDraftHandler) Create(c *gin.Context) {
	var req service.CreateDecisionDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	d, err := h.svc.Create(c.Request.Context(), req, decisionActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, d)
}

// Update godoc
// @Summary      Захирамжийн төсөл засварлах (зөвхөн "Төсөл" төлөвт)
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id} [put]
func (h *DecisionDraftHandler) Update(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateDecisionDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	d, err := h.svc.Update(c.Request.Context(), id, req, decisionActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, d)
}

// Delete godoc
// @Summary      Захирамжийн төсөл устгах (зөвхөн "Төсөл" төлөвт)
// @Tags         decision-draft
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      204
// @Router       /decision-drafts/{id} [delete]
func (h *DecisionDraftHandler) Delete(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// Confirm godoc
// @Summary      Захирамжийн явц нэмэх (баталгаажуулах)
// @Description  Захирамжийн дугаар, огноо болон хавсралт файлыг (PDF, заавал) авч баталгаажуулна. Үүнээс хойш өөрчлөлт орохгүй.
// @Description  Хавсралт нь «Захирамж» төрлөөр захирамжид холбогдсон бүх нэгж талбарын хавсралт болно.
// @Tags         decision-draft
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id             path      string  true  "Захирамжийн төслийн ID"
// @Param        decree_number  formData  string  true  "Захирамжийн дугаар"
// @Param        decision_date  formData  string  true  "Захирамжийн огноо (YYYY-MM-DD)"
// @Param        note           formData  string  false "Тайлбар"
// @Param        file           formData  file    true  "Захирамжийн хавсралт (PDF)"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/confirm [post]
func (h *DecisionDraftHandler) Confirm(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize+1<<20)
	req := service.ConfirmDecisionDraftRequest{
		DecreeNumber: c.PostForm("decree_number"),
		DecisionDate: c.PostForm("decision_date"),
		Note:         c.PostForm("note"),
	}
	fh, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return
		}
		response.Error(c, domain.ErrDecisionAttachmentRequired)
		return
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return
	}
	if err != nil || len(data) == 0 {
		response.Error(c, domain.ErrDecisionAttachmentRequired)
		return
	}
	// Захирамжийн эх баримт — нэгж талбарын хавсралтын ерөнхий дүрмээр PDF.
	contentType, ok := allowedPDFContentType(data)
	if !ok {
		response.Error(c, domain.ErrDecisionAttachmentRequired)
		return
	}
	req.Attachment = &service.DecisionAttachment{Filename: fh.Filename, ContentType: contentType, Data: data}

	d, err := h.svc.Confirm(c.Request.Context(), id, req, decisionActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, d)
}

// AddProgress godoc
// @Summary      Захирамжийн явц нэмэх (хянагдаж буй)
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      201  {object}  response.Response
// @Router       /decision-drafts/{id}/progress [post]
func (h *DecisionDraftHandler) AddProgress(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var req service.AddDecisionDraftProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	history, err := h.svc.AddProgress(c.Request.Context(), id, req, decisionActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, history)
}

// ListProgressHistory godoc
// @Summary      Захирамжийн төслийн явцын түүх
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/progress-history [get]
func (h *DecisionDraftHandler) ListProgressHistory(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	list, err := h.svc.ListProgressHistory(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// ListParcels godoc
// @Summary      Захирамжид холбогдсон нэгж талбарууд (хасагдсаныг түүхээр хамт)
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/parcels [get]
func (h *DecisionDraftHandler) ListParcels(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	list, err := h.svc.ListParcels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

type linkParcelRequest struct {
	ParcelUUID string `json:"parcel_uuid" validate:"required"`
	// Тухайн нэгж талбарыг санхүүжүүлэх эх үүсвэр (захирамжид нэмэгдсэн
	// эх үүсвэрийн холбоосын ID). Заавал.
	FundingLinkID string `json:"funding_link_id" validate:"required"`
}

type setParcelFundingRequest struct {
	FundingLinkID string `json:"funding_link_id" validate:"required"`
}

// LinkParcel godoc
// @Summary      Нэгж талбар холбох
// @Description  Зөвхөн "Чөлөөлсөн" төлөвтэй, өөр захирамжид холбогдоогүй нэгж талбар холбогдоно.
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/parcels [post]
func (h *DecisionDraftHandler) LinkParcel(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var req linkParcelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if _, err := uuid.Parse(req.ParcelUUID); err != nil {
		response.Error(c, domain.ErrInvalidInput)
		return
	}
	if err := h.svc.LinkParcel(c.Request.Context(), id, req.ParcelUUID, req.FundingLinkID, decisionActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListParcels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// UnlinkParcel godoc
// @Summary      Нэгж талбарыг захирамжаас хасах (түүх үлдээнэ)
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Захирамжийн төслийн ID"
// @Param        link_id  path  string  true  "Холбоосын ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/parcels/{link_id} [delete]
func (h *DecisionDraftHandler) UnlinkParcel(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	linkID, ok := uuidParam(c, "link_id")
	if !ok {
		return
	}
	if err := h.svc.UnlinkParcel(c.Request.Context(), id, linkID, decisionActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListParcels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// ListByParcel godoc
// @Summary      Нэгж талбарт холбогдсон захирамжууд (нэгж талбарын "Захирамж" таб)
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Нэгж талбарын ID (UUID)"
// @Success      200  {object}  response.Response
// @Router       /parcels/{id}/decision-drafts [get]
func (h *DecisionDraftHandler) ListByParcel(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	list, err := h.svc.ListByParcel(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// SetParcelFundingSource godoc
// @Summary      Холбогдсон нэгж талбарын санхүүгийн эх үүсвэрийг солих
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Захирамжийн төслийн ID"
// @Param        link_id  path  string  true  "Нэгж талбарын холбоосын ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/parcels/{link_id}/funding-source [patch]
func (h *DecisionDraftHandler) SetParcelFundingSource(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	linkID, ok := uuidParam(c, "link_id")
	if !ok {
		return
	}
	var req setParcelFundingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if _, err := uuid.Parse(req.FundingLinkID); err != nil {
		response.Error(c, domain.ErrFundingSourceRequired)
		return
	}
	if err := h.svc.SetParcelFundingSource(c.Request.Context(), id, linkID, req.FundingLinkID); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListParcels(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

type addFundingSourceRequest struct {
	FundingSourceID string  `json:"funding_source_id" validate:"required"`
	Amount          float64 `json:"amount"`
}

// AddFundingSource godoc
// @Summary      Захирамжид санхүүгийн эх үүсвэр нэмэх
// @Description  Төслийг үүсгэсний дараа олон эх үүсвэр нэмж болно.
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/funding-sources [post]
func (h *DecisionDraftHandler) AddFundingSource(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	var req addFundingSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if _, err := uuid.Parse(req.FundingSourceID); err != nil {
		response.Error(c, domain.ErrInvalidInput)
		return
	}
	if req.Amount < 0 {
		response.Error(c, domain.ErrInvalidInput)
		return
	}
	if err := h.svc.AddFundingSource(c.Request.Context(), id, req.FundingSourceID, req.Amount, decisionActor(c)); err != nil {
		response.Error(c, err)
		return
	}
	h.respondFundingSources(c, id)
}

// RemoveFundingSource godoc
// @Summary      Захирамжаас санхүүгийн эх үүсвэр хасах
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Захирамжийн төслийн ID"
// @Param        link_id  path  string  true  "Холбоосын ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/funding-sources/{link_id} [delete]
func (h *DecisionDraftHandler) RemoveFundingSource(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	linkID, ok := uuidParam(c, "link_id")
	if !ok {
		return
	}
	if err := h.svc.RemoveFundingSource(c.Request.Context(), id, linkID); err != nil {
		response.Error(c, err)
		return
	}
	h.respondFundingSources(c, id)
}

// ListDraftFundingSources godoc
// @Summary      Захирамжид нэмэгдсэн санхүүгийн эх үүсвэрүүд
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Захирамжийн төслийн ID"
// @Success      200  {object}  response.Response
// @Router       /decision-drafts/{id}/funding-sources [get]
func (h *DecisionDraftHandler) ListDraftFundingSources(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	h.respondFundingSources(c, id)
}

// respondFundingSources — нэмэх/хасах/жагсаах бүгд шинэчлэгдсэн жагсаалтыг
// буцаадаг тул нэг дор задлав (LinkParcel-ийн адил хэв маяг).
func (h *DecisionDraftHandler) respondFundingSources(c *gin.Context, draftID string) {
	list, err := h.svc.ListDraftFundingSources(c.Request.Context(), draftID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// CreateFundingSourceOption godoc
// @Summary      Шинэ санхүүгийн эх үүсвэр бүртгэх (гараар)
// @Description  Захирамжийн төслөөс шууд шинэ эх үүсвэр бичиж оруулна. funding_sources хүснэгтэд чөлөөлөлтгүй мөр болж бичигдэнэ.
// @Tags         decision-draft
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      201  {object}  response.Response
// @Router       /funding-sources [post]
func (h *DecisionDraftHandler) CreateFundingSourceOption(c *gin.Context) {
	var req service.CreateFundingSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	o, err := h.svc.CreateFundingSourceOption(c.Request.Context(), req, decisionActor(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, o)
}

// ListFundingSourceOptions godoc
// @Summary      Санхүүгийн эх үүсвэрийн сонголт (бүх чөлөөлөлтөөр)
// @Tags         decision-draft
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /funding-sources [get]
func (h *DecisionDraftHandler) ListFundingSourceOptions(c *gin.Context) {
	list, err := h.svc.ListFundingSourceOptions(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

// uuidParam — path параметрийг UUID гэж шалгаж буцаана. Буруу бол 400 бичээд
// false буцаана (дуудагч тэр дор нь return хийнэ).
func uuidParam(c *gin.Context, name string) (string, bool) {
	v := c.Param(name)
	if _, err := uuid.Parse(v); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return "", false
	}
	return v, true
}

// decisionActor — JWT context дэх хэрэглэгчийн ID-г мөр болгоно.
func decisionActor(c *gin.Context) string {
	return contextUserIDString(c)
}
