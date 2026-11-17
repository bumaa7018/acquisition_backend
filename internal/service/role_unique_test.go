package service_test

import (
	"context"
	"errors"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRoleRepo — зөвхөн хэрэглэгдэх аргууд (бусад нь nil интерфэйс).
type fakeRoleRepo struct {
	repository.RoleRepository
	roles   []*domain.Role
	codes   map[string]domain.ID
	created *domain.Role
	updated *domain.Role
}

func (f *fakeRoleRepo) List(context.Context) ([]*domain.Role, error) { return f.roles, nil }
func (f *fakeRoleRepo) CodeTaken(_ context.Context, code string, exclude domain.ID) (bool, error) {
	id, ok := f.codes[code]
	return ok && id != exclude, nil
}
func (f *fakeRoleRepo) Create(_ context.Context, r *domain.Role) error { f.created = r; return nil }
func (f *fakeRoleRepo) GetByID(_ context.Context, id domain.ID) (*domain.Role, error) {
	for _, r := range f.roles {
		if r.ID == id {
			cp := *r
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeRoleRepo) Update(_ context.Context, r *domain.Role) error { f.updated = r; return nil }

func seededRoles() *fakeRoleRepo {
	return &fakeRoleRepo{
		roles: []*domain.Role{
			{ID: 2, Name: "senior_specialist", Resource: "system", Code: "A101"},
			{ID: 6, Name: "compensation_finance_specialist", Resource: "compensation"},
			{ID: 139, Name: "compensation_mika", Resource: "compensation"},
			{ID: 300, Name: "Газрын шинжээч", Code: "GS01"},
		},
		codes: map[string]domain.ID{"A101": 2, "GS01": 300, "X999": 9999}, // X999 — өөр системийнх
	}
}

func TestRoleCreate_RejectsDuplicateOrReservedNames(t *testing.T) {
	for _, name := range []string{
		"senior_specialist",   // байгаа нэр
		"  Газрын   ШИНЖЭЭЧ ", // том/жижиг үсэг, зай
		"mika",                // compensation_mika-ийн ТҮВШИН
		"finance_specialist",  // compensation_finance_specialist-ийн түвшин
		"Ахлах мэргэжилтэн",   // системийн монгол нэр (C6)
		"Санхүү", "МИКА", "Захирамж бүртгэл", "department_head",
	} {
		repo := seededRoles()
		_, err := service.NewRoleService(repo).Create(context.Background(), service.CreateRoleRequest{Name: name})
		var rule *domain.RuleError
		assert.True(t, errors.As(err, &rule), "%q: %v", name, err)
		assert.Nil(t, repo.created, name)
	}
}

func TestRoleCreate_RejectsDuplicateCode(t *testing.T) {
	for _, code := range []string{"A101", "X999", "GS01", "TOOLONGCODE1"} {
		repo := seededRoles()
		_, err := service.NewRoleService(repo).Create(context.Background(),
			service.CreateRoleRequest{Name: "Шинэ роль", Code: code})
		var rule *domain.RuleError
		assert.True(t, errors.As(err, &rule), "%q: %v", code, err)
		assert.Nil(t, repo.created)
	}
}

func TestRoleCreate_AllowsUniqueNameAndCode(t *testing.T) {
	repo := seededRoles()
	role, err := service.NewRoleService(repo).Create(context.Background(),
		service.CreateRoleRequest{Name: " Шинэ роль ", Code: "NR01"})
	require.NoError(t, err)
	assert.Equal(t, "Шинэ роль", role.Name)
	assert.Equal(t, "NR01", role.Code)
}

func TestRoleRename_RejectsDuplicateButAllowsOwnName(t *testing.T) {
	repo := seededRoles()
	svc := service.NewRoleService(repo)
	_, err := svc.Update(context.Background(), 300, service.UpdateRoleRequest{Name: "Ахлах мэргэжилтэн"})
	var rule *domain.RuleError
	assert.True(t, errors.As(err, &rule))
	_, err = svc.Update(context.Background(), 300, service.UpdateRoleRequest{Name: "compensation_mika"})
	assert.True(t, errors.As(err, &rule))
	// Өөрийн нэрийн бичлэгийг (зай) засах — өөртэйгөө давхцал гэж тооцохгүй.
	_, err = svc.Update(context.Background(), 300, service.UpdateRoleRequest{Name: "Газрын  шинжээч"})
	require.NoError(t, err)
}
