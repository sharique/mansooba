package domain

import (
	"context"
	"time"
)

// User represents an authenticated account in the system.
// Password stores a bcrypt hash — never the plaintext value.
// AvatarURL and Timezone are optional profile fields.
// IsAdmin grants platform-wide admin privileges (global settings, project creation).
// IsActive is false when an admin has disabled the account; all token issuance is blocked.
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	AvatarURL string    // optional; full URL or empty
	Timezone  string    // IANA timezone name (e.g. "America/New_York"); empty = UTC
	IsAdmin   bool      `gorm:"not null;default:false"`
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository defines the persistence contract for User.
// Implementations live in internal/repository and must never leak ORM types here.
type UserRepository interface {
	// Create persists a new user and sets its ID on success.
	Create(ctx context.Context, user *User) error
	// FindByID returns a user by primary key, or ErrNotFound if absent.
	FindByID(ctx context.Context, id uint) (*User, error)
	// FindByEmail returns a user by email address, or ErrNotFound if absent.
	FindByEmail(ctx context.Context, email string) (*User, error)
	// FindByEmailPrefix returns the user whose email starts with the given local part
	// (everything before '@'). Returns ErrNotFound when no match exists.
	FindByEmailPrefix(ctx context.Context, prefix string) (*User, error)
	// Update persists all writable fields of an existing user (name, avatar_url, timezone).
	Update(ctx context.Context, user *User) error
	// HasAdmin returns true if at least one user with IsAdmin=true exists.
	// Used exclusively by the setup service to detect fresh-install state.
	HasAdmin(ctx context.Context) (bool, error)
	// FindFirstAdmin returns the first user with IsAdmin=true, ordered by ID.
	// Returns ErrNotFound when no admin exists.
	FindFirstAdmin(ctx context.Context) (*User, error)
	// ListAll returns a paginated slice of all users sorted by created_at DESC,
	// plus the total count. Page is 1-based; out-of-range pages return empty slice.
	ListAll(ctx context.Context, page, size int) ([]*User, int64, error)
	// CountActiveAdmins returns the number of users where is_admin=true AND is_active=true.
	CountActiveAdmins(ctx context.Context) (int64, error)
	// UpdateAdminFields writes only the is_admin and is_active columns for the given user.
	UpdateAdminFields(ctx context.Context, user *User) error
}
