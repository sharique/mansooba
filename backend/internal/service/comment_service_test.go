package service_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
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

func (s *stubActivityService) ListByIssue(_ context.Context, _ uint) ([]*dto.ActivityEventResponse, error) {
	return nil, nil
}

func (s *stubActivityService) GetMyActivity(_ context.Context, _ uint, _, _ int) ([]*dto.ActivityEventResponse, error) {
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

	svc := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc, newStubNotificationRepo(), &stubUserRepoMention{})
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

func TestCommentService_Delete_AdminCanDeleteOthersComment(t *testing.T) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})

	memberRepo := newStubProjectMemberRepo()
	// Author is user 42, admin is user 99
	memberRepo.members = append(memberRepo.members,
		&domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"},
		&domain.ProjectMember{ProjectID: 10, UserID: 99, Role: "admin"},
	)

	commentRepo := newStubCommentRepo()
	activitySvc := &stubActivityService{}
	svc := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc, newStubNotificationRepo(), &stubUserRepoMention{})

	// Author creates a comment
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "author comment"})

	// Admin deletes it
	err := svc.Delete(context.Background(), commentRepo.comments[0].ID, 99)
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

// T017: author_avatar_url enrichment tests
func TestCommentService_List_IncludesAuthorAvatarURL(t *testing.T) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})

	memberRepo := newStubProjectMemberRepo()
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})

	commentRepo := newStubCommentRepo()
	activitySvc := &stubActivityService{}

	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 42, Name: "Alice", Email: "alice@example.com", Password: "x", AvatarURL: "/uploads/avatars/avatar-42.jpg?v=1000"})

	svc := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc, newStubNotificationRepo(), userRepo)
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "hello"})

	list, err := svc.List(context.Background(), 1, 42)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "/uploads/avatars/avatar-42.jpg?v=1000", list[0].AuthorAvatarURL)
}

func TestCommentService_List_EmptyAvatarURLWhenNoAvatar(t *testing.T) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})

	memberRepo := newStubProjectMemberRepo()
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})

	commentRepo := newStubCommentRepo()
	activitySvc := &stubActivityService{}

	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 42, Name: "Bob", Email: "bob@example.com", Password: "x"})

	svc := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc, newStubNotificationRepo(), userRepo)
	_, _ = svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "hi"})

	list, err := svc.List(context.Background(), 1, 42)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "", list[0].AuthorAvatarURL)
}

func TestCommentService_List_IncludesAuthorName(t *testing.T) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})

	memberRepo := newStubProjectMemberRepo()
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})

	commentRepo := newStubCommentRepo()
	activitySvc := &stubActivityService{}

	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 42, Name: "Alice", Email: "alice@example.com", Password: "x"})

	svc2 := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc, newStubNotificationRepo(), userRepo)
	_, _ = svc2.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "hello"})

	list, err := svc2.List(context.Background(), 1, 42)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Alice", list[0].AuthorName)
}

// ── stubNotificationRepo ──────────────────────────────────────────────────────

type stubNotificationRepo struct {
	notifications []*domain.Notification
	nextID        uint
}

func newStubNotificationRepo() *stubNotificationRepo { return &stubNotificationRepo{nextID: 1} }

func (r *stubNotificationRepo) Create(_ context.Context, n *domain.Notification) error {
	n.ID = r.nextID
	r.nextID++
	cp := *n
	r.notifications = append(r.notifications, &cp)
	return nil
}

func (r *stubNotificationRepo) FindUnreadByRecipientID(_ context.Context, recipientID uint) ([]*domain.NotificationDetail, error) {
	var out []*domain.NotificationDetail
	for _, n := range r.notifications {
		if n.RecipientID == recipientID {
			out = append(out, &domain.NotificationDetail{Notification: *n})
		}
	}
	return out, nil
}

func (r *stubNotificationRepo) MarkRead(_ context.Context, id, recipientID uint) error {
	for _, n := range r.notifications {
		if n.ID == id && n.RecipientID == recipientID {
			n.Read = true
			return nil
		}
	}
	return domain.ErrNotFound
}

// ── stub user repo for mentions ───────────────────────────────────────────────

type stubUserRepoMention struct {
	users []*domain.User
}

func (r *stubUserRepoMention) Create(_ context.Context, _ *domain.User) error { return nil }
func (r *stubUserRepoMention) FindByID(_ context.Context, id uint) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *stubUserRepoMention) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *stubUserRepoMention) FindByEmailPrefix(_ context.Context, prefix string) (*domain.User, error) {
	for _, u := range r.users {
		if len(u.Email) > len(prefix) && u.Email[:len(prefix)] == prefix && u.Email[len(prefix)] == '@' {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubUserRepoMention) Update(_ context.Context, u *domain.User) error {
	return nil
}

func (r *stubUserRepoMention) HasAdmin(_ context.Context) (bool, error) {
	return false, nil
}

func (r *stubUserRepoMention) FindFirstAdmin(_ context.Context) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func newCommentTestEnvWithNotifications() (service.CommentService, *stubCommentRepo, *stubActivityService, *stubNotificationRepo) {
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})

	memberRepo := newStubProjectMemberRepo()
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})

	commentRepo := newStubCommentRepo()
	activitySvc := &stubActivityService{}
	notifRepo := newStubNotificationRepo()
	userRepo := &stubUserRepoMention{
		users: []*domain.User{
			{ID: 7, Name: "Alice Smith", Email: "alice@example.com"},
			{ID: 8, Name: "Bob Jones", Email: "bob@example.com"},
		},
	}

	svc := service.NewCommentService(commentRepo, issueRepo, memberRepo, activitySvc, notifRepo, userRepo)
	return svc, commentRepo, activitySvc, notifRepo
}

func TestCommentService_Create_ParsesMentionAndCreatesNotification(t *testing.T) {
	svc, _, _, notifRepo := newCommentTestEnvWithNotifications()
	_, err := svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "hey @alice looks good"})
	require.NoError(t, err)
	require.Len(t, notifRepo.notifications, 1)
	assert.Equal(t, uint(7), notifRepo.notifications[0].RecipientID)
	assert.Equal(t, uint(42), notifRepo.notifications[0].ActorID)
}

func TestCommentService_Create_DeduplicatesMentions(t *testing.T) {
	svc, _, _, notifRepo := newCommentTestEnvWithNotifications()
	_, err := svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "@alice @alice again"})
	require.NoError(t, err)
	assert.Len(t, notifRepo.notifications, 1)
}

func TestCommentService_Create_SkipsUnknownHandles(t *testing.T) {
	svc, _, _, notifRepo := newCommentTestEnvWithNotifications()
	_, err := svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "@nobody hello"})
	require.NoError(t, err)
	assert.Len(t, notifRepo.notifications, 0)
}

func TestCommentService_Create_MultipleMentionsCreatesMultipleNotifications(t *testing.T) {
	svc, _, _, notifRepo := newCommentTestEnvWithNotifications()
	_, err := svc.Create(context.Background(), 1, 42, dto.CreateCommentRequest{Body: "@alice and @bob review this"})
	require.NoError(t, err)
	assert.Len(t, notifRepo.notifications, 2)
}
