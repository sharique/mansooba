package repository

import (
	"context"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type notificationRepo struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepo) FindUnreadByRecipientID(ctx context.Context, recipientID uint) ([]*domain.NotificationDetail, error) {
	type row struct {
		domain.Notification
		ProjectKey  string `gorm:"column:project_key"`
		ProjectName string `gorm:"column:project_name"`
		IssueKey    string `gorm:"column:issue_key"`
		ActorName   string `gorm:"column:actor_name"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("notifications").
		Select("notifications.*, "+
			"COALESCE(projects.key, '') AS project_key, "+
			"COALESCE(projects.name, '') AS project_name, "+
			"COALESCE(issues.key, '') AS issue_key, "+
			"COALESCE(actor.name, '') AS actor_name").
		Joins("LEFT JOIN issues ON issues.id = notifications.issue_id").
		// The project comes from the notification itself (project kinds) or, for
		// issue kinds, from the issue. Portable SQL only (SQLite and PostgreSQL).
		Joins("LEFT JOIN projects ON projects.id = CASE WHEN notifications.project_id <> 0 THEN notifications.project_id ELSE issues.project_id END").
		Joins("LEFT JOIN users AS actor ON actor.id = notifications.actor_id").
		Where("notifications.recipient_id = ? AND notifications.read = false", recipientID).
		Order("notifications.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.NotificationDetail, 0, len(rows))
	for _, r := range rows {
		detail := &domain.NotificationDetail{
			Notification: r.Notification,
			ProjectKey:   r.ProjectKey, ProjectName: r.ProjectName,
			IssueKey: r.IssueKey, ActorName: r.ActorName,
		}
		result = append(result, detail)
	}
	return result, nil
}

func (r *notificationRepo) MarkRead(ctx context.Context, id, recipientID uint) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("id = ? AND recipient_id = ?", id, recipientID).
		Update("read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
