package dto

// ForgotPasswordRequest is the body for POST /auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ForgotPasswordResponse is returned for POST /auth/forgot-password.
// When the email matches a registered account, Token is the raw 64-char hex
// reset token and ExpiresAt is its RFC3339 expiry time.
// When no account matches, Token and ExpiresAt are empty strings and Message
// is identical to the success case (enumeration protection, FR-004).
type ForgotPasswordResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	Message   string `json:"message"`
}

// ResetPasswordRequest is the body for POST /auth/reset-password.
type ResetPasswordRequest struct {
	Token    string `json:"token"    validate:"required,len=64"`
	Password string `json:"password" validate:"required,min=8"`
}
