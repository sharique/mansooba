package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// ProjectService defines the projects business-logic contract.
type ProjectService interface {
	List(ctx context.Context, callerID uint) ([]*dto.ProjectResponse, error)
	Create(ctx context.Context, callerID uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	FindByKey(ctx context.Context, key string, callerID uint) (*dto.ProjectResponse, error)
	Update(ctx context.Context, key string, callerID uint, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	// Delete removes a project. It needs the delete capability, the exact project
	// name as confirmation, and a project with no tasks left.
	Delete(ctx context.Context, key string, callerID uint, confirmName string) error
	ListMembers(ctx context.Context, key string, callerID uint) ([]*dto.MemberResponse, error)
	AddMember(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error
	// ChangeMemberRole sets another member's role (or the caller's own, while
	// another active admin remains).
	ChangeMemberRole(ctx context.Context, key string, callerID, targetUserID uint, role string) (*dto.MemberResponse, error)
	// RemoveMember removes someone else from the project; an admin cannot remove themselves.
	RemoveMember(ctx context.Context, key string, callerID uint, targetUserID uint) error
	// ListWithoutAdmin lists the projects that have no active admin (system administrators only).
	ListWithoutAdmin(ctx context.Context, callerID uint) ([]*dto.AbandonedProjectResponse, error)
	// ClaimAdmin makes the calling system administrator the admin of a project that has none.
	ClaimAdmin(ctx context.Context, key string, callerID uint) (*dto.MemberResponse, error)
}

type projectService struct {
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
	userRepo    domain.UserRepository
	issueRepo   domain.IssueRepository
	notifier    NotificationService // optional (nil: no notifications)
	access      Access

	// sysLog receives the audit entries (nil: none are written).
	sysLog SystemLogService
}

// NewProjectService returns a ProjectService backed by the given repositories.
func NewProjectService(
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	userRepo domain.UserRepository,
	issueRepo domain.IssueRepository,
	notifier NotificationService,
) *projectService {
	return &projectService{
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		issueRepo:   issueRepo,
		notifier:    notifier,
		access:      NewAccess(projectRepo, memberRepo, userRepo),
	}
}

// WithSystemLog sends role, membership and deletion changes to the system log.
func (s *projectService) WithSystemLog(l SystemLogService) *projectService {
	s.sysLog = l
	return s
}

func (s *projectService) List(ctx context.Context, callerID uint) ([]*dto.ProjectResponse, error) {
	memberships, err := s.memberRepo.FindByUserID(ctx, callerID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.ProjectResponse, 0)
	for _, m := range memberships {
		p, err := s.projectRepo.FindByID(ctx, m.ProjectID)
		if err != nil {
			continue
		}
		result = append(result, toProjectResponse(p))
	}
	return result, nil
}

func (s *projectService) Create(ctx context.Context, callerID uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	key := req.Key
	if key == "" {
		key = generateKey(req.Name)
	}
	key = strings.ToLower(key)

	// Resolve key conflict by appending a digit suffix.
	if _, err := s.projectRepo.FindByKey(ctx, key); err == nil {
		for i := 2; i <= 99; i++ {
			candidate := fmt.Sprintf("%s%d", key, i)
			if _, err := s.projectRepo.FindByKey(ctx, candidate); err != nil {
				key = candidate
				break
			}
		}
	}

	// The creator is the project's first admin, saved together with the project
	// so a project never exists without them. (owner_id only records who made it.)
	project := &domain.Project{Key: key, Name: req.Name, Description: req.Description, OwnerID: callerID}
	if err := s.projectRepo.CreateWithAdmin(ctx, project, callerID); err != nil {
		return nil, err
	}

	return withMyAccess(toProjectResponse(project), domain.RoleAdmin), nil
}

func (s *projectService) FindByKey(ctx context.Context, key string, callerID uint) (*dto.ProjectResponse, error) {
	project, m, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapProjectView)
	if err != nil {
		return nil, err
	}
	return withMyAccess(toProjectResponse(project), m.Role), nil
}

func (s *projectService) Update(ctx context.Context, key string, callerID uint, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	project, m, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapProjectEdit)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}
	return withMyAccess(toProjectResponse(project), m.Role), nil
}

func (s *projectService) Delete(ctx context.Context, key string, callerID uint, confirmName string) error {
	project, _, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapProjectDelete)
	if err != nil {
		return err
	}
	if strings.TrimSpace(confirmName) != project.Name {
		return domain.NewInputError("type the project name exactly to confirm deleting it")
	}
	// The count, the cleanup and the delete happen together under the project lock.
	if err := s.projectRepo.DeleteIfEmpty(ctx, project.ID); err != nil {
		return err
	}
	s.audit(ctx, "project_deleted", callerID, 0, project.Key, "")
	return nil
}

func (s *projectService) ListMembers(ctx context.Context, key string, callerID uint) ([]*dto.MemberResponse, error) {
	project, _, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapProjectView)
	if err != nil {
		return nil, err
	}
	members, err := s.memberRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.MemberResponse, 0)
	for _, m := range members {
		user, err := s.userRepo.FindByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		result = append(result, memberResponse(m, user))
	}
	return result, nil
}

func memberResponse(m *domain.ProjectMember, user *domain.User) *dto.MemberResponse {
	return &dto.MemberResponse{
		UserID:   m.UserID,
		Name:     user.Name,
		Email:    user.Email,
		Username: usernameValue(user),
		Role:     m.Role,
		IsActive: user.IsActive,
	}
}

