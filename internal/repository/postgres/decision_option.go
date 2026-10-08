package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// decisionOptionRepo — Ажлын төрөл / Төсөв хоёр lookup хүснэгт бүтцээрээ
// ижил тул нэг хэрэгжүүлэлтийг хүснэгтийн нэрээр параметрлэв. table нь
// зөвхөн доорх constructor-уудаас ирдэг compile-time тогтмол.
type decisionOptionRepo struct {
	db    *pgxpool.Pool
	table string
	// usage — сонголтыг ашигласан бүртгэлийн тоо ($1 = id) ба тайлбар.
	// Ашиглагдаж байвал устгахгүй (өмнө FK-ийн ON DELETE SET NULL-аар
	// захирамжаас чимээгүй арилдаг байв).
	usage []optionUsage
}

type optionUsage struct {
	countSQL string
	label    string
}

// DecisionWorkTypeRepo — Ажлын төрөл
type DecisionWorkTypeRepo struct{ decisionOptionRepo }

func NewDecisionWorkTypeRepo(db *pgxpool.Pool) *DecisionWorkTypeRepo {
	return &DecisionWorkTypeRepo{decisionOptionRepo{db: db, table: "decision_work_type", usage: []optionUsage{
		{`SELECT COUNT(*) FROM decision_draft WHERE work_type_id = $1`, "захирамжийн төсөл"},
	}}}
}

// DecisionBudgetRepo — Төсөв
type DecisionBudgetRepo struct{ decisionOptionRepo }

func NewDecisionBudgetRepo(db *pgxpool.Pool) *DecisionBudgetRepo {
	return &DecisionBudgetRepo{decisionOptionRepo{db: db, table: "decision_budget", usage: []optionUsage{
		{`SELECT COUNT(*) FROM decision_draft WHERE budget_id = $1`, "захирамжийн төсөл"},
	}}}
}

// DecisionReviewUnitRepo — Төсөл хянах нэгж («Хянагдаж буй газар»)
type DecisionReviewUnitRepo struct{ decisionOptionRepo }

func NewDecisionReviewUnitRepo(db *pgxpool.Pool) *DecisionReviewUnitRepo {
	return &DecisionReviewUnitRepo{decisionOptionRepo{db: db, table: "decision_review_unit", usage: []optionUsage{
		{`SELECT COUNT(*) FROM decision_draft_progress_history WHERE review_unit_id = $1`, "захирамжийн явцын бичлэг"},
	}}}
}

func (r *decisionOptionRepo) List(ctx context.Context) ([]*domain.DecisionOption, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, name, sort_order FROM `+r.table+` ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*domain.DecisionOption{}
	for rows.Next() {
		o := &domain.DecisionOption{}
		if err := rows.Scan(&o.ID, &o.Code, &o.Name, &o.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *decisionOptionRepo) GetByID(ctx context.Context, id int) (*domain.DecisionOption, error) {
	o := &domain.DecisionOption{}
	err := r.db.QueryRow(ctx,
		`SELECT id, code, name, sort_order FROM `+r.table+` WHERE id = $1`, id).
		Scan(&o.ID, &o.Code, &o.Name, &o.SortOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return o, nil
}

func (r *decisionOptionRepo) Create(ctx context.Context, o *domain.DecisionOption) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO `+r.table+` (code, name, sort_order) VALUES ($1,$2,$3) RETURNING id`,
		o.Code, o.Name, o.SortOrder).Scan(&o.ID)
}

func (r *decisionOptionRepo) Update(ctx context.Context, o *domain.DecisionOption) error {
	res, err := r.db.Exec(ctx,
		`UPDATE `+r.table+` SET code=$1, name=$2, sort_order=$3 WHERE id=$4`,
		o.Code, o.Name, o.SortOrder, o.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *decisionOptionRepo) Delete(ctx context.Context, id int) error {
	o, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	var parts []string
	for _, u := range r.usage {
		var n int
		if err := r.db.QueryRow(ctx, u.countSQL, id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, u.label))
		}
	}
	if len(parts) > 0 {
		return &domain.InUseError{Message: fmt.Sprintf(
			"«%s»-г ашиглаж байгаа тул устгах боломжгүй: %s.", o.Name, strings.Join(parts, ", "))}
	}
	res, err := r.db.Exec(ctx, `DELETE FROM `+r.table+` WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
