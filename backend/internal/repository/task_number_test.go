package repository_test

// Task numbers: a project issues each number once and never again, whatever is
// deleted. The number comes from the project's counter and is allocated in the
// same transaction as the insert.

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// numberedIssue is a task as the application creates it: no Key, so the
// repository issues the next number.
func numberedIssue(projectID uint, title string) *domain.Issue {
	return &domain.Issue{
		ProjectID: projectID, Title: title, Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo,
		Priority: domain.IssuePriorityLow, ReporterID: 1,
	}
}

func counterOf(t *testing.T, db *gorm.DB, projectID uint) uint {
	t.Helper()
	var p domain.Project
	require.NoError(t, db.First(&p, projectID).Error)
	return p.TaskCounter
}

func mustCreate(t *testing.T, repo domain.IssueRepository, issue *domain.Issue) *domain.Issue {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), issue))
	return issue
}

func TestTaskNumber_DeletingAMiddleTaskDoesNotBreakTheNextCreate(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	one, two, three := mustCreate(t, repo, numberedIssue(1, "one")), mustCreate(t, repo, numberedIssue(1, "two")), mustCreate(t, repo, numberedIssue(1, "three"))
	assert.Equal(t, []string{"P1-1", "P1-2", "P1-3"}, []string{one.Key, two.Key, three.Key})

	require.NoError(t, repo.Delete(ctx, two.ID))
	four := mustCreate(t, repo, numberedIssue(1, "four"))
	assert.Equal(t, "P1-4", four.Key, "the next number is above every number issued, not the number of tasks plus one")
}

func TestTaskNumber_DeletingTheNewestTaskDoesNotHandItsNumberOutAgain(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	mustCreate(t, repo, numberedIssue(1, "one"))
	two := mustCreate(t, repo, numberedIssue(1, "two"))
	require.NoError(t, repo.Delete(ctx, two.ID))

	three := mustCreate(t, repo, numberedIssue(1, "three"))
	assert.Equal(t, "P1-3", three.Key, "P1-2 belonged to a task that was deleted; it must not come back")
}

func TestTaskNumber_DeletingEveryTaskContinuesTheNumbering(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	first, second := mustCreate(t, repo, numberedIssue(1, "a")), mustCreate(t, repo, numberedIssue(1, "b"))
	require.NoError(t, repo.Delete(ctx, first.ID))
	require.NoError(t, repo.Delete(ctx, second.ID))

	next := mustCreate(t, repo, numberedIssue(1, "c"))
	assert.Equal(t, "P1-3", next.Key)
	assert.Equal(t, uint(3), counterOf(t, db, 1))
}

func TestTaskNumber_EachProjectCountsOnItsOwn(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	for i := 0; i < 5; i++ {
		mustCreate(t, repo, numberedIssue(1, "p1"))
	}
	assert.Equal(t, "P2-1", mustCreate(t, repo, numberedIssue(2, "p2")).Key)
	assert.Equal(t, "P1-6", mustCreate(t, repo, numberedIssue(1, "p1 again")).Key)
	assert.Equal(t, uint(1), counterOf(t, db, 2))
}

// A create that fails after the number was taken rolls the number back: no
// gap, and the number is still free for the next task.
func TestTaskNumber_AFailedCreateLeavesTheCounterWhereItWas(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	mustCreate(t, repo, numberedIssue(1, "ok"))
	require.Equal(t, uint(1), counterOf(t, db, 1))

	missingSprint := uint(9999)
	bad := numberedIssue(1, "points at a sprint that does not exist")
	bad.SprintID = &missingSprint // violates the foreign key, after the number was taken
	require.Error(t, repo.Create(context.Background(), bad))

	assert.Equal(t, uint(1), counterOf(t, db, 1), "the failed create must not use up a number")
	assert.Equal(t, "P1-2", mustCreate(t, repo, numberedIssue(1, "next")).Key)
}

func TestTaskNumber_ACreateInAMissingProjectIsNotFound(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	err := repo.Create(context.Background(), numberedIssue(999, "nowhere"))
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Fixtures set a key by hand; such a task is inserted as given and the counter
// does not move.
func TestTaskNumber_AnExplicitKeyIsInsertedAsGivenWithoutMovingTheCounter(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	fixture := numberedIssue(1, "fixture")
	fixture.Key = "FIX-77"
	mustCreate(t, repo, fixture)

	assert.Equal(t, "FIX-77", fixture.Key)
	assert.Equal(t, uint(0), counterOf(t, db, 1))
	assert.Equal(t, "P1-1", mustCreate(t, repo, numberedIssue(1, "numbered")).Key)
}

func TestTaskNumber_TheCreatedTaskCarriesItsIDAndKey(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	created := mustCreate(t, repo, numberedIssue(1, "x"))
	assert.NotZero(t, created.ID)
	assert.Equal(t, "P1-1", created.Key)

	stored, err := repo.FindByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "P1-1", stored.Key)
}

// A project that is deleted and created again under the same key is a new
// project: its numbering starts at 1 (the promise is per project).
func TestTaskNumber_ARecreatedProjectStartsAtOne(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	projects := repository.NewProjectRepository(db)
	repo := repository.NewIssueRepository(db)

	old := &domain.Project{Key: "REUSE", Name: "Old", OwnerID: 1}
	require.NoError(t, projects.Create(ctx, old))
	a, b := mustCreate(t, repo, numberedIssue(old.ID, "a")), mustCreate(t, repo, numberedIssue(old.ID, "b"))
	assert.Equal(t, "REUSE-2", b.Key)
	require.NoError(t, repo.Delete(ctx, a.ID))
	require.NoError(t, repo.Delete(ctx, b.ID))
	require.NoError(t, projects.DeleteIfEmpty(ctx, old.ID))

	fresh := &domain.Project{Key: "REUSE", Name: "New", OwnerID: 1}
	require.NoError(t, projects.Create(ctx, fresh))
	first := mustCreate(t, repo, numberedIssue(fresh.ID, "first"))
	assert.Equal(t, "REUSE-1", first.Key)
	assert.Equal(t, uint(1), counterOf(t, db, fresh.ID))
}
