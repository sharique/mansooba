package service_test

// stubUserRepo and newStubUserRepo are defined in auth_service_test.go (same package).

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// stubPasswordResetRepo is an in-memory PasswordResetRepository for tests.
type stubPasswordResetRepo struct {
	tokens map[uint]*domain.PasswordResetToken
	byHash map[string]uint
	nextID uint
}

func newStubPasswordResetRepo() *stubPasswordResetRepo {
	return &stubPasswordResetRepo{
		tokens: make(map[uint]*domain.PasswordResetToken),
		byHash: make(map[string]uint),
		nextID: 1,
	}
}

func (r *stubPasswordResetRepo) Upsert(_ context.Context, token *domain.PasswordResetToken) error {
	for id, t := range r.tokens {
		if t.UserID == token.UserID {
			delete(r.byHash, t.TokenHash)
			delete(r.tokens, id)
		}
	}
	token.ID = r.nextID
	r.nextID++
	r.tokens[token.ID] = token
	r.byHash[token.TokenHash] = token.ID
	return nil
}

func (r *stubPasswordResetRepo) FindByHash(_ context.Context, hash string) (*domain.PasswordResetToken, error) {
	id, ok := r.byHash[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	t, ok := r.tokens[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (r *stubPasswordResetRepo) Delete(_ context.Context, id uint) error {
	t, ok := r.tokens[id]
	if !ok {
		return nil
	}
	delete(r.byHash, t.TokenHash)
	delete(r.tokens, id)
	return nil
}

func (r *stubPasswordResetRepo) PurgeExpired(_ context.Context, cutoff time.Time) (int64, error) {
	var count int64
	for id, t := range r.tokens {
		if t.CreatedAt.Before(cutoff) {
			delete(r.byHash, t.TokenHash)
			delete(r.tokens, id)
			count++
		}
	}
	return count, nil
}

type stubNoopEmailSender struct{}

func (stubNoopEmailSender) SendPasswordReset(_ context.Context, _, _ string) error { return nil }

func newPRSvc(userRepo domain.UserRepository, resetRepo domain.PasswordResetRepository) service.PasswordResetService {
	return service.NewPasswordResetService(userRepo, resetRepo, stubNoopEmailSender{})
}

func seedUser(t *testing.T, repo *stubUserRepo, email string) *domain.User {
	t.Helper()
	u := &domain.User{Email: email, Password: "placeholder"}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

// ─── ForgotPassword ──────────────────────────────────────────────────────────

func TestForgotPassword_KnownEmail_ReturnsToken(t *testing.T) {
	users := newStubUserRepo()
	seedUser(t, users, "alice@example.com")
	svc := newPRSvc(users, newStubPasswordResetRepo())

	resp, err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "alice@example.com"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Token) != 64 {
		t.Errorf("expected 64-char hex token, got len=%d", len(resp.Token))
	}
	if resp.ExpiresAt == "" {
		t.Error("expected non-empty expires_at")
	}
}

func TestForgotPassword_UnknownEmail_ReturnsErrNotFound(t *testing.T) {
	svc := newPRSvc(newStubUserRepo(), newStubPasswordResetRepo())

	resp, err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "nobody@example.com"})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for unknown email, got %v", err)
	}
	if resp != nil {
		t.Error("response must be nil when email is not found")
	}
}

func TestForgotPassword_SecondCall_InvalidatesFirstToken(t *testing.T) {
	users := newStubUserRepo()
	seedUser(t, users, "alice@example.com")
	resetRepo := newStubPasswordResetRepo()
	svc := newPRSvc(users, resetRepo)

	first, _ := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "alice@example.com"})
	_, _ = svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "alice@example.com"})

	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:    first.Token,
		Password: "newPassword8",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("first token must be invalidated after second call; got %v", err)
	}
	if len(resetRepo.tokens) != 1 {
		t.Errorf("expected exactly 1 active token after second call, found %d", len(resetRepo.tokens))
	}
}

// ─── ResetPassword ───────────────────────────────────────────────────────────

func TestResetPassword_ValidToken_Succeeds(t *testing.T) {
	users := newStubUserRepo()
	seedUser(t, users, "alice@example.com")
	resetRepo := newStubPasswordResetRepo()
	svc := newPRSvc(users, resetRepo)

	resp, _ := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "alice@example.com"})

	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:    resp.Token,
		Password: "newPassword8",
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(resetRepo.tokens) != 0 {
		t.Error("token must be deleted after successful reset (delete-on-use)")
	}
}

func TestResetPassword_ExpiredToken_ReturnsErrTokenExpired(t *testing.T) {
	users := newStubUserRepo()
	seedUser(t, users, "alice@example.com")
	resetRepo := newStubPasswordResetRepo()
	svc := newPRSvc(users, resetRepo)

	resp, _ := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "alice@example.com"})

	for _, tok := range resetRepo.tokens {
		tok.ExpiresAt = time.Now().Add(-1 * time.Hour)
	}

	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:    resp.Token,
		Password: "newPassword8",
	})
	if !errors.Is(err, service.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestResetPassword_AlreadyUsedToken_ReturnsErrNotFound(t *testing.T) {
	users := newStubUserRepo()
	seedUser(t, users, "alice@example.com")
	resetRepo := newStubPasswordResetRepo()
	svc := newPRSvc(users, resetRepo)

	resp, _ := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "alice@example.com"})
	_ = svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{Token: resp.Token, Password: "newPassword8"})

	err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
		Token:    resp.Token,
		Password: "anotherPassword8",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for already-used token, got %v", err)
	}
}
