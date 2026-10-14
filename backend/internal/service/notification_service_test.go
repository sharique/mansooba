package service_test

// Constitution III: the "members only" rule decides who can learn an issue key,
// so these tests use real in-memory SQLite and the real repositories.
// Only the "repo Create fails" case wraps the real repo.

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type notifEnv struct {
	svc       service.NotificationService
	db        *gorm.DB
	projectID uint
	issueID   uint
	actor     uint
	b, c      uint // project members
	outsider  uint // not a member
}

func newNotifEnv(t *testing.T, wrap func(domain.NotificationRepository) domain.NotificationRepository) *notifEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.ProjectMember{}, &domain.Issue{}, &domain.Notification{}))

	mk := func(name string) uint {
		u := &domain.User{Name: name, Email: name + "@x.com", Password: "h", IsActive: true}
		require.NoError(t, db.Create(u).Error)
		return u.ID
	}
	actor, b, c, out := mk("actor"), mk("bob"), mk("cat"), mk("outsider")
	p := &domain.Project{Key: "APOL", Name: "Apollo", OwnerID: actor}
	require.NoError(t, db.Create(p).Error)
	for _, uid := range []uint{actor, b, c} {
		require.NoError(t, db.Create(&domain.ProjectMember{ProjectID: p.ID, UserID: uid, Role: "member"}).Error)
	}
	issue := &domain.Issue{Key: "APOL-1", ProjectID: p.ID, Title: "T", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: actor}
	require.NoError(t, db.Create(issue).Error)

	var notifRepo domain.NotificationRepository = repository.NewNotificationRepository(db)
	if wrap != nil {
		notifRepo = wrap(notifRepo)
	}
	svc := service.NewNotificationService(notifRepo, repository.NewProjectMemberRepository(db), zap.NewNop())
	return &notifEnv{svc: svc, db: db, projectID: p.ID, issueID: issue.ID, actor: actor, b: b, c: c, outsider: out}
}

func (e *notifEnv) rows(t *testing.T) []domain.Notification {
	t.Helper()
	var rows []domain.Notification
	require.NoError(t, e.db.Order("id").Find(&rows).Error)
	return rows
}

func ptr(u uint) *uint { return &u }

func TestNotificationService_Assign(t *testing.T) {
	e := newNotifEnv(t, nil)
	e.svc.NotifyAssignmentChange(context.Background(), e.projectID, e.issueID, e.actor, nil, ptr(e.b))
	rows := e.rows(t)
	require.Len(t, rows, 1)
	assert.Equal(t, e.b, rows[0].RecipientID)
	assert.Equal(t, e.actor, rows[0].ActorID)
	assert.Equal(t, domain.NotificationKindTaskAssigned, rows[0].Kind)
	assert.Equal(t, e.issueID, rows[0].IssueID)
	assert.Equal(t, uint(0), rows[0].CommentID)
}

func TestNotificationService_Unassign(t *testing.T) {
	e := newNotifEnv(t, nil)
	e.svc.NotifyAssignmentChange(context.Background(), e.projectID, e.issueID, e.actor, ptr(e.b), nil)
	rows := e.rows(t)
	require.Len(t, rows, 1)
	assert.Equal(t, e.b, rows[0].RecipientID)
	assert.Equal(t, domain.NotificationKindTaskUnassigned, rows[0].Kind)
}

func TestNotificationService_ReassignNotifiesBoth(t *testing.T) {
	e := newNotifEnv(t, nil)
	e.svc.NotifyAssignmentChange(context.Background(), e.projectID, e.issueID, e.actor, ptr(e.b), ptr(e.c))
	rows := e.rows(t)
	require.Len(t, rows, 2)
	kinds := map[uint]string{}
	for _, r := range rows {
		kinds[r.RecipientID] = r.Kind
	}
	assert.Equal(t, domain.NotificationKindTaskAssigned, kinds[e.c])
	assert.Equal(t, domain.NotificationKindTaskUnassigned, kinds[e.b])
}

