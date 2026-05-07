package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"golang.org/x/crypto/bcrypt"
)

var errInvalidCredentials = errors.New("invalid credentials")

// AuthService defines the authentication business-logic contract.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (string, error)
}

type authService struct {
	userRepo   domain.UserRepository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService returns an AuthService backed by the given repository.
func NewAuthService(repo domain.UserRepository, jwtSecret, accessTTL, refreshTTL string) AuthService {
	aTTL, _ := time.ParseDuration(accessTTL)
	rTTL, _ := time.ParseDuration(refreshTTL)
	return &authService{userRepo: repo, jwtSecret: jwtSecret, accessTTL: aTTL, refreshTTL: rTTL}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	if _, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil {
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Name: req.FullName, Email: req.Email, Password: string(hash)}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildResponse(user)
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errInvalidCredentials
	}

	return s.buildResponse(user)
}

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

	return generateToken(userID, s.jwtSecret, s.accessTTL)
}

func (s *authService) buildResponse(user *domain.User) (*dto.AuthResponse, error) {
	accessToken, err := generateToken(user.ID, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponse{
		AccessToken: accessToken,
		User:        dto.UserDTO{ID: user.ID, Name: user.Name, Email: user.Email},
	}, nil
}

func generateToken(userID uint, secret string, ttl time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
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
