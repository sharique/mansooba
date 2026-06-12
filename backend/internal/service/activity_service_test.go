package service_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubActivityRepo ──────────────────────────────────────────────────────────

type stubActivityRepo struct {
	events []*domain.ActivityEvent
	nextID uint
}

func newStubActivityRepo() *stubActivityRepo { return &stubActivityRepo{nextID: 1} }

func (r *stubActivityRepo) Create(_ context.Context, e *domain.ActivityEvent) error {
	e.ID = r.nextID
	r.nextID++
	cp := *e
	r.events = append(r.events, &cp)
	return nil
}

func (r *stubActivityRepo) FindByIssueID(_ context.Context, issueID uint) ([]*domain.ActivityEvent, error) {
	var out []*domain.ActivityEvent
	for _, e := range r.events {
		if e.IssueID == issueID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *stubActivityRepo) FindByActorID(_ context.Context, actorID uint, limit, offset int) ([]*domain.ActivityEvent, error) {
	var out []*domain.ActivityEvent
	for _, e := range r.events {
		if e.ActorID == actorID {
			out = append(out, e)
		}
	}
	// Apply offset and limit.
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestActivityService_Record_WritesEvent(t *testing.T) {
	repo := newStubActivityRepo()
	userRepo := newStubUserRepo()
	issueRepo := newStubIssueRepo()
	svc := service.NewActivityService(repo, userRepo, issueRepo)

	event := &domain.ActivityEvent{
		IssueID:  1,
		ActorID:  2,
		Kind:     domain.ActivityStatusChanged,
		OldValue: "todo",
		NewValue: "in_progress",
	}
	err := svc.Record(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, uint(1), event.ID)
}

func TestActivityService_ListByIssue_ReturnsEnrichedEvents(t *testing.T) {
	repo := newStubActivityRepo()

	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 2, Name: "Bob", Email: "bob@test.com", Password: "x"})

	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, Key: "PROJ-1", Title: "Login bug"})

	svc := service.NewActivityService(repo, userRepo, issueRepo)

	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, ActorID: 2, Kind: domain.ActivityStatusChanged})
	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 2, ActorID: 2, Kind: domain.ActivityCommentAdded})

	events, err := svc.ListByIssue(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Bob", events[0].ActorName)
	assert.Equal(t, "PROJ-1", events[0].IssueKey)
	assert.Equal(t, "Login bug", events[0].IssueTitle)
}

func TestActivityService_GetMyActivity_ReturnsOnlyCallerEvents(t *testing.T) {
	repo := newStubActivityRepo()
	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 5, Name: "Carol", Email: "carol@test.com", Password: "x"})
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, Key: "P-1", Title: "T"})

	svc := service.NewActivityService(repo, userRepo, issueRepo)

	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, ActorID: 5, Kind: domain.ActivityStatusChanged})
	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, ActorID: 9, Kind: domain.ActivityCommentAdded})

	my, err := svc.GetMyActivity(context.Background(), 5, 20, 0)
	require.NoError(t, err)
	assert.Len(t, my, 1)
	assert.Equal(t, uint(5), my[0].ActorID)
	assert.Equal(t, "Carol", my[0].ActorName)
	assert.Equal(t, "P-1", my[0].IssueKey)
	assert.Equal(t, "T", my[0].IssueTitle)
}

// T027: actor_avatar_url enrichment tests

func TestActivityService_ListByIssue_ActorAvatarURLPopulated(t *testing.T) {
	repo := newStubActivityRepo()
	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 3, Name: "Dave", Email: "dave@test.com", Password: "x", AvatarURL: "/uploads/avatars/avatar-3.jpg?v=500"})

	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, Key: "P-1", Title: "Task"})

	svc := service.NewActivityService(repo, userRepo, issueRepo)
	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, ActorID: 3, Kind: domain.ActivityStatusChanged})

	events, err := svc.ListByIssue(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "/uploads/avatars/avatar-3.jpg?v=500", events[0].ActorAvatarURL)
}

func TestActivityService_ListByIssue_EmptyAvatarURLWhenNoAvatar(t *testing.T) {
	repo := newStubActivityRepo()
	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 4, Name: "Eve", Email: "eve@test.com", Password: "x"})

	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, Key: "P-1", Title: "Task"})

	svc := service.NewActivityService(repo, userRepo, issueRepo)
	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, ActorID: 4, Kind: domain.ActivityStatusChanged})

	events, err := svc.ListByIssue(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "", events[0].ActorAvatarURL)
}

func TestActivityService_GetMyActivity_ActorAvatarURLPopulated(t *testing.T) {
	repo := newStubActivityRepo()
	userRepo := newStubUserRepo()
	_ = userRepo.Create(context.Background(), &domain.User{ID: 6, Name: "Frank", Email: "frank@test.com", Password: "x", AvatarURL: "/uploads/avatars/avatar-6.jpg?v=777"})

	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, Key: "P-1", Title: "Task"})

	svc := service.NewActivityService(repo, userRepo, issueRepo)
	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, ActorID: 6, Kind: domain.ActivityStatusChanged})

	events, err := svc.GetMyActivity(context.Background(), 6, 20, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "/uploads/avatars/avatar-6.jpg?v=777", events[0].ActorAvatarURL)
}
