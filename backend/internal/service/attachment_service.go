package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// maxAttachmentsPerIssue is the per-issue attachment cap (FR-011).
const maxAttachmentsPerIssue = 20

// AttachmentStorage is the subset of attachmentstorage.Storage's behavior
// AttachmentService depends on. Defined here (not in the attachmentstorage
// package) so unit tests can stub it — *attachmentstorage.Storage satisfies
// this interface without any changes on its side.
type AttachmentStorage interface {
	// Save writes data under keyPrefix (e.g. "PROJ/PROJ-3" — a project key
	// and issue key) and returns the generated object key.
	Save(ctx context.Context, keyPrefix string, filename string, data []byte, contentType string) (objectKey string, err error)
	PresignGet(ctx context.Context, objectKey, filename string) (string, error)
	Delete(ctx context.Context, objectKey string) error
	DeleteAll(ctx context.Context, objectKeys []string) error
}

// AttachmentService manages file attachments on issues.
type AttachmentService interface {
	Upload(ctx context.Context, issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error)
	List(ctx context.Context, issueID, callerID uint) ([]*dto.AttachmentResponse, error)
	// GenerateDownloadURL returns a short-lived presigned URL and the
	// attachment's original filename, or an error if the caller isn't a
	// project member. No URL is ever generated for a denied caller
	// (research.md Decision 2).
	GenerateDownloadURL(ctx context.Context, issueID, attachmentID, callerID uint) (url, filename string, err error)
	Delete(ctx context.Context, issueID, attachmentID, callerID uint) error
}

type attachmentService struct {
	attachmentRepo domain.AttachmentRepository
	issueRepo      domain.IssueRepository
	projectRepo    domain.ProjectRepository
	memberRepo     domain.ProjectMemberRepository
	activitySvc    ActivityService
	userRepo       domain.UserRepository
	storage        AttachmentStorage
}

func NewAttachmentService(
	attachmentRepo domain.AttachmentRepository,
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	activitySvc ActivityService,
	userRepo domain.UserRepository,
	storage AttachmentStorage,
) AttachmentService {
	return &attachmentService{
		attachmentRepo: attachmentRepo,
		issueRepo:      issueRepo,
		projectRepo:    projectRepo,
		memberRepo:     memberRepo,
		activitySvc:    activitySvc,
		userRepo:       userRepo,
		storage:        storage,
	}
}

// Upload validates the caller's role (member/admin only — viewers are
// rejected, FR-001), then writes each file to storage before creating its
// DB row (research.md Decision 9: S3-write-before-DB-row). A per-file
// storage failure is reported in Rejected, not returned as an error — the
// batch always completes with whatever succeeded (contracts/api-contracts.md).
func (s *attachmentService) Upload(ctx context.Context, issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	membership, err := s.memberRepo.FindByProjectAndUser(ctx, issue.ProjectID, callerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, err
	}
	if membership.Role == "viewer" {
		return nil, domain.ErrForbidden
	}

	count, err := s.attachmentRepo.CountByIssueID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if count+int64(len(files)) > maxAttachmentsPerIssue {
		return nil, domain.ErrAttachmentCapReached
	}

	project, err := s.projectRepo.FindByID(ctx, issue.ProjectID)
	if err != nil {
		return nil, err
	}
	// e.g. "PROJ/PROJ-3" — objects for an issue live under its project key
	// and issue key, not an opaque numeric ID.
	keyPrefix := project.Key + "/" + issue.Key

	result := &dto.AttachmentUploadResult{
		Uploaded: []dto.AttachmentResponse{},
		Rejected: []dto.AttachmentRejection{},
	}

	for _, f := range files {
		key, err := s.storage.Save(ctx, keyPrefix, f.Filename, f.Data, f.ContentType)
		if err != nil {
			result.Rejected = append(result.Rejected, dto.AttachmentRejection{
				Filename: f.Filename,
				Reason:   err.Error(),
			})
			continue
		}

		a := &domain.Attachment{
			IssueID:     issueID,
			UploaderID:  callerID,
			Filename:    f.Filename,
			ObjectKey:   key,
			ContentType: f.ContentType,
			SizeBytes:   int64(len(f.Data)),
		}
		if err := s.attachmentRepo.Create(ctx, a); err != nil {
			result.Rejected = append(result.Rejected, dto.AttachmentRejection{
				Filename: f.Filename,
				Reason:   "failed to save attachment record",
			})
			continue
		}

		_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
			IssueID:  issueID,
			ActorID:  callerID,
			Kind:     domain.ActivityAttachmentAdded,
			NewValue: f.Filename,
		})

		result.Uploaded = append(result.Uploaded, s.toResponse(ctx, a))
	}

	return result, nil
}

// List returns an issue's attachments, most-recent-first. Any project
// member — including viewer — may list.
func (s *attachmentService) List(ctx context.Context, issueID, callerID uint) ([]*dto.AttachmentResponse, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, issue.ProjectID, callerID); err != nil {
		return nil, err
	}

	attachments, err := s.attachmentRepo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.AttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		r := s.toResponse(ctx, a)
		result = append(result, &r)
	}
	return result, nil
}

func (s *attachmentService) GenerateDownloadURL(ctx context.Context, issueID, attachmentID, callerID uint) (string, string, error) {
	a, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return "", "", err
	}
	if a.IssueID != issueID {
		return "", "", domain.ErrNotFound
	}

	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return "", "", err
	}
	if err := s.requireMember(ctx, issue.ProjectID, callerID); err != nil {
		return "", "", err
	}

	url, err := s.storage.PresignGet(ctx, a.ObjectKey, a.Filename)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", domain.ErrAttachmentStorageUnavailable, err)
	}
	return url, a.Filename, nil
}

// Delete removes an attachment. Only the original uploader or a project
// admin may delete (FR-006); this mirrors comment_service.go's Delete
// exactly — no independent re-check of the uploader's current role. The S3
// object is removed before the DB row (research.md Decision 9): a failed
// S3 delete leaves the attachment fully intact and retryable.
func (s *attachmentService) Delete(ctx context.Context, issueID, attachmentID, callerID uint) error {
	a, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if a.IssueID != issueID {
		return domain.ErrNotFound
	}

	if a.UploaderID != callerID {
		issue, err := s.issueRepo.FindByID(ctx, issueID)
		if err != nil {
			return err
		}
		membership, err := s.memberRepo.FindByProjectAndUser(ctx, issue.ProjectID, callerID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrForbidden
			}
			return err
		}
		if membership.Role != "admin" {
			return domain.ErrForbidden
		}
	}

	if err := s.storage.Delete(ctx, a.ObjectKey); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrAttachmentStorageUnavailable, err)
	}
	if err := s.attachmentRepo.Delete(ctx, attachmentID); err != nil {
		return err
	}

	_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
		IssueID:  issueID,
		ActorID:  callerID,
		Kind:     domain.ActivityAttachmentRemoved,
		OldValue: a.Filename,
	})

	return nil
}

func (s *attachmentService) requireMember(ctx context.Context, projectID, userID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

func (s *attachmentService) toResponse(ctx context.Context, a *domain.Attachment) dto.AttachmentResponse {
	r := dto.AttachmentResponse{
		ID:          a.ID,
		IssueID:     a.IssueID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		UploaderID:  a.UploaderID,
		CreatedAt:   a.CreatedAt,
	}
	if u, err := s.userRepo.FindByID(ctx, a.UploaderID); err == nil {
		r.UploaderName = u.Name
	}
	return r
}
