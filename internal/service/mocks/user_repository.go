package mocks

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) CreateWithEmployee(ctx context.Context, user *domain.User, employee *domain.Employee, person *domain.Person) error {
	args := m.Called(ctx, user, employee, person)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id domain.ID, hashed string) error {
	return m.Called(ctx, id, hashed).Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id domain.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmployeeID(ctx context.Context, employeeID domain.ID) (*domain.User, error) {
	args := m.Called(ctx, employeeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// OrgIDForUser — токен үүсгэх бүрд дуудагдана. Тестүүд ихэвчлэн тохируулдаггүй
// тул хүлээлт тавиагүй үед nil (харьяалалгүй) буцаана.
func (m *MockUserRepository) OrgIDForUser(ctx context.Context, userID domain.ID) (*domain.ID, error) {
	for _, call := range m.ExpectedCalls {
		if call.Method == "OrgIDForUser" {
			args := m.Called(ctx, userID)
			id, _ := args.Get(0).(*domain.ID)
			return id, args.Error(1)
		}
	}
	return nil, nil
}

func (m *MockUserRepository) GetRoles(ctx context.Context, userID domain.ID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserRepository) GetPermissions(ctx context.Context, userID domain.ID) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserRepository) InvalidateSessions(ctx context.Context, userID domain.ID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) IsSessionValid(ctx context.Context, userID domain.ID, issuedAt time.Time) (bool, error) {
	args := m.Called(ctx, userID, issuedAt)
	return args.Bool(0), args.Error(1)
}
