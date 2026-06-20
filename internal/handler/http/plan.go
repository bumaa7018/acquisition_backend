package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/go-api-template/internal/domain"
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "code параметр шаардлагатай"})
		return
	}

	plan, err := h.svc.Search(c.Request.Context(), code)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			c.AbortWithStatusJSON(http.StatusNotFound, response.Response{Error: "Төлөвлөгөө олдсонгүй"})
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
