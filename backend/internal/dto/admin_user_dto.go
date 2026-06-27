package dto

import "time"

// AdminUserDTO is the public representation of a user returned by admin endpoints.
type AdminUserDTO struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminUserListResponse is returned by GET /api/v1/admin/users.
type AdminUserListResponse struct {
	Users []AdminUserDTO `json:"users"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// AdminUserPatchRequest is the body for PATCH /api/v1/admin/users/:id.
// Both fields are optional pointers so the handler can distinguish "not sent" from false.
type AdminUserPatchRequest struct {
	IsAdmin  *bool `json:"is_admin"`
	IsActive *bool `json:"is_active"`
}
