package repository

import (
	"context"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type activityRepo struct{ db *gorm.DB }

func NewActivityRepository(db *gorm.DB) domain.ActivityRepository {
	return &activityRepo{db: db}
}

func (r *activityRepo) Create(ctx context.Context, e *domain.ActivityEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *activityRepo) FindByIssueID(ctx context.Context, issueID uint) ([]*domain.ActivityEvent, error) {
	var events []*domain.ActivityEvent
	if err := r.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *activityRepo) FindByActorID(ctx context.Context, actorID uint, limit, offset int) ([]*domain.ActivityEvent, error) {
	var events []*domain.ActivityEvent
	if err := r.db.WithContext(ctx).
		Where("actor_id = ?", actorID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// FindRecentByProject returns the progress, comment and deletion events of a project
// since the given time, newest first. The join to issues gives the project and sprint
// filters for ordinary events and drops the events of deleted issues; a status change
// qualifies only when it lands in progress, review or done and actually changes the
// status. A deleted task has no issue row, so its issue_deleted event is matched on the
// project and sprint it carries instead.
func (r *activityRepo) FindRecentByProject(ctx context.Context, projectID uint, since time.Time, sprintID *uint, limit int) ([]*domain.ActivityEvent, error) {
	q := r.db.WithContext(ctx).
		Table("activity_events").
		Select("activity_events.*").
		Joins("LEFT JOIN issues ON issues.id = activity_events.issue_id").
		Where("activity_events.created_at >= ?", since).
		Where(
			"((issues.project_id = ? AND (activity_events.kind = ? OR (activity_events.kind = ? AND activity_events.new_value IN ? AND activity_events.old_value <> activity_events.new_value)))"+
				" OR (activity_events.kind = ? AND activity_events.project_id = ?))",
			projectID, domain.ActivityCommentAdded, domain.ActivityStatusChanged,
			[]string{domain.IssueStatusInProgress, domain.IssueStatusInReview, domain.IssueStatusDone},
			domain.ActivityIssueDeleted, projectID,
		)
	if sprintID != nil {
		q = q.Where(
			"((activity_events.kind <> ? AND issues.sprint_id = ?) OR (activity_events.kind = ? AND activity_events.sprint_id = ?))",
			domain.ActivityIssueDeleted, *sprintID, domain.ActivityIssueDeleted, *sprintID,
		)
	}
	var events []*domain.ActivityEvent
	if err := q.Order("activity_events.created_at DESC, activity_events.id DESC").Limit(limit).Scan(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
