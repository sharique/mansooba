package service

import (
	"context"
	"errors"
	"sort"

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
	sprintRepo  domain.SprintRepository
}

// NewBoardService returns a BoardService backed by the given repositories.
func NewBoardService(
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	sprintRepo domain.SprintRepository,
) BoardService {
	return &boardService{
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		sprintRepo:  sprintRepo,
	}
}

// boardColumnOrder defines the fixed display order for the sprint board.
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

	sortBoardIssues(all)

	// One lookup for the whole board: tasks of a completed sprint are shown locked.
	sprints, err := s.sprintRepo.FindByProject(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	completed := make(map[uint]bool, len(sprints))
	for _, sp := range sprints {
		if sp.Status == domain.SprintStatusCompleted {
			completed[sp.ID] = true
		}
	}

	// Group non-backlog issues by status in a single pass.
	grouped := make(map[string][]dto.BoardIssue, len(boardColumnOrder))
	for _, status := range boardColumnOrder {
		grouped[status] = make([]dto.BoardIssue, 0)
	}
	for _, issue := range all {
		if _, onBoard := grouped[issue.Status]; !onBoard {
			continue // skip backlog and any unknown status
		}
		locked := issue.SprintID != nil && completed[*issue.SprintID]
		grouped[issue.Status] = append(grouped[issue.Status], dto.BoardIssue{IssueResponse: *toIssueResponse(issue), Locked: locked})
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

// boardPriorityRank orders priorities for the board: critical first, anything
// unrecognised last. The browser applies the same rule when a card is dropped
// (frontend/app/utils/boardOrder.ts); both are tested against
// testdata/board-order-cases.json.
func boardPriorityRank(priority string) int {
	switch priority {
	case domain.IssuePriorityCritical:
		return 0
	case domain.IssuePriorityHigh:
		return 1
	case domain.IssuePriorityMedium:
		return 2
	case domain.IssuePriorityLow:
		return 3
	}
	return 4
}

// sortBoardIssues orders issues by priority, then by issue number. Issue keys count
// up with the id inside one project, so the id is the issue number.
func sortBoardIssues(issues []*domain.Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		ri, rj := boardPriorityRank(issues[i].Priority), boardPriorityRank(issues[j].Priority)
		if ri != rj {
			return ri < rj
		}
		return issues[i].ID < issues[j].ID
	})
}
