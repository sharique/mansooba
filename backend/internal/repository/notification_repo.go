package repository

import (
	"context"

	"github.com/sharique/jira-go/internal/domain"
	"gorm.io/gorm"
)

type notificationRepo struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) domain.NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepo) FindByRecipientID(ctx context.Context, recipientID uint) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	if err := r.db.WithContext(ctx).
		Where("recipient_id = ? AND read = false", recipientID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

func (r *notificationRepo) MarkRead(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("id = ?", id).
		Update("read", true).Error
}
