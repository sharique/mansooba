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
	Delete(ctx context.Context, key string, callerID uint) error
	ListMembers(ctx context.Context, key string, callerID uint) ([]*dto.MemberResponse, error)
	AddMember(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error
	RemoveMember(ctx context.Context, key string, callerID uint, targetUserID uint) error
}

type projectService struct {
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
	userRepo    domain.UserRepository
	issueRepo   domain.IssueRepository
}

// NewProjectService returns a ProjectService backed by the given repositories.
func NewProjectService(
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	userRepo domain.UserRepository,
	issueRepo domain.IssueRepository,
) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		userRepo:    userRepo,
		issueRepo:   issueRepo,
	}
}

func (s *projectService) List(ctx context.Context, callerID uint) ([]*dto.ProjectResponse, error) {
	memberships, err := s.memberRepo.FindByUserID(ctx, callerID)
	if err != nil {
		return nil, err
	}
	var result []*dto.ProjectResponse
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
	key = strings.ToUpper(key)

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

	project := &domain.Project{Key: key, Name: req.Name, Description: req.Description, OwnerID: callerID}
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	// Owner is also an admin member so they appear in member lists.
	_ = s.memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: callerID, Role: "admin"})

	return toProjectResponse(project), nil
}

func (s *projectService) FindByKey(ctx context.Context, key string, callerID uint) (*dto.ProjectResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return nil, err
	}
	return toProjectResponse(project), nil
}

func (s *projectService) Update(ctx context.Context, key string, callerID uint, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	membership, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, callerID)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	if membership.Role != "admin" && project.OwnerID != callerID {
		return nil, domain.ErrForbidden
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
	return toProjectResponse(project), nil
}

func (s *projectService) Delete(ctx context.Context, key string, callerID uint) error {
	project, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	if project.OwnerID != callerID {
		return domain.ErrForbidden
	}
	// Cascade: issues → members → project.
	if err := s.issueRepo.DeleteByProjectID(ctx, project.ID); err != nil {
		return err
	}
	if err := s.memberRepo.DeleteByProjectID(ctx, project.ID); err != nil {
		return err
	}
	return s.projectRepo.Delete(ctx, project.ID)
}

func (s *projectService) ListMembers(ctx context.Context, key string, callerID uint) ([]*dto.MemberResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return nil, err
	}
	members, err := s.memberRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	var result []*dto.MemberResponse
	for _, m := range members {
		user, err := s.userRepo.FindByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		result = append(result, &dto.MemberResponse{
			UserID: m.UserID,
			Name:   user.Name,
			Email:  user.Email,
			Role:   m.Role,
		})
	}
	return result, nil
}

func (s *projectService) AddMember(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error {
	project, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	membership, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, callerID)
	if err != nil {
		return domain.ErrForbidden
	}
	if membership.Role != "admin" && project.OwnerID != callerID {
		return domain.ErrForbidden
	}

	target, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, target.ID); err == nil {
		return domain.ErrConflict
	}

	return s.memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: target.ID, Role: req.Role})
}

func (s *projectService) RemoveMember(ctx context.Context, key string, callerID uint, targetUserID uint) error {
	project, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	membership, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, callerID)
	if err != nil {
		return domain.ErrForbidden
	}
	if membership.Role != "admin" && project.OwnerID != callerID {
		return domain.ErrForbidden
	}
	if targetUserID == project.OwnerID {
		return domain.ErrForbidden
	}
	target, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, targetUserID)
	if err != nil {
		return err
	}
	return s.memberRepo.Delete(ctx, target.ID)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *projectService) requireMember(ctx context.Context, projectID, userID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

func generateKey(name string) string {
	var letters []rune
	for _, r := range strings.ToUpper(name) {
		if unicode.IsLetter(r) {
			letters = append(letters, r)
			if len(letters) == 4 {
				break
			}
		}
	}
	return string(letters)
}

func toProjectResponse(p *domain.Project) *dto.ProjectResponse {
	return &dto.ProjectResponse{
		ID:          p.ID,
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		OwnerID:     p.OwnerID,
	}
}

