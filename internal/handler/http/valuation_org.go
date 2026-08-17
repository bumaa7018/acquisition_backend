package http

import (
	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

// Үнэлгээний (мэргэжлийн) байгууллагын бүртгэл — зөвхөн app-owned profile.
type ValuationOrgHandler struct {
	svc service.ValuationOrgService
}

func NewValuationOrgHandler(svc service.ValuationOrgService) *ValuationOrgHandler {
	return &ValuationOrgHandler{svc: svc}
}

type valuationOrgBody struct {
	Name             string  `json:"name"`
	ShortName        string  `json:"short_name"`
	RegisterNo       string  `json:"register_no"`
	LicenseNo        string  `json:"license_no"`
	LicenseIssuedAt  *string `json:"license_issued_at"`
	LicenseExpiresAt *string `json:"license_expires_at"`
	Phone            string  `json:"phone"`
	Email            string  `json:"email"`
	Address          string  `json:"address"`
	Note             string  `json:"note"`
	IsActive         *bool   `json:"is_active"`
}

func valuationOrgReqFromBody(b valuationOrgBody) (service.ValuationOrgRequest, error) {
	issuedAt, err := parseOptionalDate(b.LicenseIssuedAt)
	if err != nil {
		return service.ValuationOrgRequest{}, err
	}
	expiresAt, err := parseOptionalDate(b.LicenseExpiresAt)
	if err != nil {
		return service.ValuationOrgRequest{}, err
	}
	isActive := true
	if b.IsActive != nil {
		isActive = *b.IsActive
	}
	req := service.ValuationOrgRequest{
		Org: domain.ValuationOrg{
			Name:             b.Name,
			ShortName:        b.ShortName,
			RegisterNo:       b.RegisterNo,
			LicenseNo:        b.LicenseNo,
			LicenseIssuedAt:  issuedAt,
			LicenseExpiresAt: expiresAt,
			Phone:            b.Phone,
			Email:            b.Email,
			Address:          b.Address,
			Note:             b.Note,
			IsActive:         isActive,
		},
	}
	return req, nil
}

// List godoc
// @Summary      Үнэлгээний байгууллагын жагсаалт
// @Tags         ValuationOrgs
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /valuation-orgs [get]
func (h *ValuationOrgHandler) List(c *gin.Context) {
	f := domain.ValuationOrgFilter{
		Search:   c.Query("search"),
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
	}
	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		f.IsActive = &active
	}
	list, total, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, f.Page, f.PageSize)
}

// GetByID godoc
// @Summary      Үнэлгээний байгууллагын profile
// @Tags         ValuationOrgs
// @Security     BearerAuth
// @Produce      json
// @Param        id   path  int  true  "Байгууллагын ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /valuation-orgs/{id} [get]
func (h *ValuationOrgHandler) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	item, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}

// Create godoc
// @Summary      Үнэлгээний байгууллагын profile бүртгэх
// @Tags         ValuationOrgs
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      201  {object}  map[string]interface{}
// @Router       /valuation-orgs [post]
func (h *ValuationOrgHandler) Create(c *gin.Context) {
	var b valuationOrgBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	req, err := valuationOrgReqFromBody(b)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}

// Update godoc
// @Summary      Үнэлгээний байгууллагын profile шинэчлэх
// @Tags         ValuationOrgs
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path  int  true  "Байгууллагын ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /valuation-orgs/{id} [put]
func (h *ValuationOrgHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b valuationOrgBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	req, err := valuationOrgReqFromBody(b)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}

// Delete godoc
// @Summary      Үнэлгээний байгууллагыг устгах
// @Tags         ValuationOrgs
// @Security     BearerAuth
// @Produce      json
// @Param        id   path  int  true  "Байгууллагын ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /valuation-orgs/{id} [delete]
func (h *ValuationOrgHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
