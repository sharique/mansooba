package dto

import "time"

type NotificationResponse struct {
	ID          uint      `json:"id"`
	RecipientID uint      `json:"recipient_id"`
	ActorID     uint      `json:"actor_id"`
	ActorName   string    `json:"actor_name"`
	Kind        string    `json:"kind"`
	IssueID     uint      `json:"issue_id"`
	IssueKey    string    `json:"issue_key"`
	ProjectKey  string    `json:"project_key"`
	ProjectName string    `json:"project_name"`
	CommentID   uint      `json:"comment_id"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"created_at"`
}
