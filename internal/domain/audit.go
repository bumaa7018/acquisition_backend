package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	AuditResourceAcquisition    = "acquisition"
	AuditResourceParcel         = "parcel"
	AuditResourceValuation      = "valuation"
	AuditResourceCompensation   = "compensation"
	AuditResourceDocument       = "document"
	AuditResourcePayment        = "payment"
	AuditResourceFundingSource  = "funding_source"
	AuditResourceRepresentative = "authorized_representative"
)

type AuditLog struct {
	ID            uuid.UUID       `json:"id"`
	ActorID       *ID             `json:"actor_id,omitempty"`
	ActorName     string          `json:"actor_name"`
	ActorPosition string          `json:"actor_position"`
	ActorRoles    []string        `json:"actor_roles"`
	Action        string          `json:"action"`
	ResourceType  string          `json:"resource_type"`
	ResourceID    string          `json:"resource_id"`
	AcquisitionID *uuid.UUID      `json:"acquisition_id,omitempty"`
	ParcelID      *uuid.UUID      `json:"parcel_id,omitempty"`
	Details       json.RawMessage `json:"details"`
	IPAddress     string          `json:"ip_address"`
	UserAgent     string          `json:"user_agent"`
	CreatedAt     time.Time       `json:"created_at"`
}

type AuditLogFilter struct {
	ActorID       *ID
	Action        string
	ResourceType  string
	ResourceID    string
	AcquisitionID *uuid.UUID
	ParcelID      *uuid.UUID
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	SortBy        string // auditLogSortColumns-ийн түлхүүр
	SortDir       string // asc | desc
	Page          int
	PageSize      int
}
