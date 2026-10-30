package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The overview reads real data and names people, so it is member-only. These
// tests use real repositories on in-memory SQLite and a fixed clock.

var overviewNow = time.Date(2026, 10, 17, 15, 0, 0, 0, time.UTC)

type overviewEnv struct {
	ctx                       context.Context
	db                        *gorm.DB
	svc                       service.OverviewService
	issues                    domain.IssueRepository
	sprints                   domain.SprintRepository
	acts                      domain.ActivityRepository
	project                   *domain.Project
	member, outsider, otherPM uint
}

func newOverviewEnv(t *testing.T) *overviewEnv {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: ":memory:"})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))

	users := repository.NewUserRepository(db)
	mk := func(name, username string) uint {
		u := &domain.User{Name: name, Email: username + "@example.com", Username: username, Password: "x", IsActive: true}
		require.NoError(t, users.Create(ctx, u))
		return u.ID
	}
	e := &overviewEnv{
		ctx: ctx, db: db,
		issues:  repository.NewIssueRepository(db),
		sprints: repository.NewSprintRepository(db),
		acts:    repository.NewActivityRepository(db),
	}
	e.member, e.outsider, e.otherPM = mk("Mia Member", "mia"), mk("Olly Outsider", "olly"), mk("Pat Other", "pat")

	projects := repository.NewProjectRepository(db)
	members := repository.NewProjectMemberRepository(db)
	e.project = &domain.Project{Key: "OVR", Name: "Overview", OwnerID: e.member}
	require.NoError(t, projects.Create(ctx, e.project))
	require.NoError(t, members.Create(ctx, &domain.ProjectMember{ProjectID: e.project.ID, UserID: e.member, Role: "admin"}))
	other := &domain.Project{Key: "OTH", Name: "Other", OwnerID: e.otherPM}
	require.NoError(t, projects.Create(ctx, other))
	require.NoError(t, members.Create(ctx, &domain.ProjectMember{ProjectID: other.ID, UserID: e.otherPM, Role: "admin"}))

	e.svc = service.NewOverviewServiceWithClock(projects, members, e.sprints, e.issues, e.acts, users, func() time.Time { return overviewNow })
	return e
}

func (e *overviewEnv) sprint(t *testing.T, status string, start, end *time.Time) *domain.Sprint {
	t.Helper()
	s := &domain.Sprint{ProjectID: e.project.ID, Name: "Sprint " + status, Status: status, StartDate: start, EndDate: end}
	require.NoError(t, e.sprints.Create(e.ctx, s))
	return s
}

func (e *overviewEnv) issue(t *testing.T, key, status, priority string, points *int, sprint *uint, assignee *uint) *domain.Issue {
	t.Helper()
	i := &domain.Issue{
		Key: key, ProjectID: e.project.ID, Title: "Title " + key, Type: domain.IssueTypeTask,
		Status: status, Priority: priority, ReporterID: e.member, SprintID: sprint, StoryPoints: points, AssigneeID: assignee,
	}
	require.NoError(t, e.issues.Create(e.ctx, i))
	return i
}

func pts(n int) *int { return &n }
func day(y int, m time.Month, d int) *time.Time {
	v := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &v
}

