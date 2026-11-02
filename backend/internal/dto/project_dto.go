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
	// OwnerID records who created the project; it carries no rights.
	OwnerID uint `json:"owner_id"`
	// MyRole and Permissions describe what the caller may do in this project.
	// They are set when a single project is fetched, created or updated, not in
	// the project list. The interface reads Permissions instead of keeping its
	// own copy of the role table.
	MyRole      string   `json:"my_role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
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

// ChangeRoleRequest is the body of PATCH /projects/:key/members/:userId.
type ChangeRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member viewer"`
}

// DeleteProjectRequest is the body of DELETE /projects/:key. ConfirmName must be
// the project's exact name.
type DeleteProjectRequest struct {
	ConfirmName string `json:"confirm_name"`
}

// AbandonedProjectResponse names a project that has no active admin. It carries
// nothing about the project's contents.
type AbandonedProjectResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
}
