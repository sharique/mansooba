package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type issueRepo struct {
	db *gorm.DB
}

// NewIssueRepository returns a GORM-backed implementation of domain.IssueRepository.
func NewIssueRepository(db *gorm.DB) domain.IssueRepository {
	return &issueRepo{db: db}
}

// Create inserts a new issue record and populates the ID field on success.
func (r *issueRepo) Create(ctx context.Context, issue *domain.Issue) error {
	return r.db.WithContext(ctx).Create(issue).Error
}

// FindByID retrieves an issue by primary key.
// Returns domain.ErrNotFound when no row matches.
func (r *issueRepo) FindByID(ctx context.Context, id uint) (*domain.Issue, error) {
	var issue domain.Issue
	if err := r.db.WithContext(ctx).First(&issue, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &issue, nil
}

// FindByProjectID returns all issues belonging to a given project.
func (r *issueRepo) FindByProjectID(ctx context.Context, projectID uint) ([]*domain.Issue, error) {
	var issues []*domain.Issue
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

// Update saves all fields of an existing issue record.
func (r *issueRepo) Update(ctx context.Context, issue *domain.Issue) error {
	return r.db.WithContext(ctx).Save(issue).Error
}

// Delete removes an issue by primary key.
func (r *issueRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Issue{}, id).Error
}

// DeleteByProjectID removes all issues belonging to a project.
func (r *issueRepo) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return r.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&domain.Issue{}).Error
}

// FindBacklog returns issues with sprint_id IS NULL, sorted by priority then created_at.
// Priority order in SQL: critical(1) > high(2) > medium(3) > low(4).
func (r *issueRepo) FindBacklog(ctx context.Context, projectID uint) ([]*domain.Issue, error) {
	var issues []*domain.Issue
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND sprint_id IS NULL", projectID).
		Order(`CASE priority
			WHEN 'critical' THEN 1
			WHEN 'high'     THEN 2
			WHEN 'medium'   THEN 3
			WHEN 'low'      THEN 4
			ELSE 5
		END ASC, created_at ASC`).
		Find(&issues).Error
	return issues, err
}

// FindBySprint returns all issues assigned to a given sprint.
func (r *issueRepo) FindBySprint(ctx context.Context, sprintID uint) ([]*domain.Issue, error) {
	var issues []*domain.Issue
	if err := r.db.WithContext(ctx).Where("sprint_id = ?", sprintID).Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

// FindIssueIDsByLabelID returns all issue IDs that have the given label attached.
func (r *issueRepo) FindIssueIDsByLabelID(ctx context.Context, labelID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.WithContext(ctx).
		Table("issue_labels").
		Where("label_id = ?", labelID).
		Pluck("issue_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// FindByAssignee returns all issues assigned to the given user, across all projects,
// ordered by created_at descending (newest first).
func (r *issueRepo) FindByAssignee(ctx context.Context, userID uint) ([]*domain.Issue, error) {
	var issues []*domain.Issue
	if err := r.db.WithContext(ctx).
		Where("assignee_id = ?", userID).
		Order("created_at DESC").
		Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

// CountBySprint returns the issue count and total story points for a sprint in one aggregate query.
func (r *issueRepo) CountBySprint(ctx context.Context, sprintID uint) (int, int, error) {
	type result struct {
		Count       int
		TotalPoints int
	}
	var res result
	err := r.db.WithContext(ctx).
		Model(&domain.Issue{}).
		Select("COUNT(*) as count, COALESCE(SUM(story_points), 0) as total_points").
		Where("sprint_id = ?", sprintID).
		Scan(&res).Error
	return res.Count, res.TotalPoints, err
}

// CountByStatusForSprint returns task and story-point totals per status for a
// sprint in one grouped query. Statuses without issues are not returned.
func (r *issueRepo) CountByStatusForSprint(ctx context.Context, sprintID uint) ([]domain.StatusTotals, error) {
	var rows []domain.StatusTotals
	err := r.db.WithContext(ctx).
		Model(&domain.Issue{}).
		Select("status AS status, COUNT(*) AS tasks, COALESCE(SUM(story_points), 0) AS points").
		Where("sprint_id = ?", sprintID).
		Group("status").
		Scan(&rows).Error
	return rows, err
}

// FindOpenBySprint returns the sprint's issues that are not done, most urgent
// first. The two CASE expressions give priority and status an explicit rank
// because neither has a natural sort order; anything unranked sorts last.
func (r *issueRepo) FindOpenBySprint(ctx context.Context, sprintID uint, limit int) ([]*domain.Issue, error) {
	var issues []*domain.Issue
	err := r.db.WithContext(ctx).
		Where("sprint_id = ? AND status <> ?", sprintID, domain.IssueStatusDone).
		Order("CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END").
		Order("CASE status WHEN 'in_review' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'todo' THEN 2 ELSE 3 END").
		Order("updated_at DESC, id ASC").
		Limit(limit).
		Find(&issues).Error
	return issues, err
}
