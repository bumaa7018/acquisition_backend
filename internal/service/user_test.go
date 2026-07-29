package service_test

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserService_GetByID_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	id := uuid.New()
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
	id := uuid.New()
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
	existing := &domain.User{ID: uuid.New(), Username: "duplicate"}
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
	id := uuid.New()
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
	users := []*domain.User{{ID: uuid.New(), Email: "a@a.com", Password: "h"}}
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
	id := uuid.New()
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
	id := uuid.New()
	repo.On("GetByID", context.Background(), id).Return(nil, domain.ErrNotFound)

	svc := service.NewUserService(repo)
	err := svc.ChangePassword(context.Background(), id, "NewPassword123!")

	require.ErrorIs(t, err, domain.ErrNotFound)
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}
