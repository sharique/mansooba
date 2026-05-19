package service_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/service"
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

// ── tests ─────────────────────────────────────────────────────────────────────

func TestActivityService_Record_WritesEvent(t *testing.T) {
	repo := newStubActivityRepo()
	svc := service.NewActivityService(repo)

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

func TestActivityService_ListByIssue_ReturnsOnlyMatchingEvents(t *testing.T) {
	repo := newStubActivityRepo()
	svc := service.NewActivityService(repo)

	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 1, Kind: domain.ActivityStatusChanged})
	_ = svc.Record(context.Background(), &domain.ActivityEvent{IssueID: 2, Kind: domain.ActivityCommentAdded})

	events, err := svc.ListByIssue(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, domain.ActivityStatusChanged, events[0].Kind)
}
