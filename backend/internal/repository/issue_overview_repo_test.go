package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// overviewTestDB is in-memory SQLite, or a throwaway PostgreSQL when
// OVERVIEW_PG_DSN is set (for example a postgres:17 container on a spare port),
// so the overview queries are also checked on the production database. The
// tables the overview reads are dropped and recreated first.
func overviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("OVERVIEW_PG_DSN")
	if dsn == "" {
		db := newTestDB(t)
		require.NoError(t, db.AutoMigrate(&domain.ActivityEvent{}))
		return db
	}
	db, err := database.Open(&config.Config{DBDriver: "postgres", DBDSN: dsn})
	require.NoError(t, err)
	require.NoError(t, db.Migrator().DropTable(&domain.ActivityEvent{}, &domain.Issue{}))
	require.NoError(t, db.AutoMigrate(&domain.Issue{}, &domain.ActivityEvent{}))
	return db
}

func seedOverviewIssue(t *testing.T, repo domain.IssueRepository, key string, sprintID *uint, status, priority string, points *int, updated time.Time) *domain.Issue {
	t.Helper()
	i := &domain.Issue{
		Key: key, ProjectID: 1, Title: key, Type: domain.IssueTypeTask,
		Status: status, Priority: priority, ReporterID: 1, SprintID: sprintID, StoryPoints: points,
	}
	require.NoError(t, repo.Create(context.Background(), i))
	// UpdatedAt is set by GORM on create; force the value the ordering test needs.
	i.UpdatedAt = updated
	require.NoError(t, repo.Update(context.Background(), i))
	return i
}

func intp(n int) *int { return &n }

func TestIssueRepository_CountByStatusForSprint(t *testing.T) {
	repo := repository.NewIssueRepository(overviewTestDB(t))
	ctx := context.Background()
	s1, s2 := uint(1), uint(2)
	now := time.Now()
	seedOverviewIssue(t, repo, "P-1", &s1, domain.IssueStatusTodo, domain.IssuePriorityLow, intp(3), now)
	seedOverviewIssue(t, repo, "P-2", &s1, domain.IssueStatusTodo, domain.IssuePriorityLow, nil, now) // no points
	seedOverviewIssue(t, repo, "P-3", &s1, domain.IssueStatusInProgress, domain.IssuePriorityLow, intp(5), now)
	seedOverviewIssue(t, repo, "P-4", &s1, domain.IssueStatusDone, domain.IssuePriorityLow, intp(8), now)
	seedOverviewIssue(t, repo, "P-5", &s2, domain.IssueStatusDone, domain.IssuePriorityLow, intp(13), now) // other sprint
	seedOverviewIssue(t, repo, "P-6", nil, domain.IssueStatusTodo, domain.IssuePriorityLow, intp(21), now) // backlog

	got, err := repo.CountByStatusForSprint(ctx, s1)
	require.NoError(t, err)
	byStatus := map[string]domain.StatusTotals{}
	for _, g := range got {
		byStatus[g.Status] = g
	}
	assert.Equal(t, domain.StatusTotals{Status: domain.IssueStatusTodo, Tasks: 2, Points: 3}, byStatus[domain.IssueStatusTodo])
	assert.Equal(t, domain.StatusTotals{Status: domain.IssueStatusInProgress, Tasks: 1, Points: 5}, byStatus[domain.IssueStatusInProgress])
	assert.Equal(t, domain.StatusTotals{Status: domain.IssueStatusDone, Tasks: 1, Points: 8}, byStatus[domain.IssueStatusDone])
	assert.Len(t, byStatus, 3)

	none, err := repo.CountByStatusForSprint(ctx, 99)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestIssueRepository_FindOpenBySprint_OrderAndFilter(t *testing.T) {
	repo := repository.NewIssueRepository(overviewTestDB(t))
	ctx := context.Background()
	sp, other := uint(1), uint(2)
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	// Seeded in a scrambled order on purpose.
	seedOverviewIssue(t, repo, "LOW-TODO", &sp, domain.IssueStatusTodo, domain.IssuePriorityLow, nil, base)
	seedOverviewIssue(t, repo, "CRIT-PROG", &sp, domain.IssueStatusInProgress, domain.IssuePriorityCritical, nil, base)
	seedOverviewIssue(t, repo, "HIGH-TODO", &sp, domain.IssueStatusTodo, domain.IssuePriorityHigh, nil, base)
	seedOverviewIssue(t, repo, "CRIT-REVIEW", &sp, domain.IssueStatusInReview, domain.IssuePriorityCritical, nil, base)
	seedOverviewIssue(t, repo, "CRIT-TODO-OLD", &sp, domain.IssueStatusTodo, domain.IssuePriorityCritical, nil, base)
	seedOverviewIssue(t, repo, "CRIT-TODO-NEW", &sp, domain.IssueStatusTodo, domain.IssuePriorityCritical, nil, base.Add(time.Hour))
	seedOverviewIssue(t, repo, "MED-PROG", &sp, domain.IssueStatusInProgress, domain.IssuePriorityMedium, nil, base)
	seedOverviewIssue(t, repo, "CRIT-BACKLOGSTATUS", &sp, domain.IssueStatusBacklog, domain.IssuePriorityCritical, nil, base)
	// Must not appear.
	seedOverviewIssue(t, repo, "DONE", &sp, domain.IssueStatusDone, domain.IssuePriorityCritical, nil, base)
	seedOverviewIssue(t, repo, "OTHER-SPRINT", &other, domain.IssueStatusInReview, domain.IssuePriorityCritical, nil, base)
	seedOverviewIssue(t, repo, "BACKLOG", nil, domain.IssueStatusInReview, domain.IssuePriorityCritical, nil, base)

	got, err := repo.FindOpenBySprint(ctx, sp, 50)
	require.NoError(t, err)
	var keys []string
	for _, i := range got {
		keys = append(keys, i.Key)
	}
	assert.Equal(t, []string{
		"CRIT-REVIEW", "CRIT-PROG", "CRIT-TODO-NEW", "CRIT-TODO-OLD", "CRIT-BACKLOGSTATUS",
		"HIGH-TODO", "MED-PROG", "LOW-TODO",
	}, keys)
}

func TestIssueRepository_FindOpenBySprint_Limit(t *testing.T) {
	repo := repository.NewIssueRepository(overviewTestDB(t))
	sp := uint(1)
	now := time.Now()
	for i := 0; i < 12; i++ {
		seedOverviewIssue(t, repo, "P-"+string(rune('A'+i)), &sp, domain.IssueStatusTodo, domain.IssuePriorityMedium, nil, now.Add(time.Duration(i)*time.Minute))
	}
	got, err := repo.FindOpenBySprint(context.Background(), sp, 10)
	require.NoError(t, err)
	assert.Len(t, got, 10)
}
