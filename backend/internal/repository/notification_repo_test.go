package repository_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newNotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.ProjectMember{}, &domain.Issue{}, &domain.Notification{}))
	return db
}

type notifFixture struct {
	db      *gorm.DB
	repo    domain.NotificationRepository
	actor   *domain.User
	bob     *domain.User
	project *domain.Project
	issue   *domain.Issue
}

func newNotifFixture(t *testing.T) *notifFixture {
	t.Helper()
	db := newNotificationTestDB(t)
	actor := &domain.User{Name: "Priya", Email: "priya@x.com", Password: "h", IsActive: true}
	bob := &domain.User{Name: "Bob", Email: "bob@x.com", Password: "h", IsActive: true}
	require.NoError(t, db.Create(actor).Error)
	require.NoError(t, db.Create(bob).Error)
	project := &domain.Project{Key: "APOL", Name: "Apollo", OwnerID: actor.ID}
	require.NoError(t, db.Create(project).Error)
	issue := &domain.Issue{Key: "APOL-1", ProjectID: project.ID, Title: "T", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: actor.ID}
	require.NoError(t, db.Create(issue).Error)
	return &notifFixture{db: db, repo: repository.NewNotificationRepository(db), actor: actor, bob: bob, project: project, issue: issue}
}

func TestNotificationRepo_MentionRowStillReturnedWithContext(t *testing.T) {
	f := newNotifFixture(t)
	ctx := context.Background()
	require.NoError(t, f.repo.Create(ctx, &domain.Notification{
		RecipientID: f.bob.ID, ActorID: f.actor.ID, Kind: domain.NotificationKindMention,
		IssueID: f.issue.ID, CommentID: 9,
	}))

	got, err := f.repo.FindUnreadByRecipientID(ctx, f.bob.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "APOL", got[0].ProjectKey)
	assert.Equal(t, "Apollo", got[0].ProjectName)
	assert.Equal(t, "APOL-1", got[0].IssueKey)
	assert.Equal(t, "Priya", got[0].ActorName)
	assert.Equal(t, domain.NotificationKindMention, got[0].Kind)
}

func TestNotificationRepo_ProjectRowWithoutIssueIsReturned(t *testing.T) {
	f := newNotifFixture(t)
	ctx := context.Background()
	require.NoError(t, f.repo.Create(ctx, &domain.Notification{
		RecipientID: f.bob.ID, ActorID: f.actor.ID, Kind: domain.NotificationKindProjectMemberRemoved,
		ProjectID: f.project.ID,
	}))

	got, err := f.repo.FindUnreadByRecipientID(ctx, f.bob.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "APOL", got[0].ProjectKey)
	assert.Equal(t, "Apollo", got[0].ProjectName)
	assert.Equal(t, "", got[0].IssueKey)
	assert.Equal(t, uint(0), got[0].IssueID)
}

func TestNotificationRepo_OtherRecipientsRowsAreNotReturned(t *testing.T) {
	f := newNotifFixture(t)
	ctx := context.Background()
	require.NoError(t, f.repo.Create(ctx, &domain.Notification{
		RecipientID: f.actor.ID, ActorID: f.bob.ID, Kind: domain.NotificationKindTaskAssigned, IssueID: f.issue.ID,
	}))

	got, err := f.repo.FindUnreadByRecipientID(ctx, f.bob.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestNotificationRepo_ReadRowsAreNotReturned(t *testing.T) {
	f := newNotifFixture(t)
	ctx := context.Background()
	n := &domain.Notification{RecipientID: f.bob.ID, ActorID: f.actor.ID, Kind: domain.NotificationKindTaskAssigned, IssueID: f.issue.ID}
	require.NoError(t, f.repo.Create(ctx, n))
	require.NoError(t, f.repo.MarkRead(ctx, n.ID, f.bob.ID))

	got, err := f.repo.FindUnreadByRecipientID(ctx, f.bob.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestNotificationRepo_DeletedActorStillReturnsRowWithEmptyName(t *testing.T) {
	f := newNotifFixture(t)
	ctx := context.Background()
	require.NoError(t, f.repo.Create(ctx, &domain.Notification{
		RecipientID: f.bob.ID, ActorID: 9999, Kind: domain.NotificationKindTaskAssigned, IssueID: f.issue.ID,
	}))

	got, err := f.repo.FindUnreadByRecipientID(ctx, f.bob.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "", got[0].ActorName)
	assert.Equal(t, "APOL-1", got[0].IssueKey)
}

func TestNotificationRepo_LegacyRowReadsAsMention(t *testing.T) {
	f := newNotifFixture(t)
	ctx := context.Background()
	// A row written before the kind and project_id columns existed.
	require.NoError(t, f.db.Exec(
		"INSERT INTO notifications (recipient_id, actor_id, issue_id, comment_id, read, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
		f.bob.ID, f.actor.ID, f.issue.ID, 3, false).Error)

	got, err := f.repo.FindUnreadByRecipientID(ctx, f.bob.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, domain.NotificationKindMention, got[0].Kind)
	assert.Equal(t, "APOL", got[0].ProjectKey)
}
