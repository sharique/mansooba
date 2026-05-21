package service_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUserService() (service.UserService, *stubUserRepo) {
	repo := newStubUserRepo()
	_ = repo.Create(context.Background(), &domain.User{
		ID: 1, Name: "Alice", Email: "alice@example.com", Password: "hash",
	})
	return service.NewUserService(repo), repo
}

func TestUserService_GetProfile_ReturnsProfile(t *testing.T) {
	svc, _ := newTestUserService()

	resp, err := svc.GetProfile(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), resp.ID)
	assert.Equal(t, "Alice", resp.Name)
	assert.Equal(t, "alice@example.com", resp.Email)
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	svc, _ := newTestUserService()

	_, err := svc.GetProfile(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_UpdateProfile_ChangesName(t *testing.T) {
	svc, _ := newTestUserService()

	resp, err := svc.UpdateProfile(context.Background(), 1, dto.UpdateProfileRequest{
		FullName: "Alice Updated",
	})
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", resp.Name)
}

func TestUserService_UpdateProfile_ChangesTimezone(t *testing.T) {
	svc, _ := newTestUserService()

	resp, err := svc.UpdateProfile(context.Background(), 1, dto.UpdateProfileRequest{
		Timezone: "America/New_York",
	})
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", resp.Timezone)
}

func TestUserService_UpdateProfile_RejectsInvalidTimezone(t *testing.T) {
	svc, _ := newTestUserService()

	_, err := svc.UpdateProfile(context.Background(), 1, dto.UpdateProfileRequest{
		Timezone: "Mars/Olympus",
	})
	assert.Error(t, err)
}

func TestUserService_UpdateProfile_NotFound(t *testing.T) {
	svc, _ := newTestUserService()

	_, err := svc.UpdateProfile(context.Background(), 999, dto.UpdateProfileRequest{
		FullName: "Ghost",
	})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
