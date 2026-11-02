package service

import (
	"context"
	"errors"
	"strings"

	"github.com/sharique/mansooba/internal/domain"
)

// Access is the one place that decides whether someone may do something. It
// finds the membership (or the account) and asks the rule table in
// domain/access.go; no other code compares a role name or the admin flag.
//
// Refusals:
//   - not a member, or no such project: domain.ErrNotFound (the same answer for
//     both, so project keys cannot be guessed);
//   - a member whose role lacks the capability: *domain.AccessDeniedError
//     (errors.Is(err, domain.ErrForbidden)), with a plain-language Reason().
//
// Roles are read on every call and never cached, so a role change applies on
// the person's next request.
type Access interface {
	// RequireProject checks that userID holds cap in the project and returns the
	// membership so callers can apply authorship rules on top.
	RequireProject(ctx context.Context, projectID, userID uint, cap domain.Capability) (*domain.ProjectMember, error)
	// RequireProjectByKey does the same for a project given by key (any case).
	RequireProjectByKey(ctx context.Context, key string, userID uint, cap domain.Capability) (*domain.Project, *domain.ProjectMember, error)
	// RequireSystem checks that the account is an active system administrator
	// holding cap. Anything else is domain.ErrForbidden.
	RequireSystem(ctx context.Context, userID uint, cap domain.SystemCapability) error
}

type access struct {
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
	userRepo    domain.UserRepository
}

// NewAccess builds the access checker. Any repository the caller will not need
// may be nil (a service that only checks membership by project id needs just
// the member repository).
func NewAccess(projectRepo domain.ProjectRepository, memberRepo domain.ProjectMemberRepository, userRepo domain.UserRepository) Access {
	return &access{projectRepo: projectRepo, memberRepo: memberRepo, userRepo: userRepo}
}

func (a *access) RequireProject(ctx context.Context, projectID, userID uint, cap domain.Capability) (*domain.ProjectMember, error) {
	m, err := a.memberRepo.FindByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if !domain.RoleCan(m.Role, cap) {
		return nil, &domain.AccessDeniedError{Capability: cap, Role: m.Role}
	}
	return m, nil
}

func (a *access) RequireProjectByKey(ctx context.Context, key string, userID uint, cap domain.Capability) (*domain.Project, *domain.ProjectMember, error) {
	// Keys are stored in lower case; accept them in any case, but look the key
	// up as given first so a store that kept another case still resolves.
	p, err := a.projectRepo.FindByKey(ctx, key)
	if errors.Is(err, domain.ErrNotFound) && strings.ToLower(key) != key {
		p, err = a.projectRepo.FindByKey(ctx, strings.ToLower(key))
	}
	if err != nil {
		return nil, nil, err
	}
	m, err := a.RequireProject(ctx, p.ID, userID, cap)
	if err != nil {
		return nil, nil, err
	}
	return p, m, nil
}

func (a *access) RequireSystem(ctx context.Context, userID uint, cap domain.SystemCapability) error {
	u, err := a.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	if !domain.SystemAdminCan(u.IsAdmin, u.IsActive, cap) {
		return domain.ErrForbidden
	}
	return nil
}
