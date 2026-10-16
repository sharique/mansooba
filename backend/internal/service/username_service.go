package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// maxUsernameAttempts bounds the search for a free numbered username.
const maxUsernameAttempts = 10000

// UsernameService owns the username rules that need the database: choosing a
// free name, checking availability and backfilling legacy accounts
// (feature 018, ADR-038). The pure rules live in domain.
type UsernameService interface {
	// Resolve returns the username to store for a new account: the supplied one
	// (validated, normalised, must be free) or, when none is supplied, one
	// generated from the email. It returns a domain username error otherwise.
	Resolve(ctx context.Context, supplied, email string) (string, error)
	// Check reports whether a username could be used, for the live feedback on
	// the create-user forms. It never writes.
	Check(ctx context.Context, raw string) dto.UsernameCheckResponse
	// CountMissing returns how many accounts have no username yet.
	CountMissing(ctx context.Context) (int, error)
	// BackfillMissing gives every account without a username a generated one
	// and returns how many it changed. Safe to run repeatedly.
	BackfillMissing(ctx context.Context) (int, error)
}

type usernameService struct {
	users domain.UserRepository
}

// NewUsernameService returns a UsernameService backed by the user repository.
func NewUsernameService(users domain.UserRepository) UsernameService {
	return &usernameService{users: users}
}

func (s *usernameService) Resolve(ctx context.Context, supplied, email string) (string, error) {
	return resolveUsername(ctx, s.users, supplied, email)
}

func (s *usernameService) Check(ctx context.Context, raw string) dto.UsernameCheckResponse {
	name := domain.NormalizeUsername(raw)
	if err := domain.ValidateUsername(name); err != nil {
		reason := "invalid"
		if errors.Is(err, domain.ErrUsernameReserved) {
			reason = "reserved"
		}
		return dto.UsernameCheckResponse{Reason: reason, Message: err.Error()}
	}
	if _, err := s.users.FindByUsername(ctx, name); err == nil {
		return dto.UsernameCheckResponse{Reason: "taken", Message: domain.ErrUsernameTaken.Error()}
	}
	return dto.UsernameCheckResponse{Available: true}
}

func (s *usernameService) CountMissing(ctx context.Context) (int, error) {
	missing, err := s.users.ListMissingUsername(ctx)
	if err != nil {
		return 0, err
	}
	return len(missing), nil
}

func (s *usernameService) BackfillMissing(ctx context.Context) (int, error) {
	missing, err := s.users.ListMissingUsername(ctx)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, u := range missing {
		if err := assignGeneratedUsername(ctx, s.users, u); err != nil {
			return changed, fmt.Errorf("username for account %d: %w", u.ID, err)
		}
		changed++
	}
	return changed, nil
}

// resolveUsername is shared by account creation and the UsernameService.
func resolveUsername(ctx context.Context, users domain.UserRepository, supplied, email string) (string, error) {
	if domain.NormalizeUsername(supplied) != "" {
		name := domain.NormalizeUsername(supplied)
		if err := domain.ValidateUsername(name); err != nil {
			return "", err
		}
		if _, err := users.FindByUsername(ctx, name); err == nil {
			return "", domain.ErrUsernameTaken
		} else if !errors.Is(err, domain.ErrNotFound) {
			return "", err
		}
		return name, nil
	}
	return uniqueUsername(ctx, users, domain.DeriveUsername(email), nil)
}

// uniqueUsername returns base, or base followed by 2, 3, ... (cut so the whole
// name stays within the length limit) until a free, non-reserved name is found.
// Names in skip are treated as taken (names that lost a race).
func uniqueUsername(ctx context.Context, users domain.UserRepository, base string, skip map[string]bool) (string, error) {
	for n := 1; n <= maxUsernameAttempts; n++ {
		candidate := base
		if n > 1 {
			suffix := strconv.Itoa(n)
			stem := base
			if len(stem)+len(suffix) > domain.UsernameMaxLen {
				stem = stem[:domain.UsernameMaxLen-len(suffix)]
			}
			candidate = stem + suffix
		}
		if domain.ReservedUsernames[candidate] || skip[candidate] {
			continue
		}
		if _, err := users.FindByUsername(ctx, candidate); errors.Is(err, domain.ErrNotFound) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("no free username found for %q", base)
}

// assignGeneratedUsername gives u a generated username, retrying with the next
// number when another writer takes the chosen name between the check and the
// write.
func assignGeneratedUsername(ctx context.Context, users domain.UserRepository, u *domain.User) error {
	base := domain.DeriveUsername(u.Email)
	skip := map[string]bool{}
	for attempt := 0; attempt < 50; attempt++ {
		name, err := uniqueUsername(ctx, users, base, skip)
		if err != nil {
			return err
		}
		err = users.SetUsername(ctx, u.ID, name)
		if err == nil {
			return nil
		}
		if !errors.Is(err, domain.ErrUsernameTaken) {
			return err
		}
		skip[name] = true // lost the race for this name; look for the next free one
	}
	return fmt.Errorf("could not settle on a username for %q", u.Email)
}

// usernameValue returns the username, or "" for a nil user.
func usernameValue(u *domain.User) string {
	if u == nil {
		return ""
	}
	return u.Username
}
