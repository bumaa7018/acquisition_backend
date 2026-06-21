package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/i18n"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
)

type PlanHandler struct {
	svc service.PlanService
}

func NewPlanHandler(svc service.PlanService) *PlanHandler {
	return &PlanHandler{svc: svc}
}

type planResponse struct {
	ID          string  `json:"id"`
	PlanCode    string  `json:"plan_code"`
	Name        string  `json:"name"`
	AreaM2      float64 `json:"area_m2"`
	Status      int     `json:"status"`
	BoundaryWKT string  `json:"boundary_wkt"`
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
		ID:          plan.ID.String(),
		PlanCode:    plan.PlanCode,
		Name:        plan.Name,
		AreaM2:      plan.AreaM2,
		Status:      plan.Status,
		BoundaryWKT: plan.BoundaryWKT,
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
			PlanCode: p.PlanCode,
			Name:     p.Name,
		})
	}
	response.OK(c, result)
}
