package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubActivitySvc struct {
	recorded []*domain.ActivityEvent
}

func (s *stubActivitySvc) Record(_ context.Context, e *domain.ActivityEvent) error {
	s.recorded = append(s.recorded, e)
	return nil
}

func (s *stubActivitySvc) ListByIssue(_ context.Context, _ uint) ([]*dto.ActivityEventResponse, error) {
	return nil, nil
}

func (s *stubActivitySvc) GetMyActivity(_ context.Context, _ uint, _, _ int) ([]*dto.ActivityEventResponse, error) {
	return nil, nil
}

// stubIssueRepo is a full in-memory IssueRepository for issue service tests.
// stubIssueRepoForProject (no-op) is defined in project_service_test.go.
type stubIssueRepo struct {
	issues            []*domain.Issue
	nextID            uint
	labelIDToIssueIDs map[uint][]uint
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

func (r *stubIssueRepo) FindIssueIDsByLabelID(_ context.Context, labelID uint) ([]uint, error) {
	if r.labelIDToIssueIDs != nil {
		return r.labelIDToIssueIDs[labelID], nil
	}
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newIssueService() (service.IssueService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo) {
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	issueRepo := newStubIssueRepo()
	activitySvc := &stubActivitySvc{}
	userRepo := newStubUserRepo()
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, activitySvc, userRepo)
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

func TestIssueService_Update_RecordsStatusChangeActivity(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	activitySvc := &stubActivitySvc{}
	userRepo := newStubUserRepo()

	projectRepo.projects["PROJ"] = &domain.Project{ID: 1, Key: "PROJ"}
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 1, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{
		ID: 1, ProjectID: 1, Key: "PROJ-1",
		Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium,
		ReporterID: 1,
	})

	newStatus := domain.IssueStatusInProgress
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, activitySvc, userRepo)
	_, err := svc.Update(context.Background(), "PROJ", 1, 1, dto.UpdateIssueRequest{Status: &newStatus})
	require.NoError(t, err)

	require.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityStatusChanged, activitySvc.recorded[0].Kind)
	assert.Equal(t, domain.IssueStatusTodo, activitySvc.recorded[0].OldValue)
	assert.Equal(t, domain.IssueStatusInProgress, activitySvc.recorded[0].NewValue)
}

func TestIssueService_Update_RecordsPriorityChangeActivity(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	activitySvc := &stubActivitySvc{}
	userRepo := newStubUserRepo()

	projectRepo.projects["PROJ"] = &domain.Project{ID: 1, Key: "PROJ"}
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 1, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{
		ID: 1, ProjectID: 1, Key: "PROJ-1",
		Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium,
		ReporterID: 1,
	})

	newPriority := domain.IssuePriorityHigh
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, activitySvc, userRepo)
	_, err := svc.Update(context.Background(), "PROJ", 1, 1, dto.UpdateIssueRequest{Priority: &newPriority})
	require.NoError(t, err)

	require.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityPriorityChanged, activitySvc.recorded[0].Kind)
}

func TestIssueService_Update_NoActivityWhenNothingChanges(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	activitySvc := &stubActivitySvc{}
	userRepo := newStubUserRepo()

	projectRepo.projects["PROJ"] = &domain.Project{ID: 1, Key: "PROJ"}
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 1, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{
		ID: 1, ProjectID: 1, Key: "PROJ-1",
		Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium,
		ReporterID: 1,
	})

	sameStatus := domain.IssueStatusTodo
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, activitySvc, userRepo)
	_, err := svc.Update(context.Background(), "PROJ", 1, 1, dto.UpdateIssueRequest{Status: &sameStatus})
	require.NoError(t, err)
	assert.Empty(t, activitySvc.recorded)
}

func TestIssueService_Update_RecordsAssigneeChangeActivity(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	activitySvc := &stubActivitySvc{}
	userRepo := newStubUserRepo()
	userRepo.users["alice@example.com"] = &domain.User{ID: 10, Name: "Alice", Email: "alice@example.com"}

	projectRepo.projects["PROJ"] = &domain.Project{ID: 1, Key: "PROJ"}
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 1, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{
		ID: 1, ProjectID: 1, Key: "PROJ-1",
		Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium,
		AssigneeID: nil, ReporterID: 1,
	})

	aliceID := uint(10)
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, activitySvc, userRepo)
	_, err := svc.Update(context.Background(), "PROJ", 1, 1, dto.UpdateIssueRequest{AssigneeID: &aliceID})
	require.NoError(t, err)

	require.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityAssigneeChanged, activitySvc.recorded[0].Kind)
	assert.Equal(t, "unassigned", activitySvc.recorded[0].OldValue)
	assert.Equal(t, "Alice", activitySvc.recorded[0].NewValue)
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

