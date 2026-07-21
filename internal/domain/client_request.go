package domain

import (
	"time"

	"github.com/google/uuid"
)

// ClientRequest — backend-ээс бусад системрүү (гадаад API) илгээсэн HTTP
// хүсэлт бүрийн бүртгэл. internal/pkg/httpclient пакет-ээр дамжсан
// хүсэлт бүр энд автоматаар хадгалагдана.
type ClientRequest struct {
	ID              uuid.UUID `json:"id"`
	Method          string    `json:"method"`
	Host            string    `json:"host"`
	Path            string    `json:"path"`
	RequestBody     string    `json:"request_body,omitempty"`
	RequestResponse string    `json:"request_response,omitempty"`
	StatusCode      int       `json:"status_code,omitempty"`
	Error           string    `json:"error,omitempty"`
	DurationMs      int64     `json:"duration_ms"`
	RequestedAt     time.Time `json:"requested_at"`
}

type ClientRequestFilter struct {
	Method      string
	Host        string
	Path        string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Page        int
	PageSize    int
}
