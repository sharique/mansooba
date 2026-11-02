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
	// members, when set, receives the admin membership CreateWithAdmin records.
	members *stubProjectMemberRepo
	// abandoned is what FindWithoutActiveAdmin returns.
	abandoned []*domain.Project
	// tasks is how many tasks DeleteIfEmpty finds.
	tasks int64
}

func (r *stubProjectRepo) DeleteIfEmpty(ctx context.Context, id uint) error {
	if r.tasks > 0 {
		return &domain.ProjectNotEmptyError{TaskCount: int(r.tasks)}
	}
	if r.members != nil {
		_ = r.members.DeleteByProjectID(ctx, id)
	}
	return r.Delete(ctx, id)
}

func (r *stubProjectRepo) CreateWithAdmin(ctx context.Context, p *domain.Project, adminUserID uint) error {
	if err := r.Create(ctx, p); err != nil {
		return err
	}
	if r.members != nil {
		return r.members.Create(ctx, &domain.ProjectMember{ProjectID: p.ID, UserID: adminUserID, Role: domain.RoleAdmin})
	}
	return nil
}

func (r *stubProjectRepo) FindWithoutActiveAdmin(_ context.Context) ([]*domain.Project, error) {
	return r.abandoned, nil
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
	// inactive marks accounts as disabled for the active-admin count.
	inactive map[uint]bool
}

func newStubProjectMemberRepo() *stubProjectMemberRepo {
	return &stubProjectMemberRepo{nextID: 1}
}

// inactive lists accounts the stub treats as disabled when counting active admins.
func (r *stubProjectMemberRepo) countActiveAdmins(projectID uint) int64 {
	var n int64
	for _, m := range r.members {
		if m.ProjectID == projectID && m.Role == domain.RoleAdmin && !r.inactive[m.UserID] {
			n++
		}
	}
	return n
}

func (r *stubProjectMemberRepo) CountActiveAdmins(_ context.Context, projectID uint) (int64, error) {
	return r.countActiveAdmins(projectID), nil
}

func (r *stubProjectMemberRepo) ChangeRole(_ context.Context, projectID, userID uint, role string) error {
	for _, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			before := r.countActiveAdmins(projectID)
			old := m.Role
			m.Role = role
			if before > 0 && r.countActiveAdmins(projectID) == 0 {
				m.Role = old
				return domain.ErrLastProjectAdmin
			}
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubProjectMemberRepo) Remove(_ context.Context, projectID, userID uint) error {
	for i, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			before := r.countActiveAdmins(projectID)
			kept := append(append([]*domain.ProjectMember{}, r.members[:i]...), r.members[i+1:]...)
			saved := r.members
			r.members = kept
			if before > 0 && r.countActiveAdmins(projectID) == 0 {
				r.members = saved
				return domain.ErrLastProjectAdmin
			}
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubProjectMemberRepo) AddAdminIfNone(_ context.Context, projectID, userID uint) (*domain.ProjectMember, error) {
	if r.countActiveAdmins(projectID) > 0 {
		return nil, domain.ErrConflict
	}
	for _, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			m.Role = domain.RoleAdmin
			return m, nil
		}
	}
	m := &domain.ProjectMember{ID: r.nextID, ProjectID: projectID, UserID: userID, Role: domain.RoleAdmin}
	r.nextID++
	r.members = append(r.members, m)
	return m, nil
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

func (s *stubIssueRepoForProject) Create(_ context.Context, _ *domain.Issue) error { return nil }
func (s *stubIssueRepoForProject) FindByID(_ context.Context, _ uint) (*domain.Issue, error) {
	return nil, domain.ErrNotFound
}
func (s *stubIssueRepoForProject) FindByProjectID(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) Update(_ context.Context, _ *domain.Issue) error   { return nil }
func (s *stubIssueRepoForProject) Delete(_ context.Context, _ uint) error            { return nil }
func (s *stubIssueRepoForProject) DeleteByProjectID(_ context.Context, _ uint) error { return nil }
func (s *stubIssueRepoForProject) FindBacklog(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) FindBySprint(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) CountBySprint(_ context.Context, _ uint) (int, int, error) {
	return 0, 0, nil
}
func (s *stubIssueRepoForProject) CountByStatusForSprint(_ context.Context, _ uint) ([]domain.StatusTotals, error) {
	return nil, nil
}

func (s *stubIssueRepoForProject) FindOpenBySprint(_ context.Context, _ uint, _ int) ([]*domain.Issue, error) {
	return nil, nil
}