func TestOverviewService_OnlyProjectMembers(t *testing.T) {
	e := newOverviewEnv(t)
	e.sprint(t, domain.SprintStatusActive, nil, nil)

	for _, caller := range []uint{e.outsider, e.otherPM} {
		_, err := e.svc.Sprint(e.ctx, "OVR", caller)
		assert.ErrorIs(t, err, domain.ErrForbidden)
		_, err = e.svc.Tasks(e.ctx, "OVR", caller)
		assert.ErrorIs(t, err, domain.ErrForbidden)
		_, err = e.svc.Activity(e.ctx, "OVR", caller, "all")
		assert.ErrorIs(t, err, domain.ErrForbidden)
	}
	_, err := e.svc.Sprint(e.ctx, "NOPE", e.member)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	_, err = e.svc.Tasks(e.ctx, "NOPE", e.member)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	_, err = e.svc.Activity(e.ctx, "NOPE", e.member, "all")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOverviewService_Sprint_NoActiveSprint(t *testing.T) {
	e := newOverviewEnv(t)
	e.sprint(t, domain.SprintStatusPlanning, nil, nil)
	e.sprint(t, domain.SprintStatusCompleted, nil, nil)

	got, err := e.svc.Sprint(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	assert.Nil(t, got.Sprint)
}

func TestOverviewService_Sprint_CountsFourGroups(t *testing.T) {
	e := newOverviewEnv(t)
	sp := e.sprint(t, domain.SprintStatusActive, day(2026, 10, 12), day(2026, 10, 26))
	id := sp.ID
	e.issue(t, "OVR-1", domain.IssueStatusTodo, domain.IssuePriorityLow, pts(2), &id, nil)
	e.issue(t, "OVR-2", domain.IssueStatusBacklog, domain.IssuePriorityLow, pts(1), &id, nil) // counts as To Do
	e.issue(t, "OVR-3", domain.IssueStatusInProgress, domain.IssuePriorityLow, pts(3), &id, nil)
	e.issue(t, "OVR-4", domain.IssueStatusInReview, domain.IssuePriorityLow, nil, &id, nil) // no points: counts as a task only
	e.issue(t, "OVR-5", domain.IssueStatusDone, domain.IssuePriorityLow, pts(5), &id, nil)
	e.issue(t, "OVR-6", domain.IssueStatusDone, domain.IssuePriorityLow, pts(8), &id, nil)
	e.issue(t, "OVR-7", domain.IssueStatusTodo, domain.IssuePriorityLow, pts(40), nil, nil) // backlog: not counted

	got, err := e.svc.Sprint(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	s := got.Sprint
	require.NotNil(t, s)
	assert.Equal(t, sp.ID, s.ID)
	assert.Equal(t, sp.Name, s.Name)
	assert.Equal(t, 2, s.Todo.Tasks)
	assert.Equal(t, 3, s.Todo.Points)
	assert.Equal(t, 1, s.InProgress.Tasks)
	assert.Equal(t, 3, s.InProgress.Points)
	assert.Equal(t, 1, s.InReview.Tasks)
	assert.Equal(t, 0, s.InReview.Points)
	assert.Equal(t, 2, s.Done.Tasks)
	assert.Equal(t, 13, s.Done.Points)
	assert.Equal(t, 6, s.Total.Tasks)
	assert.Equal(t, 19, s.Total.Points)
	assert.False(t, s.IsOverdue)
	require.NotNil(t, s.StartDate)
	require.NotNil(t, s.EndDate)
}

func TestOverviewService_Sprint_ZeroProgressIsNotAnError(t *testing.T) {
	e := newOverviewEnv(t)
	e.sprint(t, domain.SprintStatusActive, day(2026, 10, 17), day(2026, 10, 31)) // just started, no tasks
	got, err := e.svc.Sprint(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	require.NotNil(t, got.Sprint)
	assert.Zero(t, got.Sprint.Total.Tasks)
	assert.Zero(t, got.Sprint.Done.Tasks)

	id := got.Sprint.ID
	e.issue(t, "OVR-1", domain.IssueStatusTodo, domain.IssuePriorityLow, nil, &id, nil) // started, nothing done yet
	got, err = e.svc.Sprint(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Sprint.Total.Tasks)
	assert.Zero(t, got.Sprint.Done.Tasks)
}

func TestOverviewService_Sprint_Overdue(t *testing.T) {
	cases := []struct {
		name string
		end  *time.Time
		want bool
	}{
		{"ended yesterday", day(2026, 10, 16), true},
		{"ends today (not overdue during the end day)", day(2026, 10, 17), false},
		{"ends later today", func() *time.Time { v := time.Date(2026, 10, 17, 23, 59, 0, 0, time.UTC); return &v }(), false},
		{"ends tomorrow", day(2026, 10, 18), false},
		{"no end date", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newOverviewEnv(t)
			e.sprint(t, domain.SprintStatusActive, day(2026, 10, 1), tc.end)
			got, err := e.svc.Sprint(e.ctx, "OVR", e.member)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Sprint.IsOverdue)
		})
	}
}

func TestOverviewService_Tasks(t *testing.T) {
	e := newOverviewEnv(t)

	none, err := e.svc.Tasks(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	assert.False(t, none.SprintActive)
	assert.Empty(t, none.Items)

	sp := e.sprint(t, domain.SprintStatusActive, nil, nil)
	id := sp.ID
	e.issue(t, "OVR-1", domain.IssueStatusInProgress, domain.IssuePriorityCritical, nil, &id, &e.member)
	e.issue(t, "OVR-2", domain.IssueStatusInReview, domain.IssuePriorityCritical, nil, &id, nil)
	e.issue(t, "OVR-3", domain.IssueStatusDone, domain.IssuePriorityCritical, nil, &id, nil)
	e.issue(t, "OVR-4", domain.IssueStatusTodo, domain.IssuePriorityHigh, nil, nil, nil)

	got, err := e.svc.Tasks(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	assert.True(t, got.SprintActive)
	require.Len(t, got.Items, 2)
	assert.Equal(t, "OVR-2", got.Items[0].Key)
	assert.Equal(t, domain.IssueStatusInReview, got.Items[0].Status)
	assert.Nil(t, got.Items[0].Assignee)
	assert.Equal(t, "OVR-1", got.Items[1].Key)
	require.NotNil(t, got.Items[1].Assignee)
	assert.Equal(t, e.member, got.Items[1].Assignee.ID)
	assert.Equal(t, "Mia Member", got.Items[1].Assignee.Name)
	assert.NotZero(t, got.Items[1].IssueID)
	assert.Equal(t, "Title OVR-1", got.Items[1].Title)
}

func TestOverviewService_Tasks_AtMostTen(t *testing.T) {
	e := newOverviewEnv(t)
	sp := e.sprint(t, domain.SprintStatusActive, nil, nil)
	id := sp.ID
	for n := 0; n < 13; n++ {
		e.issue(t, "OVR-"+string(rune('a'+n)), domain.IssueStatusTodo, domain.IssuePriorityMedium, nil, &id, nil)
	}
	got, err := e.svc.Tasks(e.ctx, "OVR", e.member)
	require.NoError(t, err)
	assert.Len(t, got.Items, 10)
}

func TestOverviewService_Activity(t *testing.T) {
	e := newOverviewEnv(t)
	sp := e.sprint(t, domain.SprintStatusActive, nil, nil)
	id := sp.ID
	inSprint := e.issue(t, "OVR-1", domain.IssueStatusInReview, domain.IssuePriorityLow, nil, &id, nil)
	backlog := e.issue(t, "OVR-2", domain.IssueStatusTodo, domain.IssuePriorityLow, nil, nil, nil)

	rec := func(issue uint, actor uint, kind, oldV, newV string, ago time.Duration) {
		require.NoError(t, e.acts.Create(e.ctx, &domain.ActivityEvent{
			IssueID: issue, ActorID: actor, Kind: kind, OldValue: oldV, NewValue: newV, CreatedAt: overviewNow.Add(-ago),
		}))
	}
	rec(inSprint.ID, e.member, domain.ActivityStatusChanged, "in_progress", "in_review", time.Hour)
	rec(backlog.ID, e.otherPM, domain.ActivityCommentAdded, "", "", 2*time.Hour)
	rec(inSprint.ID, 9999, domain.ActivityCommentAdded, "", "", 3*time.Hour)                     // removed user
	rec(inSprint.ID, e.member, domain.ActivityCommentAdded, "", "", 119*time.Hour)               // inside the window
	rec(inSprint.ID, e.member, domain.ActivityCommentAdded, "", "", 121*time.Hour)               // outside the 120 hours
	rec(inSprint.ID, e.member, domain.ActivityPriorityChanged, "low", "high", time.Minute)       // not listed
	rec(inSprint.ID, e.member, domain.ActivityStatusChanged, "in_progress", "todo", time.Minute) // not listed

	all, err := e.svc.Activity(e.ctx, "OVR", e.member, "all")
	require.NoError(t, err)
	assert.True(t, all.SprintActive)
	require.Len(t, all.Items, 4)

	first := all.Items[0]
	assert.Equal(t, domain.ActivityStatusChanged, first.Kind)
	assert.Equal(t, domain.IssueStatusInReview, first.NewStatus)
	assert.Equal(t, "OVR-1", first.IssueKey)
	assert.Equal(t, "Title OVR-1", first.IssueTitle)
	assert.Equal(t, inSprint.ID, first.IssueID)
	assert.Equal(t, "Mia Member", first.Actor.Name)
	assert.Equal(t, overviewNow.Add(-time.Hour), first.CreatedAt.UTC())

	assert.Equal(t, domain.ActivityCommentAdded, all.Items[1].Kind)
	assert.Equal(t, "", all.Items[1].NewStatus)
	assert.Equal(t, "Pat Other", all.Items[1].Actor.Name)
	assert.Equal(t, "", all.Items[2].Actor.Name) // removed user: empty name, entry still listed
	assert.Equal(t, uint(9999), all.Items[2].Actor.ID)

	bySprint, err := e.svc.Activity(e.ctx, "OVR", e.member, "sprint")
	require.NoError(t, err)
	assert.True(t, bySprint.SprintActive)
	assert.Len(t, bySprint.Items, 3) // everything on OVR-1, not the backlog task's comment
	for _, it := range bySprint.Items {
		assert.Equal(t, "OVR-1", it.IssueKey)
	}
}

func TestOverviewService_Activity_DefaultScopeAndErrors(t *testing.T) {
	e := newOverviewEnv(t)
	i := e.issue(t, "OVR-1", domain.IssueStatusTodo, domain.IssuePriorityLow, nil, nil, nil)
	require.NoError(t, e.acts.Create(e.ctx, &domain.ActivityEvent{IssueID: i.ID, ActorID: e.member, Kind: domain.ActivityCommentAdded, CreatedAt: overviewNow.Add(-time.Hour)}))

	def, err := e.svc.Activity(e.ctx, "OVR", e.member, "")
	require.NoError(t, err)
	assert.Len(t, def.Items, 1)
	assert.False(t, def.SprintActive)

	_, err = e.svc.Activity(e.ctx, "OVR", e.member, "everything")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestOverviewService_Activity_SprintScopeWithoutSprint(t *testing.T) {
	e := newOverviewEnv(t)
	i := e.issue(t, "OVR-1", domain.IssueStatusTodo, domain.IssuePriorityLow, nil, nil, nil)
	require.NoError(t, e.acts.Create(e.ctx, &domain.ActivityEvent{IssueID: i.ID, ActorID: e.member, Kind: domain.ActivityCommentAdded, CreatedAt: overviewNow.Add(-time.Hour)}))

	got, err := e.svc.Activity(e.ctx, "OVR", e.member, "sprint")
	require.NoError(t, err)
	assert.False(t, got.SprintActive)
	assert.Empty(t, got.Items)
	assert.NotNil(t, got.Items, "items must marshal as [] not null")
}

func TestOverviewService_Activity_ListsDeletedTasks(t *testing.T) {
	e := newOverviewEnv(t)
	sp := e.sprint(t, domain.SprintStatusActive, nil, nil)
	id := sp.ID
	other := uint(9999)
	proj := e.issue(t, "OVR-1", domain.IssueStatusTodo, domain.IssuePriorityLow, nil, &id, nil).ProjectID

	del := func(issueID uint, key string, sprint *uint, ago time.Duration) {
		require.NoError(t, e.acts.Create(e.ctx, &domain.ActivityEvent{
			IssueID: issueID, ActorID: e.member, Kind: domain.ActivityIssueDeleted, CreatedAt: overviewNow.Add(-ago),
			ProjectID: proj, IssueKey: key, IssueTitle: "Title " + key, SprintID: sprint,
		}))
	}
	del(800, "OVR-8", &id, time.Hour)      // was in the current sprint
	del(801, "OVR-9", &other, 2*time.Hour) // was in another sprint
	del(802, "OVR-7", nil, 3*time.Hour)    // had no sprint

	all, err := e.svc.Activity(e.ctx, "OVR", e.member, "all")
	require.NoError(t, err)
	require.Len(t, all.Items, 3)
	first := all.Items[0]
	assert.Equal(t, domain.ActivityIssueDeleted, first.Kind)
	assert.Equal(t, "OVR-8", first.IssueKey)
	assert.Equal(t, "Title OVR-8", first.IssueTitle)
	assert.Equal(t, "Mia Member", first.Actor.Name)
	assert.Equal(t, "", first.NewStatus)

	bySprint, err := e.svc.Activity(e.ctx, "OVR", e.member, "sprint")
	require.NoError(t, err)
	require.Len(t, bySprint.Items, 1)
	assert.Equal(t, "OVR-8", bySprint.Items[0].IssueKey)
}
