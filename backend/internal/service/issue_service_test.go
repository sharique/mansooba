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

	r1, err := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: "task", Priority: "low"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r1.Key != "PROJ-1" {
		t.Errorf("expected PROJ-1, got %s", r1.Key)
	}

	r2, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 2", Type: "story", Priority: "medium"})
	if r2.Key != "PROJ-2" {
		t.Errorf("expected PROJ-2, got %s", r2.Key)
	}

	r3, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 3", Type: "bug", Priority: "high"})
	if r3.Key != "PROJ-3" {
		t.Errorf("expected PROJ-3, got %s", r3.Key)
	}
}

func TestIssueService_Update_RejectsInvalidStatus(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: "task", Priority: "low"})

	invalid := "flying"
	_, err := svc.Update(ctx, "PROJ", resp.ID, 1, dto.UpdateIssueRequest{Status: &invalid})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestIssueService_Delete_ForbiddenForNonReporter(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	project := seedProject(ctx, projectRepo, memberRepo)
	// User 2 is a member but not a reporter or admin
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 2, Role: "member"})

	resp, _ := svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Issue 1", Type: "task", Priority: "low"})

	err := svc.Delete(ctx, "PROJ", resp.ID, 2)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestIssueService_ListByProject_FiltersApplied(t *testing.T) {
	svc, projectRepo, memberRepo, _ := newIssueService()
	ctx := context.Background()
	seedProject(ctx, projectRepo, memberRepo)

	_, _ = svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Task 1", Type: "task", Priority: "low"})
	_, _ = svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Bug 1", Type: "bug", Priority: "high"})
	_, _ = svc.Create(ctx, "PROJ", 1, dto.CreateIssueRequest{Title: "Task 2", Type: "task", Priority: "medium"})

	results, err := svc.ListByProject(ctx, "PROJ", 1, dto.IssueListQuery{Type: "task"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 task issues, got %d", len(results))
	}
	for _, r := range results {
		if r.Type != "task" {
			t.Errorf("expected type task, got %s", r.Type)
		}
	}
}
