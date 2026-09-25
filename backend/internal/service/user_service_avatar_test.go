package service_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAvatarStore is an in-memory domain.AvatarStore. The service depends only
// on the interface, so its orchestration (validation vs outage classification,
// profile updated only after storage succeeds) is tested here without S3; the
// real S3 behavior is covered by the LocalStack tests in avatarstorage.
type fakeAvatarStore struct {
	objects map[string][]byte // filename -> bytes
	types   map[string]string // filename -> content type

	saveErr   error
	getErr    error
	deleteErr error

	saveCalls   int
	deleteCalls int
}

func newFakeAvatarStore() *fakeAvatarStore {
	return &fakeAvatarStore{objects: map[string][]byte{}, types: map[string]string{}}
}

var _ domain.AvatarStore = (*fakeAvatarStore)(nil)

func (f *fakeAvatarStore) Save(_ context.Context, userID uint, _ string, data []byte, contentType string) (string, error) {
	f.saveCalls++
	if f.saveErr != nil {
		return "", f.saveErr
	}
	ext, ok := map[string]string{"image/jpeg": "jpg", "image/png": "png"}[contentType]
	if !ok {
		return "", fmt.Errorf("%w: content type %q is not accepted", domain.ErrAvatarRejected, contentType)
	}
	for _, other := range []string{"jpg", "png"} {
		delete(f.objects, fmt.Sprintf("avatar-%d.%s", userID, other))
	}
	name := fmt.Sprintf("avatar-%d.%s", userID, ext)
	f.objects[name] = data
	f.types[name] = contentType
	return fmt.Sprintf("/avatars/%s?v=1", name), nil
}

func (f *fakeAvatarStore) Get(_ context.Context, filename string) (io.ReadCloser, string, error) {
	if f.getErr != nil {
		return nil, "", f.getErr
	}
	data, ok := f.objects[filename]
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), f.types[filename], nil
}

func (f *fakeAvatarStore) Delete(_ context.Context, userID uint) error {
	f.deleteCalls++
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for _, ext := range []string{"jpg", "png"} {
		delete(f.objects, fmt.Sprintf("avatar-%d.%s", userID, ext))
	}
	return nil
}

func newAvatarService(t *testing.T) (service.UserService, *stubUserRepo, *fakeAvatarStore) {
	t.Helper()
	repo := newStubUserRepo()
	require.NoError(t, repo.Create(context.Background(), &domain.User{
		ID: 1, Name: "Alice", Email: "alice@example.com", Password: "hash",
	}))
	store := newFakeAvatarStore()
	return service.NewUserService(repo, store), repo, store
}

func storedAvatarURL(t *testing.T, repo *stubUserRepo) string {
	t.Helper()
	u, err := repo.FindByID(context.Background(), 1)
	require.NoError(t, err)
	return u.AvatarURL
}

func TestUserService_UploadAvatar_SetsProfileURLFromStore(t *testing.T) {
	svc, repo, store := newAvatarService(t)

	resp, err := svc.UploadAvatar(context.Background(), 1, "me.jpg", []byte("jpeg"), "image/jpeg")
	require.NoError(t, err)

	assert.Equal(t, "/avatars/avatar-1.jpg?v=1", resp.AvatarURL)
	assert.Equal(t, resp.AvatarURL, storedAvatarURL(t, repo))
	assert.Equal(t, 1, store.saveCalls)
}

func TestUserService_UploadAvatar_UnknownUserSkipsStorage(t *testing.T) {
	svc, _, store := newAvatarService(t)

	_, err := svc.UploadAvatar(context.Background(), 999, "me.jpg", []byte("jpeg"), "image/jpeg")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Zero(t, store.saveCalls)
}

func TestUserService_UploadAvatar_RejectionKeepsMessageAndURL(t *testing.T) {
	svc, repo, _ := newAvatarService(t)

	_, err := svc.UploadAvatar(context.Background(), 1, "a.gif", []byte("gif"), "image/gif")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAvatarRejected)
	assert.NotErrorIs(t, err, domain.ErrAvatarStorageUnavailable, "a rejected file is the client's fault, not an outage")
	assert.Contains(t, err.Error(), "not accepted", "the message is safe to show the client")
	assert.Empty(t, storedAvatarURL(t, repo))
}

