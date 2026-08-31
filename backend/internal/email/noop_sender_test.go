package email

// T023 (012-change-password): NoopSender's two new EmailSender methods must
// return nil without sending anything, matching SendPasswordReset's
// existing no-op behavior. Will not compile until NoopSender gains
// SendPasswordChanged and SendSuspiciousActivityAlert — the expected TDD
// red state. See specs/012-change-password/IMPLEMENTATION_GUIDE.md.

import (
	"context"
	"testing"
)

func TestNoopSender_SendPasswordChanged_ReturnsNil(t *testing.T) {
	if err := (NoopSender{}).SendPasswordChanged(context.Background(), "user@test.com"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoopSender_SendSuspiciousActivityAlert_ReturnsNil(t *testing.T) {
	if err := (NoopSender{}).SendSuspiciousActivityAlert(context.Background(), "user@test.com"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
