package avatarstorage_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/pkg/attachmentstorage"
	"github.com/sharique/mansooba/internal/pkg/avatarstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBucket = "mansooba-attachments"

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

// minimalWebP is a RIFF container header. Go's content sniffer requires the
// "VP" after "WEBP" to classify it as image/webp.
var minimalWebP = func() []byte {
	b := make([]byte, 512)
	copy(b, []byte("RIFF"))
	copy(b[8:], []byte("WEBPVP"))
	return b
}()

var nextUserID atomic.Uint32

func init() { nextUserID.Store(9_000_000) }

func newClient(t *testing.T) *s3.Client {
	t.Helper()

	resp, err := http.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		// CI sets REQUIRE_LOCALSTACK=1 so a missing LocalStack fails the build
		// instead of letting every S3 test silently skip.
		if os.Getenv("REQUIRE_LOCALSTACK") == "1" {
			t.Fatalf("LocalStack required (REQUIRE_LOCALSTACK=1) but not reachable at localhost:4566: %v", err)
		}
		t.Skipf("LocalStack not reachable at localhost:4566 (%v) — run `docker compose up -d localstack localstack-init`", err)
	}
	resp.Body.Close()

	a, err := attachmentstorage.New(attachmentstorage.Config{
		Endpoint:        "http://localhost:4566",
		Bucket:          testBucket,
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		PresignTTL:      time.Hour,
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	return a.Client()
}

// newStorage returns a Storage, the raw client, and a user ID unique to this
// test whose avatar objects are removed on cleanup.
func newStorage(t *testing.T) (*avatarstorage.Storage, *s3.Client, uint) {
	t.Helper()
	client := newClient(t)
	userID := uint(nextUserID.Add(1))
	t.Cleanup(func() {
		for _, ext := range []string{"jpg", "png", "webp"} {
			_, _ = client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
				Bucket: aws.String(testBucket),
				Key:    aws.String(fmt.Sprintf("avatars/avatar-%d.%s", userID, ext)),
			})
		}
	})
	return avatarstorage.New(client, testBucket), client, userID
}

// listKeys returns every object key stored for the user under avatars/.
func listKeys(t *testing.T, client *s3.Client, userID uint) []string {
	t.Helper()
	out, err := client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(testBucket),
		Prefix: aws.String(fmt.Sprintf("avatars/avatar-%d.", userID)),
	})
	require.NoError(t, err)
	keys := make([]string, 0, len(out.Contents))
	for _, o := range out.Contents {
		keys = append(keys, aws.ToString(o.Key))
	}
	return keys
}

func TestSave_ValidTypesWriteExpectedObject(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		contentType string
		ext         string
	}{
		{"jpeg", minimalJPEG, "image/jpeg", "jpg"},
		{"png", minimalPNG, "image/png", "png"},
		{"webp", minimalWebP, "image/webp", "webp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, client, userID := newStorage(t)

			url, err := s.Save(context.Background(), userID, "photo."+tc.ext, tc.data, tc.contentType)
			require.NoError(t, err)

			wantKey := fmt.Sprintf("avatars/avatar-%d.%s", userID, tc.ext)
			assert.Equal(t, []string{wantKey}, listKeys(t, client, userID))

			assert.True(t, strings.HasPrefix(url, fmt.Sprintf("/avatars/avatar-%d.%s?v=", userID, tc.ext)), "url was %q", url)

			head, err := client.HeadObject(context.Background(), &s3.HeadObjectInput{
				Bucket: aws.String(testBucket),
				Key:    aws.String(wantKey),
			})
			require.NoError(t, err)
			assert.Equal(t, tc.contentType, aws.ToString(head.ContentType))
		})
	}
}

func TestSave_OversizedFileRejectedAndNothingWritten(t *testing.T) {
	s, client, userID := newStorage(t)
	big := make([]byte, 3*1024*1024) // 3 MB, over the 2 MB cap
	copy(big, minimalJPEG)

	_, err := s.Save(context.Background(), userID, "big.jpg", big, "image/jpeg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size")
	assert.ErrorIs(t, err, domain.ErrAvatarRejected)
	assert.Empty(t, listKeys(t, client, userID))
}

