package http

import (
	"net/http"
	"strconv"
	"strings"

	"acquisition_backend/internal/domain"
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ToStatusID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_status_id required"})
		return
	}
	w, err := h.svc.CreateParcelWorkflow(c.Request.Context(), req.FromStatusID, req.ToStatusID, req.SortOrder)
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.StatusID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status_id required"})
		return
	}
	createdBy := "system"
	if id := contextUserIDString(c); id != "" {
		createdBy = id
	}
	if err := h.svc.UpdateParcelStatus(c.Request.Context(), parcelID, acqID, *req.StatusID, createdBy); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "ok"})
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

// TransitionValuationSubmission — submit | approve | return. Бүх шилжилтэд тайлбар шаардлагатай.
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
	var req struct {
		Action        string `json:"action"`
		Note          string `json:"note"`
		ValuationType string `json:"valuation_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	req.Note = strings.TrimSpace(req.Note)
	if req.Note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Тайлбар бичих шаардлагатай"})
		return
	}

	// Ролийн зохицуулалт: зөвшөөрөх/буцаах = санхүү (эсвэл ахлах); илгээх = мэрг.
	// байгууллага (эсвэл ахлах), МИКА бол ЗӨВХӨН өөрийн "mika" урсгалыг илгээнэ.
	isFinance := ctxHasRole(c, "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү")
	isSenior := ctxHasRole(c, "senior_specialist", "Ахлах мэргэжилтэн")
	isProf := ctxHasRole(c, "professional_org", "Мэргэжлийн байгууллага")
	isMika := ctxHasRole(c, "mika", "МИКА")
	switch req.Action {
	case "approve", "return":
		if !isFinance && !isSenior {
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

	userID := "system"
	if id := contextUserIDString(c); id != "" {
		userID = id
	}
	sub, err := h.svc.TransitionValuationSubmission(c.Request.Context(), acqID, parcelID, req.ValuationType, req.Action, req.Note, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, sub)
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
