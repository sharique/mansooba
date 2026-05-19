package service_test

import (
	"context"
	"testing"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubLabelRepo ─────────────────────────────────────────────────────────────

type stubLabelRepo struct {
	labels      []*domain.Label
	issueLabels []domain.IssueLabel
	nextID      uint
}

func newStubLabelRepo() *stubLabelRepo { return &stubLabelRepo{nextID: 1} }

func (r *stubLabelRepo) Create(_ context.Context, l *domain.Label) error {
	l.ID = r.nextID
	r.nextID++
	cp := *l
	r.labels = append(r.labels, &cp)
	return nil
}

func (r *stubLabelRepo) FindByProjectID(_ context.Context, projectID uint) ([]*domain.Label, error) {
	var out []*domain.Label
	for _, l := range r.labels {
		if l.ProjectID == projectID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *stubLabelRepo) FindByID(_ context.Context, id uint) (*domain.Label, error) {
	for _, l := range r.labels {
		if l.ID == id {
			return l, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *stubLabelRepo) Delete(_ context.Context, id uint) error {
	for i, l := range r.labels {
		if l.ID == id {
			r.labels = append(r.labels[:i], r.labels[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *stubLabelRepo) AttachToIssue(_ context.Context, issueID, labelID uint) error {
	r.issueLabels = append(r.issueLabels, domain.IssueLabel{IssueID: issueID, LabelID: labelID})
	return nil
}

func (r *stubLabelRepo) DetachFromIssue(_ context.Context, issueID, labelID uint) error {
	for i, il := range r.issueLabels {
		if il.IssueID == issueID && il.LabelID == labelID {
			r.issueLabels = append(r.issueLabels[:i], r.issueLabels[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *stubLabelRepo) FindByIssueID(_ context.Context, issueID uint) ([]*domain.Label, error) {
	var out []*domain.Label
	for _, il := range r.issueLabels {
		if il.IssueID == issueID {
			for _, l := range r.labels {
				if l.ID == il.LabelID {
					out = append(out, l)
				}
			}
		}
	}
	return out, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newLabelTestEnv() (service.LabelService, *stubLabelRepo, *stubActivitySvc) {
	labelRepo := newStubLabelRepo()
	activitySvc := &stubActivitySvc{}
	issueRepo := newStubIssueRepo()
	issueRepo.issues = append(issueRepo.issues, &domain.Issue{ID: 1, ProjectID: 10})
	projectRepo := newStubProjectRepo()
	projectRepo.projects["PROJ"] = &domain.Project{ID: 10, Key: "PROJ"}
	memberRepo := newStubProjectMemberRepo()
	memberRepo.members = append(memberRepo.members, &domain.ProjectMember{ProjectID: 10, UserID: 42, Role: "member"})

	svc := service.NewLabelService(labelRepo, issueRepo, projectRepo, memberRepo, activitySvc)
	return svc, labelRepo, activitySvc
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestLabelService_Create_PersistsLabel(t *testing.T) {
	svc, labelRepo, _ := newLabelTestEnv()
	resp, err := svc.Create(context.Background(), "PROJ", 42, dto.CreateLabelRequest{Name: "bug", Color: "#e11d48"})
	require.NoError(t, err)
	assert.Equal(t, "bug", resp.Name)
	assert.Len(t, labelRepo.labels, 1)
}

func TestLabelService_Create_RejectsInvalidColor(t *testing.T) {
	svc, _, _ := newLabelTestEnv()
	_, err := svc.Create(context.Background(), "PROJ", 42, dto.CreateLabelRequest{Name: "bug", Color: "#badcol"})
	assert.Error(t, err)
}

func TestLabelService_ListByProject_ReturnsLabels(t *testing.T) {
	svc, _, _ := newLabelTestEnv()
	_, _ = svc.Create(context.Background(), "PROJ", 42, dto.CreateLabelRequest{Name: "a", Color: "#e11d48"})
	_, _ = svc.Create(context.Background(), "PROJ", 42, dto.CreateLabelRequest{Name: "b", Color: "#3b82f6"})

	labels, err := svc.ListByProject(context.Background(), "PROJ", 42)
	require.NoError(t, err)
	assert.Len(t, labels, 2)
}

func TestLabelService_AttachToIssue_RecordsActivity(t *testing.T) {
	svc, labelRepo, activitySvc := newLabelTestEnv()
	labelRepo.labels = append(labelRepo.labels, &domain.Label{ID: 5, ProjectID: 10, Name: "urgent", Color: "#e11d48"})

	err := svc.AttachToIssue(context.Background(), 1, 5, 42)
	require.NoError(t, err)
	require.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityLabelAdded, activitySvc.recorded[0].Kind)
	assert.Equal(t, "urgent", activitySvc.recorded[0].NewValue)
}

func TestLabelService_DetachFromIssue_RecordsActivity(t *testing.T) {
	svc, labelRepo, activitySvc := newLabelTestEnv()
	labelRepo.labels = append(labelRepo.labels, &domain.Label{ID: 5, ProjectID: 10, Name: "urgent", Color: "#e11d48"})
	_ = svc.AttachToIssue(context.Background(), 1, 5, 42)
	activitySvc.recorded = nil // reset

	err := svc.DetachFromIssue(context.Background(), 1, 5, 42)
	require.NoError(t, err)
	require.Len(t, activitySvc.recorded, 1)
	assert.Equal(t, domain.ActivityLabelRemoved, activitySvc.recorded[0].Kind)
}

func TestLabelService_AttachToIssue_RejectsLabelFromWrongProject(t *testing.T) {
	svc, labelRepo, _ := newLabelTestEnv()
	// Label belongs to project 99, but the issue belongs to project 10
	labelRepo.labels = append(labelRepo.labels, &domain.Label{ID: 7, ProjectID: 99, Name: "foreign", Color: "#e11d48"})

	err := svc.AttachToIssue(context.Background(), 1, 7, 42)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLabelService_Delete_RemovesLabel(t *testing.T) {
	svc, labelRepo, _ := newLabelTestEnv()
	_, _ = svc.Create(context.Background(), "PROJ", 42, dto.CreateLabelRequest{Name: "gone", Color: "#e11d48"})
	id := labelRepo.labels[0].ID

	err := svc.Delete(context.Background(), "PROJ", id, 42)
	require.NoError(t, err)
	assert.Len(t, labelRepo.labels, 0)
}
