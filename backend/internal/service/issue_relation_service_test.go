package service_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Relations expose the other issue's key, title and status, and creating or
// deleting them changes the project's data, so every operation is limited to
// members of the issue's project (feature 018, access fix). Real repositories
// on in-memory SQLite: this guards information disclosure.
func TestIssueRelationService_OnlyProjectMembersMayUseRelations(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: ":memory:"})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))

	issueRepo := repository.NewIssueRepository(db)
	memberRepo := repository.NewProjectMemberRepository(db)
	svc := service.NewIssueRelationService(repository.NewIssueRelationRepository(db), issueRepo, memberRepo)

	require.NoError(t, memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: 1, UserID: 10, Role: "member"}))
	require.NoError(t, memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: 2, UserID: 20, Role: "member"}))
	mk := func(project uint, key string) *domain.Issue {
		i := &domain.Issue{ProjectID: project, Key: key, Title: key, Type: "task", Status: "todo", Priority: "low", ReporterID: 10}
		require.NoError(t, issueRepo.Create(ctx, i))
		return i
	}
	a, b, other := mk(1, "A-1"), mk(1, "A-2"), mk(2, "B-1")

	const member, outsider, otherProjectMember = uint(10), uint(99), uint(20)

	rel, err := svc.Create(ctx, a.ID, member, dto.CreateRelationRequest{TargetIssueID: b.ID, RelationType: domain.RelationTypeRelatesTo})
	require.NoError(t, err)

	t.Run("member can list", func(t *testing.T) {
		list, err := svc.List(ctx, a.ID, member)
		require.NoError(t, err)
		assert.Len(t, list, 1)
	})
	t.Run("outsider cannot list, create or delete", func(t *testing.T) {
		_, err := svc.List(ctx, a.ID, outsider)
		assert.ErrorIs(t, err, domain.ErrForbidden)
		_, err = svc.Create(ctx, a.ID, outsider, dto.CreateRelationRequest{TargetIssueID: b.ID, RelationType: domain.RelationTypeBlocks})
		assert.ErrorIs(t, err, domain.ErrForbidden)
		assert.ErrorIs(t, svc.Delete(ctx, a.ID, rel.ID, outsider), domain.ErrForbidden)
	})
	t.Run("a member of another project cannot reach this project's relations", func(t *testing.T) {
		_, err := svc.List(ctx, a.ID, otherProjectMember)
		assert.ErrorIs(t, err, domain.ErrForbidden)
		assert.ErrorIs(t, svc.Delete(ctx, a.ID, rel.ID, otherProjectMember), domain.ErrForbidden)
	})
	t.Run("a relation can only be deleted through an issue it involves", func(t *testing.T) {
		// Caller is a member of project 2 and names an issue of their own project,
		// but the relation belongs to project 1.
		err := svc.Delete(ctx, other.ID, rel.ID, otherProjectMember)
		assert.ErrorIs(t, err, service.ErrRelationNotFound)
		list, err := svc.List(ctx, a.ID, member)
		require.NoError(t, err)
		assert.Len(t, list, 1, "the relation must still exist")
	})
	t.Run("unknown issue is not found", func(t *testing.T) {
		_, err := svc.List(ctx, 99999, member)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
	t.Run("member can delete", func(t *testing.T) {
		require.NoError(t, svc.Delete(ctx, a.ID, rel.ID, member))
		list, err := svc.List(ctx, a.ID, member)
		require.NoError(t, err)
		assert.Empty(t, list)
	})
}