func (s *stubIssueRepoForProject) FindIssueIDsByLabelID(_ context.Context, _ uint) ([]uint, error) {
	return nil, nil
}
func (s *stubIssueRepoForProject) FindByAssignee(_ context.Context, _ uint) ([]*domain.Issue, error) {
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// linked returns a project stub whose CreateWithAdmin records the admin in members.
func linked(members *stubProjectMemberRepo) *stubProjectRepo {
	r := newStubProjectRepo()
	r.members = members
	return r
}

func newProjectService() service.ProjectService {
	members := newStubProjectMemberRepo()
	return service.NewProjectService(
		linked(members),
		members,
		newStubUserRepo(),
		&stubIssueRepoForProject{},
		nil,
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

// FR-006: someone outside the project gets the same answer as for a project
// that does not exist, so keys cannot be guessed.
func TestProjectService_FindByKey_NonMemberSeesNotFound(t *testing.T) {
	svc := newProjectService()

	_, _ = svc.Create(context.Background(), 1, dto.CreateProjectRequest{Name: "My Project"})

	_, errHidden := svc.FindByKey(context.Background(), "mypr", 99)
	_, errMissing := svc.FindByKey(context.Background(), "nosuch", 99)

	if !errors.Is(errHidden, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for a non-member, got %v", errHidden)
	}
	if errHidden != errMissing {
		t.Errorf("hidden project and missing project must give the same error: %v vs %v", errHidden, errMissing)
	}
}

// rolesFixture is a project with one person in each role (ids 1 admin, 2 member, 3 viewer, 4 outsider).
func rolesFixture(t *testing.T) (service.ProjectService, *stubProjectMemberRepo) {
	t.Helper()
	ctx := context.Background()
	userRepo := newStubUserRepo()
	memberRepo := newStubProjectMemberRepo()
	svc := service.NewProjectService(linked(memberRepo), memberRepo, userRepo, &stubIssueRepoForProject{}, nil)
	for id, name := range map[uint]string{2: "ben", 3: "cy", 4: "dee", 5: "eve"} {
		_ = userRepo.Create(ctx, &domain.User{ID: id, Name: name, Email: name + "@example.com", Password: "x", IsActive: true})
	}
	if _, err := svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"}); err != nil {
		t.Fatal(err)
	}
	for _, m := range []struct {
		email string
		role  string
	}{{"ben@example.com", "member"}, {"cy@example.com", "viewer"}} {
		if err := svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: m.email, Role: m.role}); err != nil {
			t.Fatal(err)
		}
	}
	return svc, memberRepo
}

func TestProjectService_OnlyAdminsEditTheProjectAndManageMembers(t *testing.T) {
	svc, _ := rolesFixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		caller uint
		allow  bool
	}{{"admin", 1, true}, {"member", 2, false}, {"viewer", 3, false}} {
		_, errEdit := svc.Update(ctx, "mypr", tc.caller, dto.UpdateProjectRequest{Description: "x"})
		errAdd := svc.AddMember(ctx, "mypr", tc.caller, dto.AddMemberRequest{Email: "dee@example.com", Role: "member"})
		errRemove := svc.RemoveMember(ctx, "mypr", tc.caller, 4)
		for what, err := range map[string]error{"edit": errEdit, "add": errAdd} {
			if tc.allow && err != nil {
				t.Errorf("%s: %s should be allowed, got %v", tc.name, what, err)
			}
			if !tc.allow && !errors.Is(err, domain.ErrForbidden) {
				t.Errorf("%s: %s should be forbidden, got %v", tc.name, what, err)
			}
		}
		if !tc.allow && !errors.Is(errRemove, domain.ErrForbidden) {
			t.Errorf("%s: remove should be forbidden, got %v", tc.name, errRemove)
		}
		if tc.allow {
			_ = svc.RemoveMember(ctx, "mypr", tc.caller, 4) // tidy for the next case
		}
	}
}

func TestProjectService_ReadsCarryTheCallersRoleAndPermissions(t *testing.T) {
	svc, _ := rolesFixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		caller   uint
		role     string
		mustHave string
		mustLack string
	}{
		{1, "admin", "project.delete", ""},
		{2, "member", "task.write", "sprint.manage"},
		{3, "viewer", "project.view", "task.write"},
	} {
		p, err := svc.FindByKey(ctx, "mypr", tc.caller)
		if err != nil {
			t.Fatal(err)
		}
		if p.MyRole != tc.role {
			t.Errorf("caller %d: my_role = %q, want %q", tc.caller, p.MyRole, tc.role)
		}
		has := map[string]bool{}
		for _, c := range p.Permissions {
			has[c] = true
		}
		if !has[tc.mustHave] {
			t.Errorf("%s permissions %v should include %s", tc.role, p.Permissions, tc.mustHave)
		}
		if tc.mustLack != "" && has[tc.mustLack] {
			t.Errorf("%s permissions %v should not include %s", tc.role, p.Permissions, tc.mustLack)
		}
	}
}

