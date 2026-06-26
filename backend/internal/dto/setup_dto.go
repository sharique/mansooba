package dto

// SetupStatusResponse is returned by GET /api/v1/setup/status.
type SetupStatusResponse struct {
	SetupRequired bool `json:"setup_required"`
}

// SetupAdminRequest creates the initial admin account during wizard step 1.
type SetupAdminRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,password_complexity"`
}

// SetupAdminResponse is returned by POST /api/v1/setup/admin.
// Identical shape to AuthResponse so the frontend auth store needs no special-casing.
type SetupAdminResponse = AuthResponse

// SetupUserRequest creates an optional team member during wizard step 2.
type SetupUserRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,password_complexity"`
}

// SetupUserResponse is returned by POST /api/v1/setup/user.
type SetupUserResponse struct {
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// SetupProjectRequest creates an optional project during wizard step 3.
type SetupProjectRequest struct {
	Name        string `json:"name"        validate:"required"`
	Description string `json:"description"`
	// AddUserID is non-zero when the team member created in step 2
	// should be added to this project as a "member".
	AddUserID uint `json:"add_user_id"`
}

// SetupProjectResponse is returned by POST /api/v1/setup/project.
type SetupProjectResponse struct {
	ProjectID  uint   `json:"project_id"`
	ProjectKey string `json:"project_key"`
	Name       string `json:"name"`
}

// SetupSeedResponse is returned by POST /api/v1/setup/seed.
type SetupSeedResponse struct {
	Skipped     bool   `json:"skipped"`
	ProjectKey  string `json:"project_key"`
	ProjectName string `json:"project_name"`
}
