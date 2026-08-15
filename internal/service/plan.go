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
	planRepo repository.PlanRepository
	projects PlanProjectClient
	// store — MAIN DB-ийн plan хүснэгт. Төлөвлөгөөгөөр хайх/сонгох үйлдэл
	// (Suggest) ҮҮНЭЭС уншина; nil үед auth DB-ийн planRepo руу унана.
	store repository.PlanStoreRepository
}

func NewPlanService(planRepo repository.PlanRepository, extras ...any) PlanService {
	s := &planService{planRepo: planRepo}
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

// Search — төлөвлөгөөг дугаараар дундын сервисээс (middleware /plan/project)
// хайна. Дугаар нь ЗААВАЛ байх ёстой.
//
// Эх сурвалж нь ЗӨВХӨН дундын сервис: тэндээс олдоогүй бол хадгалагдсан
// (plan хүснэгт) ч, auth DB-ийн төсөөллийн мэдээлэл ч хайгдахгүй — алдаа шууд
// дамжина (404 → ErrNotFound, холболтын алдаа → ErrInternalServer).
func (s *planService) Search(ctx context.Context, code string) (*domain.Plan, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, domain.ErrInvalidInput
	}
	if s.projects == nil {
		return nil, domain.ErrNotFound
	}
	plan, err := s.projects.GetPlanProject(ctx, code)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, domain.ErrNotFound
	}
	// Дундын сервис нэгж талбарын дугаарыг буцаагаагүй ч хайсан дугаараар бүртгэнэ.
	if strings.TrimSpace(plan.ParcelID) == "" {
		plan.ParcelID = code
	}

	// ОЛДМОГЦ MAIN DB-д хадгална — чөлөөлөлт үүсгэх үед дахин татах шаардлагагүй.
	// Upsert нь plan_code дээр ON CONFLICT DO UPDATE хийдэг тул давхардсан
	// төлөвлөгөө шинэ мөр үүсгэхгүй, зөвхөн хоосон биш талбарууд шинэчлэгдэнэ.
	if s.store != nil {
		saved, err := s.store.Upsert(ctx, plan)
		if err != nil {
			// Хадгалалт унасан ч хайлтыг тасалдуулахгүй — чөлөөлөлт үүсгэх үед
			// төлөвлөгөө дахин татагдаж хадгалагдана.
			slog.Error("төлөвлөгөөг хадгалахад алдаа",
				"parcel_id", code, "plan_code", plan.PlanCode, "error", err)
			return plan, nil
		}
		return saved, nil
	}
	return plan, nil
}

// Suggest — хадгалагдсан төлөвлөгөөнүүдээс хайна (жагсаалт/шүүлтүүрийн сонголт).
func (s *planService) Suggest(ctx context.Context, q string) ([]*domain.Plan, error) {
	if s.store != nil {
		return s.store.Suggest(ctx, q)
	}
	return s.planRepo.Suggest(ctx, q)
}
