package postgres

import (
	"context"
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ClientRequestRepo struct {
	db *pgxpool.Pool
}

func NewClientRequestRepo(db *pgxpool.Pool) *ClientRequestRepo {
	return &ClientRequestRepo{db: db}
}

const clientRequestColumns = `id, method, host, path, request_body, request_response,
	status_code, error, duration_ms, requested_at`

func (r *ClientRequestRepo) Create(ctx context.Context, cr *domain.ClientRequest) error {
	_, err := r.db.Exec(ctx, `INSERT INTO client_request (`+clientRequestColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		cr.ID, cr.Method, cr.Host, cr.Path, cr.RequestBody, cr.RequestResponse,
		cr.StatusCode, cr.Error, cr.DurationMs, cr.RequestedAt)
	return err
}

func (r *ClientRequestRepo) List(ctx context.Context, filter domain.ClientRequestFilter) ([]*domain.ClientRequest, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 200 {
		filter.PageSize = 20
	}

	where, args := buildClientRequestWhere(filter)
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM client_request "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := r.db.Query(ctx,
		"SELECT "+clientRequestColumns+" FROM client_request "+where+
			fmt.Sprintf(" ORDER BY requested_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]*domain.ClientRequest, 0)
	for rows.Next() {
		cr := &domain.ClientRequest{}
		if err := rows.Scan(
			&cr.ID, &cr.Method, &cr.Host, &cr.Path, &cr.RequestBody, &cr.RequestResponse,
			&cr.StatusCode, &cr.Error, &cr.DurationMs, &cr.RequestedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, cr)
	}
	return out, total, rows.Err()
}

func buildClientRequestWhere(filter domain.ClientRequestFilter) (string, []any) {
	var (
		conds []string
		args  []any
	)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if filter.Method != "" {
		add("method = $%d", filter.Method)
	}
	if filter.Host != "" {
		add("host = $%d", filter.Host)
	}
	if filter.Path != "" {
		add("path = $%d", filter.Path)
	}
	if filter.CreatedFrom != nil {
		add("requested_at >= $%d", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		add("requested_at <= $%d", *filter.CreatedTo)
	}
	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}
