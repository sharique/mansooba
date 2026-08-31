package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var errInvalidCredentials = errors.New("invalid credentials")

// AuthService defines the authentication business-logic contract.
type AuthService interface {
	// Register creates a user on behalf of callerID (the authenticated admin
	// making the call — AuthHandler.Register enforces admin-only access
	// before invoking this). callerID is used only to label the resulting
	// account_created System Log entry's actor (FR-002).
	Register(ctx context.Context, req dto.RegisterRequest, callerID uint) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	// Refresh validates a refresh token and returns a new access token string.
	// Returns ErrTokenRevoked, ErrRevocationStoreUnavailable, or ErrAccountDisabled
	// on rejection; the caller should NOT issue a new access token in those cases.
	Refresh(ctx context.Context, refreshToken string) (string, error)
	// Logout revokes the supplied refresh token so the Refresh endpoint will reject it.
	// If the token is invalid or already expired, Logout returns nil (idempotent).
	Logout(ctx context.Context, refreshToken string) error
	// IssueTokens mints a fresh access/refresh token pair for an already-verified
	// caller (no password check) — used by PasswordChangeHandler (ADR-032) to
	// keep the session that performed a password change active: since
	// TokenValidAfter invalidates every refresh token issued before the
	// change, including the one the acting session already holds, that
	// session needs a replacement pair issued *after* the new boundary.
	IssueTokens(ctx context.Context, userID uint) (*dto.AuthResponse, error)
}

type authService struct {
	userRepo     domain.UserRepository
	revokedRepo  domain.RevokedTokenRepository
	systemLogSvc SystemLogService
	log          *zap.Logger
	jwtSecret    string
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// NewAuthService returns an AuthService backed by the given repositories.
// TTL strings are parsed via time.ParseDuration; invalid strings default to zero.
func NewAuthService(
	userRepo domain.UserRepository,
	revokedRepo domain.RevokedTokenRepository,
	systemLogSvc SystemLogService,
	log *zap.Logger,
	jwtSecret, accessTTL, refreshTTL string,
) AuthService {
	aTTL, _ := time.ParseDuration(accessTTL)
	rTTL, _ := time.ParseDuration(refreshTTL)
	return &authService{
		userRepo:     userRepo,
		revokedRepo:  revokedRepo,
		systemLogSvc: systemLogSvc,
		log:          log,
		jwtSecret:    jwtSecret,
		accessTTL:    aTTL,
		refreshTTL:   rTTL,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest, callerID uint) (*dto.AuthResponse, error) {
	if _, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil {
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Name: req.FullName, Email: req.Email, Password: string(hash), IsActive: true}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Register is always invoked via the admin-only create-user flow
	// (AuthHandler.Register requires an admin caller) — every successful
	// call here is an auditable admin action (FR-002).
	s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAdminAction,
		Action:   "account_created",
		Outcome:  "success",
		Actor:    s.actorLabel(ctx, callerID),
		ActorID:  callerID,
		Target:   user.Email,
		TargetID: user.ID,
	})

	return s.buildResponse(ctx, user)
}

// actorLabel resolves callerID to a human-readable label (email) for System
// Log entries — falls back to a stable "user#<id>" form if the lookup fails,
// mirroring the same pattern in admin_user_service.go and setting_service.go
// (each service owns its own copy rather than sharing one, since none of
// them otherwise depend on each other).
func (s *authService) actorLabel(ctx context.Context, callerID uint) string {
	user, err := s.userRepo.FindByID(ctx, callerID)
	if err != nil {
		return fmt.Sprintf("user#%d", callerID)
	}
	return user.Email
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// No matching user — the actor is recorded as the attempted email
		// with no ActorID, per FR-004's "unknown identifier" case.
		s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
			Category: domain.SystemLogCategoryAuthentication,
			Action:   "login_failed",
			Outcome:  "failure",
			Actor:    req.Email,
			Detail:   "no account with this email",
		})
		return nil, errInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
			Category: domain.SystemLogCategoryAuthentication,
			Action:   "login_failed",
			Outcome:  "failure",
			Actor:    req.Email,
			ActorID:  user.ID,
			Detail:   "invalid credentials",
		})
		return nil, errInvalidCredentials
	}

	if !user.IsActive {
		s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
			Category: domain.SystemLogCategoryAuthentication,
			Action:   "login_failed",
			Outcome:  "failure",
			Actor:    req.Email,
			ActorID:  user.ID,
			Detail:   "account disabled",
		})
		return nil, domain.ErrAccountDisabled
	}

	s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAuthentication,
		Action:   "login_success",
		Outcome:  "success",
		Actor:    req.Email,
		ActorID:  user.ID,
	})

	return s.buildResponse(ctx, user)
}

