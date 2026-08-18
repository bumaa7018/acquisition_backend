package response

import (
	"log/slog"
	"net/http"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message"`
}

type PaginatedResponse struct {
	Code       int    `json:"code"`
	Data       any    `json:"data"`
	Message    string `json:"message"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int64  `json:"total_pages"`
}

func lang(c *gin.Context) i18n.Lang {
	return i18n.Detect(c.GetHeader("Accept-Language"))
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Data:    data,
		Message: i18n.T(lang(c), i18n.MsgOK),
	})
}

func OKMsg(c *gin.Context, data any, key i18n.Key) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Data:    data,
		Message: i18n.T(lang(c), key),
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Code:    http.StatusCreated,
		Data:    data,
		Message: i18n.T(lang(c), i18n.MsgCreated),
	})
}

func NoContent(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}

func Paginated(c *gin.Context, data any, total int64, page, pageSize int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	totalPages := total / int64(pageSize)
	if total%int64(pageSize) != 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:       http.StatusOK,
		Data:       data,
		Message:    i18n.T(lang(c), i18n.MsgOK),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// Abort sends an error response using an i18n key.
func Abort(c *gin.Context, status int, key i18n.Key) {
	msg := i18n.T(lang(c), key)
	logError(c, status, msg)
	c.AbortWithStatusJSON(status, Response{
		Code:    status,
		Error:   msg,
		Message: msg,
	})
}

// AbortValidation sends a 400 with a raw validation error string (from go-playground/validator).
// Валидацийн алдаа зөвхөн талбарын нэр/дүрмийг агуулна (go-playground/validator), илгээсэн
// утгыг агуулдаггүй тул чөлөөтэй логлож болно.
func AbortValidation(c *gin.Context, err error) {
	logError(c, http.StatusBadRequest, err.Error())
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{
		Code:    http.StatusBadRequest,
		Error:   err.Error(),
		Message: err.Error(),
	})
}

func Error(c *gin.Context, err error) {
	status, key := mapError(err)
	msg := i18n.T(lang(c), key)
	logError(c, status, err.Error())
	c.AbortWithStatusJSON(status, Response{
		Code:    status,
		Error:   msg,
		Message: msg,
	})
}

// logError бүх төрлийн алдааны хариуг (4xx болон 5xx) статусаас хамаарсан
// түвшинд логлоно — өмнө нь зөвхөн 500-аас дээш статус л логлогддог байсан тул
// бизнесийн ихэнх алдаа (400/403/404/409 гэх мэт) чимээгүй алга болдог байв.
// Зөвхөн статус, аргумент, зам, алдааны мессежийг логлоно — хүсэлтийн биеийг
// (нууц үг, файл гэх мэт мэдээлэл агуулж болзошгүй) хэзээ ч логлохгүй.
func logError(c *gin.Context, status int, msg string) {
	attrs := []any{
		"error", msg,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", status,
	}
	switch {
	case status >= http.StatusInternalServerError:
		slog.Error("request failed", attrs...)
	default:
		slog.Warn("request failed", attrs...)
	}
}

func mapError(err error) (int, i18n.Key) {
	switch err {
	case domain.ErrNotFound:
		return http.StatusNotFound, i18n.ErrNotFound
	case domain.ErrAlreadyExists:
		return http.StatusConflict, i18n.ErrConflict
	case domain.ErrInvalidCredentials:
		return http.StatusUnauthorized, i18n.ErrInvalidCreds
	case domain.ErrTooManyRequests:
		return http.StatusTooManyRequests, i18n.ErrTooManyRequests
	case domain.ErrUnauthorized:
		return http.StatusUnauthorized, i18n.ErrUnauthorized
	case domain.ErrForbidden:
		return http.StatusForbidden, i18n.ErrForbidden
	case domain.ErrApproved:
		return http.StatusConflict, i18n.ErrApproved
	case domain.ErrValuationReportRequired:
		return http.StatusUnprocessableEntity, i18n.ErrValuationReportRequired
	case domain.ErrInvalidToken:
		return http.StatusUnauthorized, i18n.ErrInvalidToken
	case domain.ErrExpiredToken:
		return http.StatusUnauthorized, i18n.ErrExpiredToken
	case domain.ErrInvalidInput:
		return http.StatusBadRequest, i18n.ErrInvalidInput
	case domain.ErrAU2Crossing:
		return http.StatusUnprocessableEntity, i18n.ErrAU2Crossing
	case domain.ErrOutsidePlanExceeds:
		return http.StatusUnprocessableEntity, i18n.ErrOutsidePlanExceeds
	case domain.ErrOutsideParcelBoundary:
		return http.StatusUnprocessableEntity, i18n.ErrOutsideParcelBoundary
	case domain.ErrShapefileProjection:
		return http.StatusUnprocessableEntity, i18n.ErrShapefileProjection
	case domain.ErrShapefileOutsideCountry:
		return http.StatusUnprocessableEntity, i18n.ErrShapefileOutsideCountry
	case domain.ErrShapefileMultiRecord:
		return http.StatusUnprocessableEntity, i18n.ErrShapefileMultiRecord
	case domain.ErrShapefileShapeType:
		return http.StatusUnprocessableEntity, i18n.ErrShapefileShapeType
	case domain.ErrShapefileMultiPolygon:
		return http.StatusUnprocessableEntity, i18n.ErrShapefileMultiPolygon
	case domain.ErrShapefileFormat:
		return http.StatusUnprocessableEntity, i18n.ErrShapefileFormat
	case domain.ErrInvalidTransition:
		return http.StatusConflict, i18n.ErrConflict
	case domain.ErrValuationNotApproved:
		return http.StatusUnprocessableEntity, i18n.ErrValuationReportRequired
	case domain.ErrNonFinalParcels:
		return http.StatusUnprocessableEntity, i18n.ErrNonFinalParcels
	case domain.ErrAssetPhotoRequired:
		return http.StatusUnprocessableEntity, i18n.ErrAssetPhotoRequired
	case domain.ErrReportBeforeApproval:
		return http.StatusUnprocessableEntity, i18n.ErrReportBeforeApproval
	case domain.ErrParcelAppNoMissing:
		return http.StatusUnprocessableEntity, i18n.ErrParcelAppNoMissing
	case domain.ErrParcelOwnedNoPayment:
		return http.StatusUnprocessableEntity, i18n.ErrParcelOwnedNoPayment
	case domain.ErrSourceFileNotConfigured:
		return http.StatusServiceUnavailable, i18n.ErrSourceFileUnavailable
	case domain.ErrDecisionConfirmed:
		return http.StatusConflict, i18n.ErrDecisionConfirmed
	case domain.ErrParcelNotReleased:
		return http.StatusUnprocessableEntity, i18n.ErrParcelNotReleased
	case domain.ErrParcelAlreadyLinked:
		return http.StatusConflict, i18n.ErrParcelAlreadyLinked
	case domain.ErrDecisionAcqMismatch:
		return http.StatusUnprocessableEntity, i18n.ErrDecisionAcqMismatch
	case domain.ErrDecreeRequired:
		return http.StatusBadRequest, i18n.ErrDecreeRequired
	case domain.ErrFundingSourceRequired:
		return http.StatusUnprocessableEntity, i18n.ErrFundingSourceRequired
	case domain.ErrFundingSourceInUse:
		return http.StatusConflict, i18n.ErrFundingSourceInUse
	default:
		return http.StatusInternalServerError, i18n.ErrInternalServer
	}
}
