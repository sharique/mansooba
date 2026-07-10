package dto

import "time"

type RegisterRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,password_complexity"`
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
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserProfileResponse is returned by GET /auth/me and PUT /auth/me.
type UserProfileResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
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
}