// The creator holds no right except through their admin membership.
func TestProjectService_DemotedCreatorLosesAdminPowers(t *testing.T) {
	svc, memberRepo := rolesFixture(t)
	ctx := context.Background()
	for _, m := range memberRepo.members {
		if m.UserID == 1 {
			m.Role = "member"
		}
	}
	if _, err := svc.Update(ctx, "mypr", 1, dto.UpdateProjectRequest{Description: "x"}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a demoted creator must not edit the project, got %v", err)
	}
	if err := svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "dee@example.com", Role: "member"}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a demoted creator must not add members, got %v", err)
	}
}

func TestProjectService_AddMember_RejectsAnUnknownRole(t *testing.T) {
	svc, _ := rolesFixture(t)
	err := svc.AddMember(context.Background(), "mypr", 1, dto.AddMemberRequest{Email: "dee@example.com", Role: "owner"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for role owner, got %v", err)
	}
}

func TestProjectService_AddMember_ConflictIfAlreadyMember(t *testing.T) {
	userRepo := newStubUserRepo()
	memberRepo := newStubProjectMemberRepo()
	projectRepo := linked(memberRepo)
	svc := service.NewProjectService(projectRepo, memberRepo, userRepo, &stubIssueRepoForProject{}, nil)

	// Register a target user
	ctx := context.Background()
	_ = userRepo.Create(ctx, &domain.User{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "x", IsActive: true})
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

func TestProjectService_RemoveMember_AdminCannotRemoveThemselves(t *testing.T) {
	svc := newProjectService()
	ctx := context.Background()

	resp, _ := svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})

	err := svc.RemoveMember(ctx, resp.Key, 1, 1)

	if !errors.Is(err, domain.ErrSelfRemoval) {
		t.Errorf("expected ErrSelfRemoval, got %v", err)
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

// ── notifications on membership changes (feature 017) ────────────────────────

func newNotifyingProjectService() (service.ProjectService, *recordingNotifier, *stubUserRepo) {
	userRepo := newStubUserRepo()
	rec := &recordingNotifier{}
	members := newStubProjectMemberRepo()
	svc := service.NewProjectService(linked(members), members, userRepo, &stubIssueRepoForProject{}, rec)
	return svc, rec, userRepo
}

func TestProjectService_AddAndRemoveMember_Notify(t *testing.T) {
	svc, rec, userRepo := newNotifyingProjectService()
	ctx := context.Background()
	_ = userRepo.Create(ctx, &domain.User{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "x", IsActive: true})
	proj, _ := svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})

	if err := svc.AddMember(ctx, proj.Key, 1, dto.AddMemberRequest{Email: "bob@example.com", Role: "member"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(rec.memberships) != 1 || !rec.memberships[0].added || rec.memberships[0].targetID != 2 || rec.memberships[0].actorID != 1 || rec.memberships[0].projectID != proj.ID {
		t.Fatalf("unexpected add notification: %+v", rec.memberships)
	}

	if err := svc.RemoveMember(ctx, proj.Key, 1, 2); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(rec.memberships) != 2 || rec.memberships[1].added || rec.memberships[1].targetID != 2 {
		t.Fatalf("unexpected remove notification: %+v", rec.memberships)
	}
}

func TestProjectService_FailedMembershipChangesDoNotNotify(t *testing.T) {
	svc, rec, userRepo := newNotifyingProjectService()
	ctx := context.Background()
	_ = userRepo.Create(ctx, &domain.User{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "x", IsActive: true})
	proj, _ := svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})

	// forbidden: caller 9 is not a member
	_ = svc.AddMember(ctx, proj.Key, 9, dto.AddMemberRequest{Email: "bob@example.com", Role: "member"})
	// owner removal refused
	_ = svc.RemoveMember(ctx, proj.Key, 1, 1)
	// conflict
	_ = svc.AddMember(ctx, proj.Key, 1, dto.AddMemberRequest{Email: "bob@example.com", Role: "member"})
	rec.memberships = nil
	_ = svc.AddMember(ctx, proj.Key, 1, dto.AddMemberRequest{Email: "bob@example.com", Role: "member"})

	if len(rec.memberships) != 0 {
		t.Fatalf("expected no notifications for failed changes, got %+v", rec.memberships)
	}
}
