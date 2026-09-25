package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// UserService manages user profile read and update operations.
type UserService interface {
	GetProfile(ctx context.Context, userID uint) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
	UploadAvatar(ctx context.Context, userID uint, filename string, data []byte, contentType string) (*dto.UserProfileResponse, error)
	DeleteAvatar(ctx context.Context, userID uint) (*dto.UserProfileResponse, error)
	// GetAvatar streams the stored avatar named filename (e.g. "avatar-3.jpg")
	// and returns its content type. The caller must close the reader. Returns
	// domain.ErrNotFound if no such avatar exists.
	GetAvatar(ctx context.Context, filename string) (io.ReadCloser, string, error)
}

type userServiceImpl struct {
	userRepo    domain.UserRepository
	avatarStore domain.AvatarStore
}

// NewUserService returns a UserService backed by the given repository and
// avatar storage.
func NewUserService(userRepo domain.UserRepository, avatarStore domain.AvatarStore) UserService {
	return &userServiceImpl{
		userRepo:    userRepo,
		avatarStore: avatarStore,
	}
}

func (s *userServiceImpl) GetProfile(ctx context.Context, userID uint) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toProfileResponse(user), nil
}

func (s *userServiceImpl) UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FullName != "" {
		user.Name = req.FullName
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", req.Timezone, err)
		}
		user.Timezone = req.Timezone
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return toProfileResponse(user), nil
}

func (s *userServiceImpl) UploadAvatar(ctx context.Context, userID uint, filename string, data []byte, contentType string) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	url, err := s.avatarStore.Save(ctx, userID, filename, data, contentType)
	if err != nil {
		return nil, classifyAvatarStorageError(err)
	}

	user.AvatarURL = url
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return toProfileResponse(user), nil
}

func (s *userServiceImpl) DeleteAvatar(ctx context.Context, userID uint) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.avatarStore.Delete(ctx, userID); err != nil {
		return nil, classifyAvatarStorageError(err)
	}

	user.AvatarURL = ""
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return toProfileResponse(user), nil
}

func (s *userServiceImpl) GetAvatar(ctx context.Context, filename string) (io.ReadCloser, string, error) {
	return s.avatarStore.Get(ctx, filename)
}

// classifyAvatarStorageError keeps client-side rejections as-is (their message
// is safe to show) and wraps everything else as a storage outage so handlers
// can answer 502 without echoing S3 endpoint details.
func classifyAvatarStorageError(err error) error {
	if errors.Is(err, domain.ErrAvatarRejected) {
		return err
	}
	return fmt.Errorf("%w: %v", domain.ErrAvatarStorageUnavailable, err)
}

func toProfileResponse(u *domain.User) *dto.UserProfileResponse {
	return &dto.UserProfileResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Timezone:  u.Timezone,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
	}
}
