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
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	// Refresh validates a refresh token and returns a new access token string.
	// Returns ErrTokenRevoked, ErrRevocationStoreUnavailable, or ErrAccountDisabled
	// on rejection; the caller should NOT issue a new access token in those cases.
	Refresh(ctx context.Context, refreshToken string) (string, error)
	// Logout revokes the supplied refresh token so the Refresh endpoint will reject it.
	// If the token is invalid or already expired, Logout returns nil (idempotent).
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	userRepo    domain.UserRepository
	revokedRepo domain.RevokedTokenRepository
	log         *zap.Logger
	jwtSecret   string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewAuthService returns an AuthService backed by the given repositories.
// TTL strings are parsed via time.ParseDuration; invalid strings default to zero.
func NewAuthService(
	userRepo domain.UserRepository,
	revokedRepo domain.RevokedTokenRepository,
	log *zap.Logger,
	jwtSecret, accessTTL, refreshTTL string,
) AuthService {
	aTTL, _ := time.ParseDuration(accessTTL)
	rTTL, _ := time.ParseDuration(refreshTTL)
	return &authService{
		userRepo:    userRepo,
		revokedRepo: revokedRepo,
		log:         log,
		jwtSecret:   jwtSecret,
		accessTTL:   aTTL,
		refreshTTL:  rTTL,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
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

	return s.buildResponse(ctx, user)
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errInvalidCredentials
	}

	if !user.IsActive {
		return nil, domain.ErrAccountDisabled
	}

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