func (s *projectService) AddMember(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error {
	project, _, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapMemberManage)
	if err != nil {
		return err
	}
	if !domain.IsValidRole(req.Role) {
		return domain.NewInputError("role must be one of: admin, member, viewer")
	}

	target, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if !target.IsActive {
		return domain.NewInputError("that account is disabled")
	}
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, target.ID); err == nil {
		return domain.ErrAlreadyMember
	}

	// A second add of the same person is also refused by the database (unique
	// index), which covers two simultaneous requests.
	if err := s.memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: target.ID, Role: req.Role}); err != nil {
		return err
	}
	s.audit(ctx, "project_member_added", callerID, target.ID, project.Key, "role: "+req.Role)
	if s.notifier != nil {
		s.notifier.NotifyMembershipChange(ctx, project.ID, callerID, target.ID, true)
	}
	return nil
}

func (s *projectService) ChangeMemberRole(ctx context.Context, key string, callerID, targetUserID uint, role string) (*dto.MemberResponse, error) {
	project, _, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapMemberManage)
	if err != nil {
		return nil, err
	}
	if !domain.IsValidRole(role) {
		return nil, domain.NewInputError("role must be one of: admin, member, viewer")
	}
	current, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, targetUserID)
	if err != nil {
		return nil, err
	}
	oldRole := current.Role
	if oldRole != role {
		if err := s.memberRepo.ChangeRole(ctx, project.ID, targetUserID, role); err != nil {
			return nil, err
		}
		s.audit(ctx, "project_role_changed", callerID, targetUserID, project.Key, fmt.Sprintf("role: %s -> %s", oldRole, role))
		if s.notifier != nil {
			s.notifier.NotifyRoleChange(ctx, project.ID, callerID, targetUserID)
		}
	}
	user, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	return memberResponse(&domain.ProjectMember{ProjectID: project.ID, UserID: targetUserID, Role: role}, user), nil
}

func (s *projectService) RemoveMember(ctx context.Context, key string, callerID uint, targetUserID uint) error {
	project, _, err := s.access.RequireProjectByKey(ctx, key, callerID, domain.CapMemberManage)
	if err != nil {
		return err
	}
	if targetUserID == callerID {
		return domain.ErrSelfRemoval
	}
	if err := s.memberRepo.Remove(ctx, project.ID, targetUserID); err != nil {
		return err
	}
	s.audit(ctx, "project_member_removed", callerID, targetUserID, project.Key, "")
	if s.notifier != nil {
		s.notifier.NotifyMembershipChange(ctx, project.ID, callerID, targetUserID, false)
	}
	return nil
}

func (s *projectService) ListWithoutAdmin(ctx context.Context, callerID uint) ([]*dto.AbandonedProjectResponse, error) {
	if err := s.access.RequireSystem(ctx, callerID, domain.SysRescueProject); err != nil {
		return nil, err
	}
	projects, err := s.projectRepo.FindWithoutActiveAdmin(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.AbandonedProjectResponse, 0, len(projects))
	for _, p := range projects {
		members, err := s.memberRepo.FindByProjectID(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		// Only what is needed to find the project: no tasks, no member names.
		out = append(out, &dto.AbandonedProjectResponse{ID: p.ID, Key: strings.ToLower(p.Key), Name: p.Name, MemberCount: len(members)})
	}
	return out, nil
}

func (s *projectService) ClaimAdmin(ctx context.Context, key string, callerID uint) (*dto.MemberResponse, error) {
	if err := s.access.RequireSystem(ctx, callerID, domain.SysRescueProject); err != nil {
		return nil, err
	}
	// The caller is deliberately not a member, so the project is found directly.
	project, err := s.projectRepo.FindByKey(ctx, strings.ToLower(key))
	if err != nil {
		return nil, err
	}
	m, err := s.memberRepo.AddAdminIfNone(ctx, project.ID, callerID)
	if errors.Is(err, domain.ErrConflict) {
		return nil, domain.ErrProjectHasAdmin
	}
	if err != nil {
		return nil, err
	}
	s.audit(ctx, "project_admin_claimed", callerID, callerID, project.Key, "")
	if s.notifier != nil {
		s.notifier.NotifyAdminClaimed(ctx, project.ID, callerID)
	}
	user, err := s.userRepo.FindByID(ctx, callerID)
	if err != nil {
		return nil, err
	}
	return memberResponse(m, user), nil
}

// audit writes a project change to the system log (best effort, never fails the change).
func (s *projectService) audit(ctx context.Context, action string, actorID, targetID uint, projectKey, detail string) {
	if s.sysLog == nil {
		return
	}
	entry := domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAdminAction,
		Action:   action,
		Outcome:  "success",
		Actor:    fmt.Sprintf("user#%d", actorID),
		ActorID:  actorID,
		Target:   projectKey,
		TargetID: targetID,
		Detail:   detail,
	}
	if u, err := s.userRepo.FindByID(ctx, actorID); err == nil {
		entry.Actor = u.Email
	}
	if targetID != 0 {
		if u, err := s.userRepo.FindByID(ctx, targetID); err == nil {
			entry.Target = projectKey + " / " + u.Email
		}
	}
	s.sysLog.Record(ctx, entry)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func generateKey(name string) string {
	var letters []rune
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) {
			letters = append(letters, r)
			if len(letters) == 4 {
				break
			}
		}
	}
	return string(letters)
}

// withMyAccess adds the caller's role and the capabilities that role holds.
func withMyAccess(r *dto.ProjectResponse, role string) *dto.ProjectResponse {
	r.MyRole = role
	caps := domain.CapabilitiesOf(role)
	r.Permissions = make([]string, len(caps))
	for i, c := range caps {
		r.Permissions[i] = string(c)
	}
	return r
}

func toProjectResponse(p *domain.Project) *dto.ProjectResponse {
	return &dto.ProjectResponse{
		ID:          p.ID,
		Key:         strings.ToLower(p.Key),
		Name:        p.Name,
		Description: p.Description,
		OwnerID:     p.OwnerID,
	}
}
