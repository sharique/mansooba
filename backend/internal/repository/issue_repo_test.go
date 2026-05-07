package repository_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/repository"
)

func TestIssueRepository_Create_Succeeds(t *testing.T) {
	repo := repository.NewIssueRepository(newTestDB(t))
	issue := &domain.Issue{
		Key: "PROJ-1", ProjectID: 1, Title: "First issue",
		Type: "task", Status: "todo", Priority: "medium", ReporterID: 1,
	}
	if err := repo.Create(context.Background(), issue); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if issue.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestIssueRepository_FindByID(t *testing.T) {
	repo := repository.NewIssueRepository(newTestDB(t))
	created := &domain.Issue{
		Key: "PROJ-2", ProjectID: 1, Title: "Second",
		Type: "bug", Status: "todo", Priority: "high", ReporterID: 1,
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
	repo := repository.NewIssueRepository(newTestDB(t))
	for i, key := range []string{"P-1", "P-2"} {
		_ = repo.Create(context.Background(), &domain.Issue{
			Key: key, ProjectID: 10, Title: key,
			Type: "task", Status: "todo", Priority: "low", ReporterID: uint(i + 1),
		})
	}
	_ = repo.Create(context.Background(), &domain.Issue{
		Key: "Q-1", ProjectID: 99, Title: "Other",
		Type: "task", Status: "todo", Priority: "low", ReporterID: 1,
	})

	issues, err := repo.FindByProjectID(context.Background(), 10)
	if err != nil {
		t.Fatalf("FindByProjectID: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("want 2 issues, got %d", len(issues))
	}
}

func TestIssueRepository_Update(t *testing.T) {
	repo := repository.NewIssueRepository(newTestDB(t))
	issue := &domain.Issue{
		Key: "UPD-1", ProjectID: 1, Title: "Original",
		Type: "task", Status: "todo", Priority: "low", ReporterID: 1,
	}
	_ = repo.Create(context.Background(), issue)

	issue.Status = "in_progress"
	if err := repo.Update(context.Background(), issue); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.FindByID(context.Background(), issue.ID)
	if got.Status != "in_progress" {
		t.Fatalf("want status %q, got %q", "in_progress", got.Status)
	}
}

func TestIssueRepository_Delete(t *testing.T) {
	repo := repository.NewIssueRepository(newTestDB(t))
	issue := &domain.Issue{
		Key: "DEL-1", ProjectID: 1, Title: "Delete Me",
		Type: "task", Status: "todo", Priority: "low", ReporterID: 1,
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
