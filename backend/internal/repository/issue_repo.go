package repository

import (
	"context"
	"errors"

	"github.com/sharique/jira-go/internal/domain"
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
