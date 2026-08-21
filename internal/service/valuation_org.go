package service

import (
	"context"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/repository"
)

// ValuationOrgService — үнэлгээний (мэргэжлийн) байгууллагыг АЖИЛТНУУДЫНХ НЬ
// хамт бүртгэнэ. Байгууллага бүр олон ажилтантай; ажилтан бүр өөрийн нэвтрэх
// эрхтэй байж болно (заавал биш). Идэвхтэй ажилтан системд нэвтэрч, тухайн
// байгууллагад оноогдсон чөлөөлөлт дээр ажиллана.
type ValuationOrgService interface {
	List(ctx context.Context, filter domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error)
	Create(ctx context.Context, req ValuationOrgRequest) (*domain.ValuationOrg, error)
	Update(ctx context.Context, id domain.ID, req ValuationOrgRequest) (*domain.ValuationOrg, error)
	Delete(ctx context.Context, id domain.ID) error
}

type ValuationOrgRequest struct {
	Org domain.ValuationOrg
	// Employees nil бол ажилтнуудад хүрэхгүй (зөвхөн байгууллагын мэдээлэл
	// засварлана). nil биш бол жагсаалт нь БҮРЭН төлөвийг илэрхийлнэ.
	Employees []domain.ValuationOrgEmployee
}

type valuationOrgService struct {
	repo repository.ValuationOrgRepository
}

func NewValuationOrgService(repo repository.ValuationOrgRepository) ValuationOrgService {
	return &valuationOrgService{repo: repo}
}

func (s *valuationOrgService) List(ctx context.Context, filter domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *valuationOrgService) GetByID(ctx context.Context, id domain.ID) (*domain.ValuationOrg, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *valuationOrgService) Create(ctx context.Context, req ValuationOrgRequest) (*domain.ValuationOrg, error) {
	org, emps, err := prepareValuationOrg(req)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, org, emps); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, org.ID)
}

// Update — ЗӨВХӨН байгаа байгууллагыг засна. Ажилтан нэмэх нь шинэ байгууллага
// үүсгэх шалтгаан болохгүй: id-гаар мөр олдохгүй бол repo ErrNotFound буцаана.
func (s *valuationOrgService) Update(ctx context.Context, id domain.ID, req ValuationOrgRequest) (*domain.ValuationOrg, error) {
	org, emps, err := prepareValuationOrg(req)
	if err != nil {
		return nil, err
	}
	org.ID = id
	if err := s.repo.Update(ctx, org, emps); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *valuationOrgService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.Delete(ctx, id)
}

// prepareValuationOrg — оролтыг цэвэрлэж шалгана, нууц үгийг hash хийнэ
// (repo нь ЗӨВХӨН hash хүлээж авдаг — тэр нь SQL руу шууд бичдэг).
func prepareValuationOrg(req ValuationOrgRequest) (*domain.ValuationOrg, []domain.ValuationOrgEmployee, error) {
	org := req.Org
	org.Name = strings.TrimSpace(org.Name)
	if org.Name == "" {
		return nil, nil, domain.ErrInvalidInput
	}
	org.ShortName = strings.TrimSpace(org.ShortName)
	org.RegisterNo = strings.TrimSpace(org.RegisterNo)
	org.LicenseNo = strings.TrimSpace(org.LicenseNo)
	org.Phone = strings.TrimSpace(org.Phone)
	org.Email = strings.TrimSpace(org.Email)
	org.Address = strings.TrimSpace(org.Address)

	if req.Employees == nil {
		return &org, nil, nil
	}

	emps := make([]domain.ValuationOrgEmployee, 0, len(req.Employees))
	seenRegister := make(map[string]bool, len(req.Employees))
	seenUsername := make(map[string]bool, len(req.Employees))
	for _, e := range req.Employees {
		e.LastName = strings.TrimSpace(e.LastName)
		e.FirstName = strings.TrimSpace(e.FirstName)
		e.RegisterNo = strings.TrimSpace(e.RegisterNo)
		e.Phone = strings.TrimSpace(e.Phone)
		e.Email = strings.TrimSpace(e.Email)
		e.PositionName = strings.TrimSpace(e.PositionName)
		e.Username = strings.TrimSpace(e.Username)

		// Нэр, регистргүй ажилтан бүртгэх утгагүй — регистр нь хүнийг таних
		// түлхүүр бөгөөс байгууллага дотор давхардлыг мөн үүгээр шалгана.
		if e.FirstName == "" || e.RegisterNo == "" {
			return nil, nil, domain.ErrInvalidInput
		}
		// Нэг байгууллагын дотор нэг хүн хоёр удаа орж болохгүй.
		if seenRegister[e.RegisterNo] {
			return nil, nil, domain.ErrInvalidInput
		}
		seenRegister[e.RegisterNo] = true
		if e.Username != "" {
			if seenUsername[e.Username] {
				return nil, nil, domain.ErrInvalidInput
			}
			seenUsername[e.Username] = true
		}

		if e.Password != "" {
			hashed, err := hash.Password(e.Password)
			if err != nil {
				return nil, nil, err
			}
			e.Password = hashed
		}
		emps = append(emps, e)
	}
	return &org, emps, nil
}
