package dto

import "time"

type NotificationResponse struct {
	ID          uint      `json:"id"`
	RecipientID uint      `json:"recipient_id"`
	ActorID     uint      `json:"actor_id"`
	IssueID     uint      `json:"issue_id"`
	IssueKey    string    `json:"issue_key"`
	ProjectKey  string    `json:"project_key"`
	CommentID   uint      `json:"comment_id"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"created_at"`
}
