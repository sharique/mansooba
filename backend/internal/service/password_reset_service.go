package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ErrTokenExpired is returned by ResetPassword when the token's ExpiresAt is in the past.
var ErrTokenExpired = errors.New("reset token has expired")

// PasswordResetService handles the forgot-password / reset-password flow.
type PasswordResetService interface {
	// ForgotPassword generates a reset token for the given email address.
	// If no account matches, a neutral response is returned without error
	// to prevent user enumeration (FR-004).
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error)

	// ResetPassword verifies the token, updates the password, and deletes the token.
	// Returns ErrNotFound when the token does not exist (used or never issued).
	// Returns ErrTokenExpired when the token's ExpiresAt is in the past.
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
}

const resetTokenTTL = 15 * time.Minute

type passwordResetService struct {
	userRepo    domain.UserRepository
	resetRepo   domain.PasswordResetRepository
	emailSender domain.EmailSender
}

// NewPasswordResetService returns a PasswordResetService.
func NewPasswordResetService(
	userRepo domain.UserRepository,
	resetRepo domain.PasswordResetRepository,
	emailSender domain.EmailSender,
) PasswordResetService {
	return &passwordResetService{
		userRepo:    userRepo,
		resetRepo:   resetRepo,
		emailSender: emailSender,
	}
}

func (s *passwordResetService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	log := logger.Logger

	log.Info("password reset requested", zap.String("event", "password_reset_requested"))

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		log.Warn("password reset for unknown email", zap.String("event", "password_reset_unknown_email"))
		return nil, domain.ErrNotFound
	}

	// Generate 32 cryptographically random bytes → 64-char hex token.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(raw)
	hash := sha256hex(token)
	expiresAt := time.Now().Add(resetTokenTTL)

	if err := s.resetRepo.Upsert(ctx, &domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("upsert reset token: %w", err)
	}

	if err := s.emailSender.SendPasswordReset(ctx, req.Email, token); err != nil {
		log.Warn("password reset email delivery failed",
			zap.String("event", "password_reset_email_failed"),
			zap.Error(err),
		)
	}

	log.Info("password reset token issued",
		zap.String("event", "password_reset_token_issued"),
		zap.Uint("user_id", user.ID),
		zap.Time("expires_at", expiresAt),
	)

	return &dto.ForgotPasswordResponse{
		Token:     token,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		Message:   "A password reset token has been generated.",
	}, nil
}

func (s *passwordResetService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	log := logger.Logger

	hash := sha256hex(req.Token)
	record, err := s.resetRepo.FindByHash(ctx, hash)
	if err != nil {
		log.Warn("password reset token invalid",
			zap.String("event", "password_reset_token_invalid"),
			zap.String("reason", "not_found"),
		)
		return domain.ErrNotFound
	}

	if time.Now().After(record.ExpiresAt) {
		log.Warn("password reset token invalid",
			zap.String("event", "password_reset_token_invalid"),
			zap.String("reason", "expired"),
		)
		return ErrTokenExpired
	}

	user, err := s.userRepo.FindByID(ctx, record.UserID)
	if err != nil {
		return domain.ErrNotFound
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.Password = string(hashed)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := s.resetRepo.Delete(ctx, record.ID); err != nil {
		return fmt.Errorf("delete token: %w", err)
	}

	log.Info("password reset completed",
		zap.String("event", "password_reset_completed"),
		zap.Uint("user_id", user.ID),
	)
	return nil
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
