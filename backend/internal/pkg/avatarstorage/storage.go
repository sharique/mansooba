package avatarstorage

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const maxBytes = 2 * 1024 * 1024 // 2 MB

var allowedTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// Storage manages avatar files on the local filesystem.
type Storage struct {
	dir string
}

// New returns a Storage that saves files under dir.
func New(dir string) *Storage {
	return &Storage{dir: dir}
}

// Save validates data, writes it to disk, and returns the server-relative URL.
// Returns an error if the file exceeds maxBytes or the content type is not accepted.
func (s *Storage) Save(userID uint, _ string, data []byte, contentType string) (string, error) {
	if len(data) > maxBytes {
		return "", fmt.Errorf("file size %d exceeds maximum %d bytes", len(data), maxBytes)
	}

	// Validate declared content-type.
	ext, ok := allowedTypes[contentType]
	if !ok {
		return "", fmt.Errorf("content type %q is not accepted", contentType)
	}

	// Validate actual bytes via magic-byte sniff (uses first 512 bytes).
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	detected := http.DetectContentType(sniff)
	if _, valid := allowedTypes[detected]; !valid {
		return "", fmt.Errorf("detected content type %q is not an accepted image format", detected)
	}

	filename := fmt.Sprintf("avatar-%d.%s", userID, ext)
	path := filepath.Join(s.dir, filename)

	// Overwrite any existing file for this userID atomically via temp+rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return "", fmt.Errorf("write avatar: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rename avatar: %w", err)
	}

	url := fmt.Sprintf("/uploads/avatars/%s?v=%d", filename, time.Now().Unix())
	return url, nil
}

// Delete removes the avatar file for the given userID. It is a no-op if no file exists.
func (s *Storage) Delete(userID uint) error {
	// Try all known extensions — only one will exist at a time (deterministic name).
	for _, ext := range []string{"jpg", "png", "webp"} {
		filename := fmt.Sprintf("avatar-%d.%s", userID, ext)
		path := filepath.Join(s.dir, filename)
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete avatar: %w", err)
		}
	}
	return nil
}
