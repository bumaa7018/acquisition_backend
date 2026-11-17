package service_test

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserService_GetByID_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	id := domain.ID(1)
	user := &domain.User{ID: id, Email: "u@example.com", Password: "hashed"}
	repo.On("GetByID", context.Background(), id).Return(user, nil)

	svc := service.NewUserService(repo)
	got, err := svc.GetByID(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Empty(t, got.Password)
	repo.AssertExpectations(t)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	id := domain.ID(2)
	repo.On("GetByID", context.Background(), id).Return(nil, domain.ErrNotFound)

	svc := service.NewUserService(repo)
	_, err := svc.GetByID(context.Background(), id)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_Create_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("GetByUsername", context.Background(), "").Return(nil, domain.ErrNotFound)
	repo.On("Create", context.Background(), mock_any_user()).Return(nil)

	svc := service.NewUserService(repo)
	active := true
	user, err := svc.Create(context.Background(), service.CreateUserRequest{
		Email:     "new@example.com",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "User",
		IsActive:  &active,
	})

	require.NoError(t, err)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Empty(t, user.Password)
	repo.AssertExpectations(t)
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	existing := &domain.User{ID: domain.ID(3), Username: "duplicate"}
	repo.On("GetByUsername", context.Background(), "duplicate").Return(existing, nil)

	svc := service.NewUserService(repo)
	_, err := svc.Create(context.Background(), service.CreateUserRequest{
		Username: "duplicate",
		Email:    "dup@example.com",
		Password: "Password123!",
	})

	assert.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestUserService_Delete_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	id := domain.ID(4)
	repo.On("Delete", context.Background(), id).Return(nil)
	// Устгасны дараа сесс тасалдана — устгагдсан хэрэглэгчийн токен хүчинтэй
	// хэвээр байхаас сэргийлнэ.
	repo.On("InvalidateSessions", context.Background(), id).Return(nil)

	svc := service.NewUserService(repo)
	err := svc.Delete(context.Background(), id)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserService_List_DefaultPagination(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	users := []*domain.User{{ID: domain.ID(5), Email: "a@a.com", Password: "h"}}
	// Expect page=1, pageSize=20 defaults
	repo.On("List", context.Background(), domain.UserFilter{Page: 1, PageSize: 20}).
		Return(users, int64(1), nil)

	svc := service.NewUserService(repo)
	got, total, err := svc.List(context.Background(), domain.UserFilter{})

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, got, 1)
	assert.Empty(t, got[0].Password)
}

// Нууц үг солиход хуучин токенууд хүчингүй болно.
func TestUserService_ChangePassword_InvalidatesSessions(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	id := domain.ID(6)
	repo.On("GetByID", context.Background(), id).Return(&domain.User{ID: id}, nil)
	repo.On("UpdatePassword", context.Background(), id, mock.MatchedBy(func(hashed string) bool {
		// Түүхий нууц үг хадгалагдахгүй — hash болсон байх ёстой.
		return hashed != "" && hashed != "NewPassword123!"
	})).Return(nil)
	repo.On("InvalidateSessions", context.Background(), id).Return(nil)

	svc := service.NewUserService(repo)
	require.NoError(t, svc.ChangePassword(context.Background(), id, "NewPassword123!"))

	repo.AssertExpectations(t)
}

// Байхгүй хэрэглэгчийн нууц үг солигдохгүй.
func TestUserService_ChangePassword_NotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	id := domain.ID(7)
	repo.On("GetByID", context.Background(), id).Return(nil, domain.ErrNotFound)

	svc := service.NewUserService(repo)
	err := svc.ChangePassword(context.Background(), id, "NewPassword123!")

	require.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}

// Идэвхгүй болгоход токен шууд хүчингүй болно; бусад засварт сесс хөндөгдөхгүй.
func TestUserService_Update_DeactivateInvalidatesSessions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		wasActive  bool
		setActive  *bool
		invalidate bool
	}{
		{"идэвхгүй болгох", true, boolPtr(false), true},
		{"нэр л засах", true, nil, false},
		{"идэвхжүүлэх", false, boolPtr(true), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mocks.MockUserRepository{}
			id := domain.ID(9)
			repo.On("GetByID", mock.Anything, id).Return(&domain.User{ID: id, IsActive: tc.wasActive}, nil)
			repo.On("Update", mock.Anything, mock.Anything).Return(nil)
			repo.On("InvalidateSessions", mock.Anything, id).Return(nil).Maybe()
			svc := service.NewUserService(repo)

			_, err := svc.Update(context.Background(), id, service.UpdateUserRequest{FirstName: "Бат", IsActive: tc.setActive})
			require.NoError(t, err)
			if tc.invalidate {
				repo.AssertCalled(t, "InvalidateSessions", mock.Anything, id)
			} else {
				repo.AssertNotCalled(t, "InvalidateSessions", mock.Anything, mock.Anything)
			}
		})
	}
}

func boolPtr(v bool) *bool { return &v }
