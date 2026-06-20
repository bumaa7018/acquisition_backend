package response

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/go-api-template/internal/domain"
)

type Response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type PaginatedResponse struct {
	Data       any   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int64 `json:"total_pages"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Data: data})
}

func NoContent(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}

func Paginated(c *gin.Context, data any, total int64, page, pageSize int) {
	totalPages := total / int64(pageSize)
	if total%int64(pageSize) != 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func Error(c *gin.Context, err error) {
	status, msg := mapError(err)
	if status >= http.StatusInternalServerError {
		slog.Error("request failed",
			"error", err,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
		)
	}
	c.AbortWithStatusJSON(status, Response{Error: msg})
}

func mapError(err error) (int, string) {
	switch err {
	case domain.ErrNotFound:
		return http.StatusNotFound, err.Error()
	case domain.ErrAlreadyExists:
		return http.StatusConflict, err.Error()
	case domain.ErrInvalidCredentials:
		return http.StatusUnauthorized, err.Error()
	case domain.ErrUnauthorized:
		return http.StatusUnauthorized, err.Error()
	case domain.ErrForbidden:
		return http.StatusForbidden, err.Error()
	case domain.ErrInvalidToken, domain.ErrExpiredToken:
		return http.StatusUnauthorized, err.Error()
	case domain.ErrInvalidInput:
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
