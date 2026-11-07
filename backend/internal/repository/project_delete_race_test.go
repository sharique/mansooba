package repository_test

// A project delete racing an insert into the same project. Deleting takes a row
// lock on the project, counts its tasks and removes it; an insert that did not
// take the same lock could land between the count and the delete and leave a row
// that points at a project that no longer exists. Each test gets its own database
// from internal/testdb.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type projectChild struct {
	name   string
	table  string
	create func(ctx context.Context, db *gorm.DB, projectID uint, n int) error
	// blocksDelete: a task makes the delete refuse; the others are removed with the project.
	blocksDelete bool
}

var projectChildren = []projectChild{
	{"task", "issues", func(ctx context.Context, db *gorm.DB, p uint, n int) error {
		return repository.NewIssueRepository(db).Create(ctx, &domain.Issue{
			Key: fmt.Sprintf("R-%d", n), ProjectID: p, Title: "t", Type: domain.IssueTypeTask,
			Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: 1,
		})
	}, true},
	{"sprint", "sprints", func(ctx context.Context, db *gorm.DB, p uint, n int) error {
		return repository.NewSprintRepository(db).Create(ctx, &domain.Sprint{ProjectID: p, Name: fmt.Sprintf("S%d", n)})
	}, false},
	{"label", "labels", func(ctx context.Context, db *gorm.DB, p uint, n int) error {
		return repository.NewLabelRepository(db).Create(ctx, &domain.Label{ProjectID: p, Name: fmt.Sprintf("L%d", n), Color: "#fff"})
	}, false},
	{"member", "project_members", func(ctx context.Context, db *gorm.DB, p uint, n int) error {
		return repository.NewProjectMemberRepository(db).Create(ctx, &domain.ProjectMember{ProjectID: p, UserID: uint(n) + 100, Role: domain.RoleMember})
	}, false},
}

func orphans(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM `+table+` WHERE project_id NOT IN (SELECT id FROM projects)`).Scan(&n).Error)
	return n
}

func TestProjectDelete_RacingAnInsertNeverLeavesARowWithoutItsProject(t *testing.T) {
	for _, child := range projectChildren {
		t.Run(child.name, func(t *testing.T) {
			db := testdb.New(t)
			ctx := context.Background()
			projects := repository.NewProjectRepository(db)

			deleted, insertRefused, deleteRefused := 0, 0, 0
			for round := 0; round < raceRounds; round++ {
				p := &domain.Project{Key: fmt.Sprintf("K%d", round), Name: "p", OwnerID: 1}
				require.NoError(t, projects.Create(ctx, p))

				errs := runTogether(
					func() error { return projects.DeleteIfEmpty(ctx, p.ID) },
					func() error { return child.create(ctx, db, p.ID, round) },
				)
				delErr, insErr := errs[0], errs[1]

				assert.Zero(t, orphans(t, db, child.table), "round %d left a %s without a project (delete: %v, insert: %v)", round, child.name, delErr, insErr)

				switch {
				case delErr == nil:
					deleted++
				case errors.Is(delErr, domain.ErrProjectNotEmpty):
					deleteRefused++
					assert.True(t, child.blocksDelete, "only a task may block the delete")
				default:
					t.Fatalf("round %d: unexpected delete error: %v", round, delErr)
				}
				if insErr != nil {
					assert.ErrorIs(t, insErr, domain.ErrNotFound, "an insert that lost the race is 'not found', nothing else")
					insertRefused++
				}
			}
			t.Logf("%s: project deleted %d times, delete refused %d, insert refused %d of %d rounds", child.name, deleted, deleteRefused, insertRefused, raceRounds)
		})
	}
}

// The control: the same two operations with no lock between them lose the
// invariant, so the test above is able to fail. The delete counts the tasks and
// pauses; the insert commits; the delete then removes the project.
func TestProjectDelete_ControlWithoutTheLockLeavesATaskBehind(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	p := &domain.Project{Key: "CTRL", Name: "p", OwnerID: 1}
	require.NoError(t, repository.NewProjectRepository(db).Create(ctx, p))

	counted := make(chan struct{})
	inserted := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var n int64
			if err := tx.Model(&domain.Issue{}).Where("project_id = ?", p.ID).Count(&n).Error; err != nil || n > 0 {
				return fmt.Errorf("count: %v (%d)", err, n)
			}
			close(counted)
			<-inserted // a task is created and committed in between
			return tx.Delete(&domain.Project{}, p.ID).Error
		})
	}()
	<-counted
	require.NoError(t, projectChildren[0].create(ctx, db, p.ID, 1))
	close(inserted)
	require.NoError(t, <-done)

	assert.Equal(t, int64(1), orphans(t, db, "issues"), "without the lock the task is left behind; the real code must prevent exactly this")
}
