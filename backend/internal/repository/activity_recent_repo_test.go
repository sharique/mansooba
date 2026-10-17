package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type recentEnv struct {
	db     *gorm.DB
	issues domain.IssueRepository
	acts   domain.ActivityRepository
}

func newRecentEnv(t *testing.T) *recentEnv {
	t.Helper()
	db := overviewTestDB(t)
	return &recentEnv{db: db, issues: repository.NewIssueRepository(db), acts: repository.NewActivityRepository(db)}
}

func (e *recentEnv) issue(t *testing.T, key string, projectID uint, sprint *uint) *domain.Issue {
	t.Helper()
	i := &domain.Issue{
		Key: key, ProjectID: projectID, Title: key, Type: domain.IssueTypeTask,
		Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityMedium, ReporterID: 1, SprintID: sprint,
	}
	require.NoError(t, e.issues.Create(context.Background(), i))
	return i
}

func (e *recentEnv) event(t *testing.T, issueID uint, kind, oldV, newV string, at time.Time) *domain.ActivityEvent {
	t.Helper()
	ev := &domain.ActivityEvent{IssueID: issueID, ActorID: 1, Kind: kind, OldValue: oldV, NewValue: newV, CreatedAt: at}
	require.NoError(t, e.acts.Create(context.Background(), ev))
	return ev
}

func ids(evs []*domain.ActivityEvent) []uint {
	out := make([]uint, 0, len(evs))
	for _, ev := range evs {
		out = append(out, ev.ID)
	}
	return out
}

