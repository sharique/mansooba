package service_test

// T001, T014, T022: PasswordChangeService tests.
//
// T001 (US1): current-password verification, reuse-prevention, persistence,
// and the success System Log entry.
// T014 (US2): the failure-reason Detail recorded on each rejection branch.
// T022 (US3): TokenValidAfter / ConsecutiveFailedPasswordChanges side
// effects and the two new EmailSender calls.
//
// This file will not compile until service.NewPasswordChangeService,
// service.PasswordChangeService, dto.ChangePasswordRequest,
// domain.ErrCurrentPasswordMismatch, domain.ErrNewPasswordSameAsCurrent,
// domain.User.TokenValidAfter, and domain.User.ConsecutiveFailedPasswordChanges
// all exist — that is the expected TDD red state. See
// specs/012-change-password/IMPLEMENTATION_GUIDE.md.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// recordingEmailSender captures every Send* call so tests can assert on
// which methods fired and with what recipient, without a real SMTP server.
type recordingEmailSender struct {
	passwordChangedCalls         []string
	suspiciousActivityAlertCalls []string
}

func (r *recordingEmailSender) SendPasswordReset(_ context.Context, _, _ string) error {
	return nil
}

func (r *recordingEmailSender) SendPasswordChanged(_ context.Context, to string) error {
	r.passwordChangedCalls = append(r.passwordChangedCalls, to)
	return nil
}

func (r *recordingEmailSender) SendSuspiciousActivityAlert(_ context.Context, to string) error {
	r.suspiciousActivityAlertCalls = append(r.suspiciousActivityAlertCalls, to)
	return nil
}

var _ domain.EmailSender = (*recordingEmailSender)(nil)

// newPCSvc builds a PasswordChangeService wired with the given collaborators,
// substituting recording/no-op doubles for anything left nil.
func newPCSvc(userRepo domain.UserRepository, systemLogSvc service.SystemLogService, emailSender domain.EmailSender) service.PasswordChangeService {
	if systemLogSvc == nil {
		systemLogSvc = stubSystemLogService{}
	}
	if emailSender == nil {
		emailSender = &recordingEmailSender{}
	}
	return service.NewPasswordChangeService(userRepo, systemLogSvc, emailSender)
}

// seedUserWithPassword creates a user with a real bcrypt hash of password —
// unlike password_reset_service_test.go's seedUser (which stores a
// "placeholder" string), PasswordChangeService's current-password check
// requires a genuine hash to compare against.
func seedUserWithPassword(t *testing.T, repo *stubUserRepo, email, password string) *domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash seed password: %v", err)
	}
	u := &domain.User{Email: email, Password: string(hash)}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

// ─── T001: current-password verification, reuse-prevention, persistence ──────

func TestPasswordChangeService_CorrectCurrentPassword_UpdatesHash(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	svc := newPCSvc(repo, nil, nil)

	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "OldPassword1",
		NewPassword:     "NewPassword2",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	updated, _ := repo.FindByID(context.Background(), user.ID)
	if bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("OldPassword1")) == nil {
		t.Error("old password must no longer verify against the stored hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("NewPassword2")); err != nil {
		t.Errorf("new password must verify against the stored hash: %v", err)
	}
}

func TestPasswordChangeService_WrongCurrentPassword_ReturnsErrCurrentPasswordMismatch(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	svc := newPCSvc(repo, nil, nil)

	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "WrongPassword9",
		NewPassword:     "NewPassword2",
	})
	if !errors.Is(err, domain.ErrCurrentPasswordMismatch) {
		t.Errorf("expected ErrCurrentPasswordMismatch, got %v", err)
	}

	unchanged, _ := repo.FindByID(context.Background(), user.ID)
	if bcrypt.CompareHashAndPassword([]byte(unchanged.Password), []byte("OldPassword1")) != nil {
		t.Error("original password must still verify — a rejected attempt must not mutate the stored hash")
	}
}

func TestPasswordChangeService_NewPasswordSameAsCurrent_ReturnsErrNewPasswordSameAsCurrent(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "SamePassword1")
	svc := newPCSvc(repo, nil, nil)

	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "SamePassword1",
		NewPassword:     "SamePassword1",
	})
	if !errors.Is(err, domain.ErrNewPasswordSameAsCurrent) {
		t.Errorf("expected ErrNewPasswordSameAsCurrent, got %v", err)
	}
}

// ─── T001 / T014: System Log entries on success and on each failure reason ───

func TestPasswordChangeService_Success_RecordsSystemLog(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	logSvc := &recordingSystemLogService{}
	svc := newPCSvc(repo, logSvc, nil)

	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "OldPassword1",
		NewPassword:     "NewPassword2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := logSvc.all()
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 Record call, got %d", len(entries))
	}
	e := entries[0]
	if e.Category != domain.SystemLogCategoryAuthentication || e.Action != "password_change_success" || e.Outcome != "success" {
		t.Errorf("unexpected entry: %+v", e)
	}
	if e.ActorID != user.ID || e.Actor != user.Email {
		t.Errorf("entry must attribute the caller as actor: %+v", e)
	}
}

