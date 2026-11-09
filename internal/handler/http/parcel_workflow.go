package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ctxHasRole — context дахь ролиудын аль нэг нь өгсөн нэрстэй таарч байвал true.
func ctxHasRole(c *gin.Context, names ...string) bool {
	rolesAny, ok := c.Get("roles")
	if !ok {
		return false
	}
	roles, ok := rolesAny.([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		for _, n := range names {
			if r == n {
				return true
			}
		}
	}
	return false
}

type ParcelWorkflowHandler struct {
	svc service.LandService
}

func NewParcelWorkflowHandler(svc service.LandService) *ParcelWorkflowHandler {
	return &ParcelWorkflowHandler{svc: svc}
}

func (h *ParcelWorkflowHandler) List(c *gin.Context) {
	list, err := h.svc.ListParcelWorkflows(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []*domain.ParcelWorkflow{}
	}
	response.OK(c, list)
}

func (h *ParcelWorkflowHandler) Create(c *gin.Context) {
	var req struct {
		FromStatusID *int `json:"from_status_id"`
		ToStatusID   int  `json:"to_status_id"`
		SortOrder    int  `json:"sort_order"`
		// DocumentTypeIDs — энэ шилжилтэд ЗААВАЛ байх хавсралт (заавал биш).
		DocumentTypeIDs []int `json:"document_type_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ToStatusID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_status_id required"})
		return
	}
	w, err := h.svc.CreateParcelWorkflow(c.Request.Context(), req.FromStatusID, req.ToStatusID, req.SortOrder, req.DocumentTypeIDs)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 201, "data": w})
}

func (h *ParcelWorkflowHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteParcelWorkflow(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *ParcelWorkflowHandler) GetAvailableStatuses(c *gin.Context) {
	parcelIDStr := c.Param("parcel_id")
	parcelID, err := uuid.Parse(parcelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	acqIDStr := c.Param("id")
	acqID, err := uuid.Parse(acqIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	pf, err := h.svc.GetParcelDetail(c.Request.Context(), acqID, parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	statuses, err := h.svc.GetAvailableParcelStatuses(c.Request.Context(), pf.StatusID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if statuses == nil {
		statuses = []*domain.ParcelStatus{}
	}
	response.OK(c, statuses)
}

func (h *ParcelWorkflowHandler) UpdateParcelStatus(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	var req struct {
		StatusID *int `json:"status_id"`
		// Reason — төлөв солих ШАЛТГААН. "Нөлөөлөгдсөн гарсан"/"Татгалзсан"
		// үед ЗААВАЛ (доор шалгана), бусад төлөвт хоосон байж болно.
		Reason string `json:"reason"`
	}
	// ХОЁР хэлбэрийг дэмжинэ:
	//   application/json    — өмнөх шигээ (хавсралтгүй)
	//   multipart/form-data — status_id/reason + `file`
	//
	// "Татгалзсан" төлөвт хавсралт ЗААВАЛ (шаардлагыг service шалгана) тул
	// тэр шилжилт ҮРГЭЛЖ multipart-аар ирнэ.
	var attachment *service.ParcelStatusAttachment
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)
		if err := c.Request.ParseMultipartForm(maxDocumentUploadSize); err != nil {
			if isBodyTooLarge(err) {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Хүсэлтийг уншиж чадсангүй"})
			return
		}
		if raw := strings.TrimSpace(c.PostForm("status_id")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "status_id required"})
				return
			}
			req.StatusID = &n
		}
		req.Reason = c.PostForm("reason")

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
			// Хавсралт нь нэгж талбарын БАРИМТ болж бүртгэгдэх тул баримтын
			// ерөнхий дүрэмтэй ижил: PDF (агуулгаар шалгана).
			contentType, ok := allowedPDFContentType(data)
			if !ok {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
				return
			}
			// document_type_id — хавсралтын төрөл (заавал биш). Явцад заавал
			// хавсралт тохируулсан бол тэр төрлөөр бүртгэгдэж шаардлагыг хангана.
			var docTypeID int
			if raw := strings.TrimSpace(c.PostForm("document_type_id")); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "document_type_id буруу"})
					return
				}
				docTypeID = n
			}
			attachment = &service.ParcelStatusAttachment{
				DocumentTypeID: docTypeID,
				Filename:       fh.Filename,
				// file_name — хэрэглэгчийн засварласан ХАРАГДАХ нэр (заавал биш).
				DisplayName: strings.TrimSpace(c.PostForm("file_name")),
				ContentType: contentType,
				Data:        data,
			}
		}
	} else if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.StatusID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status_id required"})
		return
	}
	reason := strings.TrimSpace(req.Reason)
	// «Шалтгаан заавал» (requires_reason) төлөвт шалтгаангүй шилжихийг
	// хориглоно — дараа нь яагаад гэдгийг тайлбарлах өгөгдөл үлдэхгүй болно.
	// Өмнө Нөлөөлөгдсөн гарсан/Татгалзсан гэж ДУГААРААР шалгадаг байв.
	if reason == "" {
		if st, err := h.svc.GetParcelStatusRules(c.Request.Context(), *req.StatusID); err == nil && st.RequiresReason {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Шалтгаан заавал бөглөнө"})
			return
		}
	}
	createdBy := contextUserDisplayName(c)
	err = h.svc.UpdateParcelStatus(c.Request.Context(), domain.ParcelStatusChange{
		ParcelID:      parcelID,
		AcquisitionID: acqID,
		StatusID:      *req.StatusID,
		CreatedBy:     createdBy,
		Reason:        reason,
	}, attachment)
	if err != nil {
		requiredDocumentsError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "ok"})
}

// requiredDocumentsError — дутуу заавал хавсралтыг НЭРСТЭЙ нь 422 болгоно;
// бусад алдаа ердийн замаар.
func requiredDocumentsError(c *gin.Context, err error) {
	var missing *domain.RequiredDocumentsMissingError
	if errors.As(err, &missing) {
		msg := missing.Error()
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, response.Response{
			Code: http.StatusUnprocessableEntity, Error: msg, Message: msg,
		})
		return
	}
	response.Error(c, err)
}

// SetRequiredDocuments — шилжилтийн заавал хавсралтыг БҮХЭЛД нь солино.
// Body: {"document_type_ids": [1, 2]} (хоосон = шаардлагагүй).
func (h *ParcelWorkflowHandler) SetRequiredDocuments(c *gin.Context) {
	workflowID, err := strconv.Atoi(c.Param("id"))
	if err != nil || workflowID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		DocumentTypeIDs []int `json:"document_type_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetParcelWorkflowRequiredDocuments(c.Request.Context(), workflowID, req.DocumentTypeIDs); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "ok"})
}

// GetStatusRequirements — нэгж талбарыг ?status_id=-д шилжүүлэхэд шаардлагатай
// хавсралт бүр орсон эсэх (явц солих цонхонд).
func (h *ParcelWorkflowHandler) GetStatusRequirements(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	statusID, err := strconv.Atoi(c.Query("status_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status_id required"})
		return
	}
	// Нэгж талбар энэ чөлөөлөлтийнх эсэхийг шалгана (GetAvailableStatuses-тэй ижил).
	if _, err := h.svc.GetParcelDetail(c.Request.Context(), acqID, parcelID); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ParcelStatusRequirements(c.Request.Context(), acqID, parcelID, statusID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []domain.ParcelStatusRequirement{}
	}
	response.OK(c, list)
}

// ── Нөхөх олговрын үнэлгээний илгээх/зөвшөөрөх төлөв ────────────────────────────

func (h *ParcelWorkflowHandler) GetValuationSubmission(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	sub, err := h.svc.GetValuationSubmission(c.Request.Context(), acqID, parcelID, c.Query("valuation_type"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, sub)
}

// TransitionValuationSubmission — submit | approve | return | cancel. Бүх шилжилтэд тайлбар шаардлагатай.
func (h *ParcelWorkflowHandler) TransitionValuationSubmission(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	// ХОЁР хэлбэрийг дэмжинэ:
	//   application/json      — өмнөх шигээ (хавсралтгүй)
	//   multipart/form-data   — action/note/valuation_type + ЗААВАЛ БИШ `file`
	//
	// Хавсралт нь буцаалтын шалтгааныг гэрчилдэг (залруулга шаардсан хуудас,
	// дүнгийн зөрүүний хүснэгт) тул шилжилттэй НЭГ хүсэлтэд явна: тусад нь
	// байвал шилжилт хийгдээд хавсралт унах эсвэл эсрэгээр болно.
	var req struct {
		Action        string `json:"action"`
		Note          string `json:"note"`
		ValuationType string `json:"valuation_type"`
	}
	var attachment *service.ValuationAttachment
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize)
		if err := c.Request.ParseMultipartForm(maxDocumentUploadSize); err != nil {
			if isBodyTooLarge(err) {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Хүсэлтийг уншиж чадсангүй"})
			return
		}
		req.Action = c.PostForm("action")
		req.Note = c.PostForm("note")
		req.ValuationType = c.PostForm("valuation_type")

		// `file` — ЗААВАЛ БИШ. Огт өгөөгүй бол хавсралтгүй шилжилт.
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
			// Хавсралт нь ЗӨВХӨН буцаах/цуцлах үйлдлийнх (үндэслэлийн PDF).
			// Илгээх үед файл АВАХГҮЙ: баталгаажсан тайлан ба ажлын зураг нь
			// нэгж талбарын баримт (parcel_document) болж тусдаа хавсрагдана.
			contentType, ok := allowedPDFContentType(data)
			if !ok {
				response.Abort(c, http.StatusBadRequest, i18n.ErrFileType)
				return
			}
			attachment = &service.ValuationAttachment{
				Filename:    fh.Filename,
				ContentType: contentType,
				Data:        data,
			}
		}
	} else if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	req.Note = strings.TrimSpace(req.Note)
	if req.Note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Тайлбар бичих шаардлагатай"})
		return
	}
	// Ролийн зохицуулалт: зөвшөөрөх/буцаах = санхүү (эсвэл ахлах); цуцлах =
	// санхүү, ахлах эсвэл тухайн чөлөөлөлт дээр ажиллаж буй дотоод ажилтан;
	// илгээх = мэрг. байгууллага (эсвэл ахлах), МИКА бол ЗӨВХӨН өөрийн "mika"
	// урсгалыг илгээнэ.
	isFinance := ctxHasRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү")
	isSenior := ctxHasRole(c, "admin", "Админ", "senior_specialist", "Ахлах мэргэжилтэн", "department_head")
	// Үнэлгээний байгууллага — роль БИШ, токены `valuation_org` тэмдгээр.
	isProf := middleware.IsValuationOrgCaller(c)
	isMika := ctxHasRole(c, "mika", "МИКА")
	switch req.Action {
	case "approve", "return":
		if !isFinance && !isSenior {
			response.Error(c, domain.ErrForbidden)
			return
		}
	case "cancel":
		isAssigned := false
		if !isFinance && !isSenior && !isProf && !isMika && !ctxHasRole(c, "professional_org", "Мэргэжлийн байгууллага", "Мэргэжлийн байгуулл...") {
			if callerID, ok := middleware.GetCallerID(c); ok {
				assigned, err := h.svc.IsAssigned(c.Request.Context(), acqID, callerID)
				if err != nil {
					response.Error(c, err)
					return
				}
				isAssigned = assigned
			}
		}
		if !isFinance && !isSenior && !isAssigned {
			response.Error(c, domain.ErrForbidden)
			return
		}
	case "submit":
		mikaOwnStream := isMika && req.ValuationType == domain.ValuationTypeMika
		if !isProf && !isSenior && !mikaOwnStream {
			response.Error(c, domain.ErrForbidden)
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}
	// Хавсралт нь ЗӨВХӨН буцаалт, цуцлалтад хамаарна — бусад үйлдэлд ирвэл
	// хаяна (илгээх дээрх файл буцаалт/цуцлалтын хавсралт болж харагдахаас
	// сэргийлнэ). Цуцлалтад эрх шалгасны дараа заавал PDF хавсралт шаардана.
	if req.Action != "return" && req.Action != "cancel" {
		attachment = nil
	}
	if req.Action == "cancel" && attachment == nil {
		response.Error(c, domain.ErrValuationCancelAttachmentRequired)
		return
	}

	userID := contextUserDisplayName(c)
	sub, err := h.svc.TransitionValuationSubmission(c.Request.Context(), domain.ValuationTransition{
		AcquisitionID: acqID,
		ParcelID:      parcelID,
		ValuationType: req.ValuationType,
		Action:        req.Action,
		Note:          req.Note,
		UserID:        userID,
	}, attachment)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, sub)
}

func (h *ParcelWorkflowHandler) ListValuationSnapshots(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	list, err := h.svc.ListValuationSnapshots(c.Request.Context(), acqID, parcelID, c.Query("valuation_type"))
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []*domain.ValuationSnapshot{}
	}
	response.OK(c, list)
}

func (h *ParcelWorkflowHandler) ListValuationSubmissionHistory(c *gin.Context) {
	acqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid acquisition_id"})
		return
	}
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	list, err := h.svc.ListValuationSubmissionHistory(c.Request.Context(), acqID, parcelID, c.Query("valuation_type"))
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []*domain.ValuationSubmissionHistory{}
	}
	response.OK(c, list)
}

func (h *ParcelWorkflowHandler) ListParcelStatusHistory(c *gin.Context) {
	parcelID, err := uuid.Parse(c.Param("parcel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parcel_id"})
		return
	}
	list, err := h.svc.ListParcelStatusHistory(c.Request.Context(), parcelID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []*domain.ParcelStatusHistory{}
	}
	response.OK(c, list)
}
