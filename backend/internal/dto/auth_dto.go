package dto

import "time"

type RegisterRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,password_complexity"`
	// Username is optional; when empty one is generated from the email.
	// Validated by the service (rules, reserved names, uniqueness).
	Username string `json:"username"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         UserDTO `json:"user"`
}

type UserDTO struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// UserProfileResponse is returned by GET /auth/me and PUT /auth/me.
type UserProfileResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	Timezone  string    `json:"timezone"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateProfileRequest is the body for PUT /auth/me.
// All fields are optional — only non-empty values are applied.
type UpdateProfileRequest struct {
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
	Timezone  string `json:"timezone"`
	// Username cannot be changed; the field exists only so a request that tries
	// is refused instead of silently ignored. The current value is accepted.
	Username *string `json:"username"`
}
