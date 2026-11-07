package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueRepository_Create_Succeeds(t *testing.T) {
	repo := repository.NewIssueRepository(newDBWithProjects(t))
	issue := &domain.Issue{
		Key: "PROJ-1", ProjectID: 1, Title: "First issue",
		Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium, ReporterID: 1,
	}
	if err := repo.Create(context.Background(), issue); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if issue.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestIssueRepository_FindByID(t *testing.T) {
	repo := repository.NewIssueRepository(newDBWithProjects(t))
	created := &domain.Issue{
		Key: "PROJ-2", ProjectID: 1, Title: "Second",
		Type: domain.IssueTypeBug, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityHigh, ReporterID: 1,
	}
	_ = repo.Create(context.Background(), created)

	tests := []struct {
		name    string
		id      uint
		wantErr error
	}{
		{"found", created.ID, nil},
		{"not found", 9999, domain.ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.FindByID(context.Background(), tc.id)
			if err != tc.wantErr {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestIssueRepository_FindByProjectID(t *testing.T) {
	repo := repository.NewIssueRepository(newDBWithProjects(t))
	for i, key := range []string{"P-1", "P-2"} {
		_ = repo.Create(context.Background(), &domain.Issue{
			Key: key, ProjectID: 4, Title: key,
			Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: uint(i + 1),
		})
	}
	_ = repo.Create(context.Background(), &domain.Issue{
		Key: "Q-1", ProjectID: 5, Title: "Other",
		Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: 1,
	})

	issues, err := repo.FindByProjectID(context.Background(), 4)
	if err != nil {
		t.Fatalf("FindByProjectID: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("want 2 issues, got %d", len(issues))
	}
}

func TestIssueRepository_Update(t *testing.T) {
	repo := repository.NewIssueRepository(newDBWithProjects(t))
	issue := &domain.Issue{
		Key: "UPD-1", ProjectID: 1, Title: "Original",
		Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: 1,
	}
	_ = repo.Create(context.Background(), issue)

	issue.Status = domain.IssueStatusInProgress
	if err := repo.Update(context.Background(), issue); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.FindByID(context.Background(), issue.ID)
	if got.Status != domain.IssueStatusInProgress {
		t.Fatalf("want status %q, got %q", domain.IssueStatusInProgress, got.Status)
	}
}

func TestIssueRepository_Delete(t *testing.T) {
	repo := repository.NewIssueRepository(newDBWithProjects(t))
	issue := &domain.Issue{
		Key: "DEL-1", ProjectID: 1, Title: "Delete Me",
		Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: 1,
	}
	_ = repo.Create(context.Background(), issue)

	if err := repo.Delete(context.Background(), issue.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID(context.Background(), issue.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}

// Newest first, and a stable order when several tasks were created in the same
// instant: the database is free to return ties in any order (MariaDB stores
// milliseconds, so tasks created together tie), and the list must not shuffle.
func TestIssueRepository_FindByAssignee_TiesComeNewestFirst(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	assignee := uint(3)
	at := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	for _, key := range []string{"T-1", "T-2", "T-3", "T-4"} {
		require.NoError(t, db.Create(&domain.Issue{
			Key: key, ProjectID: 1, Title: key, Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo,
			Priority: domain.IssuePriorityLow, ReporterID: 1, AssigneeID: &assignee, CreatedAt: at,
		}).Error)
	}

	got, err := repo.FindByAssignee(ctx, assignee)
	require.NoError(t, err)
	keys := make([]string, 0, len(got))
	for _, i := range got {
		keys = append(keys, i.Key)
	}
	assert.Equal(t, []string{"T-4", "T-3", "T-2", "T-1"}, keys)
}
