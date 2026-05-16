package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

// ── stubSprintRepo ────────────────────────────────────────────────────────────

type stubSprintRepo struct {
	sprints          []*domain.Sprint
	nextID           uint
	lastMigratedIDs  []uint
	lastNextSprintID *uint
}

func newStubSprintRepo() *stubSprintRepo {
	return &stubSprintRepo{nextID: 1}
}

func (r *stubSprintRepo) Create(_ context.Context, s *domain.Sprint) error {
	s.ID = r.nextID
	r.nextID++
	cp := *s
	r.sprints = append(r.sprints, &cp)
	return nil
}

func (r *stubSprintRepo) FindByID(_ context.Context, id uint) (*domain.Sprint, error) {
	for _, s := range r.sprints {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubSprintRepo) FindByProject(_ context.Context, projectID uint) ([]*domain.Sprint, error) {
	var result []*domain.Sprint
	for _, s := range r.sprints {
		if s.ProjectID == projectID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *stubSprintRepo) Update(_ context.Context, s *domain.Sprint) error {
	for i, existing := range r.sprints {
		if existing.ID == s.ID {
			r.sprints[i] = s
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubSprintRepo) Delete(_ context.Context, id uint) error {
	for i, s := range r.sprints {
		if s.ID == id {
			r.sprints = append(r.sprints[:i], r.sprints[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubSprintRepo) FindActiveByProject(_ context.Context, projectID uint) (*domain.Sprint, error) {
	for _, s := range r.sprints {
		if s.ProjectID == projectID && s.Status == domain.SprintStatusActive {
			return s, nil
		}
	}
	return nil, nil
}

func (r *stubSprintRepo) FindWithIssues(ctx context.Context, id uint) (*domain.Sprint, error) {
	return r.FindByID(ctx, id)
}

func (r *stubSprintRepo) CompleteWithMigration(ctx context.Context, sprint *domain.Sprint, unfinishedIDs []uint, nextSprintID *uint) error {
	r.lastMigratedIDs = unfinishedIDs
	r.lastNextSprintID = nextSprintID
	return r.Update(ctx, sprint)
}

// ── test helpers ──────────────────────────────────────────────────────────────

func newSprintService() (service.SprintService, *stubProjectRepo, *stubProjectMemberRepo, *stubIssueRepo, *stubSprintRepo) {
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	issueRepo := newStubIssueRepo()
	sprintRepo := newStubSprintRepo()
	svc := service.NewSprintService(sprintRepo, issueRepo, projectRepo, memberRepo)
	return svc, projectRepo, memberRepo, issueRepo, sprintRepo
}

func seedSprintProject(ctx context.Context, projectRepo *stubProjectRepo, memberRepo *stubProjectMemberRepo, ownerID uint) *domain.Project {
	project := &domain.Project{Key: "TEST", Name: "Test", OwnerID: ownerID}
	_ = projectRepo.Create(ctx, project)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: ownerID, Role: "admin"})
	return project
}

// ── CRUD tests ────────────────────────────────────────────────────────────────

func TestSprintService_Create_HappyPath(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	seedSprintProject(ctx, projectRepo, memberRepo, 1)

	resp, err := svc.Create(ctx, "TEST", 1, dto.CreateSprintRequest{Name: "Sprint 1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Sprint 1" {
		t.Errorf("expected name Sprint 1, got %s", resp.Name)
	}
	if resp.Status != domain.SprintStatusPlanning {
		t.Errorf("expected status planning, got %s", resp.Status)
	}
	if len(sprintRepo.sprints) != 1 {
		t.Errorf("expected 1 sprint in repo, got %d", len(sprintRepo.sprints))
	}
}

func TestSprintService_Create_ProjectNotFound(t *testing.T) {
	svc, _, _, _, _ := newSprintService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "NOPE", 1, dto.CreateSprintRequest{Name: "Sprint 1"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSprintService_Create_ForbiddenForNonAdmin(t *testing.T) {
	svc, projectRepo, memberRepo, _, _ := newSprintService()
	ctx := context.Background()
	project := seedSprintProject(ctx, projectRepo, memberRepo, 1)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: 2, Role: "member"})

	_, err := svc.Create(ctx, "TEST", 2, dto.CreateSprintRequest{Name: "Sprint 1"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSprintService_List_ReturnsSprintsForProject(t *testing.T) {
	svc, projectRepo, memberRepo, _, _ := newSprintService()
	ctx := context.Background()
	seedSprintProject(ctx, projectRepo, memberRepo, 1)

	_, _ = svc.Create(ctx, "TEST", 1, dto.CreateSprintRequest{Name: "Sprint A"})
	_, _ = svc.Create(ctx, "TEST", 1, dto.CreateSprintRequest{Name: "Sprint B"})

	sprints, err := svc.List(ctx, "TEST", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sprints) != 2 {
		t.Errorf("expected 2 sprints, got %d", len(sprints))
	}
}

func TestSprintService_Get_WrongProject_ReturnsNotFound(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	otherSprint := &domain.Sprint{ProjectID: p.ID + 99, Status: "planning", Name: "Other"}
	_ = sprintRepo.Create(ctx, otherSprint)

	_, err := svc.Get(ctx, "TEST", otherSprint.ID, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSprintService_Update_BlockedForCompletedSprint(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	completed := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusCompleted, Name: "Done"}
	_ = sprintRepo.Create(ctx, completed)

	name := "new name"
	_, err := svc.Update(ctx, "TEST", completed.ID, 1, dto.UpdateSprintRequest{Name: &name})
	if !errors.Is(err, domain.ErrSprintNotEditable) {
		t.Errorf("expected ErrSprintNotEditable, got %v", err)
	}
}

func TestSprintService_Update_ForbiddenForMember(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)
	_ = memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: p.ID, UserID: 2, Role: "member"})

	sprint := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusPlanning, Name: "Sprint 1"}
	_ = sprintRepo.Create(ctx, sprint)

	name := "renamed"
	_, err := svc.Update(ctx, "TEST", sprint.ID, 2, dto.UpdateSprintRequest{Name: &name})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSprintService_Delete_BlockedForActiveSprint(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	active := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusActive, Name: "Sprint 1"}
	_ = sprintRepo.Create(ctx, active)

	err := svc.Delete(ctx, "TEST", active.ID, 1)
	if !errors.Is(err, domain.ErrSprintNotDeletable) {
		t.Errorf("expected ErrSprintNotDeletable, got %v", err)
	}
}

func TestSprintService_Delete_HappyPath(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	sprint := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusPlanning, Name: "Sprint 1"}
	_ = sprintRepo.Create(ctx, sprint)

	err := svc.Delete(ctx, "TEST", sprint.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sprintRepo.sprints) != 0 {
		t.Errorf("expected 0 sprints after delete, got %d", len(sprintRepo.sprints))
	}
}

// ── Lifecycle tests ───────────────────────────────────────────────────────────

func TestSprintService_Start_HappyPath(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	sprint := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusPlanning, Name: "Sprint 1"}
	_ = sprintRepo.Create(ctx, sprint)

	resp, err := svc.Start(ctx, "TEST", sprint.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != domain.SprintStatusActive {
		t.Errorf("expected active, got %s", resp.Status)
	}
	if resp.StartDate == nil {
		t.Error("expected StartDate to be set")
	}
}

func TestSprintService_Start_InvalidTransition_AlreadyActive(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	active := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusActive, Name: "Sprint 1"}
	_ = sprintRepo.Create(ctx, active)

	_, err := svc.Start(ctx, "TEST", active.ID, 1)
	if !errors.Is(err, domain.ErrSprintInvalidTransition) {
		t.Errorf("expected ErrSprintInvalidTransition, got %v", err)
	}
}

func TestSprintService_Start_AnotherSprintAlreadyActive(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	active := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusActive, Name: "Sprint 1"}
	planning := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusPlanning, Name: "Sprint 2"}
	_ = sprintRepo.Create(ctx, active)
	_ = sprintRepo.Create(ctx, planning)

	_, err := svc.Start(ctx, "TEST", planning.ID, 1)
	if !errors.Is(err, domain.ErrSprintAlreadyActive) {
		t.Errorf("expected ErrSprintAlreadyActive, got %v", err)
	}
}

func TestSprintService_Complete_MovesToBacklog(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	sprint := &domain.Sprint{
		ProjectID: p.ID,
		Status:    domain.SprintStatusActive,
		Name:      "Sprint 1",
		Issues: []domain.Issue{
			{ID: 1, Status: "todo"},
			{ID: 2, Status: "done"},
		},
	}
	_ = sprintRepo.Create(ctx, sprint)

	resp, err := svc.Complete(ctx, "TEST", sprint.ID, 1, dto.CompleteSprintRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != domain.SprintStatusCompleted {
		t.Errorf("expected completed, got %s", resp.Status)
	}
	if len(sprintRepo.lastMigratedIDs) != 1 || sprintRepo.lastMigratedIDs[0] != 1 {
		t.Errorf("expected [1] to be migrated, got %v", sprintRepo.lastMigratedIDs)
	}
	if sprintRepo.lastNextSprintID != nil {
		t.Errorf("expected nil nextSprintID, got %v", sprintRepo.lastNextSprintID)
	}
}

func TestSprintService_Complete_MovesToNextSprint(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	nextSprint := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusPlanning, Name: "Sprint 2"}
	_ = sprintRepo.Create(ctx, nextSprint)

	active := &domain.Sprint{
		ProjectID: p.ID,
		Status:    domain.SprintStatusActive,
		Name:      "Sprint 1",
		Issues:    []domain.Issue{{ID: 10, Status: "in_progress"}},
	}
	_ = sprintRepo.Create(ctx, active)

	resp, err := svc.Complete(ctx, "TEST", active.ID, 1, dto.CompleteSprintRequest{NextSprintID: &nextSprint.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != domain.SprintStatusCompleted {
		t.Errorf("expected completed, got %s", resp.Status)
	}
	if sprintRepo.lastNextSprintID == nil || *sprintRepo.lastNextSprintID != nextSprint.ID {
		t.Errorf("expected nextSprintID=%d, got %v", nextSprint.ID, sprintRepo.lastNextSprintID)
	}
}

func TestSprintService_Complete_SkipsDoneIssues(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	active := &domain.Sprint{
		ProjectID: p.ID,
		Status:    domain.SprintStatusActive,
		Name:      "Sprint 1",
		Issues:    []domain.Issue{{ID: 5, Status: "done"}},
	}
	_ = sprintRepo.Create(ctx, active)

	_, err := svc.Complete(ctx, "TEST", active.ID, 1, dto.CompleteSprintRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sprintRepo.lastMigratedIDs) != 0 {
		t.Errorf("expected empty migrated IDs, got %v", sprintRepo.lastMigratedIDs)
	}
}

func TestSprintService_Complete_InvalidTransition_NotActive(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	planning := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusPlanning, Name: "Sprint 1"}
	_ = sprintRepo.Create(ctx, planning)

	_, err := svc.Complete(ctx, "TEST", planning.ID, 1, dto.CompleteSprintRequest{})
	if !errors.Is(err, domain.ErrSprintInvalidTransition) {
		t.Errorf("expected ErrSprintInvalidTransition, got %v", err)
	}
}

func TestSprintService_Complete_NextSprintMustBePlanning(t *testing.T) {
	svc, projectRepo, memberRepo, _, sprintRepo := newSprintService()
	ctx := context.Background()
	p := seedSprintProject(ctx, projectRepo, memberRepo, 1)

	completedNext := &domain.Sprint{ProjectID: p.ID, Status: domain.SprintStatusCompleted, Name: "Old Sprint"}
	_ = sprintRepo.Create(ctx, completedNext)

	active := &domain.Sprint{
		ProjectID: p.ID,
		Status:    domain.SprintStatusActive,
		Name:      "Sprint 1",
		Issues:    []domain.Issue{{ID: 3, Status: "todo"}},
	}
	_ = sprintRepo.Create(ctx, active)

	_, err := svc.Complete(ctx, "TEST", active.ID, 1, dto.CompleteSprintRequest{NextSprintID: &completedNext.ID})
	if !errors.Is(err, domain.ErrSprintInvalidTransition) {
		t.Errorf("expected ErrSprintInvalidTransition, got %v", err)
	}
}