func TestSave_DisallowedDeclaredTypeRejected(t *testing.T) {
	s, client, userID := newStorage(t)

	_, err := s.Save(context.Background(), userID, "anim.gif", minimalJPEG, "image/gif")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAvatarRejected)
	assert.Empty(t, listKeys(t, client, userID))
}

func TestSave_NonImageBytesRejectedAndNothingWritten(t *testing.T) {
	s, client, userID := newStorage(t)

	_, err := s.Save(context.Background(), userID, "hack.jpg", []byte("this is not an image"), "image/jpeg")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAvatarRejected)
	assert.Empty(t, listKeys(t, client, userID))
}

func TestGet_ReturnsExactBytesAndContentType(t *testing.T) {
	s, _, userID := newStorage(t)
	_, err := s.Save(context.Background(), userID, "photo.png", minimalPNG, "image/png")
	require.NoError(t, err)

	body, contentType, err := s.Get(context.Background(), fmt.Sprintf("avatar-%d.png", userID))
	require.NoError(t, err)
	defer body.Close()

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, minimalPNG, got)
	assert.Equal(t, "image/png", contentType)
}

func TestStorage_SatisfiesDomainAvatarStore(t *testing.T) {
	var _ domain.AvatarStore = (*avatarstorage.Storage)(nil)
}

func TestGet_MissingKeyReturnsErrNotFound(t *testing.T) {
	s, _, userID := newStorage(t)

	_, _, err := s.Get(context.Background(), fmt.Sprintf("avatar-%d.jpg", userID))
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNotFound), "err was %v", err)
}

func TestDelete_RemovesUsersObject(t *testing.T) {
	s, client, userID := newStorage(t)
	_, err := s.Save(context.Background(), userID, "photo.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)
	require.NotEmpty(t, listKeys(t, client, userID))

	require.NoError(t, s.Delete(context.Background(), userID))

	assert.Empty(t, listKeys(t, client, userID))
}

func TestDelete_NoopWhenNoAvatar(t *testing.T) {
	s, _, userID := newStorage(t)

	assert.NoError(t, s.Delete(context.Background(), userID))
}

// ── replace semantics (exactly one object per user) ──────────────────────────

func TestSave_SameFormatReuploadLeavesExactlyOneObject(t *testing.T) {
	s, client, userID := newStorage(t)

	_, err := s.Save(context.Background(), userID, "first.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)
	_, err = s.Save(context.Background(), userID, "second.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	assert.Equal(t, []string{fmt.Sprintf("avatars/avatar-%d.jpg", userID)}, listKeys(t, client, userID))
}

func TestSave_DifferentFormatReuploadRemovesOldFormatObject(t *testing.T) {
	s, client, userID := newStorage(t)

	_, err := s.Save(context.Background(), userID, "first.png", minimalPNG, "image/png")
	require.NoError(t, err)
	_, err = s.Save(context.Background(), userID, "second.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	assert.Equal(t, []string{fmt.Sprintf("avatars/avatar-%d.jpg", userID)}, listKeys(t, client, userID),
		"the old .png object must be gone")
}

func TestSave_ConcurrentSameFormatUploadsLeaveExactlyOneObject(t *testing.T) {
	s, client, userID := newStorage(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Save(context.Background(), userID, "x.png", minimalPNG, "image/png")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.Equal(t, []string{fmt.Sprintf("avatars/avatar-%d.png", userID)}, listKeys(t, client, userID))
}

func TestSave_RejectedReuploadLeavesPreviousObjectIntact(t *testing.T) {
	s, client, userID := newStorage(t)
	_, err := s.Save(context.Background(), userID, "first.jpg", minimalJPEG, "image/jpeg")
	require.NoError(t, err)

	_, err = s.Save(context.Background(), userID, "hack.png", []byte("not an image"), "image/png")
	require.Error(t, err)

	assert.Equal(t, []string{fmt.Sprintf("avatars/avatar-%d.jpg", userID)}, listKeys(t, client, userID))
	body, _, err := s.Get(context.Background(), fmt.Sprintf("avatar-%d.jpg", userID))
	require.NoError(t, err)
	defer body.Close()
	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, minimalJPEG, got)
}
