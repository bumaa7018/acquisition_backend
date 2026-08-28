package http

import (
	"net/http"
	"strings"

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
	ID           string `json:"id"`
	ProjectID    string `json:"project_id,omitempty"`
	Code         string `json:"code"`
	PlanCode     string `json:"plan_code"`
	Name         string `json:"name"`
	PlanTypeName string `json:"plan_type_name,omitempty"`
	// ParcelID — төлөвлөгөөний нэгж талбарын дугаар (үүгээр хайдаг)
	ParcelID string `json:"parcel_id,omitempty"`
	// Gazner — бүтээн байгуулалтын ажлын нэр
	Gazner string  `json:"gazner,omitempty"`
	AreaM2 float64 `json:"area_m2"`
	Status int     `json:"status"`
	// HasBoundary — хил байгаа ЭСЭХ. Чөлөөлөлтийн хил нь төлөвлөгөөнөөс
	// хуулагддаг тул хилгүй төлөвлөгөөгөөр цааш үргэлжлэх боломжгүй; дэлгэц
	// үүнийг хайлтын мөчид анхааруулахад ашиглана (Search-д л утга дүүрнэ).
	HasBoundary bool `json:"has_boundary"`
	// BoundaryWKT — хилийн геометр. Анхдагчаар буцаагддаггүй: хариултыг олон
	// зуун КБ болгож хүндрүүлдэг ба хайлтын сонголтод (suggest) хэрэггүй.
	// ЗӨВХӨН `?with_boundary=1` үед дүүрнэ — дэлгэц олдсон төлөвлөгөөний хилийг
	// газрын зураг дээр зурахад ашиглана.
	BoundaryWKT  string  `json:"boundary_wkt,omitempty"`
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
// @Param        with_boundary  query  bool  false  "1/true бол хилийн геометрийг (boundary_wkt) хамт буцаана — дэлгэц дээр зурахад"
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
	// Хил нь том тул ЗӨВХӨН хүссэн үед буцаана (дэлгэц зурахад л хэрэгтэй).
	withBoundary := c.Query("with_boundary") == "1" || c.Query("with_boundary") == "true"

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

	boundary := strings.TrimSpace(plan.BoundaryWKT)
	responseBoundary := ""
	if withBoundary {
		responseBoundary = boundary
	}

	response.OK(c, planResponse{
		ID:           plan.ID.String(),
		ProjectID:    plan.ProjectID,
		Code:         planCode(plan),
		PlanCode:     plan.PlanCode,
		Name:         plan.Name,
		PlanTypeName: plan.PlanTypeName,
		ParcelID:     plan.ParcelID,
		Gazner:       plan.Gazner,
		AreaM2:       plan.AreaM2,
		Status:       plan.Status,
		HasBoundary:  boundary != "",
		BoundaryWKT:  responseBoundary,
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
			ParcelID: p.ParcelID,
			Gazner:   p.Gazner,
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
