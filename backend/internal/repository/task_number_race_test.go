package repository_test

// Simultaneous creates: every task gets its own number, a delete running beside
// a create changes nothing, and a create racing a project delete never leaves a
// task without its project. The controls show the old rules failing under the
// same conditions, so a green run means something.

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func keyNumber(t *testing.T, key string) int {
	t.Helper()
	n, err := strconv.Atoi(key[strings.LastIndex(key, "-")+1:])
	require.NoError(t, err)
	return n
}

func TestTaskNumberRace_TwentySimultaneousCreatesGetTwentyDifferentNumbers(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	for round := 0; round < 20; round++ {
		issues := make([]*domain.Issue, 20)
		fns := make([]func() error, 20)
		for i := range fns {
			issues[i] = numberedIssue(1, "racer")
			fns[i] = func() error { return repo.Create(ctx, issues[i]) }
		}
		for i, err := range runTogether(fns...) {
			require.NoError(t, err, "round %d, create %d", round, i)
		}
		seen := map[string]bool{}
		for _, issue := range issues {
			require.False(t, seen[issue.Key], "round %d: %s issued twice", round, issue.Key)
			seen[issue.Key] = true
		}
	}
	assert.Equal(t, uint(400), counterOf(t, db, 1))
}

func TestTaskNumberRace_ACreateBesideADeleteGetsANumberAboveEveryEarlierOne(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	for round := 0; round < 20; round++ {
		before := counterOf(t, db, 1)
		victim := mustCreate(t, repo, numberedIssue(1, "victim"))
		created := numberedIssue(1, "new")
		errs := runTogether(
			func() error { return repo.Delete(ctx, victim.ID) },
			func() error { return repo.Create(ctx, created) },
		)
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])
		assert.Greater(t, uint(keyNumber(t, created.Key)), before+1, "round %d: %s must be above the deleted task's number", round, created.Key)
	}
}

func TestTaskNumberRace_ACreateBesideAProjectDeleteNeverLeavesATaskWithoutItsProject(t *testing.T) {
	ctx := context.Background()
	for round := 0; round < 20; round++ {
		db := newTestDB(t)
		projects := repository.NewProjectRepository(db)
		repo := repository.NewIssueRepository(db)
		p := &domain.Project{Key: "RACE", Name: "Race", OwnerID: 1}
		require.NoError(t, projects.Create(ctx, p))

		errs := runTogether(
			func() error { return projects.DeleteIfEmpty(ctx, p.ID) },
			func() error { return repo.Create(ctx, numberedIssue(p.ID, "late")) },
		)
		if errs[1] != nil {
			assert.ErrorIs(t, errs[1], domain.ErrNotFound, "round %d", round)
		}
		assert.Zero(t, orphans(t, db, "issues"), "round %d left a task without a project (delete: %v, create: %v)", round, errs[0], errs[1])
	}
}

// ── controls: the old rules, which must lose ─────────────────────────────────

// countPlusOne is the allocation this feature replaced.
func countPlusOne(db *gorm.DB, issue *domain.Issue) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&domain.Issue{}).Where("project_id = ?", issue.ProjectID).Count(&n).Error; err != nil {
			return err
		}
		issue.Key = "P1-" + strconv.FormatInt(n+1, 10)
		return tx.Create(issue).Error
	})
}

func TestTaskNumberRaceControl_CountPlusOneFailsUnderTheSameDeleteSequence(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	mustCreate(t, repo, numberedIssue(1, "one"))
	two := mustCreate(t, repo, numberedIssue(1, "two"))
	mustCreate(t, repo, numberedIssue(1, "three"))
	require.NoError(t, repo.Delete(ctx, two.ID))
	assert.Error(t, countPlusOne(db, numberedIssue(1, "four")), "the old rule collides with P1-3")
}

func TestTaskNumberRaceControl_CountPlusOneUnderConcurrencyProducesAnErrorOrADuplicate(t *testing.T) {
	db := newDBWithProjects(t)
	failures := 0
	for round := 0; round < 5; round++ {
		fns := make([]func() error, 10)
		for i := range fns {
			fns[i] = func() error { return countPlusOne(db, numberedIssue(1, "racer")) }
		}
		for _, err := range runTogether(fns...) {
			if err != nil {
				failures++
			}
		}
	}
	assert.Positive(t, failures, "the control must collide; if it never does, the race tests prove nothing")
}
