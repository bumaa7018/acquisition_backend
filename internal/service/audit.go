package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/actorctx"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

type AuditService interface {
	Log(ctx context.Context, entry AuditEntry)
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int64, error)
}

type AuditEntry struct {
	Action        string
	ResourceType  string
	ResourceID    string
	AcquisitionID *uuid.UUID
	ParcelID      *uuid.UUID
	Details       any
}

type auditService struct {
	repo repository.AuditLogRepository
}

func NewAuditService(repo repository.AuditLogRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *auditService) Log(ctx context.Context, entry AuditEntry) {
	if s == nil || s.repo == nil || entry.Action == "" || entry.ResourceType == "" {
		return
	}
	actor, _ := actorctx.From(ctx)
	details, err := json.Marshal(entry.Details)
	if err != nil || len(details) == 0 || string(details) == "null" {
		details = []byte("{}")
	}
	var actorID *uuid.UUID
	if actor.ID != uuid.Nil {
		id := actor.ID
		actorID = &id
	}
	log := &domain.AuditLog{
		ID:            uuid.New(),
		ActorID:       actorID,
		ActorName:     actor.Name,
		ActorPosition: actor.Position,
		ActorRoles:    actor.Roles,
		Action:        entry.Action,
		ResourceType:  entry.ResourceType,
		ResourceID:    entry.ResourceID,
		AcquisitionID: entry.AcquisitionID,
		ParcelID:      entry.ParcelID,
		Details:       details,
		IPAddress:     actor.IP,
		UserAgent:     actor.UserAgent,
		CreatedAt:     time.Now(),
	}
	go func() {
		release, ok := tryAcquireBackgroundSlot()
		if !ok {
			slog.Warn("audit log dropped — background write queue full", "action", entry.Action, "resource_type", entry.ResourceType)
			return
		}
		defer release()

		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.Create(saveCtx, log); err != nil {
			slog.Error("audit log save failed", "action", entry.Action, "resource_type", entry.ResourceType, "error", err)
		}
	}()
}
