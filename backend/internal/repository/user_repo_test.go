package repository_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.ProjectMember{}, &domain.Issue{}, &domain.Sprint{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUserRepository_Create_Succeeds(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	user := &domain.User{Name: "Alice", Email: "alice@example.com", Password: "hash"}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	repo := repository.NewUserRepository(newTestDB(t))
	created := &domain.User{Name: "Bob", Email: "bob@example.com", Password: "hash"}
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
	_ = repo.Create(context.Background(), &domain.User{Name: "Carol", Email: "carol@example.com", Password: "hash"})

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
