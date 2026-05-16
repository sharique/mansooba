package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/repository"
)

// seedSprint creates a sprint in the test DB and returns it.
func seedSprint(t *testing.T, repo domain.SprintRepository, projectID uint, name, status string) *domain.Sprint {
	t.Helper()
	s := &domain.Sprint{ProjectID: projectID, Name: name, Status: status}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("seedSprint: %v", err)
	}
	return s
}

func TestSprintRepository_Create_SetsID(t *testing.T) {
	repo := repository.NewSprintRepository(newTestDB(t))
	s := &domain.Sprint{ProjectID: 1, Name: "Sprint 1", Status: "planning"}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestSprintRepository_FindByID_NotFound(t *testing.T) {
	repo := repository.NewSprintRepository(newTestDB(t))
	_, err := repo.FindByID(context.Background(), 9999)
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSprintRepository_FindByProject_ReturnsOrdered(t *testing.T) {
	repo := repository.NewSprintRepository(newTestDB(t))
	seedSprint(t, repo, 1, "Sprint 1", "planning")
	seedSprint(t, repo, 1, "Sprint 2", "planning")

	sprints, err := repo.FindByProject(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByProject: %v", err)
	}
	if len(sprints) != 2 {
		t.Fatalf("want 2 sprints, got %d", len(sprints))
	}
	if sprints[0].Name != "Sprint 1" {
		t.Errorf("want Sprint 1 first, got %s", sprints[0].Name)
	}
}

func TestSprintRepository_FindActiveByProject_NilWhenNone(t *testing.T) {
	repo := repository.NewSprintRepository(newTestDB(t))
	active, err := repo.FindActiveByProject(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindActiveByProject: %v", err)
	}
	if active != nil {
		t.Fatal("expected nil when no active sprint")
	}
}

func TestSprintRepository_FindActiveByProject_ReturnsActive(t *testing.T) {
	repo := repository.NewSprintRepository(newTestDB(t))
	s := seedSprint(t, repo, 1, "Sprint 1", "active")

	active, err := repo.FindActiveByProject(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindActiveByProject: %v", err)
	}
	if active == nil || active.ID != s.ID {
		t.Fatalf("want sprint ID %d, got %v", s.ID, active)
	}
}

func TestSprintRepository_Delete_RemovesRow(t *testing.T) {
	repo := repository.NewSprintRepository(newTestDB(t))
	s := seedSprint(t, repo, 1, "Sprint 1", "planning")

	if err := repo.Delete(context.Background(), s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID(context.Background(), s.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}

func TestSprintRepository_CompleteWithMigration_MovesToBacklog(t *testing.T) {
	db := newTestDB(t)
	sprintRepo := repository.NewSprintRepository(db)
	issueRepo := repository.NewIssueRepository(db)
	ctx := context.Background()

	sprint := &domain.Sprint{ProjectID: 1, Name: "S1", Status: "active"}
	if err := sprintRepo.Create(ctx, sprint); err != nil {
		t.Fatalf("create sprint: %v", err)
	}

	issue := &domain.Issue{
		Key:        "TEST-1",
		ProjectID:  1,
		SprintID:   &sprint.ID,
		Title:      "Open issue",
		Type:       "task",
		Status:     "todo",
		Priority:   "medium",
		ReporterID: 1,
	}
	if err := issueRepo.Create(ctx, issue); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	now := time.Now()
	sprint.Status = "completed"
	sprint.EndDate = &now

	if err := sprintRepo.CompleteWithMigration(ctx, sprint, []uint{issue.ID}, nil); err != nil {
		t.Fatalf("CompleteWithMigration: %v", err)
	}

	updated, err := issueRepo.FindByID(ctx, issue.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if updated.SprintID != nil {
		t.Fatalf("expected sprint_id to be NULL after migration to backlog, got %v", updated.SprintID)
	}
}

func TestIssueRepository_FindBacklog_SortsByPriority(t *testing.T) {
	repo := repository.NewIssueRepository(newTestDB(t))
	ctx := context.Background()

	// Create issues in reverse priority order.
	for _, priority := range []string{"low", "critical", "medium", "high"} {
		_ = repo.Create(ctx, &domain.Issue{
			Key: "P-" + priority, ProjectID: 1, Title: priority,
			Type: "task", Status: "todo", Priority: priority, ReporterID: 1,
		})
	}

	issues, err := repo.FindBacklog(ctx, 1)
	if err != nil {
		t.Fatalf("FindBacklog: %v", err)
	}
	if len(issues) != 4 {
		t.Fatalf("want 4 issues, got %d", len(issues))
	}
	want := []string{"critical", "high", "medium", "low"}
	for i, w := range want {
		if issues[i].Priority != w {
			t.Errorf("position %d: want %s, got %s", i, w, issues[i].Priority)
		}
	}
}
