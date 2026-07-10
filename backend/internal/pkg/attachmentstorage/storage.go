// Package attachmentstorage manages issue attachment files in S3-compatible
// object storage. Real AWS S3 in production, LocalStack in local dev — the
// same code path talks to both; only endpoint/credential configuration
// differs (see docs/decisions/ADR-029 in the docs repo).
package attachmentstorage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

const maxBytes = 10 * 1024 * 1024 // 10 MB, per FR-004 / spec.md Assumptions

// allowedTypes maps accepted declared content types to a file extension used
// when generating an object key. Covers common document, image, text, and
// archive formats per FR-005; excludes executables and other higher-risk
// formats.
var allowedTypes = map[string]string{
	"image/jpeg":         "jpg",
	"image/png":          "png",
	"image/webp":         "webp",
	"image/gif":          "gif",
	"application/pdf":    "pdf",
	"text/plain":         "txt",
	"text/csv":           "csv",
	"application/zip":    "zip",
	"application/msword": "doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
	"application/vnd.ms-excel": "xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
	"application/vnd.ms-powerpoint":                                             "ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
}

// oleCompoundFileType is a synthetic content type this package assigns to
// data starting with the OLE Compound File Binary signature — the container
// format legacy Office files (.doc/.xls/.ppt) use. Go's stdlib
// http.DetectContentType has no built-in signature for it and would
// otherwise fall back to "application/octet-stream", indistinguishable from
// an arbitrary (and potentially dangerous) binary.
const oleCompoundFileType = "application/x-ole-compound-document"

var oleCompoundFileMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// compatibleDeclaredTypes maps a magic-byte-sniffed base content type to the
// set of declared content types it's consistent with. Several formats share
// an underlying container signature that byte-sniffing alone can't further
// disambiguate — OOXML documents (.docx/.xlsx/.pptx) are all ZIP containers,
// and legacy Office documents (.doc/.xls/.ppt) all share the OLE Compound
// File signature. Grouping them here still catches the actual threat FR-005
// guards against (a dangerous file disguised with a safe declared type),
// since none of these signatures overlap with an executable's.
var compatibleDeclaredTypes = map[string]map[string]bool{
	"image/jpeg":      {"image/jpeg": true},
	"image/png":       {"image/png": true},
	"image/webp":      {"image/webp": true},
	"image/gif":       {"image/gif": true},
	"application/pdf": {"application/pdf": true},
	"text/plain":      {"text/plain": true, "text/csv": true},
	"application/zip": {
		"application/zip": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	},
	oleCompoundFileType: {
		"application/msword":            true,
		"application/vnd.ms-excel":      true,
		"application/vnd.ms-powerpoint": true,
	},
}

// sniffContentType detects the base content type of data by magic bytes,
// extending http.DetectContentType with an OLE Compound File check.
func sniffContentType(data []byte) string {
	if bytes.HasPrefix(data, oleCompoundFileMagic) {
		return oleCompoundFileType
	}
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	detected := http.DetectContentType(sniff)
	base, _, _ := strings.Cut(detected, ";")
	return base
}

// Config configures the S3 client. Endpoint is set for LocalStack in local
// dev and left empty in production so the AWS SDK resolves the real regional
// S3 endpoint and falls through to the EC2 instance's IAM role for
// credentials. AccessKeyID/SecretAccessKey are LocalStack-only.
type Config struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	PresignTTL      time.Duration
	UsePathStyle    bool

	// PresignEndpoint overrides Endpoint for presigned URL generation only.
	// Needed for local dev: the backend container reaches LocalStack via the
	// Docker-internal hostname ("http://localstack:4566"), but a presigned
	// URL is followed by the browser on the host machine, which can't
	// resolve that hostname — it needs "http://localhost:4566" instead.
	// Defaults to Endpoint when empty (the production case: real S3's
	// public hostname is reachable identically from the backend and the
	// browser, so no override is needed there).
	PresignEndpoint string
}

// Storage manages attachment objects in S3-compatible storage.
type Storage struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	presignTTL time.Duration
}

// New returns a Storage configured against cfg.
func New(cfg Config) (*Storage, error) {
	ctx := context.Background()

	var optFns []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		optFns = append(optFns, awsconfig.WithRegion(cfg.Region))
	}
	if cfg.AccessKeyID != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	presignEndpoint := cfg.PresignEndpoint
	if presignEndpoint == "" {
		presignEndpoint = cfg.Endpoint
	}
	presignClient := client
	if presignEndpoint != cfg.Endpoint {
		presignClient = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if presignEndpoint != "" {
				o.BaseEndpoint = aws.String(presignEndpoint)
			}
			o.UsePathStyle = cfg.UsePathStyle
		})
	}

	presignTTL := cfg.PresignTTL
	if presignTTL <= 0 {
		presignTTL = time.Hour
	}

	return &Storage{
		client:     client,
		presign:    s3.NewPresignClient(presignClient),
		bucket:     cfg.Bucket,
		presignTTL: presignTTL,
	}, nil
}

// Save validates data, writes it to S3 under a generated key within
// keyPrefix, and returns that key. Returns an error if the file exceeds
// maxBytes or the content type is not accepted.
//
// keyPrefix is caller-supplied (e.g. "PROJ/PROJ-3", a project key and issue
// key) and used verbatim as the S3 key's directory portion — the object
// itself always gets a generated, collision-proof filename
// ("<uuid>.<ext>") within it, so multiple attachments on the same issue,
// even with identical original filenames, never collide (data-model.md).
//
// Validation happens entirely before the S3 write (ADR-029, research.md
// Decision 2 & 9): the object key is only returned — and only ever
// persisted by the caller — once the write to S3 has succeeded.
func (s *Storage) Save(ctx context.Context, keyPrefix string, _ string, data []byte, contentType string) (string, error) {
	if len(data) > maxBytes {
		return "", fmt.Errorf("file size %d exceeds maximum %d bytes", len(data), maxBytes)
	}

	ext, ok := allowedTypes[contentType]
	if !ok {
		return "", fmt.Errorf("content type %q is not accepted", contentType)
	}

	detected := sniffContentType(data)
	compatible, known := compatibleDeclaredTypes[detected]
	if !known || !compatible[contentType] {
		return "", fmt.Errorf("declared content type %q does not match detected content (sniffed as %q)", contentType, detected)
	}

	key := path.Join(keyPrefix, fmt.Sprintf("%s.%s", uuid.NewString(), ext))

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("write attachment to storage: %w", err)
	}

	return key, nil
}

// PresignGet returns a short-lived, presigned GET URL for the object at key.
// The URL sets a Content-Disposition header so the browser downloads with
// filename regardless of the (opaque) S3 object key.
func (s *Storage) PresignGet(ctx context.Context, key, filename string) (string, error) {
	disposition := fmt.Sprintf(`attachment; filename="%s"`, filename)
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(disposition),
	}, s3.WithPresignExpires(s.presignTTL))
	if err != nil {
		return "", fmt.Errorf("presign download url: %w", err)
	}
	return req.URL, nil
}

// Delete removes the object at key. It is a no-op if the object doesn't exist.
func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	return nil
}

// DeleteAll removes multiple objects in a single batched request (used for
// cascade delete when an issue is removed — research.md Decision 4). A nil
// or empty keys slice is a no-op. The AWS API supports up to 1000 keys per
// call, comfortably covering the per-issue attachment cap (FR-011).
func (s *Storage) DeleteAll(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
	}

	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{Objects: objects},
	})
	if err != nil {
		return fmt.Errorf("batch delete attachments: %w", err)
	}
	return nil
}
