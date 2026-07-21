package service

import (
	"context"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

// ClientRequestEntry — internal/pkg/httpclient гэх мэт бусад системрүү хандах
// wrapper-ийн дамжуулах бэлэн бүртгэлийн өгөгдөл.
type ClientRequestEntry struct {
	Method          string
	Host            string
	Path            string
	RequestBody     string
	RequestResponse string
	StatusCode      int
	Error           string
	DurationMs      int64
	RequestedAt     time.Time
}

type ClientRequestService interface {
	// Log бичихийг дэвсгэрт нь (fire-and-forget) хийнэ — гадаад системрүү хийсэн
	// дуудлагын хариуг хүлээх урсгалыг лог бичих үйлдэл хэзээ ч удаашруулахгүй.
	Log(entry ClientRequestEntry)
	List(ctx context.Context, filter domain.ClientRequestFilter) ([]*domain.ClientRequest, int64, error)
}

type clientRequestService struct {
	repo repository.ClientRequestRepository
}

func NewClientRequestService(repo repository.ClientRequestRepository) ClientRequestService {
	return &clientRequestService{repo: repo}
}

func (s *clientRequestService) List(ctx context.Context, filter domain.ClientRequestFilter) ([]*domain.ClientRequest, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *clientRequestService) Log(entry ClientRequestEntry) {
	if s == nil || s.repo == nil {
		return
	}
	if entry.RequestedAt.IsZero() {
		entry.RequestedAt = time.Now()
	}
	record := &domain.ClientRequest{
		ID:              uuid.New(),
		Method:          entry.Method,
		Host:            entry.Host,
		Path:            entry.Path,
		RequestBody:     entry.RequestBody,
		RequestResponse: entry.RequestResponse,
		StatusCode:      entry.StatusCode,
		Error:           entry.Error,
		DurationMs:      entry.DurationMs,
		RequestedAt:     entry.RequestedAt,
	}
	go func() {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.Create(saveCtx, record); err != nil {
			slog.Error("client_request log save failed", "host", entry.Host, "path", entry.Path, "error", err)
		}
	}()
}
