package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
)

// stubAdminUserRepo is a controllable stand-in for domain.UserRepository used
// by admin user service tests. Only methods relevant to admin operations are wired.
type stubAdminUserRepo struct {
	users             []*domain.User
	listAllFn         func(page, size int) ([]*domain.User, int64, error)
	countAdminsFn     func() (int64, error)
	updateAdminFn     func(u *domain.User) error
	findByIDFn        func(id uint) (*domain.User, error)
}

func (r *stubAdminUserRepo) Create(_ context.Context, _ *domain.User) error { return nil }
func (r *stubAdminUserRepo) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *stubAdminUserRepo) FindByEmailPrefix(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *stubAdminUserRepo) FindByID(_ context.Context, id uint) (*domain.User, error) {
	if r.findByIDFn != nil {
		return r.findByIDFn(id)
	}
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *stubAdminUserRepo) Update(_ context.Context, u *domain.User) error {
	for i, existing := range r.users {
		if existing.ID == u.ID {
			cp := *u
			r.users[i] = &cp
			return nil
		}
	}
	return domain.ErrNotFound
}
func (r *stubAdminUserRepo) HasAdmin(_ context.Context) (bool, error)          { return false, nil }
func (r *stubAdminUserRepo) FindFirstAdmin(_ context.Context) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *stubAdminUserRepo) ListAll(_ context.Context, page, size int) ([]*domain.User, int64, error) {
	if r.listAllFn != nil {
		return r.listAllFn(page, size)
	}
	return nil, 0, nil
}
func (r *stubAdminUserRepo) CountActiveAdmins(_ context.Context) (int64, error) {
	if r.countAdminsFn != nil {
		return r.countAdminsFn()
	}
	return 0, nil
}
func (r *stubAdminUserRepo) UpdateAdminFields(_ context.Context, u *domain.User) error {
	if r.updateAdminFn != nil {
		return r.updateAdminFn(u)
	}
	for i, existing := range r.users {
		if existing.ID == u.ID {
			cp := *u
			r.users[i] = &cp
			return nil
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// T025: AdminUserService tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAdminUserService_ListUsers_MapsToDTO(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	repo := &stubAdminUserRepo{
		listAllFn: func(page, size int) ([]*domain.User, int64, error) {
			return []*domain.User{
				{ID: 1, Name: "Alice", Email: "alice@test.com", IsAdmin: true, IsActive: true, CreatedAt: now},
				{ID: 2, Name: "Bob", Email: "bob@test.com", IsAdmin: false, IsActive: true, CreatedAt: now},
			}, 2, nil
		},
	}
	svc := service.NewAdminUserService(repo)

	resp, err := svc.ListUsers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
	if len(resp.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Users))
	}
	if resp.Users[0].Name != "Alice" || !resp.Users[0].IsAdmin {
		t.Error("first user DTO is wrong")
	}
	if resp.Users[1].Email != "bob@test.com" || resp.Users[1].IsAdmin {
		t.Error("second user DTO is wrong")
	}
}

func TestAdminUserService_ListUsers_PassesPaginationThrough(t *testing.T) {
	var capturedPage, capturedSize int
	repo := &stubAdminUserRepo{
		listAllFn: func(page, size int) ([]*domain.User, int64, error) {
			capturedPage = page
			capturedSize = size
			return nil, 0, nil
		},
	}
	svc := service.NewAdminUserService(repo)

	if _, err := svc.ListUsers(context.Background(), 3, 50); err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if capturedPage != 3 || capturedSize != 50 {
		t.Errorf("expected page=3 size=50, got page=%d size=%d", capturedPage, capturedSize)
	}
}