func TestIssueService_ListByProject_SearchByText(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	userRepo := newStubUserRepo()

	proj := &domain.Project{ID: 10, Key: "SRCH"}
	projectRepo.projects["SRCH"] = proj
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues,
		&domain.Issue{ID: 1, ProjectID: 10, Key: "SRCH-1", Title: "Login bug", Description: "auth fails"},
		&domain.Issue{ID: 2, ProjectID: 10, Key: "SRCH-2", Title: "Signup flow", Description: "registration"},
	)

	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, &stubActivitySvc{}, userRepo)

	result, err := svc.ListByProject(context.Background(), "SRCH", 1, dto.IssueListQuery{Q: "login"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "SRCH-1", result[0].Key)
}

func TestIssueService_ListByProject_FilterByPriority(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	userRepo := newStubUserRepo()

	proj := &domain.Project{ID: 11, Key: "PRI"}
	projectRepo.projects["PRI"] = proj
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 11, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues,
		&domain.Issue{ID: 3, ProjectID: 11, Key: "PRI-1", Title: "A", Priority: domain.IssuePriorityHigh},
		&domain.Issue{ID: 4, ProjectID: 11, Key: "PRI-2", Title: "B", Priority: domain.IssuePriorityLow},
	)

	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, &stubActivitySvc{}, userRepo)

	result, err := svc.ListByProject(context.Background(), "PRI", 1, dto.IssueListQuery{Priority: "high"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "PRI-1", result[0].Key)
}

func TestIssueService_ListByProject_FilterByLabelID(t *testing.T) {
	issueRepo := newStubIssueRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	userRepo := newStubUserRepo()

	proj := &domain.Project{ID: 12, Key: "LBL"}
	projectRepo.projects["LBL"] = proj
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 12, UserID: 1, Role: "member"})
	issueRepo.issues = append(issueRepo.issues,
		&domain.Issue{ID: 5, ProjectID: 12, Key: "LBL-1", Title: "Has label"},
		&domain.Issue{ID: 6, ProjectID: 12, Key: "LBL-2", Title: "No label"},
	)
	// Stub FindIssueIDsByLabelID to return only issue 5 for label 7.
	issueRepo.labelIDToIssueIDs = map[uint][]uint{7: {5}}

	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, &stubActivitySvc{}, userRepo)

	result, err := svc.ListByProject(context.Background(), "LBL", 1, dto.IssueListQuery{LabelID: 7})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "LBL-1", result[0].Key)
}

// ── GetMyIssues ───────────────────────────────────────────────────────────────

func (r *stubIssueRepo) FindByAssignee(_ context.Context, userID uint) ([]*domain.Issue, error) {
	var result []*domain.Issue
	for _, i := range r.issues {
		if i.AssigneeID != nil && *i.AssigneeID == userID {
			result = append(result, i)
		}
	}
	return result, nil
}

func TestIssueService_GetMyIssues_returns_assigned_issues(t *testing.T) {
	issueRepo := newStubIssueRepo()
	uid := uint(42)
	other := uint(99)
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "A-1", Title: "Mine", AssigneeID: &uid, Status: domain.IssueStatusInProgress},
		{ID: 2, Key: "A-2", Title: "Not mine", AssigneeID: &other, Status: domain.IssueStatusTodo},
		{ID: 3, Key: "A-3", Title: "Unassigned", AssigneeID: nil, Status: domain.IssueStatusTodo},
	}

	svc := service.NewIssueService(issueRepo, newStubProjectRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, newStubUserRepo())

	result, err := svc.GetMyIssues(context.Background(), uid, dto.IssueListQuery{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "A-1", result[0].Key)
}

func TestIssueService_GetMyIssues_filters_by_status(t *testing.T) {
	issueRepo := newStubIssueRepo()
	uid := uint(7)
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "B-1", Title: "Active", AssigneeID: &uid, Status: domain.IssueStatusInProgress},
		{ID: 2, Key: "B-2", Title: "Done", AssigneeID: &uid, Status: domain.IssueStatusDone},
	}

	svc := service.NewIssueService(issueRepo, newStubProjectRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, newStubUserRepo())

	result, err := svc.GetMyIssues(context.Background(), uid, dto.IssueListQuery{Status: domain.IssueStatusInProgress})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "B-1", result[0].Key)
}

