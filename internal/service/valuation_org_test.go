package service_test

// Үнэлгээний байгууллагыг АЖИЛТНУУДЫНХ НЬ хамт бүртгэх урсгал:
//   - Байгууллага олон ажилтантай байж болно
//   - Ажилтны нэвтрэх эрх ЗААВАЛ БИШ (username хоосон бол зүгээр бүртгэл)
//   - Засварлах нь ХЭЗЭЭ Ч шинэ байгууллага үүсгэхгүй (Update → Update)
//   - Нууц үг repo руу ХЭЗЭЭ Ч цэвэр текстээр хүрэхгүй (hash хийгдэнэ)

import (
	"context"
	"errors"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeValuationOrgRepo — дуудлагыг бүртгэж, оролтыг шалгах хамгийн бага repo.
type fakeValuationOrgRepo struct {
	createCalls int
	updateCalls int
	lastOrg     *domain.ValuationOrg
	lastEmps    []domain.ValuationOrgEmployee
	updateErr   error
}

func (f *fakeValuationOrgRepo) List(context.Context, domain.ValuationOrgFilter) ([]*domain.ValuationOrg, int64, error) {
	return nil, 0, nil
}

func (f *fakeValuationOrgRepo) GetByID(_ context.Context, id domain.ID) (*domain.ValuationOrg, error) {
	return &domain.ValuationOrg{ID: id, Name: "Тест Үнэлгээ ХХК"}, nil
}

func (f *fakeValuationOrgRepo) Create(_ context.Context, org *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	f.createCalls++
	f.lastOrg, f.lastEmps = org, emps
	org.ID = 7
	return nil
}

func (f *fakeValuationOrgRepo) Update(_ context.Context, org *domain.ValuationOrg, emps []domain.ValuationOrgEmployee) error {
	f.updateCalls++
	f.lastOrg, f.lastEmps = org, emps
	return f.updateErr
}

func (f *fakeValuationOrgRepo) Delete(context.Context, domain.ID) error { return nil }

func validEmployee() domain.ValuationOrgEmployee {
	return domain.ValuationOrgEmployee{
		LastName:     "Бат",
		FirstName:    "Дорж",
		RegisterNo:   "УБ11111111",
		Phone:        "99110011",
		Email:        "dorj@example.mn",
		PositionName: "Үнэлгээчин",
	}
}

func orgRequest(emps ...domain.ValuationOrgEmployee) service.ValuationOrgRequest {
	return service.ValuationOrgRequest{
		Org:       domain.ValuationOrg{Name: "Тест Үнэлгээ ХХК", IsActive: true},
		Employees: emps,
	}
}

// Засварлах үед ажилтан нэмэхэд ШИНЭ БАЙГУУЛЛАГА ҮҮСЭХГҮЙ — энэ л мөчлөг
// эвдэрсэн байсан (frontend employees[] илгээдэг, backend уншдаггүй байв).
func TestValuationOrg_UpdateWithEmployeesNeverCreates(t *testing.T) {
	repo := &fakeValuationOrgRepo{}
	svc := service.NewValuationOrgService(repo)

	emp := validEmployee()
	emp.Username = "dorj"
	emp.Password = "Testpass123!"

	_, err := svc.Update(context.Background(), 42, orgRequest(emp))
	require.NoError(t, err)

	assert.Equal(t, 0, repo.createCalls, "засварлах нь Create дуудах ёсгүй")
	assert.Equal(t, 1, repo.updateCalls)
	assert.Equal(t, domain.ID(42), repo.lastOrg.ID, "URL-ийн id тэр чигээр дамжина")
	require.Len(t, repo.lastEmps, 1)
}

// Байгууллага олон ажилтантай байна; нэвтрэх эрх нь ажилтан тус бүрт заавал биш.
func TestValuationOrg_ManyEmployeesLoginOptional(t *testing.T) {
	repo := &fakeValuationOrgRepo{}
	svc := service.NewValuationOrgService(repo)

	withLogin := validEmployee()
	withLogin.Username = "dorj"
	withLogin.Password = "Testpass123!"

	noLogin := validEmployee()
	noLogin.RegisterNo = "УБ22222222"
	noLogin.FirstName = "Оюун"

	_, err := svc.Create(context.Background(), orgRequest(withLogin, noLogin))
	require.NoError(t, err)

	require.Len(t, repo.lastEmps, 2)
	assert.Equal(t, "dorj", repo.lastEmps[0].Username)
	assert.Empty(t, repo.lastEmps[1].Username, "нэвтрэх эрхгүй ажилтан ч бүртгэгдэнэ")
	assert.Empty(t, repo.lastEmps[1].Password)
}

// Нууц үг repo руу цэвэр текстээр хүрэх ёсгүй.
func TestValuationOrg_PasswordIsHashedBeforeRepo(t *testing.T) {
	repo := &fakeValuationOrgRepo{}
	svc := service.NewValuationOrgService(repo)

	emp := validEmployee()
	emp.Username = "dorj"
	emp.Password = "Testpass123!"

	_, err := svc.Create(context.Background(), orgRequest(emp))
	require.NoError(t, err)

	require.Len(t, repo.lastEmps, 1)
	assert.NotEqual(t, "Testpass123!", repo.lastEmps[0].Password)
	assert.NotEmpty(t, repo.lastEmps[0].Password)
}

// employees талбарыг ОГТ илгээхгүй бол ажилтнуудад хүрэхгүй (nil дамжина) —
// байгууллагын мэдээллийг л засах тохиолдол ажилтныг санамсаргүй хасах ёсгүй.
func TestValuationOrg_NilEmployeesLeavesStaffUntouched(t *testing.T) {
	repo := &fakeValuationOrgRepo{}
	svc := service.NewValuationOrgService(repo)

	_, err := svc.Update(context.Background(), 5, service.ValuationOrgRequest{
		Org: domain.ValuationOrg{Name: "Тест Үнэлгээ ХХК", IsActive: true},
	})
	require.NoError(t, err)
	assert.Nil(t, repo.lastEmps, "nil бол repo ажилтнуудад хүрэхгүй")
}

// Хоосон массив нь "бүх ажилтныг хас" гэсэн ЗОРИУДЫН үйлдэл — nil-ээс ялгаатай.
func TestValuationOrg_EmptyEmployeesClearsStaff(t *testing.T) {
	repo := &fakeValuationOrgRepo{}
	svc := service.NewValuationOrgService(repo)

	_, err := svc.Update(context.Background(), 5, service.ValuationOrgRequest{
		Org:       domain.ValuationOrg{Name: "Тест Үнэлгээ ХХК", IsActive: true},
		Employees: []domain.ValuationOrgEmployee{},
	})
	require.NoError(t, err)
	assert.NotNil(t, repo.lastEmps)
	assert.Empty(t, repo.lastEmps)
}

func TestValuationOrg_RejectsInvalidEmployees(t *testing.T) {
	dupRegister := validEmployee()
	other := validEmployee()
	other.FirstName = "Оюун"

	noFirstName := validEmployee()
	noFirstName.FirstName = "  "

	noRegister := validEmployee()
	noRegister.RegisterNo = ""

	dupUserA := validEmployee()
	dupUserA.Username = "dorj"
	dupUserA.Password = "Testpass123!"
	dupUserB := validEmployee()
	dupUserB.RegisterNo = "УБ22222222"
	dupUserB.Username = "dorj"
	dupUserB.Password = "Testpass123!"

	cases := []struct {
		name string
		emps []domain.ValuationOrgEmployee
	}{
		{"нэр хоосон", []domain.ValuationOrgEmployee{noFirstName}},
		{"регистр хоосон", []domain.ValuationOrgEmployee{noRegister}},
		{"регистр давхардсан", []domain.ValuationOrgEmployee{dupRegister, other}},
		{"username давхардсан", []domain.ValuationOrgEmployee{dupUserA, dupUserB}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeValuationOrgRepo{}
			svc := service.NewValuationOrgService(repo)

			_, err := svc.Update(context.Background(), 1, orgRequest(tc.emps...))
			assert.ErrorIs(t, err, domain.ErrInvalidInput)
			assert.Equal(t, 0, repo.updateCalls, "шалгалт өнгөрөөгүй бол бичихгүй")
			assert.Equal(t, 0, repo.createCalls)
		})
	}
}

// Байгууллагын нэр хоосон бол бүртгэхгүй.
func TestValuationOrg_RejectsEmptyName(t *testing.T) {
	repo := &fakeValuationOrgRepo{}
	svc := service.NewValuationOrgService(repo)

	_, err := svc.Create(context.Background(), service.ValuationOrgRequest{
		Org: domain.ValuationOrg{Name: "   "},
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Equal(t, 0, repo.createCalls)
}

// Repo-ийн ErrNotFound (id-гаар мөр олдсонгүй) дуудагч руу тэр чигээр гарна —
// "олдсонгүй"-г "шинээр үүсгэ" болгож хувиргах ёсгүй.
func TestValuationOrg_UpdateNotFoundPropagates(t *testing.T) {
	repo := &fakeValuationOrgRepo{updateErr: domain.ErrNotFound}
	svc := service.NewValuationOrgService(repo)

	_, err := svc.Update(context.Background(), 404, orgRequest(validEmployee()))
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	assert.Equal(t, 0, repo.createCalls)
}
