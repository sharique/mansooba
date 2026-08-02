package service

import (
	"context"
	"fmt"

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
	userRepo     domain.UserRepository
	systemLogSvc SystemLogService
}

// NewAdminUserService returns an AdminUserService backed by the given repository.
func NewAdminUserService(userRepo domain.UserRepository, systemLogSvc SystemLogService) AdminUserService {
	return &adminUserService{userRepo: userRepo, systemLogSvc: systemLogSvc}
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

func (s *adminUserService) SetRole(ctx context.Context, callerID, targetID uint, isAdmin bool) error {
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
	if err := s.userRepo.UpdateAdminFields(ctx, target); err != nil {
		return err
	}

	s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAdminAction,
		Action:   "role_changed",
		Outcome:  "success",
		Actor:    s.actorLabel(ctx, callerID),
		ActorID:  callerID,
		Target:   target.Email,
		TargetID: target.ID,
		Detail:   fmt.Sprintf("is_admin: %t -> %t", !isAdmin, isAdmin),
	})
	return nil
}

func (s *adminUserService) SetActive(ctx context.Context, callerID, targetID uint, isActive bool) error {
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
	if err := s.userRepo.UpdateAdminFields(ctx, target); err != nil {
		return err
	}

	action := "account_disabled"
	if isActive {
		action = "account_enabled"
	}
	s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAdminAction,
		Action:   action,
		Outcome:  "success",
		Actor:    s.actorLabel(ctx, callerID),
		ActorID:  callerID,
		Target:   target.Email,
		TargetID: target.ID,
	})
	return nil
}

// actorLabel resolves callerID to a human-readable label (email) for
// System Logs (FR-004). Falls back to a numeric placeholder if the caller's
// own record can't be read — this must never block the action it's
// describing (FR-012's best-effort principle extends to this lookup too).
func (s *adminUserService) actorLabel(ctx context.Context, callerID uint) string {
	caller, err := s.userRepo.FindByID(ctx, callerID)
	if err != nil {
		return fmt.Sprintf("user#%d", callerID)
	}
	return caller.Email
}
