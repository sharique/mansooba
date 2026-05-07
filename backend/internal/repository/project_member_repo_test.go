package repository_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/repository"
)

func TestProjectMemberRepository_Create_Succeeds(t *testing.T) {
	repo := repository.NewProjectMemberRepository(newTestDB(t))
	m := &domain.ProjectMember{ProjectID: 1, UserID: 10, Role: "member"}
	if err := repo.Create(context.Background(), m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestProjectMemberRepository_FindByProjectID(t *testing.T) {
	repo := repository.NewProjectMemberRepository(newTestDB(t))
	_ = repo.Create(context.Background(), &domain.ProjectMember{ProjectID: 5, UserID: 1, Role: "admin"})
	_ = repo.Create(context.Background(), &domain.ProjectMember{ProjectID: 5, UserID: 2, Role: "member"})
	_ = repo.Create(context.Background(), &domain.ProjectMember{ProjectID: 9, UserID: 3, Role: "member"})

	members, err := repo.FindByProjectID(context.Background(), 5)
	if err != nil {
		t.Fatalf("FindByProjectID: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("want 2 members, got %d", len(members))
	}
}

func TestProjectMemberRepository_FindByProjectAndUser(t *testing.T) {
	repo := repository.NewProjectMemberRepository(newTestDB(t))
	_ = repo.Create(context.Background(), &domain.ProjectMember{ProjectID: 7, UserID: 20, Role: "admin"})

	tests := []struct {
		name      string
		projectID uint
		userID    uint
		wantErr   error
	}{
		{"found", 7, 20, nil},
		{"not found", 7, 99, domain.ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.FindByProjectAndUser(context.Background(), tc.projectID, tc.userID)
			if err != tc.wantErr {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestProjectMemberRepository_Delete(t *testing.T) {
	repo := repository.NewProjectMemberRepository(newTestDB(t))
	m := &domain.ProjectMember{ProjectID: 3, UserID: 5, Role: "viewer"}
	_ = repo.Create(context.Background(), m)

	if err := repo.Delete(context.Background(), m.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByProjectAndUser(context.Background(), 3, 5)
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}
