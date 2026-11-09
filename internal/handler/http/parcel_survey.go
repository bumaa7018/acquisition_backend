package http

import (
	"net/http"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ParcelSurveyHandler — чөлөөлөгдсөн газрын судалгаа (нэгж талбарт нэг бүртгэл,
// нэгж талбар үүсэхэд автоматаар үүснэ).
type ParcelSurveyHandler struct {
	svc service.ParcelSurveyService
}

func NewParcelSurveyHandler(svc service.ParcelSurveyService) *ParcelSurveyHandler {
	return &ParcelSurveyHandler{svc: svc}
}

type parcelSurveyRequest struct {
	ChimneyCount      *int   `json:"chimney_count"`
	ToiletCount       *int   `json:"toilet_count"`
	CohabitantCount   *int   `json:"cohabitant_count"`
	RelocationType    string `json:"relocation_type"`
	RelocationAddress string `json:"relocation_address"`
	Note              string `json:"note"`
}

type parcelSurveyResponse struct {
	ID                string    `json:"id"`
	ParcelID          string    `json:"parcel_id"`
	ChimneyCount      *int      `json:"chimney_count"`
	ToiletCount       *int      `json:"toilet_count"`
	CohabitantCount   *int      `json:"cohabitant_count"`
	RelocationType    string    `json:"relocation_type"`
	RelocationAddress string    `json:"relocation_address"`
	Note              string    `json:"note"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         string    `json:"updated_by"`
}

type parcelSurveyRowResponse struct {
	parcelSurveyResponse
	// ParcelCode — нэгж талбарын дугаар (parcel.parcel_id); ParcelID нь uuid.
	ParcelCode      string `json:"parcel_code"`
	AcquisitionID   string `json:"acquisition_id"`
	AcquisitionName string `json:"acquisition_name"`
	AU3Code         string `json:"au3_code"`
}

type parcelSurveySummaryResponse struct {
	Total            int64            `json:"total"`
	Answered         int64            `json:"answered"`
	ChimneyCounts    map[string]int64 `json:"chimney_counts"`
	ToiletCounts     map[string]int64 `json:"toilet_counts"`
	CohabitantCounts map[string]int64 `json:"cohabitant_counts"`
	RelocationCounts map[string]int64 `json:"relocation_counts"`
}

func toParcelSurveyResponse(s *domain.ParcelSurvey) parcelSurveyResponse {
	return parcelSurveyResponse{
		ID:                s.ID.String(),
		ParcelID:          s.ParcelID.String(),
		ChimneyCount:      s.ChimneyCount,
		ToiletCount:       s.ToiletCount,
		CohabitantCount:   s.CohabitantCount,
		RelocationType:    s.RelocationType,
		RelocationAddress: s.RelocationAddress,
		Note:              s.Note,
		CreatedAt:         s.CreatedAt,
		CreatedBy:         s.CreatedBy,
		UpdatedAt:         s.UpdatedAt,
		UpdatedBy:         s.UpdatedBy,
	}
}

func parcelSurveyFilterFromQuery(c *gin.Context) (domain.ParcelSurveyFilter, bool) {
	filter := domain.ParcelSurveyFilter{
		ParcelCode: c.Query("parcel_id"),
		SortBy:     c.Query("sort"),
		SortDir:    c.Query("order"),
		Page:       queryInt(c, "page", 1),
		PageSize:   queryInt(c, "page_size", 20),
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 200 {
		filter.PageSize = 20
	}
	switch c.Query("answered") {
	case "true", "1":
		answered := true
		filter.Answered = &answered
	case "false", "0":
		answered := false
		filter.Answered = &answered
	}
	// acquisition_id-г давтан өгч болно (Хяналтын самбарын шүүсэн чөлөөлөлтүүд).
	for _, raw := range c.QueryArray("acquisition_id") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return filter, false
		}
		filter.AcquisitionIDs = append(filter.AcquisitionIDs, id)
	}
	return filter, true
}

// GetParcelSurvey godoc
//
//	@Summary		Чөлөөлөгдсөн газрын судалгаа
//	@Description	Нэгж талбарын судалгааны дүн. Бүртгэл ороогүй бол data=null.
//	@Tags			parcels
//	@Produce		json
//	@Param			id	path		string	true	"Нэгж талбарын ID (uuid)"
//	@Success		200	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/survey [get]
func (h *ParcelSurveyHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	survey, err := h.svc.GetParcelSurvey(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	if survey == nil {
		response.OK(c, nil)
		return
	}
	response.OK(c, toParcelSurveyResponse(survey))
}

// SaveParcelSurvey godoc
//
//	@Summary		Чөлөөлөгдсөн газрын судалгаа хадгалах
//	@Description	Нэгж талбарт НЭГ бүртгэл — дахин хадгалахад орлогдоно. Тоон хариулт 1..999, relocation_type: ger_district | apartment.
//	@Tags			parcels
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Нэгж талбарын ID (uuid)"
//	@Param			body	body		parcelSurveyRequest	true	"Судалгааны хариулт"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		404		{object}	response.Response
//	@Security		BearerAuth
//	@Router			/parcels/{id}/survey [put]
func (h *ParcelSurveyHandler) Save(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req parcelSurveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	survey, err := h.svc.SaveParcelSurvey(c.Request.Context(), id, service.ParcelSurveyInput{
		ChimneyCount:      req.ChimneyCount,
		ToiletCount:       req.ToiletCount,
		CohabitantCount:   req.CohabitantCount,
		RelocationType:    req.RelocationType,
		RelocationAddress: req.RelocationAddress,
		Note:              req.Note,
		SavedBy:           contextUserIDString(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, toParcelSurveyResponse(survey), i18n.MsgUpdated)
}

// ClearParcelSurvey godoc
//
//	@Summary	Чөлөөлөгдсөн газрын судалгааны хариултыг цэвэрлэх
//	@Description	Бүртгэл нэгж талбартайгаа үлдэж, хариултууд хоосорно.
//	@Tags		parcels
//	@Param		id	path	string	true	"Нэгж талбарын ID (uuid)"
//	@Success	204
//	@Failure	404	{object}	response.Response
//	@Security	BearerAuth
//	@Router		/parcels/{id}/survey [delete]
func (h *ParcelSurveyHandler) Clear(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.ClearParcelSurvey(c.Request.Context(), id, contextUserIDString(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// ListParcelSurveys godoc
//
//	@Summary	Чөлөөлөгдсөн газрын судалгааны жагсаалт
//	@Tags		parcel-surveys
//	@Produce	json
//	@Param		acquisition_id	query		string	false	"Чөлөөлөлтийн ID"
//	@Param		parcel_id		query		string	false	"Нэгж талбарын дугаар (ILIKE)"
//	@Param		answered		query		bool	false	"true: бөглөсөн, false: хоосон"
//	@Param		page			query		int		false	"Хуудас (default: 1)"
//	@Param		page_size		query		int		false	"Хуудасны хэмжээ (default: 20, max: 200)"
//	@Success	200				{object}	response.PaginatedResponse
//	@Security	BearerAuth
//	@Router		/parcel-surveys [get]
func (h *ParcelSurveyHandler) List(c *gin.Context) {
	filter, ok := parcelSurveyFilterFromQuery(c)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	list, total, err := h.svc.ListParcelSurveys(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	out := make([]parcelSurveyRowResponse, 0, len(list))
	for _, row := range list {
		out = append(out, parcelSurveyRowResponse{
			parcelSurveyResponse: toParcelSurveyResponse(&row.ParcelSurvey),
			ParcelCode:           row.ParcelCode,
			AcquisitionID:        row.AcquisitionID.String(),
			AcquisitionName:      row.AcquisitionName,
			AU3Code:              row.AU3Code,
		})
	}
	response.Paginated(c, out, total, filter.Page, filter.PageSize)
}

// ParcelSurveySummary godoc
//
//	@Summary	Чөлөөлөгдсөн газрын судалгааны дүн (статистик)
//	@Tags		parcel-surveys
//	@Produce	json
//	@Param		acquisition_id	query		[]string	false	"Чөлөөлөлтийн ID (давтан өгч болно)"	collectionFormat(multi)
//	@Success	200				{object}	response.Response
//	@Security	BearerAuth
//	@Router		/parcel-surveys/summary [get]
func (h *ParcelSurveyHandler) Summary(c *gin.Context) {
	filter, ok := parcelSurveyFilterFromQuery(c)
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	sum, err := h.svc.ParcelSurveySummary(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, parcelSurveySummaryResponse{
		Total:            sum.Total,
		Answered:         sum.Answered,
		ChimneyCounts:    sum.ChimneyCounts,
		ToiletCounts:     sum.ToiletCounts,
		CohabitantCounts: sum.CohabitantCounts,
		RelocationCounts: sum.RelocationCounts,
	})
}
