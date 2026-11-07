package service_test

import (
	"context"
	"fmt"
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

// Relations expose the other issue's key, title and status, and creating or
// deleting them changes the project's data, so every operation is limited to
// members of the issue's project (feature 018, access fix). Real repositories
// on a real database: this guards information disclosure.
func TestIssueRelationService_OnlyProjectMembersMayUseRelations(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	seedProjects(t, db, 1, 2)

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
		assert.ErrorIs(t, err, domain.ErrNotFound)
		_, err = svc.Create(ctx, a.ID, outsider, dto.CreateRelationRequest{TargetIssueID: b.ID, RelationType: domain.RelationTypeBlocks})
		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.ErrorIs(t, svc.Delete(ctx, a.ID, rel.ID, outsider), domain.ErrNotFound)
	})
	t.Run("a member of another project cannot reach this project's relations", func(t *testing.T) {
		_, err := svc.List(ctx, a.ID, otherProjectMember)
		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.ErrorIs(t, svc.Delete(ctx, a.ID, rel.ID, otherProjectMember), domain.ErrNotFound)
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

// To someone who cannot see the other project, a task there simply does not
// exist; only a person who can see both is told it is a cross-project link.
func TestIssueRelationService_CrossProjectTargetIsNotFoundToOutsiders(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	seedProjects(t, db, 1, 2)
	issueRepo := repository.NewIssueRepository(db)
	memberRepo := repository.NewProjectMemberRepository(db)
	svc := service.NewIssueRelationService(repository.NewIssueRelationRepository(db), issueRepo, memberRepo)

	require.NoError(t, memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: 1, UserID: 10, Role: "member"}))
	require.NoError(t, memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: 1, UserID: 20, Role: "member"}))
	require.NoError(t, memberRepo.Create(ctx, &domain.ProjectMember{ProjectID: 2, UserID: 20, Role: "member"}))
	mk := func(project uint, key string) *domain.Issue {
		i := &domain.Issue{ProjectID: project, Key: key, Title: key, Type: "task", Status: "todo", Priority: "low", ReporterID: 10}
		require.NoError(t, issueRepo.Create(ctx, i))
		return i
	}
	a, b := mk(1, "A-1"), mk(2, "B-1")

	req := dto.CreateRelationRequest{TargetIssueID: b.ID, RelationType: domain.RelationTypeRelatesTo}
	_, err := svc.Create(ctx, a.ID, 10, req)
	assert.ErrorIs(t, err, domain.ErrNotFound, "10 cannot see project 2")
	_, err = svc.Create(ctx, a.ID, 20, req)
	assert.ErrorIs(t, err, service.ErrCrossProjectRelation, "20 is in both projects")
}

// seedProjects creates projects with the given ids. Tasks and members can only be
// added to a project that exists.
func seedProjects(t *testing.T, db *gorm.DB, ids ...uint) {
	t.Helper()
	for _, id := range ids {
		require.NoError(t, db.Create(&domain.Project{ID: id, Key: fmt.Sprintf("P%d", id), Name: fmt.Sprintf("Project %d", id), OwnerID: 1}).Error)
	}
}
