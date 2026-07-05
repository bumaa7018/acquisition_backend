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
	c.AbortWithStatusJSON(status, Response{
		Code:    status,
		Error:   msg,
		Message: msg,
	})
}

// AbortValidation sends a 400 with a raw validation error string (from go-playground/validator).
func AbortValidation(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{
		Code:    http.StatusBadRequest,
		Error:   err.Error(),
		Message: err.Error(),
	})
}

func Error(c *gin.Context, err error) {
	status, key := mapError(err)
	msg := i18n.T(lang(c), key)
	if status >= http.StatusInternalServerError {
		slog.Error("request failed",
			"error", err,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
		)
	}
	c.AbortWithStatusJSON(status, Response{
		Code:    status,
		Error:   msg,
		Message: msg,
	})
}

func mapError(err error) (int, i18n.Key) {
	switch err {
	case domain.ErrNotFound:
		return http.StatusNotFound, i18n.ErrNotFound
	case domain.ErrAlreadyExists:
		return http.StatusConflict, i18n.ErrConflict
	case domain.ErrInvalidCredentials:
		return http.StatusUnauthorized, i18n.ErrInvalidCreds
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
	case domain.ErrInvalidTransition:
		return http.StatusConflict, i18n.ErrConflict
	case domain.ErrValuationNotApproved:
		return http.StatusUnprocessableEntity, i18n.ErrValuationReportRequired
	case domain.ErrNonFinalParcels:
		return http.StatusUnprocessableEntity, i18n.ErrNonFinalParcels
	default:
		return http.StatusInternalServerError, i18n.ErrInternalServer
	}
}
