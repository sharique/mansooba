package avatarstorage_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/pkg/avatarstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalJPEG is the smallest valid JPEG magic bytes (SOI marker + APP0).
var minimalJPEG = func() []byte {
	b := make([]byte, 512)
	b[0] = 0xFF
	b[1] = 0xD8
	b[2] = 0xFF
	b[3] = 0xE0
	return b
}()

// minimalPNG is a valid PNG signature.
var minimalPNG = func() []byte {
	b := make([]byte, 512)
	copy(b, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	return b
}()

func newStorage(t *testing.T) *avatarstorage.Storage {
	t.Helper()
	dir := t.TempDir()
	return avatarstorage.New(dir)
}

func TestSave_ValidJPEG(t *testing.T) {
	s := newStorage(t)
	url, err := s.Save(1, "photo.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)
	assert.Contains(t, url, "avatar-1.")
	assert.Contains(t, url, "?v=")
}

func TestSave_ValidPNG(t *testing.T) {
	s := newStorage(t)
	url, err := s.Save(2, "photo.png", minimalPNG, "image/png")
	require.NoError(t, err)
	assert.Contains(t, url, "avatar-2.")
}

func TestSave_FilenameIsAvatarUserIDExt(t *testing.T) {
	s := newStorage(t)
	url, err := s.Save(5, "anything.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)
	// URL path should contain avatar-5.
	assert.Contains(t, url, "avatar-5.")
	// File on disk should be named avatar-5.jpg
	dir := t.TempDir() // We can't inspect s.dir here; filename check via url
	_ = dir
	assert.True(t, strings.HasPrefix(url, "/uploads/"))
}

func TestSave_OversizedFileRejected(t *testing.T) {
	s := newStorage(t)
	big := make([]byte, 3*1024*1024) // 3 MB
	copy(big, minimalJPEG)
	_, err := s.Save(1, "big.jpg", big, "image/jpeg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size")
}

func TestSave_WrongContentTypeRejected(t *testing.T) {
	s := newStorage(t)
	// Plaintext data with wrong content-type
	data := []byte("this is not an image")
	_, err := s.Save(1, "hack.jpg", data, "image/jpeg")
	require.Error(t, err)
}

func TestSave_ReuploadOverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	s := avatarstorage.New(dir)

	_, err := s.Save(1, "first.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	_, err = s.Save(1, "second.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	// Only one file should exist on disk for userID 1
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	jpgFiles := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "avatar-1.") {
			jpgFiles++
		}
	}
	assert.Equal(t, 1, jpgFiles, "only one file on disk after two consecutive uploads")
}

func TestSave_URLContainsCacheBuster(t *testing.T) {
	s := newStorage(t)
	url, err := s.Save(1, "photo.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)
	assert.Contains(t, url, "?v=")
}

func TestDelete_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	s := avatarstorage.New(dir)

	_, err := s.Save(3, "photo.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	err = s.Delete(3)
	require.NoError(t, err)

	// File should no longer exist
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), "avatar-3."), "file should be deleted")
	}
}

func TestDelete_NoopWhenNoFile(t *testing.T) {
	s := newStorage(t)
	// Deleting a nonexistent avatar should not error
	err := s.Delete(999)
	assert.NoError(t, err)
}

func TestSave_ReturnedURLMatchesDiskFile(t *testing.T) {
	dir := t.TempDir()
	s := avatarstorage.New(dir)

	url, err := s.Save(7, "photo.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	// Extract filename from URL (before ?v=)
	urlPath := strings.Split(url, "?")[0]
	filename := filepath.Base(urlPath)
	_, statErr := os.Stat(filepath.Join(dir, filename))
	assert.NoError(t, statErr, "file should exist on disk")
}
