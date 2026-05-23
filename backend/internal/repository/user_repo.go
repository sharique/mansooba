package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

// NewUserRepository returns a GORM-backed implementation of domain.UserRepository.
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepo{db: db}
}

// Create inserts a new user record and populates the ID field on success.
func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

// FindByID retrieves a user by primary key.
// Returns domain.ErrNotFound when no row matches.
func (r *userRepo) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email address.
// Returns domain.ErrNotFound when no row matches.
func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmailPrefix retrieves a user whose email starts with the given local part (before '@').
// Returns domain.ErrNotFound when no row matches.
func (r *userRepo) FindByEmailPrefix(ctx context.Context, prefix string) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).
		Where("email LIKE ?", prefix+"@%").
		First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// Update persists name, avatar_url, and timezone for an existing user.
func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}
