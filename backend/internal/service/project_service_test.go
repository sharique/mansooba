package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// ── stub repositories ────────────────────────────────────────────────────────

type stubProjectRepo struct {
	projects map[string]*domain.Project
	nextID   uint
}

func newStubProjectRepo() *stubProjectRepo {
	return &stubProjectRepo{projects: make(map[string]*domain.Project), nextID: 1}
}

func (r *stubProjectRepo) Create(_ context.Context, p *domain.Project) error {
	if _, exists := r.projects[p.Key]; exists {
		return domain.ErrConflict
	}
	p.ID = r.nextID
	r.nextID++
	r.projects[p.Key] = p
	return nil
}

func (r *stubProjectRepo) FindByID(_ context.Context, id uint) (*domain.Project, error) {
	for _, p := range r.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubProjectRepo) FindByKey(_ context.Context, key string) (*domain.Project, error) {
	p, ok := r.projects[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (r *stubProjectRepo) FindByUserID(_ context.Context, userID uint) ([]*domain.Project, error) {
	var result []*domain.Project
	for _, p := range r.projects {
		if p.OwnerID == userID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *stubProjectRepo) Update(_ context.Context, p *domain.Project) error {
	if _, ok := r.projects[p.Key]; !ok {
		return domain.ErrNotFound
	}
	r.projects[p.Key] = p
	return nil
}

func (r *stubProjectRepo) Delete(_ context.Context, id uint) error {
	for key, p := range r.projects {
		if p.ID == id {
			delete(r.projects, key)
			return nil
		}
	}
	return domain.ErrNotFound
}

type stubProjectMemberRepo struct {
	members []*domain.ProjectMember
	nextID  uint
}

func newStubProjectMemberRepo() *stubProjectMemberRepo {
	return &stubProjectMemberRepo{nextID: 1}
}

func (r *stubProjectMemberRepo) Create(_ context.Context, m *domain.ProjectMember) error {
	m.ID = r.nextID
	r.nextID++
	r.members = append(r.members, m)
	return nil
}

func (r *stubProjectMemberRepo) FindByProjectID(_ context.Context, projectID uint) ([]*domain.ProjectMember, error) {
	var result []*domain.ProjectMember
	for _, m := range r.members {
		if m.ProjectID == projectID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *stubProjectMemberRepo) FindByUserID(_ context.Context, userID uint) ([]*domain.ProjectMember, error) {
	var result []*domain.ProjectMember
	for _, m := range r.members {
		if m.UserID == userID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *stubProjectMemberRepo) FindByProjectAndUser(_ context.Context, projectID, userID uint) (*domain.ProjectMember, error) {
	for _, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			return m, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubProjectMemberRepo) Delete(_ context.Context, id uint) error {
	for i, m := range r.members {
		if m.ID == id {
			r.members = append(r.members[:i], r.members[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubProjectMemberRepo) DeleteByProjectID(_ context.Context, projectID uint) error {
	var kept []*domain.ProjectMember
	for _, m := range r.members {
		if m.ProjectID != projectID {
			kept = append(kept, m)
		}
	}
	r.members = kept
	return nil
}

type stubIssueRepoForProject struct{}

func (s *stubIssueRepoForProject) Create(_ context.Context, _ *domain.Issue) error        { return nil }
func (s *stubIssueRepoForProject) FindByID(_ context.Context, _ uint) (*domain.Issue, error) {
	return nil, domain.ErrNotFound
}
func (s *stubIssueRepoForProject) FindByProjectID(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) Update(_ context.Context, _ *domain.Issue) error        { return nil }
func (s *stubIssueRepoForProject) Delete(_ context.Context, _ uint) error                 { return nil }
func (s *stubIssueRepoForProject) DeleteByProjectID(_ context.Context, _ uint) error      { return nil }
func (s *stubIssueRepoForProject) FindBacklog(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) FindBySprint(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) CountBySprint(_ context.Context, _ uint) (int, int, error) {
	return 0, 0, nil
}
func (s *stubIssueRepoForProject) FindIssueIDsByLabelID(_ context.Context, _ uint) ([]uint, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) FindByAssignee(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newProjectService() service.ProjectService {
	return service.NewProjectService(
		newStubProjectRepo(),
		newStubProjectMemberRepo(),
		newStubUserRepo(),
		&stubIssueRepoForProject{},
	)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestProjectService_Create_AutoGeneratesKey(t *testing.T) {
	svc := newProjectService()

	resp, err := svc.Create(context.Background(), 1, dto.CreateProjectRequest{
		Name: "My Project",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Key == "" {
		t.Fatal("expected auto-generated key")
	}
	if resp.Key != "mypr" {
		t.Errorf("expected key mypr, got %s", resp.Key)
	}
}

func TestProjectService_Create_KeyConflictAppendsDigit(t *testing.T) {
	svc := newProjectService()

	_, _ = svc.Create(context.Background(), 1, dto.CreateProjectRequest{Name: "My Project"})
	resp, err := svc.Create(context.Background(), 1, dto.CreateProjectRequest{Name: "My Project"})

	if err != nil {
		t.Fatalf("unexpected error on second create: %v", err)
	}
	if resp.Key == "mypr" {
		t.Error("second project must not reuse the same key")
	}
	if resp.Key != "mypr2" {
		t.Errorf("expected key mypr2, got %s", resp.Key)
	}
}

func TestProjectService_FindByKey_ForbiddenForNonMember(t *testing.T) {
	svc := newProjectService()

	_, _ = svc.Create(context.Background(), 1, dto.CreateProjectRequest{Name: "My Project"})

	_, err := svc.FindByKey(context.Background(), "mypr", 99)

	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectService_AddMember_ConflictIfAlreadyMember(t *testing.T) {
	userRepo := newStubUserRepo()
	projectRepo := newStubProjectRepo()
	memberRepo := newStubProjectMemberRepo()
	svc := service.NewProjectService(projectRepo, memberRepo, userRepo, &stubIssueRepoForProject{})

	// Register a target user
	ctx := context.Background()
	_ = userRepo.Create(ctx, &domain.User{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "x"})
	_, _ = svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})

	err1 := svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "bob@example.com", Role: "member"})
	if err1 != nil {
		t.Fatalf("first add failed: %v", err1)
	}

	err2 := svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "bob@example.com", Role: "member"})
	if !errors.Is(err2, domain.ErrConflict) {
		t.Errorf("expected ErrConflict on duplicate add, got %v", err2)
	}
}

func TestProjectService_RemoveMember_CannotRemoveOwner(t *testing.T) {
	svc := newProjectService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})

	err := svc.RemoveMember(ctx, resp.Key, 1, 1)

	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden when removing owner, got %v", err)
	}
}

func TestProjectService_Create_KeyIsLowercase(t *testing.T) {
	svc := newProjectService()

	resp, err := svc.Create(context.Background(), 1, dto.CreateProjectRequest{
		Name: "My Project",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Key != strings.ToLower(resp.Key) {
		t.Errorf("expected lowercase key, got %s", resp.Key)
	}
}

func TestProjectService_Create_ExplicitKeyIsLowercased(t *testing.T) {
	svc := newProjectService()

	resp, err := svc.Create(context.Background(), 1, dto.CreateProjectRequest{
		Name: "My Project",
		Key:  "MYPR",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Key != "mypr" {
		t.Errorf("expected key mypr, got %s", resp.Key)
	}
}

func TestProjectService_FindByKey_UppercaseInputNormalizedToLowercase(t *testing.T) {
	svc := newProjectService()
	ctx := context.Background()

	// Create a project (key stored as lowercase "mypr")
	created, err := svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Key != "mypr" {
		t.Fatalf("expected lowercase key mypr on create, got %s", created.Key)
	}

	// Lookup with uppercase key — service normalizes to lowercase before querying
	found, err := svc.FindByKey(ctx, "MYPR", 1)
	if err != nil {
		t.Fatalf("FindByKey with uppercase input failed: %v", err)
	}
	if found.Key != "mypr" {
		t.Errorf("expected found key mypr, got %s", found.Key)
	}
}
