package dto

// ChangePasswordRequest is the body for PUT /auth/me/password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,password_complexity"`
}