func TestPasswordChangeService_WrongCurrentPassword_RecordsSystemLogWithReason(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	logSvc := &recordingSystemLogService{}
	svc := newPCSvc(repo, logSvc, nil)

	_ = svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "WrongPassword9",
		NewPassword:     "NewPassword2",
	})

	entries := logSvc.all()
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 Record call, got %d", len(entries))
	}
	e := entries[0]
	if e.Action != "password_change_failed" || e.Outcome != "failure" {
		t.Errorf("unexpected entry: %+v", e)
	}
	if e.Detail != "incorrect current password" {
		t.Errorf("expected Detail %q, got %q", "incorrect current password", e.Detail)
	}
}

func TestPasswordChangeService_SamePassword_RecordsSystemLogWithReason(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "SamePassword1")
	logSvc := &recordingSystemLogService{}
	svc := newPCSvc(repo, logSvc, nil)

	_ = svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "SamePassword1",
		NewPassword:     "SamePassword1",
	})

	entries := logSvc.all()
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 Record call, got %d", len(entries))
	}
	if entries[0].Detail != "new password reused current password" {
		t.Errorf("expected Detail %q, got %q", "new password reused current password", entries[0].Detail)
	}
}

// ─── T022: TokenValidAfter, ConsecutiveFailedPasswordChanges, and email ───────

func TestPasswordChangeService_Success_SetsTokenValidAfter(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	svc := newPCSvc(repo, nil, nil)

	before := time.Now()
	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "OldPassword1",
		NewPassword:     "NewPassword2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := repo.FindByID(context.Background(), user.ID)
	if updated.TokenValidAfter == nil {
		t.Fatal("expected TokenValidAfter to be set on a successful change")
	}
	// TokenValidAfter is truncated to whole seconds (matches jwt.NewNumericDate's
	// own truncation — see password_change_service.go) so compare against
	// before's own truncated floor, not before itself.
	if updated.TokenValidAfter.Before(before.Truncate(time.Second)) {
		t.Errorf("TokenValidAfter must be set to (approximately) the change time, got %v (before test start %v)", updated.TokenValidAfter, before)
	}
}

func TestPasswordChangeService_Success_ResetsConsecutiveFailedCount(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	user.ConsecutiveFailedPasswordChanges = 2
	_ = repo.Update(context.Background(), user)
	svc := newPCSvc(repo, nil, nil)

	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "OldPassword1",
		NewPassword:     "NewPassword2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := repo.FindByID(context.Background(), user.ID)
	if updated.ConsecutiveFailedPasswordChanges != 0 {
		t.Errorf("expected ConsecutiveFailedPasswordChanges reset to 0, got %d", updated.ConsecutiveFailedPasswordChanges)
	}
}

func TestPasswordChangeService_Failure_IncrementsConsecutiveFailedCount(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	svc := newPCSvc(repo, nil, nil)

	_ = svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "WrongPassword9",
		NewPassword:     "NewPassword2",
	})

	updated, _ := repo.FindByID(context.Background(), user.ID)
	if updated.ConsecutiveFailedPasswordChanges != 1 {
		t.Errorf("expected ConsecutiveFailedPasswordChanges = 1 after one failure, got %d", updated.ConsecutiveFailedPasswordChanges)
	}
}

func TestPasswordChangeService_ThirdConsecutiveFailure_SendsSuspiciousActivityAlert(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	user.ConsecutiveFailedPasswordChanges = 2
	_ = repo.Update(context.Background(), user)
	emailSvc := &recordingEmailSender{}
	svc := newPCSvc(repo, nil, emailSvc)

	_ = svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "WrongPassword9",
		NewPassword:     "NewPassword2",
	})

	if len(emailSvc.suspiciousActivityAlertCalls) != 1 {
		t.Fatalf("expected exactly 1 suspicious-activity alert on the 3rd consecutive failure, got %d", len(emailSvc.suspiciousActivityAlertCalls))
	}
	if emailSvc.suspiciousActivityAlertCalls[0] != "alice@example.com" {
		t.Errorf("expected alert sent to alice@example.com, got %q", emailSvc.suspiciousActivityAlertCalls[0])
	}
}

func TestPasswordChangeService_BelowThirdConsecutiveFailure_DoesNotSendAlert(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	emailSvc := &recordingEmailSender{}
	svc := newPCSvc(repo, nil, emailSvc)

	// Two failures only — must not reach the 3-in-a-row threshold.
	_ = svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{CurrentPassword: "Wrong1", NewPassword: "NewPassword2"})
	_ = svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{CurrentPassword: "Wrong2", NewPassword: "NewPassword2"})

	if len(emailSvc.suspiciousActivityAlertCalls) != 0 {
		t.Errorf("expected no suspicious-activity alert below 3 consecutive failures, got %d calls", len(emailSvc.suspiciousActivityAlertCalls))
	}
}

func TestPasswordChangeService_Success_SendsPasswordChangedEmail(t *testing.T) {
	repo := newStubUserRepo()
	user := seedUserWithPassword(t, repo, "alice@example.com", "OldPassword1")
	emailSvc := &recordingEmailSender{}
	svc := newPCSvc(repo, nil, emailSvc)

	err := svc.ChangePassword(context.Background(), user.ID, dto.ChangePasswordRequest{
		CurrentPassword: "OldPassword1",
		NewPassword:     "NewPassword2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(emailSvc.passwordChangedCalls) != 1 || emailSvc.passwordChangedCalls[0] != "alice@example.com" {
		t.Errorf("expected exactly 1 SendPasswordChanged call to alice@example.com, got %+v", emailSvc.passwordChangedCalls)
	}
}
