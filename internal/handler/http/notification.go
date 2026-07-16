package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	svc service.NotificationService
}

func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// List godoc
// @Summary  Өөрийн мэдэгдлийн жагсаалт
// @Tags     notifications
// @Security BearerAuth
// @Param    unread    query bool false "зөвхөн уншаагүй"
// @Param    page      query int  false "хуудас"
// @Param    page_size query int  false "хуудасны хэмжээ (max 100)"
// @Success  200 {object} response.Response
// @Router   /notifications [get]
func (h *NotificationHandler) List(c *gin.Context) {
	uid, ok := middleware.GetCallerID(c)
	if !ok {
		response.Error(c, middleware.ErrMissingToken)
		return
	}
	onlyUnread := c.Query("unread") == "true"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.svc.List(c.Request.Context(), uid, onlyUnread, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// UnreadCount godoc
// @Summary  Уншаагүй мэдэгдлийн тоо
// @Tags     notifications
// @Security BearerAuth
// @Success  200 {object} response.Response
// @Router   /notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	uid, ok := middleware.GetCallerID(c)
	if !ok {
		response.Error(c, middleware.ErrMissingToken)
		return
	}
	count, err := h.svc.UnreadCount(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"count": count})
}

// MarkRead godoc
// @Summary  Мэдэгдлийг уншсан болгох
// @Tags     notifications
// @Security BearerAuth
// @Param    id path string true "notification id"
// @Success  200 {object} response.Response
// @Router   /notifications/{id}/read [post]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	uid, ok := middleware.GetCallerID(c)
	if !ok {
		response.Error(c, middleware.ErrMissingToken)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), uid, id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// MarkAllRead godoc
// @Summary  Бүх мэдэгдлийг уншсан болгох
// @Tags     notifications
// @Security BearerAuth
// @Success  200 {object} response.Response
// @Router   /notifications/read-all [post]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	uid, ok := middleware.GetCallerID(c)
	if !ok {
		response.Error(c, middleware.ErrMissingToken)
		return
	}
	if err := h.svc.MarkAllRead(c.Request.Context(), uid); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// Stream — Server-Sent Events: шинэ мэдэгдлийг бодит цагт түгээнэ.
// Клиент Authorization header-тэй fetch стриймээр холбогдоно (JWT middleware
// хамгаална, token query string-ээр дамждаггүй тул лог/түүхэд үлдэхгүй).
// @Summary  Мэдэгдлийн SSE стрийм
// @Tags     notifications
// @Security BearerAuth
// @Router   /notifications/stream [get]
func (h *NotificationHandler) Stream(c *gin.Context) {
	uid, ok := middleware.GetCallerID(c)
	if !ok {
		response.Error(c, middleware.ErrMissingToken)
		return
	}

	ch, unsubscribe := h.svc.Subscribe(uid)
	if ch == nil {
		// Нэг хэрэглэгчийн зэрэгцээ холболтын хязгаар хэтэрсэн.
		response.Abort(c, http.StatusTooManyRequests, i18n.ErrTooManyRequests)
		return
	}
	defer unsubscribe()

	w := c.Writer
	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("Connection", "keep-alive")
	// Reverse proxy (nginx г.м.) стриймийг буферлэхээс сэргийлнэ.
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Холболт амьд болохыг клиентэд шууд мэдэгдэнэ.
	fmt.Fprint(w, "event: ready\ndata: {}\n\n")
	w.Flush()

	// Идэвхгүй үед 25 секунд тутам heartbeat илгээж завсрын
	// прокси/LB-ийн idle timeout-оос холболтоо хамгаална.
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			w.Flush()
		case n, open := <-ch:
			if !open {
				return
			}
			payload, err := json.Marshal(n)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: notification\ndata: %s\n\n", payload); err != nil {
				return
			}
			w.Flush()
		}
	}
}
