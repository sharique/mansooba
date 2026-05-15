package dto

type BoardColumn struct {
	Status string          `json:"status"`
	Issues []IssueResponse `json:"issues"`
}

type BoardResponse struct {
	Columns []BoardColumn `json:"columns"`
}
