package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// stubProjectService satisfies service.ProjectService with controllable fns.
type stubProjectService struct {
	createFn    func(ctx context.Context, callerID uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	addMemberFn func(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error
}

func (s *stubProjectService) List(_ context.Context, _ uint) ([]*dto.ProjectResponse, error) {
	return nil, nil
}
func (s *stubProjectService) Create(ctx context.Context, callerID uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	return s.createFn(ctx, callerID, req)
}
func (s *stubProjectService) FindByKey(_ context.Context, _ string, _ uint) (*dto.ProjectResponse, error) {
	return nil, nil
}
func (s *stubProjectService) Update(_ context.Context, _ string, _ uint, _ dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	return nil, nil
}
func (s *stubProjectService) Delete(_ context.Context, _ string, _ uint) error { return nil }
func (s *stubProjectService) ListMembers(_ context.Context, _ string, _ uint) ([]*dto.MemberResponse, error) {
	return nil, nil
}
func (s *stubProjectService) AddMember(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error {
	if s.addMemberFn != nil {
		return s.addMemberFn(ctx, key, callerID, req)
	}
	return nil
}
func (s *stubProjectService) RemoveMember(_ context.Context, _ string, _ uint, _ uint) error {
	return nil
}

const setupTestSecret = "test-setup-secret-long-enough-for-hmac"

func newSetupSvc(repo domain.UserRepository, projSvc service.ProjectService) service.SetupService {
	return service.NewSetupService(repo, projSvc, setupTestSecret, 15*time.Minute, zap.NewNop(), nil)
}

// --- SetupRequired ---

func TestSetupService_SetupRequired_TrueWhenNoAdmin(t *testing.T) {
	svc := newSetupSvc(newStubUserRepo(), &stubProjectService{})
	required, err := svc.SetupRequired(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !required {
		t.Error("expected setup required=true when no admin exists")
	}
}

func TestSetupService_SetupRequired_FalseWhenAdminExists(t *testing.T) {
	repo := newStubUserRepo()
	repo.Create(context.Background(), &domain.User{ //nolint:errcheck
		Name: "Alice", Email: "alice@example.com", Password: "hash", IsAdmin: true,
	})
	svc := newSetupSvc(repo, &stubProjectService{})
	required, err := svc.SetupRequired(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if required {
		t.Error("expected setup required=false when admin exists")
	}
}

// --- CreateAdmin ---

func TestSetupService_CreateAdmin_Succeeds(t *testing.T) {
	svc := newSetupSvc(newStubUserRepo(), &stubProjectService{})
	resp, err := svc.CreateAdmin(context.Background(), dto.SetupAdminRequest{
		FullName: "Alice Admin",
		Email:    "alice@example.com",
		Password: "Secret123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("unexpected email: %s", resp.User.Email)
	}
}

func TestSetupService_CreateAdmin_Returns_ErrSetupComplete_WhenAdminExists(t *testing.T) {
	repo := newStubUserRepo()
	repo.Create(context.Background(), &domain.User{ //nolint:errcheck
		Name: "Alice", Email: "alice@example.com", Password: "hash", IsAdmin: true,
	})
	svc := newSetupSvc(repo, &stubProjectService{})
	_, err := svc.CreateAdmin(context.Background(), dto.SetupAdminRequest{
		FullName: "Bob", Email: "bob@example.com", Password: "Secret123",
	})
	if !errors.Is(err, service.ErrSetupComplete) {
		t.Errorf("expected ErrSetupComplete, got %v", err)
	}
}

// --- CreateUser ---

func TestSetupService_CreateUser_Succeeds(t *testing.T) {
	svc := newSetupSvc(newStubUserRepo(), &stubProjectService{})
	resp, err := svc.CreateUser(context.Background(), dto.SetupUserRequest{
		FullName: "Bob Member",
		Email:    "bob@example.com",
		Password: "Secret456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Email != "bob@example.com" {
		t.Errorf("unexpected email: %s", resp.Email)
	}
}

func TestSetupService_CreateUser_Returns_ErrConflict_OnDuplicateEmail(t *testing.T) {
	repo := newStubUserRepo()
	repo.Create(context.Background(), &domain.User{ //nolint:errcheck
		Name: "Bob", Email: "bob@example.com", Password: "hash",
	})
	svc := newSetupSvc(repo, &stubProjectService{})
	_, err := svc.CreateUser(context.Background(), dto.SetupUserRequest{
		FullName: "Bob2", Email: "bob@example.com", Password: "Secret456",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// --- CreateProject ---

func TestSetupService_CreateProject_Succeeds_NoMembership(t *testing.T) {
	projSvc := &stubProjectService{
		createFn: func(_ context.Context, _ uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
			return &dto.ProjectResponse{ID: 1, Key: "mfp", Name: req.Name}, nil
		},
	}
	svc := newSetupSvc(newStubUserRepo(), projSvc)
	resp, err := svc.CreateProject(context.Background(), 1, dto.SetupProjectRequest{Name: "My First Project"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProjectKey != "mfp" {
		t.Errorf("unexpected key: %s", resp.ProjectKey)
	}
}

func TestSetupService_CreateProject_Succeeds_WithMembership(t *testing.T) {
	repo := newStubUserRepo()
	repo.Create(context.Background(), &domain.User{ //nolint:errcheck
		Name: "Bob", Email: "bob@example.com", Password: "hash",
	})
	// The stub assigns ID=1 on create; retrieve it to get the actual ID
	bob, _ := repo.FindByEmail(context.Background(), "bob@example.com")

	memberCalled := false
	projSvc := &stubProjectService{
		createFn: func(_ context.Context, _ uint, _ dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
			return &dto.ProjectResponse{ID: 1, Key: "mfp", Name: "proj"}, nil
		},
		addMemberFn: func(_ context.Context, key string, _ uint, req dto.AddMemberRequest) error {
			memberCalled = true
			if req.Role != "member" {
				t.Errorf("expected role=member, got %s", req.Role)
			}
			if req.Email != "bob@example.com" {
				t.Errorf("expected bob's email, got %s", req.Email)
			}
			return nil
		},
	}
	svc := newSetupSvc(repo, projSvc)
	_, err := svc.CreateProject(context.Background(), 1, dto.SetupProjectRequest{
		Name:      "My Project",
		AddUserID: bob.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !memberCalled {
		t.Error("expected AddMember to be called")
	}
}

func TestSetupService_CreateProject_Returns_NotFound_WhenUserMissing(t *testing.T) {
	projSvc := &stubProjectService{
		createFn: func(_ context.Context, _ uint, _ dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
			t.Error("Create should not be called when user lookup fails")
			return nil, nil
		},
	}
	svc := newSetupSvc(newStubUserRepo(), projSvc)
	_, err := svc.CreateProject(context.Background(), 1, dto.SetupProjectRequest{
		Name:      "My Project",
		AddUserID: 999,
	})
	if err == nil {
		t.Fatal("expected error when user not found")
	}
}

// --- SeedData integration tests (real SQLite) ---

var setupSvcDBCounter int

func newSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	setupSvcDBCounter++
	dsn := fmt.Sprintf("file:setup_svc_test_%d?mode=memory&cache=shared", setupSvcDBCounter)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Project{},
		&domain.ProjectMember{},
		&domain.Sprint{},
		&domain.Issue{},
		&domain.Label{},
		&domain.IssueLabel{},
		&domain.Comment{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newSetupSvcWithDB(db *gorm.DB) service.SetupService {
	userRepo := repository.NewUserRepository(db)
	return service.NewSetupService(userRepo, &stubProjectService{}, setupTestSecret, 15*time.Minute, zap.NewNop(), db)
}

func TestSetupService_SeedData_SuccessMapsSeedResult(t *testing.T) {
	db := newSeedTestDB(t)
	admin := &domain.User{Name: "Admin", Email: "admin@example.com", Password: "hash", IsAdmin: true}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	svc := newSetupSvcWithDB(db)
	resp, err := svc.SeedData(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("SeedData: %v", err)
	}
	if resp.Skipped {
		t.Error("expected skipped=false on first seed")
	}
	if resp.ProjectKey != "DEMO" {
		t.Errorf("expected project_key=DEMO, got %s", resp.ProjectKey)
	}
	if resp.ProjectName != "Mansooba Demo" {
		t.Errorf("expected project_name=Mansooba Demo, got %s", resp.ProjectName)
	}
}

func TestSetupService_SeedData_SkippedOnSecondCall(t *testing.T) {
	db := newSeedTestDB(t)
	admin := &domain.User{Name: "Admin", Email: "admin@example.com", Password: "hash", IsAdmin: true}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	svc := newSetupSvcWithDB(db)
	if _, err := svc.SeedData(context.Background(), admin.ID); err != nil {
		t.Fatalf("first SeedData: %v", err)
	}

	resp, err := svc.SeedData(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("second SeedData: %v", err)
	}
	if !resp.Skipped {
		t.Error("expected skipped=true on second seed")
	}
}
