package domain

import "context"

// EmailSender is the outbound port for email delivery.
// The no-op implementation (internal/email.NoopSender) is used in feature 006.
// Feature 007-email-delivery replaces it with an AWS SES implementation
// at the composition root without touching this interface or the service layer.
type EmailSender interface {
	// SendPasswordReset delivers a password reset token to the given address.
	// to is the recipient email; token is the raw 64-char hex value.
	// The no-op implementation returns nil immediately.
	SendPasswordReset(ctx context.Context, to, token string) error
}
