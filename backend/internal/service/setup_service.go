package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/seed"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SetupService orchestrates the first-run wizard steps.
type SetupService interface {
	// SetupRequired returns true when no admin user exists (fresh install).
	SetupRequired(ctx context.Context) (bool, error)
	// CreateAdmin creates the initial admin account and issues a JWT.
	// Returns ErrSetupComplete if an admin already exists.
	CreateAdmin(ctx context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error)
	// CreateUser creates an optional non-admin team member.
	CreateUser(ctx context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error)
	// CreateProject creates an optional first project. If req.AddUserID > 0,
	// adds that user as a "member" of the project (atomic — project not created on 404).
	CreateProject(ctx context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error)
	// SeedData populates the workspace with demo content. Delegates to the seed package.
	// Returns Skipped=true if seed data already exists (idempotent).
	SeedData(ctx context.Context, adminID uint) (*dto.SetupSeedResponse, error)
}

// ErrSetupComplete is returned when a wizard step is called after setup finished.
var ErrSetupComplete = errors.New("setup is already complete")

type setupService struct {
	userRepo   domain.UserRepository
	projectSvc ProjectService
	jwtSecret  string
	accessTTL  time.Duration
	log        *zap.Logger
	db         *gorm.DB
}

// NewSetupService returns a SetupService implementation.
func NewSetupService(
	userRepo domain.UserRepository,
	projectSvc ProjectService,
	jwtSecret string,
	accessTTL time.Duration,
	log *zap.Logger,
	db *gorm.DB,
) SetupService {
	return &setupService{
		userRepo:   userRepo,
		projectSvc: projectSvc,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		log:        log,
		db:         db,
	}
}

func (s *setupService) SetupRequired(ctx context.Context) (bool, error) {
	s.log.Info("setup status check initiated")
	hasAdmin, err := s.userRepo.HasAdmin(ctx)
	if err != nil {
		return false, err
	}
	return !hasAdmin, nil
}

func (s *setupService) CreateAdmin(ctx context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error) {
	hasAdmin, err := s.userRepo.HasAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if hasAdmin {
		return nil, ErrSetupComplete
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Name: req.FullName, Email: req.Email, Password: string(hash), IsAdmin: true}
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error("admin creation failed", zap.Error(err))
		return nil, err
	}

	accessToken, err := generateAccessToken(user.ID, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, err
	}

	s.log.Info("admin account created", zap.Uint("user_id", user.ID))
	return &dto.AuthResponse{
		AccessToken: accessToken,
		User:        dto.UserDTO{ID: user.ID, Name: user.Name, Email: user.Email},
	}, nil
}

func (s *setupService) CreateUser(ctx context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error) {
	if _, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil {
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Name: req.FullName, Email: req.Email, Password: string(hash), IsAdmin: false}
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error("user creation failed", zap.Error(err))
		return nil, err
	}

	s.log.Info("team member created", zap.Uint("user_id", user.ID))
	return &dto.SetupUserResponse{UserID: user.ID, Name: user.Name, Email: user.Email}, nil
}

func (s *setupService) CreateProject(ctx context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error) {
	// If AddUserID is set, validate the user exists before creating the project.
	var targetEmail string
	if req.AddUserID > 0 {
		target, err := s.userRepo.FindByID(ctx, req.AddUserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
			}
			return nil, err
		}
		targetEmail = target.Email
	}

	proj, err := s.projectSvc.Create(ctx, adminID, dto.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		s.log.Error("project creation failed", zap.Error(err))
		return nil, err
	}

	if targetEmail != "" {
		if err := s.projectSvc.AddMember(ctx, proj.Key, adminID, dto.AddMemberRequest{
			Email: targetEmail,
			Role:  "member",
		}); err != nil {
			s.log.Error("project membership failed", zap.Error(err))
			return nil, err
		}
	}

	s.log.Info("project created", zap.Uint("project_id", proj.ID), zap.String("key", proj.Key))
	return &dto.SetupProjectResponse{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       proj.Name,
	}, nil
}

func (s *setupService) SeedData(ctx context.Context, adminID uint) (*dto.SetupSeedResponse, error) {
	result, err := seed.Seed(ctx, s.db, adminID)
	if err != nil {
		s.log.Error("seed failed", zap.Error(err))
		return nil, err
	}
	return &dto.SetupSeedResponse{
		Skipped:     result.Skipped,
		ProjectKey:  result.ProjectKey,
		ProjectName: result.ProjectName,
	}, nil
}
