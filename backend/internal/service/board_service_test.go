package service_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

func newBoardService() (service.BoardService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo) {
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	issueRepo := newStubIssueRepo()
	svc := service.NewBoardService(issueRepo, projectRepo, memberRepo)
	return svc, projectRepo, memberRepo, issueRepo
}

func TestBoardService_GetBoard_AllColumnsPresent(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newBoardService()
	ctx := context.Background()

	project := &domain.Project{Key: "PROJ", Name: "Test", OwnerID: 1}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})

	// Only one issue, in "todo" — other columns must still appear
	_ = issueRepo.Create(ctx, &domain.Issue{
		Key: "PROJ-1", ProjectID: project.ID, Title: "Fix bug",
		Type: domain.IssueTypeBug, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityHigh, ReporterID: 1,
	})

	resp, err := svc.GetBoard(ctx, "PROJ", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(resp.Columns))
	}

	wantOrder := []string{domain.IssueStatusTodo, domain.IssueStatusInProgress, domain.IssueStatusInReview, domain.IssueStatusDone}
	for i, col := range resp.Columns {
		if col.Status != wantOrder[i] {
			t.Errorf("column %d: expected status %q, got %q", i, wantOrder[i], col.Status)
		}
	}
}

func TestBoardService_GetBoard_BacklogExcluded(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newBoardService()
	ctx := context.Background()

	project := &domain.Project{Key: "PROJ", Name: "Test", OwnerID: 1}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})

	_ = issueRepo.Create(ctx, &domain.Issue{
		Key: "PROJ-1", ProjectID: project.ID, Title: "Backlog item",
		Type: domain.IssueTypeTask, Status: domain.IssueStatusBacklog, Priority: domain.IssuePriorityLow, ReporterID: 1,
	})

	resp, err := svc.GetBoard(ctx, "PROJ", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, col := range resp.Columns {
		if len(col.Issues) > 0 {
			t.Errorf("column %q: expected no issues (backlog should be excluded), got %d", col.Status, len(col.Issues))
		}
		// Each empty column must be an empty array, not null
		if col.Issues == nil {
			t.Errorf("column %q: Issues must be an empty slice, not nil (marshals to null)", col.Status)
		}
	}
}

func TestBoardService_GetBoard_CorrectGrouping(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newBoardService()
	ctx := context.Background()

	project := &domain.Project{Key: "PROJ", Name: "Test", OwnerID: 1}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})

	_ = issueRepo.Create(ctx, &domain.Issue{Key: "PROJ-1", ProjectID: project.ID, Title: "T1", Type: domain.IssueTypeBug, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityHigh, ReporterID: 1})
	_ = issueRepo.Create(ctx, &domain.Issue{Key: "PROJ-2", ProjectID: project.ID, Title: "T2", Type: domain.IssueTypeTask, Status: domain.IssueStatusInProgress, Priority: domain.IssuePriorityLow, ReporterID: 1})
	_ = issueRepo.Create(ctx, &domain.Issue{Key: "PROJ-3", ProjectID: project.ID, Title: "T3", Type: domain.IssueTypeStory, Status: domain.IssueStatusInProgress, Priority: domain.IssuePriorityMedium, ReporterID: 1})
	_ = issueRepo.Create(ctx, &domain.Issue{Key: "PROJ-4", ProjectID: project.ID, Title: "T4", Type: domain.IssueTypeTask, Status: domain.IssueStatusDone, Priority: domain.IssuePriorityLow, ReporterID: 1})
	_ = issueRepo.Create(ctx, &domain.Issue{Key: "PROJ-5", ProjectID: project.ID, Title: "T5", Type: domain.IssueTypeBug, Status: domain.IssueStatusBacklog, Priority: domain.IssuePriorityLow, ReporterID: 1})

	resp, err := svc.GetBoard(ctx, "PROJ", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counts := map[string]int{}
	for _, col := range resp.Columns {
		counts[col.Status] = len(col.Issues)
	}

	if counts[domain.IssueStatusTodo] != 1 {
		t.Errorf("todo: expected 1, got %d", counts[domain.IssueStatusTodo])
	}
	if counts[domain.IssueStatusInProgress] != 2 {
		t.Errorf("in_progress: expected 2, got %d", counts[domain.IssueStatusInProgress])
	}
	if counts[domain.IssueStatusInReview] != 0 {
		t.Errorf("in_review: expected 0, got %d", counts[domain.IssueStatusInReview])
	}
	if counts[domain.IssueStatusDone] != 1 {
		t.Errorf("done: expected 1, got %d", counts[domain.IssueStatusDone])
	}

	// Backlog issue must not appear in any column
	total := counts[domain.IssueStatusTodo] + counts[domain.IssueStatusInProgress] + counts[domain.IssueStatusInReview] + counts[domain.IssueStatusDone]
	if total != 4 {
		t.Errorf("total board issues: expected 4 (backlog excluded), got %d", total)
	}
}

func TestBoardService_GetBoard_ForbiddenForNonMember(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newBoardService()
	ctx := context.Background()

	project := &domain.Project{Key: "PROJ", Name: "Test", OwnerID: 1}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})

	_, err := svc.GetBoard(ctx, "PROJ", 99)
	if err == nil {
		t.Fatal("expected error for non-member")
	}
}

// ── helper for BoardColumn lookup ─────────────────────────────────────────────

func boardColumn(resp *dto.BoardResponse, status string) *dto.BoardColumn {
	for i := range resp.Columns {
		if resp.Columns[i].Status == status {
			return &resp.Columns[i]
		}
	}
	return nil
}
