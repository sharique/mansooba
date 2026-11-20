package service_test

// With real repositories and the real services: a deleted task's key stays in
// the activity history, no later task takes it, and a task's key never changes.

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type numberingFixture struct {
	db      *gorm.DB
	svc     service.IssueService
	project *domain.Project
	userID  uint
}

func newNumberingFixture(t *testing.T) *numberingFixture {
	t.Helper()
	db := testdb.New(t)
	ctx := context.Background()
	users := repository.NewUserRepository(db)
	projects := repository.NewProjectRepository(db)
	members := repository.NewProjectMemberRepository(db)
	issues := repository.NewIssueRepository(db)
	sprints := repository.NewSprintRepository(db)
	activity := service.NewActivityService(repository.NewActivityRepository(db), users, issues, members)

	user := &domain.User{Name: "Ada", Email: "ada@example.com", Username: "ada", Password: "h", IsActive: true}
	require.NoError(t, users.Create(ctx, user))
	project := &domain.Project{Key: "HIST", Name: "History", OwnerID: user.ID}
	require.NoError(t, projects.Create(ctx, project))
	require.NoError(t, members.Create(ctx, &domain.ProjectMember{ProjectID: project.ID, UserID: user.ID, Role: domain.RoleAdmin}))

	return &numberingFixture{db: db, project: project, userID: user.ID,
		svc: service.NewIssueService(issues, projects, members, activity, users, sprints)}
}

func (f *numberingFixture) create(t *testing.T, title string) *dto.IssueResponse {
	t.Helper()
	resp, err := f.svc.Create(context.Background(), "HIST", f.userID, dto.CreateIssueRequest{Title: title, Type: domain.IssueTypeTask, Priority: domain.IssuePriorityLow})
	require.NoError(t, err)
	return resp
}

func TestTaskNumber_ADeletedTasksKeyStaysInTheHistoryAndIsNeverTakenAgain(t *testing.T) {
	f := newNumberingFixture(t)
	ctx := context.Background()
	f.create(t, "one")
	f.create(t, "two")
	three := f.create(t, "three")
	require.Equal(t, "HIST-3", three.Key)

	require.NoError(t, f.svc.Delete(ctx, "HIST", three.ID, f.userID))

	var deleted []domain.ActivityEvent
	require.NoError(t, f.db.Where("kind = ?", domain.ActivityIssueDeleted).Find(&deleted).Error)
	require.Len(t, deleted, 1)
	assert.Equal(t, "HIST-3", deleted[0].IssueKey, "the history keeps the key the deleted task had")

	four := f.create(t, "four")
	assert.Equal(t, "HIST-4", four.Key)
	var sameKey int64
	require.NoError(t, f.db.Model(&domain.Issue{}).Where(&domain.Issue{ProjectID: f.project.ID, Key: "HIST-3"}).Count(&sameKey).Error)
	assert.Zero(t, sameKey, "no live task may carry the key named in the history")
}

func TestTaskNumber_AKeyNeverChangesWhenTheTaskIsEdited(t *testing.T) {
	f := newNumberingFixture(t)
	ctx := context.Background()
	task := f.create(t, "original title")
	counterBefore := func() uint {
		var p domain.Project
		require.NoError(t, f.db.First(&p, f.project.ID).Error)
		return p.TaskCounter
	}
	before := counterBefore()

	str := func(s string) *string { return &s }
	sprint := &domain.Sprint{ProjectID: f.project.ID, Name: "S1"}
	require.NoError(t, repository.NewSprintRepository(f.db).Create(ctx, sprint))
	updates := []dto.UpdateIssueRequest{
		{Title: str("a new title")},
		{Status: str(domain.IssueStatusInProgress)},
		{Priority: str(domain.IssuePriorityCritical)},
		{AssigneeID: &f.userID},
		{SprintID: &sprint.ID},
	}
	for _, u := range updates {
		resp, err := f.svc.Update(ctx, "HIST", task.ID, f.userID, u)
		require.NoError(t, err)
		assert.Equal(t, "HIST-1", resp.Key, "an edit must not change the key")
	}
	assert.Equal(t, before, counterBefore(), "editing a task never touches the project's counter")
}
