package repository_test

// A long, deterministic mix of creates and deletes in one project: no key is
// ever issued twice, and the numbers only go up.

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskNumber_NoKeyIsEverIssuedTwiceOverAThousandOperations(t *testing.T) {
	db := newDBWithProjects(t)
	repo := repository.NewIssueRepository(db)
	ctx := context.Background()
	rng := rand.New(rand.NewSource(42)) // fixed seed: the same run every time

	issued := map[string]bool{} // every key the project has ever shown, live or deleted
	var live []*domain.Issue
	creates, lastNumber := 0, 0

	for op := 0; op < 1000; op++ {
		if len(live) > 0 && rng.Intn(10) < 4 { // delete a random live task, in any position
			i := rng.Intn(len(live))
			require.NoError(t, repo.Delete(ctx, live[i].ID))
			live = append(live[:i], live[i+1:]...)
			continue
		}
		issue := numberedIssue(1, "task "+strconv.Itoa(op))
		require.NoError(t, repo.Create(ctx, issue), "operation %d", op)
		creates++

		require.False(t, issued[issue.Key], "operation %d issued %s a second time", op, issue.Key)
		issued[issue.Key] = true

		n, err := strconv.Atoi(strings.TrimPrefix(issue.Key, "P1-"))
		require.NoError(t, err)
		require.Greater(t, n, lastNumber, "numbers must only go up (operation %d: %s)", op, issue.Key)
		lastNumber = n
		live = append(live, issue)
	}

	assert.Equal(t, uint(creates), counterOf(t, db, 1), "the counter equals the number of tasks ever created")
	assert.Equal(t, creates, len(issued))
	for _, issue := range live {
		assert.True(t, issued[issue.Key])
	}
}
