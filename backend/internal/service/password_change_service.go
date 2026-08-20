package service

import (
	"context"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// suspiciousActivityThreshold is the number of consecutive failed
// change-password attempts that triggers an alert email (FR-013).
const suspiciousActivityThreshold = 3

// PasswordChangeService lets an authenticated user change their own password.
type PasswordChangeService interface {
	ChangePassword(ctx context.Context, userID uint, req dto.ChangePasswordRequest) error
}

type passwordChangeService struct {
	userRepo     domain.UserRepository
	systemLogSvc SystemLogService
	emailSender  domain.EmailSender
}

// NewPasswordChangeService returns a PasswordChangeService.
func NewPasswordChangeService(
	userRepo domain.UserRepository,
	systemLogSvc SystemLogService,
	emailSender domain.EmailSender,
) PasswordChangeService {
	return &passwordChangeService{userRepo: userRepo, systemLogSvc: systemLogSvc, emailSender: emailSender}
}

func (s *passwordChangeService) ChangePassword(ctx context.Context, userID uint, req dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)) != nil {
		s.recordFailure(ctx, user, "incorrect current password")
		return domain.ErrCurrentPasswordMismatch
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)) == nil {
		s.recordFailure(ctx, user, "new password reused current password")
		return domain.ErrNewPasswordSameAsCurrent
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)

	// Truncated to whole seconds to match jwt.NewNumericDate's own
	// truncation (golang-jwt/jwt/v5's TimePrecision defaults to
	// time.Second) — otherwise a refresh token reissued for the acting
	// session within the same second as this change would compare as
	// "issued before TokenValidAfter" purely from sub-second rounding and
	// get incorrectly rejected on its very next use.
	now := time.Now().Truncate(time.Second)
	user.TokenValidAfter = &now
	user.ConsecutiveFailedPasswordChanges = 0

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAuthentication,
		Action:   "password_change_success",
		Outcome:  "success",
		Actor:    user.Email,
		ActorID:  user.ID,
	})

	if err := s.emailSender.SendPasswordChanged(ctx, user.Email); err != nil {
		logger.Logger.Warn("password-changed email delivery failed",
			zap.String("event", "password_changed_email_failed"), zap.Error(err))
	}

	return nil
}

// recordFailure logs the rejection, increments the account's
// consecutive-failure counter, and — on the 3rd consecutive failure —
// sends a suspicious-activity alert (FR-013).
func (s *passwordChangeService) recordFailure(ctx context.Context, user *domain.User, reason string) {
	s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAuthentication,
		Action:   "password_change_failed",
		Outcome:  "failure",
		Actor:    user.Email,
		ActorID:  user.ID,
		Detail:   reason,
	})

	user.ConsecutiveFailedPasswordChanges++
	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.Logger.Warn("failed to persist consecutive-failure counter",
			zap.String("event", "password_change_counter_update_failed"), zap.Error(err))
		return
	}

	if user.ConsecutiveFailedPasswordChanges == suspiciousActivityThreshold {
		if err := s.emailSender.SendSuspiciousActivityAlert(ctx, user.Email); err != nil {
			logger.Logger.Warn("suspicious-activity alert email delivery failed",
				zap.String("event", "suspicious_activity_email_failed"), zap.Error(err))
		}
	}
}
