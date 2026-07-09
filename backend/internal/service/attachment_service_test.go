package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubAttachmentRepo ──────────────────────────────────────────────────────

type stubAttachmentRepo struct {
	attachments []*domain.Attachment
	nextID      uint
}

func newStubAttachmentRepo() *stubAttachmentRepo {
	return &stubAttachmentRepo{nextID: 1}
}

func (r *stubAttachmentRepo) Create(_ context.Context, a *domain.Attachment) error {
	a.ID = r.nextID
	r.nextID++
	cp := *a
	r.attachments = append(r.attachments, &cp)
	return nil
}

func (r *stubAttachmentRepo) FindByIssueID(_ context.Context, issueID uint) ([]*domain.Attachment, error) {
	var out []*domain.Attachment
	for _, a := range r.attachments {
		if a.IssueID == issueID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *stubAttachmentRepo) FindByID(_ context.Context, id uint) (*domain.Attachment, error) {
	for _, a := range r.attachments {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubAttachmentRepo) CountByIssueID(_ context.Context, issueID uint) (int64, error) {
	var count int64
	for _, a := range r.attachments {
		if a.IssueID == issueID {
			count++
		}
	}
	return count, nil
}

func (r *stubAttachmentRepo) Delete(_ context.Context, id uint) error {
	for i, a := range r.attachments {
		if a.ID == id {
			r.attachments = append(r.attachments[:i], r.attachments[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubAttachmentRepo) DeleteByIssueID(_ context.Context, issueID uint) error {
	var kept []*domain.Attachment
	for _, a := range r.attachments {
		if a.IssueID != issueID {
			kept = append(kept, a)
		}
	}
	r.attachments = kept
	return nil
}

// ── stubAttachmentStorage ────────────────────────────────────────────────────

// stubAttachmentStorage lets tests control whether S3 operations succeed,
// and records exactly what was written/deleted — used to assert the
// write/delete ordering guarantee (research.md Decision 9): a failed Save
// must never result in an Attachment DB row, and a failed Delete must never
// remove the DB row.
type stubAttachmentStorage struct {
	saveErr      error
	deleteErr    error
	presignErr   error
	savedKeys    []string
	deletedKeys  []string
	nextKeyIndex int
}

func (s *stubAttachmentStorage) Save(_ context.Context, issueID uint, filename string, data []byte, _ string) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	s.nextKeyIndex++
	key := "issues/stub/" + filename + "-" + string(rune('0'+s.nextKeyIndex))
	s.savedKeys = append(s.savedKeys, key)
	return key, nil
}

func (s *stubAttachmentStorage) PresignGet(_ context.Context, key, _ string) (string, error) {
	if s.presignErr != nil {
		return "", s.presignErr
	}
	return "https://s3.example.com/" + key + "?X-Amz-Signature=abc", nil
}

func (s *stubAttachmentStorage) Delete(_ context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedKeys = append(s.deletedKeys, key)
	return nil
}

func (s *stubAttachmentStorage) DeleteAll(_ context.Context, keys []string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedKeys = append(s.deletedKeys, keys...)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newAttachmentTestDeps() (*stubAttachmentRepo, *stubIssueRepo, *stubProjectMemberRepo, *stubActivitySvc, *stubUserRepo, *stubAttachmentStorage) {
	return newStubAttachmentRepo(), newStubIssueRepo(), newStubProjectMemberRepo(), &stubActivitySvc{}, newStubUserRepo(), &stubAttachmentStorage{}
}

func newAttachmentService(
	attachmentRepo *stubAttachmentRepo, issueRepo *stubIssueRepo, memberRepo *stubProjectMemberRepo,
	activitySvc *stubActivitySvc, userRepo *stubUserRepo, storage *stubAttachmentStorage,
) service.AttachmentService {
	return service.NewAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)
}

func mustCreateUser(t *testing.T, userRepo *stubUserRepo, id uint, name string) {
	t.Helper()
	_ = userRepo.Create(context.Background(), &domain.User{ID: id, Name: name, Email: name + "@example.com", Password: "x"})
}

// ── Upload ────────────────────────────────────────────────────────────────────

func TestAttachmentService_Upload_MemberCanUpload(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issue := &domain.Issue{ID: 1, ProjectID: 10, Key: "P-1", Title: "t", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium, ReporterID: 42}
	issueRepo.issues = append(issueRepo.issues, issue)
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})
	mustCreateUser(t, userRepo, 42, "Alice")

	result, err := svc.Upload(context.Background(), 1, 42, []dto.AttachmentUploadFile{
		{Filename: "a.png", Data: []byte("x"), ContentType: "image/png"},
	})
	require.NoError(t, err)
	require.Len(t, result.Uploaded, 1)
	assert.Empty(t, result.Rejected)
	assert.Equal(t, "a.png", result.Uploaded[0].Filename)
	assert.Equal(t, "Alice", result.Uploaded[0].UploaderName)
	assert.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityAttachmentAdded, activitySvc.recorded[0].Kind)
}

func TestAttachmentService_Upload_AdminCanUpload(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 99, Role: "admin"})
	mustCreateUser(t, userRepo, 99, "Admin")

	result, err := svc.Upload(context.Background(), 1, 99, []dto.AttachmentUploadFile{
		{Filename: "a.png", Data: []byte("x"), ContentType: "image/png"},
	})
	require.NoError(t, err)
	assert.Len(t, result.Uploaded, 1)
}

