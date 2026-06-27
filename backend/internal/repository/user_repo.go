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

// HasAdmin returns true if at least one user with is_admin=true exists.
func (r *userRepo) HasAdmin(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.User{}).Where("is_admin = ?", true).Count(&count).Error
	return count > 0, err
}

// FindFirstAdmin returns the admin user with the lowest ID, or ErrNotFound when none exists.
func (r *userRepo) FindFirstAdmin(ctx context.Context) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("is_admin = ?", true).Order("id ASC").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// ListAll returns a page of all users sorted by created_at DESC plus the total count.
// Page is 1-based; out-of-range pages return an empty slice and the total count.
func (r *userRepo) ListAll(ctx context.Context, page, size int) ([]*domain.User, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * size
	var users []*domain.User
	if err := r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(size).Find(&users).Error; err != nil {
		return nil, total, err
	}
	return users, total, nil
}

// CountActiveAdmins returns the number of users where is_admin=true AND is_active=true.
func (r *userRepo) CountActiveAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("is_admin = ? AND is_active = ?", true, true).Count(&count).Error
	return count, err
}

// UpdateAdminFields writes only the is_admin and is_active columns for the given user.
func (r *userRepo) UpdateAdminFields(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Model(user).
		Select("is_admin", "is_active").
		Updates(map[string]any{
			"is_admin":  user.IsAdmin,
			"is_active": user.IsActive,
		}).Error
}
