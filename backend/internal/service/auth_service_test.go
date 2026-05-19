package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

// stubUserRepo is a minimal in-memory stand-in for domain.UserRepository.
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
	s.users[u.Email] = u
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

const testSecret = "test-secret-key-that-is-long-enough"

func newTestService(repo domain.UserRepository) service.AuthService {
	return service.NewAuthService(repo, testSecret, "15m", "168h")
}

func TestAuthService_Register_Succeeds(t *testing.T) {
	svc := newTestService(newStubUserRepo())

	resp, err := svc.Register(context.Background(), dto.RegisterRequest{
		FullName: "Alice",
		Email:    "alice@example.com",
		Password: "password123",
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

	// Must NOT return ErrNotFound — that would leak whether the email exists.
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