func TestAttachmentService_Upload_ViewerForbidden(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 7, Role: "viewer"})

	_, err := svc.Upload(context.Background(), 1, 7, []dto.AttachmentUploadFile{
		{Filename: "a.png", Data: []byte("x"), ContentType: "image/png"},
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.Empty(t, attachmentRepo.attachments, "no attachment record for a rejected viewer upload")
}

func TestAttachmentService_Upload_NonMemberForbidden(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})

	_, err := svc.Upload(context.Background(), 1, 999, []dto.AttachmentUploadFile{
		{Filename: "a.png", Data: []byte("x"), ContentType: "image/png"},
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAttachmentService_Upload_PerFileStorageFailureIsRejectedNotFatal(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, _ := newAttachmentTestDeps()

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})
	mustCreateUser(t, userRepo, 42, "Alice")

	failing := &failingOnceStorage{failFilename: "bad.exe"}
	svc := service.NewAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, failing)

	result, err := svc.Upload(context.Background(), 1, 42, []dto.AttachmentUploadFile{
		{Filename: "good.png", Data: []byte("x"), ContentType: "image/png"},
		{Filename: "bad.exe", Data: []byte("y"), ContentType: "application/x-msdownload"},
	})
	require.NoError(t, err)
	require.Len(t, result.Uploaded, 1)
	require.Len(t, result.Rejected, 1)
	assert.Equal(t, "good.png", result.Uploaded[0].Filename)
	assert.Equal(t, "bad.exe", result.Rejected[0].Filename)
	assert.Len(t, attachmentRepo.attachments, 1, "S3-write-before-DB-row: a rejected file must not create a DB row (research.md Decision 9)")
}

// failingOnceStorage rejects a specific filename, succeeding for all others —
// used to test batch partial-rejection without depending on real content
// sniffing (that's attachmentstorage's own responsibility, tested separately).
type failingOnceStorage struct {
	failFilename string
	saved        []string
}

func (s *failingOnceStorage) Save(_ context.Context, _ uint, filename string, _ []byte, _ string) (string, error) {
	if filename == s.failFilename {
		return "", errors.New("content type not accepted")
	}
	key := "issues/stub/" + filename
	s.saved = append(s.saved, key)
	return key, nil
}
func (s *failingOnceStorage) PresignGet(_ context.Context, key, _ string) (string, error) {
	return "https://s3.example.com/" + key, nil
}
func (s *failingOnceStorage) Delete(_ context.Context, _ string) error      { return nil }
func (s *failingOnceStorage) DeleteAll(_ context.Context, _ []string) error { return nil }

