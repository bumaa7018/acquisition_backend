package http

// ГАДААД системд зориулсан нөхөх олговрын API (төлөвлөгөөний 2.8.4).
//
// Зорилго: чөлөөлсөн газрын нөхөх олговрын мэдээг бусад систем (ТЕГ, ШШГЕГ,
// үнийн мэдээний сан г.м.) машинаар уншиж авах.
//
// ЯАГААД ТАЙЛАНГИЙН ЭХ СУРВАЛЖИЙГ ДАХИН ХЭРЭГЛЭВ: гадагш гарах тоо нь дэлгэц
// дээрх тайлантай ЯГ ижил байх ёстой. Тусдаа query бичвэл цаг хугацаанд
// хоёулаа зөрж, "тайлан нэг тоо, API өөр тоо" гэсэн эндүүрэл үүсдэг. Иймд
// LandService.ListReportParcels-ийг (үндсэн үнэлгээний урсгал —
// compensation.valuation_type = parcel.selected_valuation_type) хэвээр
// хэрэглэж, ЗӨВХӨН гадагш харагдах хэлбэрт хөрвүүлнэ.
//
// ГУРВАН хэлбэрээр дуудна (аль нэгийг ЗААВАЛ):
//
//  1. parcel_id        — тухайлсан НЭГЖ ТАЛБАРЫН нөхөх олговор + өртсөн талбай
//  2. plan_parcel_id   — ЧӨЛӨӨЛӨЛТИЙН төлөвлөгөөний нэгж талбарын дугаар:
//     тэр талбараар үүссэн чөлөөлөлтийн БҮХ нэгж талбарын мэдээлэл
//  3. plan_code        — ТӨЛӨВЛӨГӨӨНИЙ дугаар: тэр төлөвлөгөөнд хамаарах БҮХ
//     чөлөөлөлтийн нэгж талбарууд
//
// Хэлбэр 2, 3-д нэгж талбаруудын ТӨЛӨВИЙН ТООЛОЛ ба нийт дүн (summary) хамт
// гарна — тэр нь тайлангийн ижил эх сурвалж (GetReportSummary).
//
// Хамгаалалт: middleware.ExternalAPIKeyAuth (X-API-Key) + rate limit. Шүүлтүүр
// заавал — бүх сангаа нэг хүсэлтээр гадагш дамжуулахгүй.

import (
	"log/slog"
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

const (
	externalDefaultPageSize = 50
	externalMaxPageSize     = 200
)

type ExternalCompensationHandler struct {
	svc service.LandService
}

func NewExternalCompensationHandler(svc service.LandService) *ExternalCompensationHandler {
	return &ExternalCompensationHandler{svc: svc}
}

// externalCompensationRow — гадаад системд гарах НЭГ мөр.
//
// Зөвхөн тайланд харагддаг ҮНДСЭН үнэлгээний мэдээлэл. Дотоод үнэлгээний
// урсгалууд (хөндлөнгийн, МИКА), баримт, эзэмшигчийн холбоо барих мэдээлэл
// ЗОРИУДААР гарахгүй.
type externalCompensationRow struct {
	ParcelID string `json:"parcel_id"`
	// AreaM2 — нэгж талбарын бүтэн талбай, AcquisitionAreaM2 — чөлөөлөгдсөн хэсэг
	AreaM2            float64 `json:"area_m2"`
	AcquisitionAreaM2 float64 `json:"acquisition_area_m2"`
	RemainingAreaM2   float64 `json:"remaining_area_m2"`
	RightType         int     `json:"right_type"`
	RightTypeName     string  `json:"right_type_name"`
	Status            int     `json:"status"`
	StatusName        string  `json:"status_name"`
	// StatusDate — төлөв сүүлд өөрчлөгдсөн огноо (чөлөөлсөн огноо болж уншигдана)
	StatusDate string `json:"status_date,omitempty"`

	Acquisition  externalAcquisitionRef  `json:"acquisition"`
	Holder       externalHolderRef       `json:"holder"`
	Compensation externalCompensationSum `json:"compensation"`
}

type externalAcquisitionRef struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PlanCode     string `json:"plan_code"`
	Category     string `json:"category,omitempty"`
	SubCategory  string `json:"sub_category,omitempty"`
	DecreeNumber string `json:"decree_number,omitempty"`
	DecreeDate   string `json:"decree_date,omitempty"`
}