func TestIssueService_GetMyIssues_empty_when_none_assigned(t *testing.T) {
	issueRepo := newStubIssueRepo()
	other := uint(5)
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "C-1", AssigneeID: &other, Status: domain.IssueStatusTodo},
	}

	svc := service.NewIssueService(issueRepo, newStubProjectRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, newStubUserRepo())

	result, err := svc.GetMyIssues(context.Background(), 999, dto.IssueListQuery{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

// T022: assignee_avatar_url and assignee_name enrichment tests

func newIssueServiceWithUsers(userRepo domain.UserRepository) (service.IssueService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo) {
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	issueRepo := newStubIssueRepo()
	activitySvc := &stubActivitySvc{}
	svc := service.NewIssueService(issueRepo, projectRepo, memberRepo, activitySvc, userRepo)
	return svc, projectRepo, memberRepo, issueRepo
}

func TestIssueService_GetMyIssues_AssigneeAvatarURLPopulated(t *testing.T) {
	userRepo := newStubUserRepo()
	assigneeID := uint(10)
	_ = userRepo.Create(context.Background(), &domain.User{ID: assigneeID, Name: "Alice", Email: "alice@example.com", Password: "x", AvatarURL: "/uploads/avatars/avatar-10.jpg?v=1000"})

	issueRepo := newStubIssueRepo()
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "P-1", Title: "Issue", AssigneeID: &assigneeID, Status: domain.IssueStatusTodo},
	}
	svc := service.NewIssueService(issueRepo, newStubProjectRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, userRepo)

	result, err := svc.GetMyIssues(context.Background(), assigneeID, dto.IssueListQuery{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].AssigneeAvatarURL)
	assert.Equal(t, "/uploads/avatars/avatar-10.jpg?v=1000", *result[0].AssigneeAvatarURL)
	require.NotNil(t, result[0].AssigneeName)
	assert.Equal(t, "Alice", *result[0].AssigneeName)
}

func TestIssueService_GetMyIssues_AssigneeAvatarURLNilWhenNoAvatar(t *testing.T) {
	userRepo := newStubUserRepo()
	assigneeID := uint(11)
	_ = userRepo.Create(context.Background(), &domain.User{ID: assigneeID, Name: "Bob", Email: "bob@example.com", Password: "x"})

	issueRepo := newStubIssueRepo()
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "P-1", Title: "Issue", AssigneeID: &assigneeID, Status: domain.IssueStatusTodo},
	}
	svc := service.NewIssueService(issueRepo, newStubProjectRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, userRepo)

	result, err := svc.GetMyIssues(context.Background(), assigneeID, dto.IssueListQuery{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Nil(t, result[0].AssigneeAvatarURL)
}

func TestIssueService_GetMyIssues_AssigneeNilWhenUnassigned(t *testing.T) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "P-1", Title: "Unassigned", AssigneeID: nil, Status: domain.IssueStatusTodo},
	}
	svc := service.NewIssueService(issueRepo, newStubProjectRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, newStubUserRepo())

	result, err := svc.GetMyIssues(context.Background(), 99, dto.IssueListQuery{})
	require.NoError(t, err)
	// No issues assigned to 99, so result should be empty — that's correct.
	assert.Empty(t, result)
}

func TestIssueService_ListByProject_AssigneeAvatarURLPopulated(t *testing.T) {
	userRepo := newStubUserRepo()
	assigneeID := uint(20)
	_ = userRepo.Create(context.Background(), &domain.User{ID: assigneeID, Name: "Carol", Email: "carol@example.com", Password: "x", AvatarURL: "/uploads/avatars/avatar-20.jpg?v=999"})

	svc, projectRepo, memberRepo, issueRepo := newIssueServiceWithUsers(userRepo)
	ctx := context.Background()
	project := seedProject(ctx, projectRepo, memberRepo)
	issueRepo.issues = []*domain.Issue{
		{ID: 1, Key: "PROJ-1", ProjectID: project.ID, Title: "Issue", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, AssigneeID: &assigneeID, ReporterID: 1},
	}

	result, err := svc.ListByProject(ctx, "PROJ", 1, dto.IssueListQuery{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].AssigneeAvatarURL)
	assert.Equal(t, "/uploads/avatars/avatar-20.jpg?v=999", *result[0].AssigneeAvatarURL)
}
