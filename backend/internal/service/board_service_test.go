package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

func newBoardService() (service.BoardService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo) {
	svc, projectRepo, memberRepo, issueRepo, _ := newBoardServiceWithSprints()
	return svc, projectRepo, memberRepo, issueRepo
}

func newBoardServiceWithSprints() (service.BoardService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo, *stubSprintRepo) {
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	issueRepo := newStubIssueRepo()
	sprintRepo := newStubSprintRepo()
	svc := service.NewBoardService(issueRepo, projectRepo, memberRepo, sprintRepo)
	return svc, projectRepo, memberRepo, issueRepo, sprintRepo
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

// ── column ordering (priority, then issue number) ────────────────────────────

type boardOrderCase struct {
	Name  string `json:"name"`
	Cards []struct {
		ID       uint   `json:"id"`
		Priority string `json:"priority"`
	} `json:"cards"`
	Expected []uint `json:"expected"`
}

// loadBoardOrderCases reads the table that the browser's boardOrder test also uses,
// so the server and the client cannot drift apart.
func loadBoardOrderCases(t *testing.T) []boardOrderCase {
	t.Helper()
	raw, err := os.ReadFile("../../../testdata/board-order-cases.json")
	if err != nil {
		t.Fatalf("read shared order cases: %v", err)
	}
	var file struct {
		Cases []boardOrderCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse shared order cases: %v", err)
	}
	if len(file.Cases) == 0 {
		t.Fatal("no shared order cases found")
	}
	return file.Cases
}

func TestBoardService_GetBoard_OrdersColumnsByPriorityThenID(t *testing.T) {
	for _, tc := range loadBoardOrderCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			svc, projectRepo, memberRepo, issueRepo := newBoardService()
			ctx := context.Background()

			project := &domain.Project{Key: "PROJ", Name: "Test", OwnerID: 1}
			_ = projectRepo.Create(ctx, project)
			_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})

			// Insert in the order the case lists them, then force the ids: the stub hands
			// out ids in insertion order, which would hide an unsorted result.
			for _, c := range tc.Cards {
				issue := &domain.Issue{
					Key: fmt.Sprintf("PROJ-%d", c.ID), ProjectID: project.ID, Title: "t",
					Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo,
					Priority: c.Priority, ReporterID: 1,
				}
				_ = issueRepo.Create(ctx, issue)
				issue.ID = c.ID
			}

			resp, err := svc.GetBoard(ctx, "PROJ", 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			col := boardColumn(resp, domain.IssueStatusTodo)
			if col == nil {
				t.Fatal("todo column missing")
			}
			if col.Issues == nil {
				t.Fatal("column issues must be an empty slice, not nil")
			}
			got := make([]uint, len(col.Issues))
			for i, issue := range col.Issues {
				got[i] = issue.ID
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.Expected) {
				t.Errorf("order = %v, want %v", got, tc.Expected)
			}
		})
	}
}

// ── locked cards (tasks of a completed sprint) ───────────────────────────────

func TestBoardService_GetBoard_LocksCardsOfCompletedSprintsOnly(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo, sprintRepo := newBoardServiceWithSprints()
	ctx := context.Background()

	project := &domain.Project{Key: "PROJ", Name: "Test", OwnerID: 1}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})

	completed := &domain.Sprint{ProjectID: project.ID, Name: "S1", Status: domain.SprintStatusCompleted}
	active := &domain.Sprint{ProjectID: project.ID, Name: "S2", Status: domain.SprintStatusActive}
	planning := &domain.Sprint{ProjectID: project.ID, Name: "S3", Status: domain.SprintStatusPlanning}
	other := &domain.Sprint{ProjectID: project.ID + 1, Name: "Other project", Status: domain.SprintStatusCompleted}
	for _, sp := range []*domain.Sprint{completed, active, planning, other} {
		_ = sprintRepo.Create(ctx, sp)
	}

	mk := func(key string, status string, sprintID *uint) *domain.Issue {
		issue := &domain.Issue{Key: key, ProjectID: project.ID, Title: key, Type: domain.IssueTypeTask,
			Status: status, Priority: domain.IssuePriorityMedium, ReporterID: 1, SprintID: sprintID}
		_ = issueRepo.Create(ctx, issue)
		return issue
	}
	doneOld := mk("PROJ-1", domain.IssueStatusDone, &completed.ID)
	doneNow := mk("PROJ-2", domain.IssueStatusDone, &active.ID)
	todoActive := mk("PROJ-3", domain.IssueStatusTodo, &active.ID)
	todoPlanning := mk("PROJ-4", domain.IssueStatusTodo, &planning.ID)
	noSprint := mk("PROJ-5", domain.IssueStatusInProgress, nil)

	resp, err := svc.GetBoard(ctx, "PROJ", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(resp.Columns))
	}

	locked := map[uint]bool{}
	for _, col := range resp.Columns {
		for _, card := range col.Issues {
			locked[card.ID] = card.Locked
		}
	}
	want := map[uint]bool{doneOld.ID: true, doneNow.ID: false, todoActive.ID: false, todoPlanning.ID: false, noSprint.ID: false}
	for id, w := range want {
		got, ok := locked[id]
		if !ok {
			t.Errorf("issue %d missing from the board", id)
			continue
		}
		if got != w {
			t.Errorf("issue %d: locked = %v, want %v", id, got, w)
		}
	}
	if sprintRepo.findByProjectCalls != 1 {
		t.Errorf("sprints looked up %d times, want exactly 1 for the whole board", sprintRepo.findByProjectCalls)
	}
}
