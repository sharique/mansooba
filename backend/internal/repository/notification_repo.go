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
		ProjectKey string `gorm:"column:project_key"`
		IssueKey   string `gorm:"column:issue_key"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("notifications").
		Select("notifications.*, projects.key AS project_key, issues.key AS issue_key").
		Joins("JOIN issues ON issues.id = notifications.issue_id").
		Joins("JOIN projects ON projects.id = issues.project_id").
		Where("notifications.recipient_id = ? AND notifications.read = false", recipientID).
		Order("notifications.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.NotificationDetail, 0, len(rows))
	for _, r := range rows {
		detail := &domain.NotificationDetail{Notification: r.Notification, ProjectKey: r.ProjectKey, IssueKey: r.IssueKey}
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
