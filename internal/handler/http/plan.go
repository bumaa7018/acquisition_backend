package http

import (
	"net/http"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type PlanHandler struct {
	svc service.PlanService
}

func NewPlanHandler(svc service.PlanService) *PlanHandler {
	return &PlanHandler{svc: svc}
}

type planResponse struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"project_id,omitempty"`
	Code         string  `json:"code"`
	PlanCode     string  `json:"plan_code"`
	Name         string  `json:"name"`
	PlanTypeName string  `json:"plan_type_name,omitempty"`
	AreaM2       float64 `json:"area_m2"`
	Status       int     `json:"status"`
	BoundaryWKT  string  `json:"boundary_wkt"`
	StartDate    *string `json:"start_date,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
	StartData    *string `json:"start_data,omitempty"`
	EndData      *string `json:"end_data,omitempty"`
	ApprovedDate *string `json:"approved_date,omitempty"`
}

// Search godoc
// @Summary      Төлөвлөгөө дугаараар хайх
// @Tags         plans
// @Produce      json
// @Security     BearerAuth
// @Param        code  query  string  true  "Төлөвлөгөөний дугаар"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /plans/search [get]
func (h *PlanHandler) Search(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrCodeRequired)
		return
	}

	plan, err := h.svc.Search(c.Request.Context(), code)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			response.Abort(c, http.StatusNotFound, i18n.ErrPlanNotFound)
		default:
			response.Error(c, err)
		}
		return
	}

	response.OK(c, planResponse{
		ID:           plan.ID.String(),
		ProjectID:    plan.ProjectID,
		Code:         planCode(plan),
		PlanCode:     plan.PlanCode,
		Name:         plan.Name,
		PlanTypeName: plan.PlanTypeName,
		AreaM2:       plan.AreaM2,
		Status:       plan.Status,
		BoundaryWKT:  plan.BoundaryWKT,
		StartDate:    plan.StartData,
		EndDate:      plan.EndData,
		StartData:    plan.StartData,
		EndData:      plan.EndData,
		ApprovedDate: plan.ApprovedDate,
	})
}

// Suggest godoc
// @Summary      Төлөвлөгөө дугаар/нэрээр хайх (ILIKE)
// @Tags         plans
// @Produce      json
// @Security     BearerAuth
// @Param        q  query  string  true  "Хайх үг"
// @Success      200  {object}  response.Response
// @Router       /plans/suggest [get]
func (h *PlanHandler) Suggest(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		response.OK(c, []planResponse{})
		return
	}
	plans, err := h.svc.Suggest(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	result := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		result = append(result, planResponse{
			ID:       p.PlanCode,
			Code:     planCode(p),
			PlanCode: p.PlanCode,
			Name:     p.Name,
		})
	}
	response.OK(c, result)
}

func planCode(plan *domain.Plan) string {
	if plan.Code != "" {
		return plan.Code
	}
	return plan.PlanCode
}