func TestAdminUserService_ListUsers_EmptyResult(t *testing.T) {
	repo := &stubAdminUserRepo{
		listAllFn: func(_, _ int) ([]*domain.User, int64, error) {
			return nil, 0, nil
		},
	}
	svc := service.NewAdminUserService(repo)

	resp, err := svc.ListUsers(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(resp.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp.Users))
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// T035: SetRole and SetActive tests
// ──────────────────────────────────────────────────────────────────────────────

func makeUser(id uint, name string, isAdmin, isActive bool) *domain.User {
	return &domain.User{ID: id, Name: name, Email: name + "@test.com", IsAdmin: isAdmin, IsActive: isActive}
}

func TestAdminUserService_SetRole_PromotesUser(t *testing.T) {
	target := makeUser(2, "bob", false, true)
	repo := &stubAdminUserRepo{
		users:         []*domain.User{makeUser(1, "admin", true, true), target},
		countAdminsFn: func() (int64, error) { return 1, nil }, // 1 active admin
	}
	svc := service.NewAdminUserService(repo)

	if err := svc.SetRole(context.Background(), 1, 2, true); err != nil {
		t.Fatalf("SetRole promote: %v", err)
	}
	reloaded, _ := repo.FindByID(context.Background(), 2)
	if !reloaded.IsAdmin {
		t.Error("expected target to be promoted to admin")
	}
}

func TestAdminUserService_SetRole_DemotesNonLastAdmin(t *testing.T) {
	repo := &stubAdminUserRepo{
		users:         []*domain.User{makeUser(1, "admin1", true, true), makeUser(2, "admin2", true, true)},
		countAdminsFn: func() (int64, error) { return 2, nil }, // 2 active admins → safe to demote
	}
	svc := service.NewAdminUserService(repo)

	if err := svc.SetRole(context.Background(), 1, 2, false); err != nil {
		t.Fatalf("SetRole demote: %v", err)
	}
}

func TestAdminUserService_SetRole_DemotesLastAdmin_ReturnsErrLastAdmin(t *testing.T) {
	repo := &stubAdminUserRepo{
		users:         []*domain.User{makeUser(1, "admin", true, true)},
		countAdminsFn: func() (int64, error) { return 1, nil }, // only 1 admin
	}
	svc := service.NewAdminUserService(repo)

	err := svc.SetRole(context.Background(), 1, 1, false)
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Errorf("expected ErrLastAdmin, got %v", err)
	}
}

func TestAdminUserService_SetActive_DisablesUser(t *testing.T) {
	repo := &stubAdminUserRepo{
		users:         []*domain.User{makeUser(1, "admin", true, true), makeUser(2, "user", false, true)},
		countAdminsFn: func() (int64, error) { return 1, nil },
	}
	svc := service.NewAdminUserService(repo)

	if err := svc.SetActive(context.Background(), 1, 2, false); err != nil {
		t.Fatalf("SetActive disable: %v", err)
	}
	reloaded, _ := repo.FindByID(context.Background(), 2)
	if reloaded.IsActive {
		t.Error("expected target to be disabled")
	}
}

func TestAdminUserService_SetActive_DisablesLastAdmin_ReturnsErrLastAdmin(t *testing.T) {
	repo := &stubAdminUserRepo{
		users:         []*domain.User{makeUser(1, "admin", true, true)},
		countAdminsFn: func() (int64, error) { return 1, nil },
	}
	svc := service.NewAdminUserService(repo)

	err := svc.SetActive(context.Background(), 1, 1, false)
	if !errors.Is(err, domain.ErrLastAdmin) {
		t.Errorf("expected ErrLastAdmin, got %v", err)
	}
}

func TestAdminUserService_SetActive_ReEnablesDisabledUser(t *testing.T) {
	repo := &stubAdminUserRepo{
		users:         []*domain.User{makeUser(1, "admin", true, true), makeUser(2, "disabled", false, false)},
		countAdminsFn: func() (int64, error) { return 1, nil },
	}
	svc := service.NewAdminUserService(repo)

	if err := svc.SetActive(context.Background(), 1, 2, true); err != nil {
		t.Fatalf("SetActive re-enable: %v", err)
	}
	reloaded, _ := repo.FindByID(context.Background(), 2)
	if !reloaded.IsActive {
		t.Error("expected target to be re-enabled")
	}
}

func TestAdminUserService_SetActive_TargetNotFound(t *testing.T) {
	repo := &stubAdminUserRepo{users: []*domain.User{makeUser(1, "admin", true, true)}}
	svc := service.NewAdminUserService(repo)

	err := svc.SetActive(context.Background(), 1, 99, false)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
