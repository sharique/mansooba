package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────────────
// Stubs
// ──────────────────────────────────────────────────────────────────────────────

type stubUserRepo struct {
	users  map[string]*domain.User
	nextID uint
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{users: make(map[string]*domain.User), nextID: 1}
}

func (s *stubUserRepo) Create(_ context.Context, u *domain.User) error {
	if _, exists := s.users[u.Email]; exists {
		return domain.ErrConflict
	}
	if u.ID == 0 {
		u.ID = s.nextID
		s.nextID++
	}
	copied := *u
	s.users[u.Email] = &copied
	return nil
}

func (s *stubUserRepo) FindByID(_ context.Context, id uint) (*domain.User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *stubUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := s.users[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (s *stubUserRepo) FindByEmailPrefix(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (s *stubUserRepo) Update(_ context.Context, u *domain.User) error {
	for email, existing := range s.users {
		if existing.ID == u.ID {
			cp := *u
			s.users[email] = &cp
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *stubUserRepo) HasAdmin(_ context.Context) (bool, error) {
	for _, u := range s.users {
		if u.IsAdmin {
			return true, nil
		}
	}
	return false, nil
}

func (s *stubUserRepo) FindFirstAdmin(_ context.Context) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (s *stubUserRepo) ListAll(_ context.Context, _, _ int) ([]*domain.User, int64, error) {
	return nil, 0, nil
}

func (s *stubUserRepo) CountActiveAdmins(_ context.Context) (int64, error) { return 0, nil }

func (s *stubUserRepo) UpdateAdminFields(_ context.Context, _ *domain.User) error { return nil }

// stubRevokedRepo controls Exists and Create outcomes.
type stubRevokedRepo struct {
	existsFn func(jti string) (bool, error)
	records  map[string]bool
}

func newStubRevokedRepo() *stubRevokedRepo {
	return &stubRevokedRepo{records: make(map[string]bool)}
}

func (r *stubRevokedRepo) Create(_ context.Context, token *domain.RevokedToken) error {
	r.records[token.JTI] = true
	return nil
}

func (r *stubRevokedRepo) Exists(_ context.Context, jti string) (bool, error) {
	if r.existsFn != nil {
		return r.existsFn(jti)
	}
	return r.records[jti], nil
}

func (r *stubRevokedRepo) DeleteExpired(_ context.Context) (int64, error) { return 0, nil }

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

const testSecret = "test-secret-key-that-is-long-enough"

func newTestService(userRepo domain.UserRepository) service.AuthService {
	return service.NewAuthService(userRepo, newStubRevokedRepo(), zap.NewNop(), testSecret, "15m", "168h")
}

func newTestServiceWith(userRepo domain.UserRepository, revokedRepo domain.RevokedTokenRepository) service.AuthService {
	return service.NewAuthService(userRepo, revokedRepo, zap.NewNop(), testSecret, "15m", "168h")
}

func newObservedService(userRepo domain.UserRepository, revokedRepo domain.RevokedTokenRepository) (service.AuthService, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.WarnLevel)
	return service.NewAuthService(userRepo, revokedRepo, zap.New(core), testSecret, "15m", "168h"), logs
}

func openServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.RevokedToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// ──────────────────────────────────────────────────────────────────────────────
// Existing tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAuthService_Register_Succeeds(t *testing.T) {
	svc := newTestService(newStubUserRepo())
	resp, err := svc.Register(context.Background(), dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", resp.User.Email)
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	repo := newStubUserRepo()
	svc := newTestService(repo)
	req := dto.RegisterRequest{FullName: "Alice", Email: "alice@example.com", Password: "password123"}
	_, _ = svc.Register(context.Background(), req)
	_, err := svc.Register(context.Background(), req)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestAuthService_Login_Succeeds(t *testing.T) {
	repo := newStubUserRepo()
	svc := newTestService(repo)
	_, _ = svc.Register(context.Background(), dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	resp, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "alice@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newStubUserRepo()
	svc := newTestService(repo)
	_, _ = svc.Register(context.Background(), dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "alice@example.com", Password: "wrongpassword",
	})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestAuthService_Login_UnknownEmail_ReturnsGenericError(t *testing.T) {
	svc := newTestService(newStubUserRepo())
	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})
	if errors.Is(err, domain.ErrNotFound) {
		t.Error("login must not reveal whether the email exists")
	}
	if err == nil {
		t.Fatal("expected an error for unknown email")
	}
}

func TestAuthService_Refresh_ExpiredToken(t *testing.T) {
	svc := newTestService(newStubUserRepo())
	_, err := svc.Refresh(context.Background(), "this.is.not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for invalid/expired refresh token")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// T013: Revocation and logout tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAuthService_Register_ResponseContainsRefreshToken(t *testing.T) {
	svc := newTestService(newStubUserRepo())
	resp, err := svc.Register(context.Background(), dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh_token in AuthResponse")
	}
}

func TestAuthService_Logout_StoresRevocationRecord(t *testing.T) {
	// Use real SQLite — auth paths must not use mocks per Constitution Principle III.
	db := openServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	revokedRepo := repository.NewRevokedTokenRepository(db)
	svc := newTestServiceWith(userRepo, revokedRepo)
	ctx := context.Background()

	resp, err := svc.Register(ctx, dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.Logout(ctx, resp.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Subsequent refresh must fail with ErrTokenRevoked
	_, err = svc.Refresh(ctx, resp.RefreshToken)
	if !errors.Is(err, domain.ErrTokenRevoked) {
		t.Errorf("expected ErrTokenRevoked after logout, got %v", err)
	}
}

func TestAuthService_Logout_Idempotent_OnInvalidToken(t *testing.T) {
	svc := newTestService(newStubUserRepo())
	if err := svc.Logout(context.Background(), "not-a-valid-jwt"); err != nil {
		t.Errorf("Logout with invalid token should return nil, got %v", err)
	}
}

func TestAuthService_Logout_Idempotent_OnEmptyToken(t *testing.T) {
	svc := newTestService(newStubUserRepo())
	if err := svc.Logout(context.Background(), ""); err != nil {
		t.Errorf("Logout with empty token should return nil, got %v", err)
	}
}

func TestAuthService_Refresh_ReturnsErrTokenRevoked_ForRevokedJTI(t *testing.T) {
	revokedRepo := newStubRevokedRepo()
	repo := newStubUserRepo()
	svc := newTestServiceWith(repo, revokedRepo)
	ctx := context.Background()

	resp, _ := svc.Register(ctx, dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	_ = svc.Logout(ctx, resp.RefreshToken)

	_, err := svc.Refresh(ctx, resp.RefreshToken)
	if !errors.Is(err, domain.ErrTokenRevoked) {
		t.Errorf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestAuthService_Refresh_ReturnsErrRevocationStoreUnavailable_OnStoreError(t *testing.T) {
	revokedRepo := newStubRevokedRepo()
	revokedRepo.existsFn = func(_ string) (bool, error) {
		return false, errors.New("db connection refused")
	}
	repo := newStubUserRepo()
	// Register with a separate (working) repo to get a valid token
	workingSvc := newTestServiceWith(repo, newStubRevokedRepo())
	ctx := context.Background()
	resp, _ := workingSvc.Register(ctx, dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})

	// Now use the broken store — user exists and is active, but store errors
	brokenSvc := newTestServiceWith(repo, revokedRepo)
	_, err := brokenSvc.Refresh(ctx, resp.RefreshToken)
	if !errors.Is(err, domain.ErrRevocationStoreUnavailable) {
		t.Errorf("expected ErrRevocationStoreUnavailable, got %v", err)
	}
}

func TestAuthService_Refresh_ReturnsErrAccountDisabled_ForDisabledUser(t *testing.T) {
	repo := newStubUserRepo()
	svc := newTestService(repo)
	ctx := context.Background()

	resp, _ := svc.Register(ctx, dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	user, _ := repo.FindByEmail(ctx, "alice@example.com")
	user.IsActive = false
	_ = repo.Update(ctx, user)

	_, err := svc.Refresh(ctx, resp.RefreshToken)
	if !errors.Is(err, domain.ErrAccountDisabled) {
		t.Errorf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestAuthService_Refresh_LogsWARN_OnRevocation(t *testing.T) {
	revokedRepo := newStubRevokedRepo()
	repo := newStubUserRepo()
	svc, logs := newObservedService(repo, revokedRepo)
	ctx := context.Background()

	resp, _ := svc.Register(ctx, dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	_ = svc.Logout(ctx, resp.RefreshToken)
	_, _ = svc.Refresh(ctx, resp.RefreshToken)

	var found bool
	for _, entry := range logs.All() {
		if entry.Level != zapcore.WarnLevel {
			continue
		}
		hasUserID, hasJTI := false, false
		for _, f := range entry.Context {
			if f.Key == "user_id" {
				hasUserID = true
			}
			if f.Key == "jti" {
				hasJTI = true
			}
		}
		if hasUserID {
			if hasJTI {
				t.Error("WARN log must NOT include jti field (security: redact JTI)")
			}
			found = true
		}
	}
	if !found {
		t.Error("expected WARN-level log with user_id field after revoked token rejection")
	}
}

func TestAuthService_Login_IsActive_BlockedAtLogin(t *testing.T) {
	repo := newStubUserRepo()
	svc := newTestService(repo)
	ctx := context.Background()

	_, _ = svc.Register(ctx, dto.RegisterRequest{
		FullName: "Alice", Email: "alice@example.com", Password: "password123",
	})
	user, _ := repo.FindByEmail(ctx, "alice@example.com")
	user.IsActive = false
	_ = repo.Update(ctx, user)

	_, err := svc.Login(ctx, dto.LoginRequest{
		Email: "alice@example.com", Password: "password123",
	})
	if !errors.Is(err, domain.ErrAccountDisabled) {
		t.Errorf("expected ErrAccountDisabled for disabled user at login, got %v", err)
	}
}
