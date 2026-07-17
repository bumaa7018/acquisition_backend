package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogRepo struct {
	db *pgxpool.Pool
}

func NewAuditLogRepo(db *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

const auditLogColumns = `id, actor_id, actor_name, actor_position, actor_roles, action,
	resource_type, resource_id, acquisition_id, parcel_id, details, ip_address, user_agent, created_at`

func (r *AuditLogRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	roles, err := json.Marshal(log.ActorRoles)
	if err != nil {
		return err
	}
	details := log.Details
	if len(details) == 0 {
		details = []byte("{}")
	}
	_, err = r.db.Exec(ctx, `INSERT INTO audit_log (`+auditLogColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		log.ID, log.ActorID, log.ActorName, log.ActorPosition, roles, log.Action,
		log.ResourceType, log.ResourceID, log.AcquisitionID, log.ParcelID, details,
		log.IPAddress, log.UserAgent, log.CreatedAt)
	return err
}

func (r *AuditLogRepo) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 200 {
		filter.PageSize = 20
	}

	where, args := buildAuditWhere(filter)
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM audit_log "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.db.Query(ctx,
		"SELECT "+auditLogColumns+" FROM audit_log "+where+
			fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]*domain.AuditLog, 0)
	for rows.Next() {
		log := &domain.AuditLog{}
		var rolesRaw []byte
		if err := rows.Scan(
			&log.ID, &log.ActorID, &log.ActorName, &log.ActorPosition, &rolesRaw, &log.Action,
			&log.ResourceType, &log.ResourceID, &log.AcquisitionID, &log.ParcelID, &log.Details,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(rolesRaw) > 0 {
			_ = json.Unmarshal(rolesRaw, &log.ActorRoles)
		}
		out = append(out, log)
	}
	return out, total, rows.Err()
}

func buildAuditWhere(filter domain.AuditLogFilter) (string, []any) {
	var (
		conds []string
		args  []any
	)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if filter.ActorID != nil && *filter.ActorID != uuid.Nil {
		add("actor_id = $%d", *filter.ActorID)
	}
	if filter.Action != "" {
		add("action = $%d", filter.Action)
	}
	if filter.ResourceType != "" {
		add("resource_type = $%d", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		add("resource_id = $%d", filter.ResourceID)
	}
	if filter.AcquisitionID != nil && *filter.AcquisitionID != uuid.Nil {
		add("acquisition_id = $%d", *filter.AcquisitionID)
	}
	if filter.ParcelID != nil && *filter.ParcelID != uuid.Nil {
		add("parcel_id = $%d", *filter.ParcelID)
	}
	if filter.CreatedFrom != nil {
		add("created_at >= $%d", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		add("created_at <= $%d", *filter.CreatedTo)
	}
	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}
