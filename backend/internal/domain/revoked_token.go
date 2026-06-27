package domain

import (
	"context"
	"time"
)

// RevokedToken records an invalidated refresh token so the refresh endpoint
// can reject it even before its natural expiry. The JTI (JWT ID) is the
// revocation key; it is a raw UUID v4 stored as a string with a unique index.
type RevokedToken struct {
	ID        uint      `gorm:"primaryKey"`
	JTI       string    `gorm:"uniqueIndex;not null"` // JWT ID claim from the refresh token
	UserID    uint      `gorm:"index;not null"`       // owner; indexed for future "revoke all" use
	ExpiresAt time.Time `gorm:"index;not null"`       // mirrors JWT exp; used by cleanup goroutine
	RevokedAt time.Time `gorm:"not null"`             // time.Now() at logout
}

// RevokedTokenRepository is the persistence contract for revocation records.
// Implementations live in internal/repository.
type RevokedTokenRepository interface {
	// Create inserts a revocation record. Duplicate JTI is silently ignored (idempotent).
	Create(ctx context.Context, token *RevokedToken) error
	// Exists returns true if the JTI appears in the revocation table.
	// A DB error is returned as the second value; the boolean is false on error.
	Exists(ctx context.Context, jti string) (bool, error)
	// DeleteExpired removes all records whose ExpiresAt is in the past.
	// Returns the number of deleted rows.
	DeleteExpired(ctx context.Context) (int64, error)
}
