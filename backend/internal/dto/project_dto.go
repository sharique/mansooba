package dto

type CreateProjectRequest struct {
	Name        string `json:"name"        validate:"required"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role"  validate:"required,oneof=admin member viewer"`
}

type ProjectResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     uint   `json:"owner_id"`
}

type MemberResponse struct {
	UserID   uint   `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	// IsActive is false for a deactivated account; the mention suggestions leave those out.
	IsActive bool `json:"is_active"`
}