type externalHolderRef struct {
	LastName   string `json:"last_name,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	RegisterNo string `json:"register_no,omitempty"`
}

// externalListResponse — гурван хэлбэрийн НЭГДСЭН хариулт.
type externalListResponse struct {
	// Mode — аль хэлбэрээр дуудсан ("parcel" | "plan_parcel" | "plan").
	Mode       string                    `json:"mode"`
	Filter     externalFilterEcho        `json:"filter"`
	Summary    externalSummary           `json:"summary"`
	Parcels    []externalCompensationRow `json:"parcels"`
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int64                     `json:"total_pages"`
}

// externalFilterEcho — ямар шүүлтүүрээр хайсныг хариултад давтана (дуудагч
// систем логтоо тааруулахад).
type externalFilterEcho struct {
	ParcelID      string `json:"parcel_id,omitempty"`
	PlanParcelID  string `json:"plan_parcel_id,omitempty"`
	PlanCode      string `json:"plan_code,omitempty"`
	AcquisitionID string `json:"acquisition_id,omitempty"`
	Years         []int  `json:"years,omitempty"`
}

// externalSummary — шүүлтэд тохирох БҮХ мөрийн нэгтгэл (хуудаслалтаас
// хамаарахгүй). Тайлангийн статистиктай ижил эх сурвалж.
type externalSummary struct {
	AcquisitionCount int `json:"acquisition_count"`
	ParcelCount      int `json:"parcel_count"`
	// AcquisitionAreaM2 — чөлөөлөгдсөн (өртсөн) талбайн нийлбэр,
	// ParcelAreaM2 — нэгж талбаруудын бүтэн талбайн нийлбэр.
	AcquisitionAreaM2 float64                 `json:"acquisition_area_m2"`
	ParcelAreaM2      float64                 `json:"parcel_area_m2"`
	Compensation      externalCompensationSum `json:"compensation"`
	// StatusCounts — чөлөөлөлтийн үндсэн нэгж талбаруудын ТӨЛӨВИЙН тоолол.
	StatusCounts []externalStatusCount `json:"status_counts"`
}

type externalStatusCount struct {
	Status     int    `json:"status"`
	StatusName string `json:"status_name"`
	Count      int    `json:"count"`
}

// externalCompensationSum — нөхөх олговор ЗОРИЛТООР задалсан дүн (₮).
type externalCompensationSum struct {
	// Land — газрын нөхөх олговор, RealEstate — үл хөдлөх, Property — бусад хөрөнгө
	Land       float64 `json:"land"`
	RealEstate float64 `json:"real_estate"`
	Property   float64 `json:"property"`
	// Other — дээрх 3 ангилалд ороогүй үлдэгдэл (задаргааны нийлбэр нь Total-тай
	// ҮРГЭЛЖ тэнцүү байхын тулд). Мөрийн түвшинд үргэлж 0.
	Other    float64 `json:"other"`
	Total    float64 `json:"total"`
	Currency string  `json:"currency"`
}

func toExternalCompensationRow(r *domain.ReportParcelRow) externalCompensationRow {
	row := externalCompensationRow{
		ParcelID:          r.ParcelID,
		AreaM2:            r.AreaM2,
		AcquisitionAreaM2: r.AcquisitionAreaM2,
		RemainingAreaM2:   r.RemainingAreaM2,
		RightType:         r.RightType,
		RightTypeName:     domain.RightTypeName(r.RightType),
		Status:            r.Status,
		StatusName:        r.StatusName,
		Acquisition: externalAcquisitionRef{
			ID:           r.AcquisitionID.String(),
			Name:         r.AcquisitionName,
			PlanCode:     r.PlanCode,
			Category:     r.GeneralCategoryName,
			SubCategory:  r.SubCategoryName,
			DecreeNumber: r.DecreeNumber,
		},
		Holder: externalHolderRef{
			LastName:   r.HolderLastName,
			FirstName:  r.HolderName,
			RegisterNo: r.HolderRegisterNo,
		},
		Compensation: externalCompensationSum{
			Land:       r.LandComp,
			RealEstate: r.RealStateComp,
			Property:   r.PropertyComp,
			Total:      r.LandComp + r.RealStateComp + r.PropertyComp,
			Currency:   "MNT",
		},
	}
	if r.StatusDate != nil {
		row.StatusDate = r.StatusDate.Format("2006-01-02")
	}
	if r.DecreeDate != nil {
		row.Acquisition.DecreeDate = r.DecreeDate.Format("2006-01-02")
	}
	return row
}

// ListCompensations godoc
//
//	@Summary		Нөхөх олговрын мэдээлэл (гадаад систем)
//	@Description	Чөлөөлсөн газрын нөхөх олговрын ҮНДСЭН үнэлгээний мэдээлэл (дэлгэцийн тайлантай ижил тоо). Танилт: X-API-Key. ГУРВАН хэлбэр — parcel_id (тухайлсан нэгж талбар) / plan_parcel_id (чөлөөлөлтийн төлөвлөгөөний нэгж талбар) / plan_code (төлөвлөгөөний дугаар). Аль нэгийг заавал. Хэлбэр 2, 3-д төлөвийн тоолол ба нийт дүн (summary) хамт гарна.
//	@Tags			external
//	@Produce		json
//	@Param			parcel_id		query		string	false	"Нэгж талбарын дугаар"
//	@Param			plan_parcel_id	query		string	false	"Чөлөөлөлтийн төлөвлөгөөний нэгж талбарын дугаар"
//	@Param			plan_code		query		string	false	"Төлөвлөгөөний дугаар (яг таарал)"
//	@Param			acquisition_id	query		string	false	"Чөлөөлөлтийн ID (UUID)"
//	@Param			year			query		int		false	"Төлөв өөрчлөгдсөн он (олон удаа зааж болно)"
//	@Param			page			query		int		false	"Хуудас (1-ээс)"
//	@Param			page_size		query		int		false	"Хуудасны хэмжээ (дээд тал 200)"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response
//	@Failure		401				{object}	response.Response
//	@Router			/external/compensations [get]
func (h *ExternalCompensationHandler) ListCompensations(c *gin.Context) {
	parcelID := strings.TrimSpace(c.Query("parcel_id"))
	planParcelID := strings.TrimSpace(c.Query("plan_parcel_id"))
	planCode := strings.TrimSpace(c.Query("plan_code"))
	acquisitionID := strings.TrimSpace(c.Query("acquisition_id"))

	// Шүүлтүүргүй хүсэлт = бүх бүртгэлийг гадагш дамжуулах. Зориудаар хориглов.
	if parcelID == "" && planParcelID == "" && planCode == "" && acquisitionID == "" {
		response.Abort(c, http.StatusBadRequest, i18n.ErrExternalFilterRequired)
		return
	}
	if acquisitionID != "" {
		if _, err := uuid.Parse(acquisitionID); err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidAcqUUID)
			return
		}
	}

	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize < 1 {
		pageSize = externalDefaultPageSize
	}
	if pageSize > externalMaxPageSize {
		pageSize = externalMaxPageSize
	}

	years := []int{}
	for _, raw := range c.QueryArray("year") {
		for _, part := range strings.Split(raw, ",") {
			if y, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && y > 1900 {
				years = append(years, y)
			}
		}
	}

	filter := domain.ReportParcelFilter{
		ParcelID:      parcelID,
		PlanParcelID:  planParcelID,
		PlanCodeExact: planCode,
		AcquisitionID: acquisitionID,
		Years:         years,
		Page:          page,
		PageSize:      pageSize,
	}

	rows, total, err := h.svc.ListReportParcels(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	// Нэгтгэл (нийт дүн + ТӨЛӨВИЙН ТООЛОЛ) — тайлангийн ижил эх сурвалж тул
	// жагсаалттай тоо зөрөхгүй. Хуудаслалтаас хамаарахгүй.
	summary, err := h.svc.GetReportSummary(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	parcels := make([]externalCompensationRow, 0, len(rows))
	for _, r := range rows {
		parcels = append(parcels, toExternalCompensationRow(r))
	}

	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = total / int64(pageSize)
		if total%int64(pageSize) != 0 {
			totalPages++
		}
	}

	out := externalListResponse{
		Mode: externalRequestMode(parcelID, planParcelID, planCode),
		Filter: externalFilterEcho{
			ParcelID:      parcelID,
			PlanParcelID:  planParcelID,
			PlanCode:      planCode,
			AcquisitionID: acquisitionID,
			Years:         years,
		},
		Summary:    toExternalSummary(summary),
		Parcels:    parcels,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	slog.Info("external compensation api",
		"client", middleware.ExternalClient(c),
		"mode", out.Mode,
		"parcel_id", parcelID,
		"plan_parcel_id", planParcelID,
		"plan_code", planCode,
		"acquisition_id", acquisitionID,
		"rows", len(parcels),
		"total", total,
	)
	response.OK(c, out)
}

// externalRequestMode — дуудлагын хэлбэр (хариултад давтагдана).
func externalRequestMode(parcelID, planParcelID, planCode string) string {
	switch {
	case parcelID != "":
		return "parcel"
	case planParcelID != "":
		return "plan_parcel"
	case planCode != "":
		return "plan"
	default:
		return "acquisition"
	}
}

func toExternalSummary(s *domain.ReportSummary) externalSummary {
	if s == nil {
		return externalSummary{
			Compensation: externalCompensationSum{Currency: "MNT"},
			StatusCounts: []externalStatusCount{},
		}
	}
	counts := make([]externalStatusCount, 0, len(s.StatusBreakdown))
	for _, st := range s.StatusBreakdown {
		counts = append(counts, externalStatusCount{
			Status:     st.Status,
			StatusName: st.StatusName,
			Count:      st.Count,
		})
	}
	return externalSummary{
		AcquisitionCount:  s.AcquisitionCount,
		ParcelCount:       s.ParcelCount,
		AcquisitionAreaM2: s.TotalAreaM2,
		ParcelAreaM2:      s.TotalParcelAreaM2,
		Compensation: externalCompensationSum{
			Land:       s.LandCompensation,
			RealEstate: s.RealStateCompensation,
			Property:   s.PropertyCompensation,
			Other:      s.OtherCompensation,
			Total:      s.TotalCompensation,
			Currency:   "MNT",
		},
		StatusCounts: counts,
	}
}
