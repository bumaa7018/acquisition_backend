package service

import (
	"context"

	"github.com/google/uuid"
)

func (s *landService) auditLog(ctx context.Context, action, resourceType, resourceID string, acqID, parcelID *uuid.UUID, details any) {
	if s.audit == nil {
		return
	}
	s.audit.Log(ctx, AuditEntry{
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		AcquisitionID: acqID,
		ParcelID:      parcelID,
		Details:       details,
	})
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
