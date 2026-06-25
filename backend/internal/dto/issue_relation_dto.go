package dto

// CreateRelationRequest is the body for POST /api/v1/issues/:id/relations.
type CreateRelationRequest struct {
	TargetIssueID uint   `json:"target_issue_id"`
	RelationType  string `json:"relation_type"`
}

// RelatedIssueInfo summarises the other side of a relation.
type RelatedIssueInfo struct {
	ID     uint   `json:"id"`
	Key    string `json:"key"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// RelationResponse is returned by GET and POST /api/v1/issues/:id/relations.
type RelationResponse struct {
	ID           uint             `json:"id"`
	RelationType string           `json:"relation_type"`
	RelatedIssue RelatedIssueInfo `json:"related_issue"`
}