func TestActivityRepository_FindRecentByProject_EventRules(t *testing.T) {
	e := newRecentEnv(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	since := now.Add(-120 * time.Hour)
	i := e.issue(t, "P-1", 1, nil)

	toProgress := e.event(t, i.ID, domain.ActivityStatusChanged, "todo", "in_progress", now.Add(-1*time.Hour))
	toReview := e.event(t, i.ID, domain.ActivityStatusChanged, "in_progress", "in_review", now.Add(-2*time.Hour))
	toDone := e.event(t, i.ID, domain.ActivityStatusChanged, "in_review", "done", now.Add(-3*time.Hour))
	reopened := e.event(t, i.ID, domain.ActivityStatusChanged, "done", "in_review", now.Add(-4*time.Hour)) // backward move still counts
	comment := e.event(t, i.ID, domain.ActivityCommentAdded, "", "", now.Add(-5*time.Hour))
	// None of these qualify.
	e.event(t, i.ID, domain.ActivityStatusChanged, "in_progress", "todo", now.Add(-6*time.Hour))
	e.event(t, i.ID, domain.ActivityStatusChanged, "todo", "backlog", now.Add(-6*time.Hour))
	e.event(t, i.ID, domain.ActivityStatusChanged, "done", "done", now.Add(-6*time.Hour))
	for _, k := range []string{
		domain.ActivityAssigneeChanged, domain.ActivityPriorityChanged, domain.ActivitySprintChanged,
		domain.ActivityStoryPointsChanged, domain.ActivityLabelAdded, domain.ActivityLabelRemoved,
		domain.ActivityAttachmentAdded, domain.ActivityAttachmentRemoved,
	} {
		e.event(t, i.ID, k, "a", "in_progress", now.Add(-7*time.Hour))
	}

	got, err := e.acts.FindRecentByProject(ctx, 1, since, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []uint{toProgress.ID, toReview.ID, toDone.ID, reopened.ID, comment.ID}, ids(got))
}

func TestActivityRepository_FindRecentByProject_WindowAndProject(t *testing.T) {
	e := newRecentEnv(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	since := now.Add(-120 * time.Hour)
	mine := e.issue(t, "P-1", 1, nil)
	theirs := e.issue(t, "Q-1", 2, nil)

	inside := e.event(t, mine.ID, domain.ActivityCommentAdded, "", "", since.Add(time.Minute))
	e.event(t, mine.ID, domain.ActivityCommentAdded, "", "", since.Add(-time.Minute)) // just outside
	exact := e.event(t, mine.ID, domain.ActivityCommentAdded, "", "", since)          // boundary is inclusive
	e.event(t, theirs.ID, domain.ActivityCommentAdded, "", "", now)                   // other project
	e.event(t, 9999, domain.ActivityCommentAdded, "", "", now)                        // deleted issue

	got, err := e.acts.FindRecentByProject(ctx, 1, since, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []uint{inside.ID, exact.ID}, ids(got))
}

func TestActivityRepository_FindRecentByProject_SprintScope(t *testing.T) {
	e := newRecentEnv(t)
	ctx := context.Background()
	now := time.Now()
	since := now.Add(-time.Hour * 120)
	sp, other := uint(7), uint(8)
	inSprint := e.issue(t, "P-1", 1, &sp)
	elsewhere := e.issue(t, "P-2", 1, &other)
	backlog := e.issue(t, "P-3", 1, nil)

	a := e.event(t, inSprint.ID, domain.ActivityCommentAdded, "", "", now.Add(-time.Minute))
	e.event(t, elsewhere.ID, domain.ActivityCommentAdded, "", "", now.Add(-time.Minute))
	e.event(t, backlog.ID, domain.ActivityCommentAdded, "", "", now.Add(-time.Minute))

	got, err := e.acts.FindRecentByProject(ctx, 1, since, &sp, 10)
	require.NoError(t, err)
	assert.Equal(t, []uint{a.ID}, ids(got))

	// A task that leaves the sprint no longer shows under the sprint scope.
	inSprint.SprintID = nil
	require.NoError(t, e.issues.Update(ctx, inSprint))
	got, err = e.acts.FindRecentByProject(ctx, 1, since, &sp, 10)
	require.NoError(t, err)
	assert.Empty(t, got)
	all, err := e.acts.FindRecentByProject(ctx, 1, since, nil, 10)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestActivityRepository_FindRecentByProject_NewestFirstAndLimit(t *testing.T) {
	e := newRecentEnv(t)
	now := time.Now()
	i := e.issue(t, "P-1", 1, nil)
	var newest *domain.ActivityEvent
	for n := 0; n < 12; n++ {
		newest = e.event(t, i.ID, domain.ActivityCommentAdded, "", "", now.Add(-time.Duration(12-n)*time.Minute))
	}
	got, err := e.acts.FindRecentByProject(context.Background(), 1, now.Add(-time.Hour), nil, 10)
	require.NoError(t, err)
	require.Len(t, got, 10)
	assert.Equal(t, newest.ID, got[0].ID)
	for n := 1; n < len(got); n++ {
		assert.True(t, !got[n].CreatedAt.After(got[n-1].CreatedAt), "not newest first")
	}
}

func TestActivityRepository_FindRecentByProject_LargeProjectStaysFast(t *testing.T) {
	e := newRecentEnv(t)
	now := time.Now()
	sp := uint(1)
	var issueIDs []uint
	for n := 0; n < 1000; n++ {
		i := e.issue(t, fmt.Sprintf("P-%d", n), 1, &sp)
		issueIDs = append(issueIDs, i.ID)
	}
	events := make([]*domain.ActivityEvent, 0, 3000)
	for n := 0; n < 3000; n++ {
		events = append(events, &domain.ActivityEvent{
			IssueID: issueIDs[n%1000], ActorID: 1, Kind: domain.ActivityCommentAdded,
			CreatedAt: now.Add(-time.Duration(n) * time.Minute),
		})
	}
	require.NoError(t, e.db.CreateInBatches(events, 500).Error)

	start := time.Now()
	got, err := e.acts.FindRecentByProject(context.Background(), 1, now.Add(-120*time.Hour), &sp, 10)
	elapsed := time.Since(start)
	require.NoError(t, err)
	assert.Len(t, got, 10)
	assert.Less(t, elapsed, 250*time.Millisecond)
}
