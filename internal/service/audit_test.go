package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/actorctx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type captureAuditRepo struct {
	ch chan *domain.AuditLog
}

func newCaptureAuditRepo() *captureAuditRepo {
	return &captureAuditRepo{ch: make(chan *domain.AuditLog, 1)}
}

func (r *captureAuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	select {
	case r.ch <- log:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (r *captureAuditRepo) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int64, error) {
	return nil, 0, nil
}

func TestAuditService_LogPersistsActorMetadataAndDetails(t *testing.T) {
	repo := newCaptureAuditRepo()
	svc := NewAuditService(repo)
	actorID := uuid.New()
	acqID := uuid.New()
	parcelID := uuid.New()
	ctx := actorctx.With(context.Background(), actorctx.Actor{
		ID:        actorID,
		Name:      "Admin User",
		Position:  "Ахлах мэргэжилтэн",
		Roles:     []string{"admin"},
		IP:        "127.0.0.1",
		UserAgent: "audit-test",
	})

	svc.Log(ctx, AuditEntry{
		Action:        "status_changed",
		ResourceType:  domain.AuditResourceAcquisition,
		ResourceID:    acqID.String(),
		AcquisitionID: &acqID,
		ParcelID:      &parcelID,
		Details: map[string]any{
			"from_status": 1,
			"to_status":   2,
		},
	})

	select {
	case got := <-repo.ch:
		require.NotEqual(t, uuid.Nil, got.ID)
		require.NotNil(t, got.ActorID)
		require.Equal(t, actorID, *got.ActorID)
		require.Equal(t, "Admin User", got.ActorName)
		require.Equal(t, "Ахлах мэргэжилтэн", got.ActorPosition)
		require.Equal(t, []string{"admin"}, got.ActorRoles)
		require.Equal(t, "127.0.0.1", got.IPAddress)
		require.Equal(t, "audit-test", got.UserAgent)
		require.Equal(t, "status_changed", got.Action)
		require.Equal(t, domain.AuditResourceAcquisition, got.ResourceType)
		require.Equal(t, acqID.String(), got.ResourceID)
		require.Equal(t, acqID, *got.AcquisitionID)
		require.Equal(t, parcelID, *got.ParcelID)

		var details map[string]int
		require.NoError(t, json.Unmarshal(got.Details, &details))
		require.Equal(t, 1, details["from_status"])
		require.Equal(t, 2, details["to_status"])
	case <-time.After(time.Second):
		t.Fatal("audit log was not persisted")
	}
}

func TestAuditService_LogSkipsInvalidEntries(t *testing.T) {
	repo := newCaptureAuditRepo()
	svc := NewAuditService(repo)

	svc.Log(context.Background(), AuditEntry{Action: "", ResourceType: domain.AuditResourceAcquisition})
	svc.Log(context.Background(), AuditEntry{Action: "updated", ResourceType: ""})

	select {
	case got := <-repo.ch:
		t.Fatalf("unexpected audit log persisted: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}
