package service_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubCommentRepo ───────────────────────────────────────────────────────────

type stubCommentRepo struct {
	comments []*domain.Comment
	nextID   uint
}

func newStubCommentRepo() *stubCommentRepo { return &stubCommentRepo{nextID: 1} }

func (r *stubCommentRepo) Create(_ context.Context, c *domain.Comment) error {
	c.ID = r.nextID
	r.nextID++
	cp := *c
	r.comments = append(r.comments, &cp)
	return nil
}

func (r *stubCommentRepo) FindByIssueID(_ context.Context, issueID uint) ([]*domain.Comment, error) {
	var out []*domain.Comment
	for _, c := range r.comments {
		if c.IssueID == issueID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *stubCommentRepo) FindByID(_ context.Context, id uint) (*domain.Comment, error) {
	for _, c := range r.comments {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubCommentRepo) Update(_ context.Context, c *domain.Comment) error {
	for i, existing := range r.comments {
		if existing.ID == c.ID {
			r.comments[i] = c
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubCommentRepo) Delete(_ context.Context, id uint) error {
	for i, c := range r.comments {
		if c.ID == id {
			r.comments = append(r.comments[:i], r.comments[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

// ── stubActivityService ───────────────────────────────────────────────────────

type stubActivityService struct {
	recorded []*domain.ActivityEvent
}

func (s *stubActivityService) Record(_ context.Context, e *domain.ActivityEvent) error {
	s.recorded = append(s.recorded, e)
	return nil
}

func (s *stubActivityService) ListByIssue(_ context.Context, issueID uint) ([]*domain.ActivityEvent, error) {
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newCommentTestEnv() (service.CommentService, *stubCommentRepo, *stubActivityService) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})

	memberRepo := newStubProjectMemberRepo()
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})

	commentRepo := newStubCommentRepo()
	activitySvc := &stubActivityService{}

	svc := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc)
	return svc, commentRepo, activitySvc
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCommentService_Create_PersistsAndRecordsActivity(t *testing.T) {
	svc, commentRepo, activitySvc := newCommentTestEnv()

	resp, err := svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "looks good"})
	require.NoError(t, err)
	assert.Equal(t, "looks good", resp.Body)
	assert.Len(t, commentRepo.comments, 1)
	assert.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityCommentAdded, activitySvc.recorded[0].Kind)
}

func TestCommentService_Create_ForbiddenWhenNotMember(t *testing.T) {
	svc, _, _ := newCommentTestEnv()
	_, err := svc.Create(context.Background(), 1, 99, dto.CreateCommentRequest{Body: "x"})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCommentService_Create_NotFoundWhenIssueAbsent(t *testing.T) {
	svc, _, _ := newCommentTestEnv()
	_, err := svc.Create(context.Background(), 999, 42, dto.CreateCommentRequest{Body: "x"})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCommentService_Update_OwnerCanEdit(t *testing.T) {
	svc, commentRepo, _ := newCommentTestEnv()
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "original"})

	updated, err := svc.Update(context.Background(), commentRepo.comments[0].ID, 42, dto.UpdateCommentRequest{Body: "edited"})
	require.NoError(t, err)
	assert.Equal(t, "edited", updated.Body)
}

func TestCommentService_Update_ForbiddenForNonOwner(t *testing.T) {
	svc, commentRepo, _ := newCommentTestEnv()
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "original"})
	_, err := svc.Update(context.Background(), commentRepo.comments[0].ID, 99, dto.UpdateCommentRequest{Body: "edit"})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestCommentService_Delete_OwnerCanDelete(t *testing.T) {
	svc, commentRepo, _ := newCommentTestEnv()
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "bye"})
	err := svc.Delete(context.Background(), commentRepo.comments[0].ID, 42)
	require.NoError(t, err)
	assert.Len(t, commentRepo.comments, 0)
}

func TestCommentService_List_ReturnsByIssue(t *testing.T) {
	svc, _, _ := newCommentTestEnv()
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "a"})
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "b"})

	list, err := svc.List(context.Background(), 1, 42)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}
