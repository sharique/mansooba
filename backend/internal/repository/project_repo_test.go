package repository_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
)

func TestProjectRepository_Create_Succeeds(t *testing.T) {
	repo := repository.NewProjectRepository(newTestDB(t))
	p := &domain.Project{Key: "PROJ", Name: "Project One", OwnerID: 1}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestProjectRepository_FindByKey(t *testing.T) {
	repo := repository.NewProjectRepository(newTestDB(t))
	_ = repo.Create(context.Background(), &domain.Project{Key: "ALPHA", Name: "Alpha", OwnerID: 1})

	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"found", "ALPHA", nil},
		{"not found", "NOPE", domain.ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.FindByKey(context.Background(), tc.key)
			if err != tc.wantErr {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestProjectRepository_FindByUserID(t *testing.T) {
	repo := repository.NewProjectRepository(newTestDB(t))
	_ = repo.Create(context.Background(), &domain.Project{Key: "P1", Name: "P1", OwnerID: 42})
	_ = repo.Create(context.Background(), &domain.Project{Key: "P2", Name: "P2", OwnerID: 42})
	_ = repo.Create(context.Background(), &domain.Project{Key: "P3", Name: "P3", OwnerID: 99})

	projects, err := repo.FindByUserID(context.Background(), 42)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("want 2 projects, got %d", len(projects))
	}
}

func TestProjectRepository_Update(t *testing.T) {
	repo := repository.NewProjectRepository(newTestDB(t))
	p := &domain.Project{Key: "UPD", Name: "Original", OwnerID: 1}
	_ = repo.Create(context.Background(), p)

	p.Name = "Updated"
	if err := repo.Update(context.Background(), p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.FindByKey(context.Background(), "UPD")
	if got.Name != "Updated" {
		t.Fatalf("want name %q, got %q", "Updated", got.Name)
	}
}

func TestProjectRepository_Delete(t *testing.T) {
	repo := repository.NewProjectRepository(newTestDB(t))
	p := &domain.Project{Key: "DEL", Name: "Delete Me", OwnerID: 1}
	_ = repo.Create(context.Background(), p)

	if err := repo.Delete(context.Background(), p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByKey(context.Background(), "DEL")
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}
