package service

import (
	"context"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// AdminUserService provides admin-only operations on user accounts.
type AdminUserService interface {
	// ListUsers returns a paginated list of all users with admin metadata.
	ListUsers(ctx context.Context, page, size int) (*dto.AdminUserListResponse, error)
	// GetUser returns a single user's admin DTO by ID.
	GetUser(ctx context.Context, id uint) (*dto.AdminUserDTO, error)
	// SetRole promotes or demotes a user's admin status.
	// Returns ErrLastAdmin if the action would leave zero active admins.
	SetRole(ctx context.Context, callerID, targetID uint, isAdmin bool) error
	// SetActive enables or disables a user account.
	// Returns ErrLastAdmin if disabling would leave zero active admins.
	SetActive(ctx context.Context, callerID, targetID uint, isActive bool) error
}

type adminUserService struct {
	userRepo domain.UserRepository
}

// NewAdminUserService returns an AdminUserService backed by the given repository.
func NewAdminUserService(userRepo domain.UserRepository) AdminUserService {
	return &adminUserService{userRepo: userRepo}
}

func (s *adminUserService) GetUser(ctx context.Context, id uint) (*dto.AdminUserDTO, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &dto.AdminUserDTO{
		ID: u.ID, Name: u.Name, Email: u.Email,
		IsAdmin: u.IsAdmin, IsActive: u.IsActive, CreatedAt: u.CreatedAt,
	}, nil
}

func (s *adminUserService) ListUsers(ctx context.Context, page, size int) (*dto.AdminUserListResponse, error) {
	users, total, err := s.userRepo.ListAll(ctx, page, size)
	if err != nil {
		return nil, err
	}

	dtos := make([]dto.AdminUserDTO, len(users))
	for i, u := range users {
		dtos[i] = dto.AdminUserDTO{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			IsAdmin:   u.IsAdmin,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
		}
	}

	return &dto.AdminUserListResponse{
		Users: dtos,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

func (s *adminUserService) SetRole(ctx context.Context, _, targetID uint, isAdmin bool) error {
	target, err := s.userRepo.FindByID(ctx, targetID)
	if err != nil {
		return err
	}

	// Guard: cannot demote the last active admin.
	if target.IsAdmin && !isAdmin && target.IsActive {
		count, err := s.userRepo.CountActiveAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrLastAdmin
		}
	}

	target.IsAdmin = isAdmin
	return s.userRepo.UpdateAdminFields(ctx, target)
}

func (s *adminUserService) SetActive(ctx context.Context, _, targetID uint, isActive bool) error {
	target, err := s.userRepo.FindByID(ctx, targetID)
	if err != nil {
		return err
	}

	// Guard: cannot disable the last active admin.
	if target.IsAdmin && target.IsActive && !isActive {
		count, err := s.userRepo.CountActiveAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrLastAdmin
		}
	}

	target.IsActive = isActive
	return s.userRepo.UpdateAdminFields(ctx, target)
}
