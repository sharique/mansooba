package repository_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
)

func TestAttachmentRepository_Create_Succeeds(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewAttachmentRepository(db)
	a := &domain.Attachment{
		IssueID: 1, UploaderID: 1, Filename: "screenshot.png",
		ObjectKey: "issues/1/abc-png", ContentType: "image/png", SizeBytes: 1024,
	}
	if err := repo.Create(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("expected ID to be set after Create")
	}
}

func TestAttachmentRepository_FindByID(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewAttachmentRepository(db)
	created := &domain.Attachment{
		IssueID: 1, UploaderID: 1, Filename: "log.txt",
		ObjectKey: "issues/1/def-txt", ContentType: "text/plain", SizeBytes: 512,
	}
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
			_, err := repo.FindByID(context.Background(), tc.id)
			if err != tc.wantErr {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestAttachmentRepository_FindByIssueID(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewAttachmentRepository(db)
	for i := 0; i < 2; i++ {
		_ = repo.Create(context.Background(), &domain.Attachment{
			IssueID: 10, UploaderID: 1, Filename: "a.txt",
			ObjectKey: "issues/10/x", ContentType: "text/plain", SizeBytes: 1,
		})
	}
	_ = repo.Create(context.Background(), &domain.Attachment{
		IssueID: 99, UploaderID: 1, Filename: "other.txt",
		ObjectKey: "issues/99/y", ContentType: "text/plain", SizeBytes: 1,
	})

	got, err := repo.FindByIssueID(context.Background(), 10)
	if err != nil {
		t.Fatalf("FindByIssueID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 attachments, got %d", len(got))
	}
}

func TestAttachmentRepository_CountByIssueID(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewAttachmentRepository(db)
	for i := 0; i < 3; i++ {
		_ = repo.Create(context.Background(), &domain.Attachment{
			IssueID: 20, UploaderID: 1, Filename: "a.txt",
			ObjectKey: "issues/20/x", ContentType: "text/plain", SizeBytes: 1,
		})
	}

	count, err := repo.CountByIssueID(context.Background(), 20)
	if err != nil {
		t.Fatalf("CountByIssueID: %v", err)
	}
	if count != 3 {
		t.Fatalf("want 3, got %d", count)
	}

	zero, err := repo.CountByIssueID(context.Background(), 999)
	if err != nil {
		t.Fatalf("CountByIssueID (no attachments): %v", err)
	}
	if zero != 0 {
		t.Fatalf("want 0, got %d", zero)
	}
}

func TestAttachmentRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewAttachmentRepository(db)
	created := &domain.Attachment{
		IssueID: 30, UploaderID: 1, Filename: "a.txt",
		ObjectKey: "issues/30/x", ContentType: "text/plain", SizeBytes: 1,
	}
	_ = repo.Create(context.Background(), created)

	if err := repo.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), created.ID); err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}

func TestAttachmentRepository_DeleteByIssueID(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewAttachmentRepository(db)
	for i := 0; i < 2; i++ {
		_ = repo.Create(context.Background(), &domain.Attachment{
			IssueID: 40, UploaderID: 1, Filename: "a.txt",
			ObjectKey: "issues/40/x", ContentType: "text/plain", SizeBytes: 1,
		})
	}
	other := &domain.Attachment{
		IssueID: 41, UploaderID: 1, Filename: "b.txt",
		ObjectKey: "issues/41/y", ContentType: "text/plain", SizeBytes: 1,
	}
	_ = repo.Create(context.Background(), other)

	if err := repo.DeleteByIssueID(context.Background(), 40); err != nil {
		t.Fatalf("DeleteByIssueID: %v", err)
	}

	remaining, err := repo.FindByIssueID(context.Background(), 40)
	if err != nil {
		t.Fatalf("FindByIssueID: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("want 0 remaining, got %d", len(remaining))
	}

	// Other issue's attachment must be untouched.
	if _, err := repo.FindByID(context.Background(), other.ID); err != nil {
		t.Fatalf("expected other issue's attachment to survive, got %v", err)
	}
}
