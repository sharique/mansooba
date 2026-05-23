package service

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// BoardService defines the board aggregation contract.
type BoardService interface {
	GetBoard(ctx context.Context, projectKey string, callerID uint) (*dto.BoardResponse, error)
}

type boardService struct {
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
}

// NewBoardService returns a BoardService backed by the given repositories.
func NewBoardService(
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
) BoardService {
	return &boardService{
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
	}
}

// boardColumnOrder defines the fixed display order for the kanban board.
// "backlog" is intentionally absent — backlog issues live in the backlog view (MVP 2).
var boardColumnOrder = []string{
	domain.IssueStatusTodo,
	domain.IssueStatusInProgress,
	domain.IssueStatusInReview,
	domain.IssueStatusDone,
}

func (s *boardService) GetBoard(ctx context.Context, projectKey string, callerID uint) (*dto.BoardResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireBoardMember(ctx, project.ID, callerID); err != nil {
		return nil, err
	}

	all, err := s.issueRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	// Group non-backlog issues by status in a single pass.
	grouped := make(map[string][]dto.IssueResponse, len(boardColumnOrder))
	for _, status := range boardColumnOrder {
		grouped[status] = make([]dto.IssueResponse, 0)
	}
	for _, issue := range all {
		if _, onBoard := grouped[issue.Status]; !onBoard {
			continue // skip backlog and any unknown status
		}
		grouped[issue.Status] = append(grouped[issue.Status], *toIssueResponse(issue))
	}

	// Assemble columns in the fixed order — empty slices marshal to [] not null.
	columns := make([]dto.BoardColumn, len(boardColumnOrder))
	for i, status := range boardColumnOrder {
		columns[i] = dto.BoardColumn{Status: status, Issues: grouped[status]}
	}

	return &dto.BoardResponse{Columns: columns}, nil
}

func (s *boardService) requireBoardMember(ctx context.Context, projectID, userID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}
