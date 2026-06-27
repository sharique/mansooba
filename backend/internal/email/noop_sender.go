// Package email provides EmailSender implementations.
// NoopSender is the implementation for feature 006 (token returned in API response).
// Feature 007-email-delivery replaces it with SesSender at the composition root.
package email

import (
	"context"

	"github.com/sharique/mansooba/internal/domain"
)

// NoopSender implements domain.EmailSender as a no-op.
// SendPasswordReset returns nil without sending any email;
// the raw token is returned directly in the API response instead.
type NoopSender struct{}

var _ domain.EmailSender = NoopSender{}

func (NoopSender) SendPasswordReset(_ context.Context, _, _ string) error {
	return nil
}