// Refresh validates the refresh token, checks revocation, and returns a new access token.
// Checks are ordered: signature/expiry → is_active → revocation store.
func (s *authService) Refresh(ctx context.Context, refreshToken string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return "", err
	}

	userID, err := parseUintSubject(claims.Subject)
	if err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", errInvalidCredentials
	}

	if !user.IsActive {
		return "", domain.ErrAccountDisabled
	}

	// Fail-closed: a store error is safer than silently issuing tokens.
	revoked, err := s.revokedRepo.Exists(ctx, claims.ID)
	if err != nil {
		return "", domain.ErrRevocationStoreUnavailable
	}
	if revoked {
		s.log.Warn("refresh rejected: token revoked",
			zap.Uint("user_id", userID),
			zap.String("reason", "jti_revoked"),
		)
		// A revoked-token refresh attempt is a signal of potential token
		// theft/reuse (FR-001) — durably recorded, unlike an ordinary expired
		// or malformed token, which isn't security-relevant on its own.
		s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
			Category: domain.SystemLogCategoryAuthentication,
			Action:   "refresh_rejected",
			Outcome:  "failure",
			Actor:    user.Email,
			ActorID:  userID,
			Detail:   "revoked token reuse attempt",
		})
		return "", domain.ErrTokenRevoked
	}

	if user.TokenValidAfter != nil && claims.IssuedAt.Time.Before(*user.TokenValidAfter) {
		s.log.Warn("refresh rejected: session invalidated by password change",
			zap.Uint("user_id", userID),
		)
		s.systemLogSvc.Record(ctx, domain.SystemLogEntry{
			Category: domain.SystemLogCategoryAuthentication,
			Action:   "refresh_rejected",
			Outcome:  "failure",
			Actor:    user.Email,
			ActorID:  userID,
			Detail:   "session invalidated by password change",
		})
		return "", domain.ErrTokenRevoked
	}

	return generateAccessToken(userID, s.jwtSecret, s.accessTTL)
}

// Logout stores the token's JTI in the revocation table.
// Invalid or already-expired tokens are silently ignored (idempotent logout).
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		// Invalid/expired tokens are treated as already-revoked — no error.
		return nil
	}

	if claims.ID == "" {
		return nil
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		expiresAt = time.Now().Add(s.refreshTTL)
	}

	userID, _ := parseUintSubject(claims.Subject)

	record := &domain.RevokedToken{
		JTI:       claims.ID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		RevokedAt: time.Now(),
	}
	return s.revokedRepo.Create(ctx, record)
}

// buildResponse generates both access and refresh tokens for the given user.
func (s *authService) IssueTokens(ctx context.Context, userID uint) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, user)
}

func (s *authService) buildResponse(ctx context.Context, user *domain.User) (*dto.AuthResponse, error) {
	accessToken, err := generateAccessToken(user.ID, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshToken(user.ID, s.jwtSecret, s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         dto.UserDTO{ID: user.ID, Name: user.Name, Email: user.Email},
	}, nil
}

// generateAccessToken creates a short-lived JWT for API authentication.
func generateAccessToken(userID uint, secret string, ttl time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		Subject:   uintToString(userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// generateRefreshToken creates a long-lived JWT for obtaining new access tokens.
// The JTI is a UUID v4 stored in the revocation table on logout.
func generateRefreshToken(userID uint, secret string, ttl time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		Subject:   uintToString(userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func uintToString(id uint) string {
	return fmt.Sprintf("%d", id)
}

func parseUintSubject(s string) (uint, error) {
	var id uint64
	_, err := fmt.Sscanf(s, "%d", &id)
	return uint(id), err
}
