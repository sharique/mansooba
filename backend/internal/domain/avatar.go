package domain

import (
	"context"
	"io"
)

// AvatarStore persists user avatar images. The service layer depends on this
// port; internal/pkg/avatarstorage provides the S3-backed implementation.
//
// Error contract: Save wraps ErrAvatarRejected when the file itself is
// unacceptable (its message is safe to show a client); Get returns ErrNotFound
// when no such avatar exists. Any other error means the store failed.
type AvatarStore interface {
	// Save validates and stores the user's avatar, replacing any previous one,
	// and returns the server-relative URL to record on the profile.
	Save(ctx context.Context, userID uint, filename string, data []byte, contentType string) (string, error)
	// Get streams the avatar stored under filename (e.g. "avatar-3.jpg") and
	// returns its content type. The caller must close the reader.
	Get(ctx context.Context, filename string) (io.ReadCloser, string, error)
	// Delete removes the user's avatar. It is a no-op if none exists.
	Delete(ctx context.Context, userID uint) error
}
