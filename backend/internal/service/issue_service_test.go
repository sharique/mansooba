package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

// stubIssueRepo is a full in-memory IssueRepository for issue service tests.
// stubIssueRepoForProject (no-op) is defined in project_service_test.go.
type stubIssueRepo struct {
	issues []*domain.Issue
	nextID uint
}

func newStubIssueRepo() *stubIssueRepo {
	return &stubIssueRepo{nextID: 1}
}

func (r *stubIssueRepo) Create(_ context.Context, issue *domain.Issue) error {
	issue.ID = r.nextID
	r.nextID++
	r.issues = append(r.issues, issue)
	return nil
}

func (r *stubIssueRepo) FindByID(_ context.Context, id uint) (*domain.Issue, error) {
	for _, i := range r.issues {
		if i.ID == id {
			return i, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubIssueRepo) FindByProjectID(_ context.Context, projectID uint) ([]*domain.Issue, error) {
	var result []*domain.Issue
	for _, i := range r.issues {
		if i.ProjectID == projectID {
			result = append(result, i)
		}
	}
	return result, nil
}

func (r *stubIssueRepo) Update(_ context.Context, issue *domain.Issue) error {
	for i, existing := range r.issues {
		if existing.ID == issue.ID {
			r.issues[i] = issue
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubIssueRepo) Delete(_ context.Context, id uint) error {
	for i, issue := range r.issues {
		if issue.ID == id {
			r.issues = append(r.issues[:i], r.issues[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubIssueRepo) DeleteByProjectID(_ context.Context, projectID uint) error {
	var kept []*domain.Issue
	for _, i := range r.issues {
		if i.ProjectID != projectID {
			kept = append(kept, i)
		}
	}
	r.issues = kept
	return nil
}

func (r *stubIssueRepo) FindBacklog(_ context.Context, projectID uint) ([]*domain.Issue, error) {
	return nil, nil
}

func (r *stubIssueRepo) FindBySprint(_ context.Context, sprintID uint) ([]*domain.Issue, error) {
	return nil, nil
}

func (r *stubIssueRepo) CountBySprint(_ context.Context, sprintID uint) (int, int, error) {
	return 0, 0, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newIssueService() (service.IssueService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo) {
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	issueRepo := newStubIssueRepo()
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo)
	return svc, projectRepo, memberRepo, issueRepo
}

// seedProject creates a project owned by userID=1 and makes them a member.
func seedProject(ctx context.Context, projectRepo *stubProjectRepo, memberRepo *stubProjectMemberRepo) *domain.Project {
	project := &domain.Project{Key: "PROJ", Name: "Test Project", OwnerID: 1}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 1, Role: "admin"})
	return project
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestIssueService_Create_GeneratesSequentialKey(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	r1, err := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r1.Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", r1.Key)
	}

	r2, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 2", Type: domain.IssueTypeStory, Priority: domain.IssuePriorityMedium})
	if r2.Key != "PROJ-2" {
		t.Errorf("expected PROJ-2, got %s", r2.Key)
	}

	r3, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 3", Type: domain.IssueTypeBug, Priority: domain.IssuePriorityHigh})
	if r3.Key != "PROJ-3" {
		t.Errorf("expected PROJ-3, got %s", r3.Key)
	}
}

func TestIssueService_Update_RejectsInvalidStatus(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	invalid := "flying"
	_, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &invalid})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestIssueService_Update_AssignsIssueToSprint(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	sprintID := uint(5)
	updated, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{SprintID: &sprintID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.SprintID == nil || *updated.SprintID != 5 {
		t.Errorf("expected SprintID=5, got %v", updated.SprintID)
	}
	// Verify persisted value matches
	stored, _ := issueRepo.FindByID(ctx, resp.ID)
	if stored.SprintID == nil || *stored.SprintID != 5 {
		t.Errorf("stored SprintID expected 5, got %v", stored.SprintID)
	}
}

func TestIssueService_Update_ClearsSprintID(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	// First assign to a sprint
	sprintID := uint(3)
	_, _ = svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{SprintID: &sprintID})

	// Sentinel value 0 means "move to backlog"
	zero := uint(0)
	updated, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{SprintID: &zero})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.SprintID != nil {
		t.Errorf("expected SprintID=nil after clearing, got %v", updated.SprintID)
	}
	stored, _ := issueRepo.FindByID(ctx, resp.ID)
	if stored.SprintID != nil {
		t.Errorf("stored SprintID expected nil, got %v", stored.SprintID)
	}
}

func TestIssueService_Delete_ForbiddenForNonReporter(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	project := seedProject(ctx, projectRepo, memberRepo)
	// User 2 is a member but not a reporter or admin
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 2, Role: "member"})

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	err := svc.Delete(ctx, "PROJ", resp.ID, 2)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestIssueService_Update_SetsCompletedAtWhenTransitioningToDone(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	done := domain.IssueStatusDone
	updated, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &done})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set after transitioning to done")
	}
	stored, _ := issueRepo.FindByID(ctx, resp.ID)
	if stored.CompletedAt == nil {
		t.Error("expected persisted CompletedAt to be set")
	}
}

func TestIssueService_Update_ClearsCompletedAtWhenLeavingDone(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	done := domain.IssueStatusDone
	_, _ = svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &done})

	inProgress := domain.IssueStatusInProgress
	updated, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &inProgress})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.CompletedAt != nil {
		t.Fatal("expected CompletedAt to be cleared after leaving done")
	}
	stored, _ := issueRepo.FindByID(ctx, resp.ID)
	if stored.CompletedAt != nil {
		t.Error("expected persisted CompletedAt to be nil after leaving done")
	}
}

func TestIssueService_Update_PreservesCompletedAtWhenAlreadyDone(t *testing.T) {
	svc, projectRepo, memberRepo, issueRepo := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})

	done := domain.IssueStatusDone
	_, _ = svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &done})

	first, _ := issueRepo.FindByID(ctx, resp.ID)
	originalCompletedAt := first.CompletedAt

	// Re-mark as done — CompletedAt must not be reset.
	_, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &done})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, _ := issueRepo.FindByID(ctx, resp.ID)
	if stored.CompletedAt == nil || !stored.CompletedAt.Equal(*originalCompletedAt) {
		t.Error("expected CompletedAt to be unchanged when issue was already done")
	}
}

func TestIssueService_ListByProject_FiltersApplied(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	_, _ = svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Task 1", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})
	_, _ = svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Bug 1", Type: domain.IssueTypeBug, Priority: domain.IssuePriorityHigh})
	_, _ = svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Task 2", Type: domain.IssueTypeTask, Priority: domain.IssuePriorityMedium})

	results, err := svc.ListByProject(ctx, "PROJ", 1, dto.IssueListQuery{Type: domain.IssueTypeTask})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 task issues, got %d", len(results))
	}
	for _, r := range results {
		if r.Type != domain.IssueTypeTask {
			t.Errorf("expected type task, got %s", r.Type)
		}
	}
}