func TestNotificationService_NeverNotifiesTheActor(t *testing.T) {
	e := newNotifEnv(t, nil)
	ctx := context.Background()
	e.svc.NotifyAssignmentChange(ctx, e.projectID, e.issueID, e.actor, nil, ptr(e.actor)) // self-assign
	e.svc.NotifyAssignmentChange(ctx, e.projectID, e.issueID, e.actor, ptr(e.actor), nil) // self-unassign
	assert.Empty(t, e.rows(t))
}

func TestNotificationService_SkipsNonMembers(t *testing.T) {
	e := newNotifEnv(t, nil)
	ctx := context.Background()
	e.svc.NotifyAssignmentChange(ctx, e.projectID, e.issueID, e.actor, nil, ptr(e.outsider)) // never a member
	e.svc.NotifyAssignmentChange(ctx, e.projectID, e.issueID, e.actor, ptr(e.outsider), nil)
	assert.Empty(t, e.rows(t), "nothing may be written for a non-member")

	// A member who was removed is no longer notified either.
	var m domain.ProjectMember
	require.NoError(t, e.db.Where("user_id = ?", e.b).First(&m).Error)
	require.NoError(t, e.db.Delete(&m).Error)
	e.svc.NotifyAssignmentChange(ctx, e.projectID, e.issueID, e.actor, nil, ptr(e.b))
	assert.Empty(t, e.rows(t))
}

func TestNotificationService_NoChangeCreatesNothing(t *testing.T) {
	e := newNotifEnv(t, nil)
	e.svc.NotifyAssignmentChange(context.Background(), e.projectID, e.issueID, e.actor, ptr(e.b), ptr(e.b))
	e.svc.NotifyAssignmentChange(context.Background(), e.projectID, e.issueID, e.actor, nil, nil)
	assert.Empty(t, e.rows(t))
}

func TestNotificationService_MembershipChange(t *testing.T) {
	e := newNotifEnv(t, nil)
	ctx := context.Background()
	e.svc.NotifyMembershipChange(ctx, e.projectID, e.actor, e.outsider, true)
	e.svc.NotifyMembershipChange(ctx, e.projectID, e.actor, e.b, false)
	rows := e.rows(t)
	require.Len(t, rows, 2)
	assert.Equal(t, domain.NotificationKindProjectMemberAdded, rows[0].Kind)
	assert.Equal(t, e.outsider, rows[0].RecipientID)
	assert.Equal(t, e.projectID, rows[0].ProjectID)
	assert.Equal(t, uint(0), rows[0].IssueID)
	assert.Equal(t, domain.NotificationKindProjectMemberRemoved, rows[1].Kind)
	assert.Equal(t, e.b, rows[1].RecipientID)
}

func TestNotificationService_MembershipChangeSelfCreatesNothing(t *testing.T) {
	e := newNotifEnv(t, nil)
	e.svc.NotifyMembershipChange(context.Background(), e.projectID, e.actor, e.actor, true)
	e.svc.NotifyMembershipChange(context.Background(), e.projectID, e.b, e.b, false)
	assert.Empty(t, e.rows(t))
}

type failingNotifRepo struct{ domain.NotificationRepository }

func (failingNotifRepo) Create(context.Context, *domain.Notification) error {
	return errors.New("db down")
}

func TestNotificationService_RepoErrorIsSwallowed(t *testing.T) {
	e := newNotifEnv(t, func(r domain.NotificationRepository) domain.NotificationRepository {
		return failingNotifRepo{r}
	})
	assert.NotPanics(t, func() {
		e.svc.NotifyAssignmentChange(context.Background(), e.projectID, e.issueID, e.actor, nil, ptr(e.b))
		e.svc.NotifyMembershipChange(context.Background(), e.projectID, e.actor, e.outsider, true)
	})
}
