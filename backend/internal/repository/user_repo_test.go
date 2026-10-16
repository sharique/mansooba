package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.ProjectMember{}, &domain.Issue{}, &domain.Sprint{}, &domain.Attachment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// T023 helpers
func createUser(t *testing.T, repo domain.UserRepository, email string, isAdmin, isActive bool) *domain.User {
	t.Helper()
	u := &domain.User{Name: email, Email: email, Username: email, Password: "hash", IsAdmin: isAdmin, IsActive: isActive}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return u
}

// T023: ListAll tests

func TestUserRepository_ListAll_ReturnsPaginatedResults(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = createUser(t, repo, fmt.Sprintf("u%d@test.com", i), false, true)
	}

	// Page 1, size 3 → 3 users
	users, total, err := repo.ListAll(ctx, 1, 3)
	if err != nil {
		t.Fatalf("ListAll p1: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users on page 1, got %d", len(users))
	}

	// Page 2, size 3 → 2 users (remainder)
	users2, total2, err := repo.ListAll(ctx, 2, 3)
	if err != nil {
		t.Fatalf("ListAll p2: %v", err)
	}
	if total2 != 5 {
		t.Errorf("expected total=5 on page 2, got %d", total2)
	}
	if len(users2) != 2 {
		t.Errorf("expected 2 users on page 2, got %d", len(users2))
	}
}

func TestUserRepository_ListAll_OutOfRangePage_ReturnsEmptySlice(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	_ = createUser(t, repo, "only@test.com", false, true)

	users, total, err := repo.ListAll(context.Background(), 99, 20)
	if err != nil {
		t.Fatalf("ListAll out-of-range: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users on out-of-range page, got %d", len(users))
	}
}

func TestUserRepository_CountActiveAdmins(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// Initially 0 admins
	n, err := repo.CountActiveAdmins(ctx)
	if err != nil {
		t.Fatalf("CountActiveAdmins: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 active admins initially, got %d", n)
	}

	// Create all users as active admins first, then disable one via UpdateAdminFields.
	// This avoids GORM's zero-value bool handling at Create time.
	admin := createUser(t, repo, "admin@test.com", true, true)
	_ = createUser(t, repo, "member@test.com", false, true)
	inactiveAdmin := createUser(t, repo, "inactive_admin@test.com", true, true)

	// Disable the third user via UpdateAdminFields (explicit column write)
	inactiveAdmin.IsActive = false
	if err := repo.UpdateAdminFields(ctx, inactiveAdmin); err != nil {
		t.Fatalf("UpdateAdminFields disable: %v", err)
	}

	n, err = repo.CountActiveAdmins(ctx)
	if err != nil {
		t.Fatalf("CountActiveAdmins after seed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 active admin, got %d", n)
	}

	// Add a second active admin
	_ = admin // suppress unused warning
	_ = createUser(t, repo, "admin2@test.com", true, true)

	n, _ = repo.CountActiveAdmins(ctx)
	if n != 2 {
		t.Errorf("expected 2 active admins after adding second, got %d", n)
	}
}

func TestUserRepository_UpdateAdminFields_OnlyWritesAdminAndActive(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	u := createUser(t, repo, "user@test.com", false, true)
	originalName := u.Name

	u.IsAdmin = true
	if err := repo.UpdateAdminFields(ctx, u); err != nil {
		t.Fatalf("UpdateAdminFields: %v", err)
	}

	reloaded, _ := repo.FindByID(ctx, u.ID)
	if !reloaded.IsAdmin {
		t.Error("expected IsAdmin=true after UpdateAdminFields")
	}
	if reloaded.Name != originalName {
		t.Errorf("UpdateAdminFields must not change Name: got %q", reloaded.Name)
	}
}

