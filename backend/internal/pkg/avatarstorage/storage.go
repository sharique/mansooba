// Package avatarstorage manages user avatar images in the S3-compatible bucket
// shared with issue attachments, under the avatars/ key prefix. Avatars are
// served by the backend (GET /avatars/:filename), never via a public bucket
// URL — see ADR-033.
package avatarstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sharique/mansooba/internal/domain"
)

const maxBytes = 2 * 1024 * 1024 // 2 MB

const keyPrefix = "avatars/"

var allowedTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// knownExts lists every extension Save can produce, in the order Delete tries
// them. S3 has no "does any of these exist" query, and deleting a missing key
// is a no-op, so cleanup simply targets all of them.
var knownExts = []string{"jpg", "png", "webp"}

var _ domain.AvatarStore = (*Storage)(nil)

// Storage manages avatar objects in S3-compatible storage.
type Storage struct {
	client *s3.Client
	bucket string
}

// New returns a Storage that reads and writes bucket through client. The
// client is shared with attachmentstorage (attachmentstorage.Storage.Client).
func New(client *s3.Client, bucket string) *Storage {
	return &Storage{client: client, bucket: bucket}
}

func key(userID uint, ext string) string {
	return fmt.Sprintf("%savatar-%d.%s", keyPrefix, userID, ext)
}

// Save validates data, writes it to S3, and returns the server-relative URL.
// Returns an error if the file exceeds maxBytes or the content type is not
// accepted. Validation happens entirely before the S3 write.
func (s *Storage) Save(ctx context.Context, userID uint, _ string, data []byte, contentType string) (string, error) {
	if len(data) > maxBytes {
		return "", fmt.Errorf("%w: file size %d exceeds maximum %d bytes", domain.ErrAvatarRejected, len(data), maxBytes)
	}

	ext, ok := allowedTypes[contentType]
	if !ok {
		return "", fmt.Errorf("%w: content type %q is not accepted", domain.ErrAvatarRejected, contentType)
	}

	// Validate actual bytes via magic-byte sniff (uses first 512 bytes).
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	detected := http.DetectContentType(sniff)
	if _, valid := allowedTypes[detected]; !valid {
		return "", fmt.Errorf("%w: detected content type %q is not an accepted image format", domain.ErrAvatarRejected, detected)
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key(userID, ext)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("write avatar: %w", err)
	}

	// Put first, clean up second: a failed put never removes the previous
	// avatar. If cleanup fails the caller does not update the profile, so it
	// keeps pointing at a still-valid object.
	for _, other := range knownExts {
		if other == ext {
			continue
		}
		if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key(userID, other)),
		}); err != nil {
			return "", fmt.Errorf("remove previous avatar: %w", err)
		}
	}

	return fmt.Sprintf("/avatars/avatar-%d.%s?v=%d", userID, ext, time.Now().Unix()), nil
}

// Get streams the avatar stored under filename (e.g. "avatar-3.jpg", resolved
// inside the avatars/ prefix — callers never see the key layout) and returns
// its content type. The caller must close the returned reader. Returns
// domain.ErrNotFound when no such avatar exists.
func (s *Storage) Get(ctx context.Context, filename string) (io.ReadCloser, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(keyPrefix + filename),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, "", domain.ErrNotFound
		}
		return nil, "", fmt.Errorf("read avatar: %w", err)
	}
	return out.Body, aws.ToString(out.ContentType), nil
}

// Delete removes the user's avatar object. It is a no-op if none exists.
func (s *Storage) Delete(ctx context.Context, userID uint) error {
	for _, ext := range knownExts {
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key(userID, ext)),
		})
		if err != nil {
			return fmt.Errorf("delete avatar: %w", err)
		}
	}
	return nil
}
