package dto

// BoardIssue is one card on the board: a task plus whether the board lets it be moved.
type BoardIssue struct {
	IssueResponse
	// Locked is true when the task belongs to a completed sprint. The board shows such a
	// card with a lock and does not let it be dragged (the task page is unchanged).
	Locked bool `json:"locked"`
}

type BoardColumn struct {
	Status string       `json:"status"`
	Issues []BoardIssue `json:"issues"`
}

type BoardResponse struct {
	Columns []BoardColumn `json:"columns"`
}
