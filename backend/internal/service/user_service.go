package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/pkg/avatarstorage"
)

// UserService manages user profile read and update operations.
type UserService interface {
	GetProfile(ctx context.Context, userID uint) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
	UploadAvatar(ctx context.Context, userID uint, filename string, data []byte, contentType string) (*dto.UserProfileResponse, error)
	DeleteAvatar(ctx context.Context, userID uint) (*dto.UserProfileResponse, error)
}

type userServiceImpl struct {
	userRepo    domain.UserRepository
	avatarStore *avatarstorage.Storage
}

// NewUserService returns a UserService backed by the given repository.
func NewUserService(userRepo domain.UserRepository) UserService {
	return &userServiceImpl{
		userRepo:    userRepo,
		avatarStore: avatarstorage.New("uploads/avatars"),
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

	url, err := s.avatarStore.Save(userID, filename, data, contentType)
	if err != nil {
		return nil, err
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

	if err := s.avatarStore.Delete(userID); err != nil {
		return nil, err
	}

	user.AvatarURL = ""
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return toProfileResponse(user), nil
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
