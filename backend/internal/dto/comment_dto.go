package dto

import "time"

type CreateCommentRequest struct {
	Body string `json:"body" validate:"required"`
}

type UpdateCommentRequest struct {
	Body string `json:"body" validate:"required"`
}

type CommentResponse struct {
	ID              uint      `json:"id"`
	IssueID         uint      `json:"issue_id"`
	AuthorID        uint      `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	AuthorAvatarURL string    `json:"author_avatar_url"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ActivityEventResponse struct {
	ID             uint      `json:"id"`
	IssueID        uint      `json:"issue_id"`
	ActorID        uint      `json:"actor_id"`
	ActorName      string    `json:"actor_name"`
	ActorAvatarURL string    `json:"actor_avatar_url"`
	IssueKey       string    `json:"issue_key"`
	IssueTitle     string    `json:"issue_title"`
	Kind           string    `json:"kind"`
	OldValue       string    `json:"old_value,omitempty"`
	NewValue       string    `json:"new_value,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