func TestUserRepository_Create_Succeeds(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	user := &domain.User{Name: "Alice", Email: "alice@example.com", Username: "alice", Password: "hash"}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	created := &domain.User{Name: "Bob", Email: "bob@example.com", Username: "bob", Password: "hash"}
	_ = repo.Create(context.Background(), created)

	tests := []struct {
		name    string
		id      uint
		wantErr error
	}{
		{"found", created.ID, nil},
		{"not found", 9999, domain.ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := repo.FindByID(context.Background(), tc.id)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("want %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u.Email != created.Email {
				t.Fatalf("want email %q, got %q", created.Email, u.Email)
			}
		})
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	_ = repo.Create(context.Background(), &domain.User{Name: "Carol", Email: "carol@example.com", Username: "carol", Password: "hash"})

	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"found", "carol@example.com", nil},
		{"not found", "nobody@example.com", domain.ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.FindByEmail(context.Background(), tc.email)
			if err != tc.wantErr {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

// ── usernames (feature 018, story 2) ─────────────────────────────────────────

func TestUserRepository_FindByUsername(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	ctx := context.Background()
	u := &domain.User{Name: "Sam", Email: "sam@x.com", Password: "h", IsActive: true, Username: "sam"}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByUsername(ctx, "sam")
	if err != nil || got.ID != u.ID {
		t.Fatalf("want user %d, got %v err %v", u.ID, got, err)
	}
	if _, err := repo.FindByUsername(ctx, "nobody"); err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// The column is NOT NULL and unique: an account without a username cannot be
// created at all, and a duplicate comes back as the domain error.
func TestUserRepository_UsernameIsRequiredAndUnique(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, &domain.User{Name: "a", Email: "a@x.com", Password: "h"}); err != domain.ErrUsernameInvalid {
		t.Fatalf("an account without a username must be refused, got %v", err)
	}
	if err := repo.Create(ctx, &domain.User{Name: "1", Email: "1@x.com", Password: "h", Username: "sam"}); err != nil {
		t.Fatal(err)
	}
	err := repo.Create(ctx, &domain.User{Name: "2", Email: "2@x.com", Password: "h", Username: "sam"})
	if err != domain.ErrUsernameTaken {
		t.Fatalf("want ErrUsernameTaken, got %v", err)
	}
}

func TestUserRepository_SetUsernameAndListMissing(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()
	a := createUser(t, repo, "a@x.com", false, true)
	c := &domain.User{Name: "c", Email: "c@x.com", Password: "h", IsActive: true, Username: "cee"}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	// A row as it looks mid-upgrade: no username yet (written directly, since
	// Create refuses it).
	if err := db.Exec(`UPDATE users SET username = '' WHERE id = ?`, a.ID).Error; err != nil {
		t.Fatal(err)
	}

	missing, err := repo.ListMissingUsername(ctx)
	if err != nil || len(missing) != 1 || missing[0].ID != a.ID {
		t.Fatalf("want only a, got %v err %v", missing, err)
	}
	if err := repo.SetUsername(ctx, a.ID, "aaa"); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.FindByID(ctx, a.ID)
	if got.Username != "aaa" {
		t.Fatalf("username not stored: %q", got.Username)
	}
	if err := repo.SetUsername(ctx, c.ID, "aaa"); err != domain.ErrUsernameTaken {
		t.Fatalf("want ErrUsernameTaken, got %v", err)
	}
	if missing, _ = repo.ListMissingUsername(ctx); len(missing) != 0 {
		t.Fatalf("want none missing, got %v", missing)
	}
}

// Update persists the account's other fields (password change and reset rely on
// it) but must never write the username, even from a stale or altered struct
// (feature 018, SC-006).
func TestUserRepository_UpdateNeverChangesUsername(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	ctx := context.Background()
	u := &domain.User{Name: "Sam", Email: "sam@x.com", Password: "old", IsActive: true, Username: "sam"}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	u.Username = "hijacked"
	u.Name = "Renamed"
	u.Password = "new-hash"
	if err := repo.Update(ctx, u); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.FindByID(ctx, u.ID)
	if got.Username != "sam" {
		t.Fatalf("username changed to %q", got.Username)
	}
	if got.Name != "Renamed" || got.Password != "new-hash" {
		t.Fatalf("other fields were not persisted: %+v", got)
	}
}
