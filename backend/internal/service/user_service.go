package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

// UserService manages user profile read and update operations.
type UserService interface {
	GetProfile(ctx context.Context, userID uint) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
}

type userServiceImpl struct {
	userRepo domain.UserRepository
}

// NewUserService returns a UserService backed by the given repository.
func NewUserService(userRepo domain.UserRepository) UserService {
	return &userServiceImpl{userRepo: userRepo}
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

func toProfileResponse(u *domain.User) *dto.UserProfileResponse {
	return &dto.UserProfileResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Timezone:  u.Timezone,
		CreatedAt: u.CreatedAt,
	}
}
