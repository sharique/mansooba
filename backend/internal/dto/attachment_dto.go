package dto

import "time"

type AttachmentResponse struct {
	ID           uint      `json:"id"`
	IssueID      uint      `json:"issue_id"`
	Filename     string    `json:"filename"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	UploaderID   uint      `json:"uploader_id"`
	UploaderName string    `json:"uploader_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type AttachmentListResponse struct {
	Attachments []AttachmentResponse `json:"attachments"`
}

// AttachmentDownloadResponse carries a short-lived presigned S3 URL. Returned
// as JSON (not a 302) so the client's authenticated fetch can read it before
// navigating — see the Download handler's doc comment for why.
type AttachmentDownloadResponse struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

// AttachmentUploadFile is one parsed multipart file, handed from the
// handler to AttachmentService.Upload.
type AttachmentUploadFile struct {
	Filename    string
	Data        []byte
	ContentType string
}

// AttachmentRejection reports why one file in a batch upload was rejected —
// either a validation failure (size, type) or a storage-write failure
// (spec.md Edge Cases: mixed valid/invalid files in one batch).
type AttachmentRejection struct {
	Filename string `json:"filename"`
	Reason   string `json:"reason"`
}

// AttachmentUploadResult is the response body for a batch upload — always
// 200 OK, even with partial rejections (contracts/api-contracts.md).
type AttachmentUploadResult struct {
	Uploaded []AttachmentResponse  `json:"uploaded"`
	Rejected []AttachmentRejection `json:"rejected"`
}
