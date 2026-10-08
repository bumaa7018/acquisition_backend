package service

import (
	"context"
	"regexp"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type ParcelStatusService interface {
	List(ctx context.Context) ([]*domain.ParcelStatus, error)
	GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error)
	Create(ctx context.Context, req CreateParcelStatusRequest) (*domain.ParcelStatus, error)
	Update(ctx context.Context, id int, req UpdateParcelStatusRequest) (*domain.ParcelStatus, error)
	Delete(ctx context.Context, id int) error
}

// ТАГИЙН СОНГОЛТ: `binding:` (gin) — `validate:` БИШ.
//
// Энэ багцын handler нь `ShouldBindJSON`-оор задалдаг ба өөрийн
// `validator.New()` ХЭРЭГЛЭДЭГГҮЙ (бусад handler-уудаас ялгаатай). gin нь
// зөвхөн `binding:` тагийг уншдаг тул `validate:` таг бичвэл шалгалт ОГТ
// ажиллахгүй — таг нь зөвхөн чимэг болно.
type CreateParcelStatusRequest struct {
	Code      string `json:"code"       binding:"required"`
	Name      string `json:"name"       binding:"required"`
	SortOrder int    `json:"sort_order"`
	// Color — `#rrggbb`. Хоосон зөвшөөрнө: өнгө сонгоогүй төлөв нь дуудагч
	// талдаа саарал өгөгдмөлөөр харагдана.
	//
	// ТАГААР ШАЛГАХГҮЙ: validHexColor нь шалгахаас гадна ЖИЖИГ үсэг рүү
	// нэгтгэдэг тул шалгалт тэнд байх ёстой (HTTP-ээс гадуур дуудагдсан ч
	// ижил дүрэм үйлчилнэ). Хоёр газар шалгавал эх сурвалж хоёр болно.
	Color string `json:"color"`
	// Төлөвийн дүрэм (эцсийн, явц түгжих, үнэлгээний шат, чөлөөлөгдсөн,
	// шалтгаан заавал) — илгээгээгүй бол false.
	domain.ParcelStatusRules
}

type UpdateParcelStatusRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	// Color — ЗААВАЛ заагч. Гурван утга ЯЛГАГДАНА:
	//
	//	талбар байхгүй  → nil  → өнгийг ХЭВЭЭР үлдээнэ
	//	"color": ""     → ""   → өнгийг АРИЛГАНА (NULL)
	//	"color": "#..." → өнгө → шинэчилнэ
	//
	// Заагчгүй бол `""` нь "мэдэхгүй" ба "арилга" хоёрын аль нь болох нь
	// ялгагдахгүй тул өнгө арилгах гарц ОГТ байхгүй болно. Өнгийг мэддэггүй
	// клиент талбарыг ОГТ илгээхгүй тул санамсаргүй арилгахгүй.
	Color *string `json:"color"`
	// Төлөвийн дүрэм — Color-той ИЖИЛ: талбар ирээгүй (nil) бол ХЭВЭЭР.
	// Дүрмийг мэддэггүй хуучин клиент тэдгээрийг санамсаргүй арилгахгүй.
	IsFinal          *bool `json:"is_final"`
	LocksProgress    *bool `json:"locks_progress"`
	IsValuationStage *bool `json:"is_valuation_stage"`
	IsReleased       *bool `json:"is_released"`
	RequiresReason   *bool `json:"requires_reason"`
}

type parcelStatusService struct {
	repo repository.ParcelStatusRepository
}

func NewParcelStatusService(repo repository.ParcelStatusRepository) ParcelStatusService {
	return &parcelStatusService{repo: repo}
}

func (s *parcelStatusService) List(ctx context.Context) ([]*domain.ParcelStatus, error) {
	return s.repo.List(ctx)
}

func (s *parcelStatusService) GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *parcelStatusService) Create(ctx context.Context, req CreateParcelStatusRequest) (*domain.ParcelStatus, error) {
	color, err := validHexColor(req.Color)
	if err != nil {
		return nil, err
	}
	ps := &domain.ParcelStatus{
		Code:              req.Code,
		Name:              req.Name,
		SortOrder:         req.SortOrder,
		Color:             color,
		ParcelStatusRules: req.ParcelStatusRules,
	}
	if err := s.repo.Create(ctx, ps); err != nil {
		return nil, err
	}
	return ps, nil
}

func (s *parcelStatusService) Update(ctx context.Context, id int, req UpdateParcelStatusRequest) (*domain.ParcelStatus, error) {
	ps, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Code != "" {
		ps.Code = req.Code
	}
	if req.Name != "" {
		ps.Name = req.Name
	}
	ps.SortOrder = req.SortOrder
	// Өнгө: талбар ИРЭЭГҮЙ бол (nil) хэвээр; ирсэн бол хоосон нь ч хүчинтэй
	// утга — арилгана. Дэлгэрэнгүйг UpdateParcelStatusRequest.Color дээр үз.
	if req.Color != nil {
		color, err := validHexColor(*req.Color)
		if err != nil {
			return nil, err
		}
		ps.Color = color
	}
	for _, f := range []struct {
		in  *bool
		out *bool
	}{
		{req.IsFinal, &ps.IsFinal},
		{req.LocksProgress, &ps.LocksProgress},
		{req.IsValuationStage, &ps.IsValuationStage},
		{req.IsReleased, &ps.IsReleased},
		{req.RequiresReason, &ps.RequiresReason},
	} {
		if f.in != nil {
			*f.out = *f.in
		}
	}
	if err := s.repo.Update(ctx, ps); err != nil {
		return nil, err
	}
	return ps, nil
}

func (s *parcelStatusService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

// validHexColor — өнгийг шалгаж, ЖИЖИГ үсгээр нэгтгэнэ.
//
// ЯАГААД ЭНД (handler-т биш): энэ багц нь `ShouldBindJSON`-оор задалдаг
// бөгөөд gin нь `binding:` тагийг л уншдаг, `validate:`-ийг ҮЛ ХЭРЭГСДЭГ.
// Тиймээс struct-ийн таг дээр найдвал шалгалт ОГТ ажиллахгүй: буруу утга
// DB-ийн CHECK хүртэл очиж 500 (400 биш) болж байлаа.
//
// Нэгтгэлт нь чухал: DB-ийн CHECK `#AABBCC`-г ч зөвшөөрдөг тул ижил өнгө
// хоёр янзаар хадгалагдвал frontend-ийн харьцуулалт таарахгүй болно.
func validHexColor(c string) (string, error) {
	if c == "" {
		return "", nil
	}
	if !hexColorRe.MatchString(c) {
		// БООХГҮЙ: response.Error нь алдааг УТГААР нь (switch err) харьцуулдаг
		// тул `fmt.Errorf("%w", …)` боолт таарахгүй бөгөөд 400-ын оронд 500
		// буцаана.
		return "", domain.ErrInvalidInput
	}
	return strings.ToLower(c), nil
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
