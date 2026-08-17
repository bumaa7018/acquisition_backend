package service

import (
	"context"
	"log/slog"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type PlanService interface {
	Search(ctx context.Context, code string) (*domain.Plan, error)
	Suggest(ctx context.Context, q string) ([]*domain.Plan, error)
}

type planService struct {
	projects PlanProjectClient
	// store — appdb-ийн plan хүснэгт. Төлөвлөгөө ЗӨВХӨН энд хадгалагдаж,
	// ЗӨВХӨН эндээс уншигдана.
	store repository.PlanStoreRepository
}

// NewPlanService — store нь appdb-ийн plan хүснэгт (цорын ганц эх сурвалж).
func NewPlanService(store repository.PlanStoreRepository, extras ...any) PlanService {
	s := &planService{store: store}
	for _, extra := range extras {
		switch v := extra.(type) {
		case repository.PlanStoreRepository:
			s.store = v
		}
		// PlanProjectClient-ийг ParcelInfoClient-тай ижил обьект хэрэгжүүлдэг тул
		// switch-ийн case биш, тусад нь шалгана.
		if v, ok := extra.(PlanProjectClient); ok {
			s.projects = v
		}
	}
	return s
}

// Search — төлөвлөгөөг middleware /plan/project-ээс татаж appdb-ийн plan
// хүснэгтэд хадгална, дараа нь хадгалагдсан мөрийг appdb-ээс уншиж буцаана.
//
// Middleware нь шинэ өгөгдлийн цорын ганц эх сурвалж. Appdb-ийн plan хүснэгт
// нь зөвхөн middleware-ээс ирсэн датаны хадгалалт ба унших байрлал. Middleware
// дээр олдоогүй/алдаа гарсан үед appdb-ээс нөхөж хайхгүй.
func (s *planService) Search(ctx context.Context, code string) (*domain.Plan, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, domain.ErrInvalidInput
	}
	if s.projects == nil {
		return nil, domain.ErrNotFound
	}
	if s.store == nil {
		return nil, domain.ErrInternalServer
	}
	plan, err := s.projects.GetPlanProject(ctx, code)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, domain.ErrNotFound
	}
	if strings.TrimSpace(plan.PlanCode) == "" {
		plan.PlanCode = code
	}
	// Дундын сервис нэгж талбарын дугаарыг буцаагаагүй ч хайсан дугаараар бүртгэнэ.
	if strings.TrimSpace(plan.ParcelID) == "" {
		plan.ParcelID = code
	}

	saved, err := s.store.Upsert(ctx, plan)
	if err != nil {
		slog.Error("төлөвлөгөөг appdb-д хадгалахад алдаа",
			"parcel_id", code, "plan_code", plan.PlanCode, "error", err)
		return nil, domain.ErrInternalServer
	}

	stored, err := s.store.GetByCode(ctx, saved.PlanCode)
	if err != nil {
		slog.Error("төлөвлөгөөг appdb-ээс уншихад алдаа",
			"parcel_id", code, "plan_code", saved.PlanCode, "error", err)
		return nil, domain.ErrInternalServer
	}
	return stored, nil
}

// Suggest — appdb-д хадгалагдсан төлөвлөгөөнүүдээс хайна (жагсаалт/шүүлтүүрийн
// сонголт). Өөр эх сурвалж БАЙХГҮЙ: plan хүснэгт зөвхөн appdb дээр байна.
func (s *planService) Suggest(ctx context.Context, q string) ([]*domain.Plan, error) {
	if s.store == nil {
		return nil, domain.ErrInternalServer
	}
	return s.store.Suggest(ctx, q)
}
