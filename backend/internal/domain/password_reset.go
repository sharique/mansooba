package domain

import (
	"context"
	"time"
)

// PasswordResetToken is a single-use, time-limited credential that authorises
// a password change. The raw token is never stored — only its SHA-256 hex hash.
// Lifecycle: deleted immediately on successful use; purged after 1 hour if unused.
type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	TokenHash string    `gorm:"not null;uniqueIndex;size:64"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

// PasswordResetRepository defines persistence for PasswordResetToken.
// Implementations live in internal/repository and must not leak ORM types here.
type PasswordResetRepository interface {
	// Upsert deletes any existing token for UserID then inserts the new one,
	// both within a single transaction to prevent two active tokens per account.
	Upsert(ctx context.Context, token *PasswordResetToken) error

	// FindByHash returns the token whose TokenHash matches hash.
	// Returns ErrNotFound when absent (never issued, already used, or purged).
	FindByHash(ctx context.Context, hash string) (*PasswordResetToken, error)

	// Delete removes a token by ID. Called immediately after successful use.
	Delete(ctx context.Context, id uint) error

	// PurgeExpired deletes all tokens where created_at < cutoff.
	// The caller passes time.Now().Add(-1 * time.Hour) so the 1-hour window
	// is measured from created_at, not from expires_at.
	// Returns the number of rows deleted.
	PurgeExpired(ctx context.Context, cutoff time.Time) (int64, error)
}