func TestUserService_UploadAvatar_StorageFailureIsUnavailableAndKeepsURL(t *testing.T) {
	svc, repo, store := newAvatarService(t)
	seedAvatarURL(t, repo, "/avatars/avatar-1.jpg?v=0")
	store.saveErr = errors.New("operation error S3: PutObject, connection refused")

	_, err := svc.UploadAvatar(context.Background(), 1, "me.jpg", []byte("jpeg"), "image/jpeg")

	assert.ErrorIs(t, err, domain.ErrAvatarStorageUnavailable)
	assert.NotErrorIs(t, err, domain.ErrAvatarRejected)
	assert.Equal(t, "/avatars/avatar-1.jpg?v=0", storedAvatarURL(t, repo))
}

func TestUserService_GetAvatar_ReturnsStoredBytesAndType(t *testing.T) {
	svc, _, _ := newAvatarService(t)
	_, err := svc.UploadAvatar(context.Background(), 1, "me.png", []byte("png-bytes"), "image/png")
	require.NoError(t, err)

	body, contentType, err := svc.GetAvatar(context.Background(), "avatar-1.png")
	require.NoError(t, err)
	defer body.Close()

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "png-bytes", string(got))
	assert.Equal(t, "image/png", contentType)
}

func TestUserService_GetAvatar_MissingIsDomainNotFound(t *testing.T) {
	svc, _, _ := newAvatarService(t)

	_, _, err := svc.GetAvatar(context.Background(), "avatar-1.jpg")

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_GetAvatar_OtherStoreErrorIsNotReportedAsNotFound(t *testing.T) {
	svc, _, store := newAvatarService(t)
	store.getErr = errors.New("read avatar: connection refused")

	_, _, err := svc.GetAvatar(context.Background(), "avatar-1.jpg")

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrNotFound, "an outage must surface as a 500, not a 404")
}

func TestUserService_DeleteAvatar_RemovesObjectAndClearsURL(t *testing.T) {
	svc, repo, _ := newAvatarService(t)
	_, err := svc.UploadAvatar(context.Background(), 1, "me.jpg", []byte("jpeg"), "image/jpeg")
	require.NoError(t, err)

	resp, err := svc.DeleteAvatar(context.Background(), 1)
	require.NoError(t, err)

	assert.Empty(t, resp.AvatarURL)
	assert.Empty(t, storedAvatarURL(t, repo))
	_, _, err = svc.GetAvatar(context.Background(), "avatar-1.jpg")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_DeleteAvatar_NoAvatarIsNotAnError(t *testing.T) {
	svc, _, _ := newAvatarService(t)

	resp, err := svc.DeleteAvatar(context.Background(), 1)

	require.NoError(t, err)
	assert.Empty(t, resp.AvatarURL)
}

func TestUserService_DeleteAvatar_StorageFailureIsUnavailableAndKeepsURL(t *testing.T) {
	svc, repo, store := newAvatarService(t)
	seedAvatarURL(t, repo, "/avatars/avatar-1.jpg?v=0")
	store.deleteErr = errors.New("operation error S3: DeleteObject, connection refused")

	_, err := svc.DeleteAvatar(context.Background(), 1)

	assert.ErrorIs(t, err, domain.ErrAvatarStorageUnavailable)
	assert.True(t, strings.HasPrefix(storedAvatarURL(t, repo), "/avatars/avatar-1.jpg"), "profile must be unchanged")
}

func seedAvatarURL(t *testing.T, repo *stubUserRepo, url string) {
	t.Helper()
	u, err := repo.FindByID(context.Background(), 1)
	require.NoError(t, err)
	u.AvatarURL = url
	require.NoError(t, repo.Update(context.Background(), u))
}
