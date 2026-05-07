package domain

import (
	"context"
	"time"
)

// User represents an authenticated account in the system.
// Password stores a bcrypt hash — never the plaintext value.
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
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
}