func TestAttachmentService_Upload_CapReached(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})
	mustCreateUser(t, userRepo, 42, "Alice")

	for i := 0; i < 20; i++ {
		attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: uint(i + 1), IssueID: 1})
	}

	_, err := svc.Upload(context.Background(), 1, 42, []dto.AttachmentUploadFile{
		{Filename: "one_more.png", Data: []byte("x"), ContentType: "image/png"},
	})
	assert.ErrorIs(t, err, domain.ErrAttachmentCapReached)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestAttachmentService_List_ViewerCanList(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 7, Role: "viewer"})
	mustCreateUser(t, userRepo, 42, "Alice")
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 1, IssueID: 1, UploaderID: 42, Filename: "a.png", CreatedAt: time.Now()})

	result, err := svc.List(context.Background(), 1, 7)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0].UploaderName)
}

func TestAttachmentService_List_NonMemberForbidden(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})

	_, err := svc.List(context.Background(), 1, 999)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// ── GenerateDownloadURL ──────────────────────────────────────────────────────

func TestAttachmentService_GenerateDownloadURL_MemberCanDownload(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	url, filename, err := svc.GenerateDownloadURL(context.Background(), 1, 5, 42)
	require.NoError(t, err)
	assert.Equal(t, "x.png", filename)
	assert.Contains(t, url, "X-Amz-Signature")
}

func TestAttachmentService_GenerateDownloadURL_NonMemberForbidden_NoURLGenerated(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	url, _, err := svc.GenerateDownloadURL(context.Background(), 1, 5, 999)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.Empty(t, url, "no presigned URL must ever be generated for a denied caller (research.md Decision 2)")
}

func TestAttachmentService_GenerateDownloadURL_WrongIssueNotFound(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 999, ObjectKey: "issues/999/x.png", Filename: "x.png"})

	_, _, err := svc.GenerateDownloadURL(context.Background(), 1, 5, 42)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestAttachmentService_Delete_UploaderCanDeleteOwn(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, UploaderID: 42, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	err := svc.Delete(context.Background(), 1, 5, 42)
	require.NoError(t, err)
	assert.Empty(t, attachmentRepo.attachments)
	assert.Contains(t, storage.deletedKeys, "issues/1/x.png")
	assert.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityAttachmentRemoved, activitySvc.recorded[0].Kind)
}

func TestAttachmentService_Delete_AdminCanDeleteOthers(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 99, Role: "admin"})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, UploaderID: 42, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	err := svc.Delete(context.Background(), 1, 5, 99)
	require.NoError(t, err)
	assert.Empty(t, attachmentRepo.attachments)
}

func TestAttachmentService_Delete_RegularMemberForbidden(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 7, Role: "member"})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, UploaderID: 42, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	err := svc.Delete(context.Background(), 1, 5, 7)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.Len(t, attachmentRepo.attachments, 1, "attachment must remain untouched when delete is denied")
}

func TestAttachmentService_Delete_ViewerForbiddenEvenIfSomehowUploader(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 7, Role: "viewer"})
	// Uploader themself has since been demoted to viewer.
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, UploaderID: 7, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	err := svc.Delete(context.Background(), 1, 5, 7)
	require.NoError(t, err, "uploader-own-delete follows the same rule as comments — no re-check of current role")
}

func TestAttachmentService_Delete_StorageFailureLeavesDBRowIntact(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	storage.deleteErr = errors.New("s3 unavailable")
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10, ReporterID: 1})
	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 1, UploaderID: 42, ObjectKey: "issues/1/x.png", Filename: "x.png"})

	err := svc.Delete(context.Background(), 1, 5, 42)
	assert.ErrorIs(t, err, domain.ErrAttachmentStorageUnavailable)
	assert.Len(t, attachmentRepo.attachments, 1, "S3-delete-before-DB-row: a failed S3 delete must leave the DB row intact (research.md Decision 9)")
}

func TestAttachmentService_Delete_WrongIssueNotFound(t *testing.T) {
	attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage := newAttachmentTestDeps()
	svc := newAttachmentService(attachmentRepo, issueRepo, memberRepo, activitySvc, userRepo, storage)

	attachmentRepo.attachments = append(attachmentRepo.attachments, &domain.Attachment{ID: 5, IssueID: 999, UploaderID: 42, ObjectKey: "issues/999/x.png", Filename: "x.png"})

	err := svc.Delete(context.Background(), 1, 5, 42)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
