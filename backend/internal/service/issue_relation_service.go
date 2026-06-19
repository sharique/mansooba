package service

import (
	"context"
	"errors"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// Sentinel errors for issue-relation business rule violations.
var (
	ErrSelfRelation        = errors.New("self_relation")
	ErrCircularRelation    = errors.New("circular_relation")
	ErrCrossProjectRelation = errors.New("cross_project_relation")
	ErrDuplicateRelation   = errors.New("duplicate_relation")
	ErrInvalidRelationType  = errors.New("invalid_relation_type")
	ErrRelationNotFound    = errors.New("relation_not_found")
)

// userSelectableTypes are the three types a user can request; "is_blocked_by" is auto-managed.
var userSelectableTypes = map[string]bool{
	domain.RelationTypeBlocks:    true,
	domain.RelationTypeRelatesTo: true,
	domain.RelationTypeDuplicates: true,
}

// IssueRelationService manages task-to-task relationships.
type IssueRelationService interface {
	List(ctx context.Context, issueID uint) ([]*dto.RelationResponse, error)
	Create(ctx context.Context, issueID, userID uint, req dto.CreateRelationRequest) (*dto.RelationResponse, error)
	Delete(ctx context.Context, relationID, userID uint) error
}

type issueRelationSvcImpl struct {
	relRepo   domain.IssueRelationRepository
	issueRepo domain.IssueRepository
}

// NewIssueRelationService returns an IssueRelationService.
func NewIssueRelationService(relRepo domain.IssueRelationRepository, issueRepo domain.IssueRepository) IssueRelationService {
	return &issueRelationSvcImpl{relRepo: relRepo, issueRepo: issueRepo}
}

func (s *issueRelationSvcImpl) List(ctx context.Context, issueID uint) ([]*dto.RelationResponse, error) {
	rows, err := s.relRepo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.RelationResponse, 0, len(rows))
	for _, row := range rows {
		resp, err := s.toResponse(ctx, issueID, row)
		if err != nil {
			continue
		}
		result = append(result, resp)
	}
	return result, nil
}

func (s *issueRelationSvcImpl) Create(ctx context.Context, issueID, userID uint, req dto.CreateRelationRequest) (*dto.RelationResponse, error) {
	if !userSelectableTypes[req.RelationType] {
		return nil, ErrInvalidRelationType
	}
	if req.TargetIssueID == issueID {
		return nil, ErrSelfRelation
	}

	source, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	target, err := s.issueRepo.FindByID(ctx, req.TargetIssueID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if source.ProjectID != target.ProjectID {
		return nil, ErrCrossProjectRelation
	}

	// Circular check: if A wants to block B but B already blocks A, reject.
	if req.RelationType == domain.RelationTypeBlocks {
		exists, err := s.relRepo.ExistsBlock(ctx, req.TargetIssueID, issueID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrCircularRelation
		}
	}

	rel := &domain.IssueRelation{
		IssueID:        issueID,
		RelatedIssueID: req.TargetIssueID,
		RelationType:   req.RelationType,
		CreatedByID:    userID,
		CreatedAt:      time.Now(),
	}
	if err := s.relRepo.Create(ctx, rel); err != nil {
		return nil, ErrDuplicateRelation
	}

	// For "blocks", also insert the reciprocal "is_blocked_by" row.
	if req.RelationType == domain.RelationTypeBlocks {
		reciprocal := &domain.IssueRelation{
			IssueID:        req.TargetIssueID,
			RelatedIssueID: issueID,
			RelationType:   domain.RelationTypeIsBlockedBy,
			CreatedByID:    userID,
			CreatedAt:      time.Now(),
		}
		_ = s.relRepo.Create(ctx, reciprocal) // best-effort; pair is enforced by unique constraint
	}

	return s.toResponse(ctx, issueID, rel)
}

func (s *issueRelationSvcImpl) Delete(ctx context.Context, relationID, userID uint) error {
	rel, err := s.relRepo.FindByID(ctx, relationID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrRelationNotFound
		}
		return err
	}

	// For "blocks", find and delete the reciprocal "is_blocked_by" row.
	if rel.RelationType == domain.RelationTypeBlocks {
		var mirror []*domain.IssueRelation
		rows, _ := s.relRepo.FindByIssueID(ctx, rel.RelatedIssueID)
		for _, r := range rows {
			if r.RelationType == domain.RelationTypeIsBlockedBy && r.RelatedIssueID == rel.IssueID {
				mirror = append(mirror, r)
			}
		}
		for _, m := range mirror {
			_ = s.relRepo.Delete(ctx, m.ID)
		}
	}

	return s.relRepo.Delete(ctx, rel.ID)
}

// toResponse converts a stored IssueRelation row into a RelationResponse from issueID's perspective.
func (s *issueRelationSvcImpl) toResponse(ctx context.Context, issueID uint, rel *domain.IssueRelation) (*dto.RelationResponse, error) {
	otherID := rel.RelatedIssueID
	relType := rel.RelationType
	// For symmetric types stored with this issue as related_issue_id, flip to show the other side.
	if rel.IssueID != issueID {
		otherID = rel.IssueID
	}

	other, err := s.issueRepo.FindByID(ctx, otherID)
	if err != nil {
		return nil, err
	}
	return &dto.RelationResponse{
		ID:           rel.ID,
		RelationType: relType,
		RelatedIssue: dto.RelatedIssueInfo{
			ID:     other.ID,
			Key:    other.Key,
			Title:  other.Title,
			Status: other.Status,
		},
	}, nil
}
