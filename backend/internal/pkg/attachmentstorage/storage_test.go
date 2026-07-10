package attachmentstorage_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/pkg/attachmentstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalPNG is a valid PNG signature followed by padding, well under the
// size cap and matching image/png via magic-byte sniffing.
var minimalPNG = func() []byte {
	b := make([]byte, 512)
	copy(b, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	return b
}()

// minimalZip is a valid (empty) ZIP local-file-header signature — the same
// container format OOXML documents (.docx/.xlsx/.pptx) use.
var minimalZip = func() []byte {
	b := make([]byte, 512)
	copy(b, []byte{0x50, 0x4B, 0x03, 0x04})
	return b
}()

// minimalOLE is the OLE Compound File Binary signature legacy Office
// documents (.doc/.xls/.ppt) use.
var minimalOLE = func() []byte {
	b := make([]byte, 512)
	copy(b, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	return b
}()

func testConfig() attachmentstorage.Config {
	return attachmentstorage.Config{
		Endpoint:        "http://localhost:4566",
		Bucket:          "mansooba-attachments",
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		PresignTTL:      time.Hour,
		UsePathStyle:    true,
	}
}

func newStorage(t *testing.T) *attachmentstorage.Storage {
	t.Helper()
	s, err := attachmentstorage.New(testConfig())
	require.NoError(t, err)

	// LocalStack must be reachable for these tests — they exercise the real
	// S3 API/SDK path per Constitution Principle III (no mocks for
	// infra-adjacent paths). Skip with a clear message if it isn't running,
	// rather than failing with an opaque connection-refused error.
	resp, err := http.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		t.Skipf("LocalStack not reachable at localhost:4566 (%v) — run `docker compose up -d localstack localstack-init`", err)
	}
	resp.Body.Close()

	return s
}

func TestSave_ValidPNG(t *testing.T) {
	s := newStorage(t)
	key, err := s.Save(context.Background(), "PROJ/PROJ-1", "screenshot.png", minimalPNG, "image/png")
	require.NoError(t, err)
	assert.Contains(t, key, "PROJ/PROJ-1/")
}

func TestSave_OversizedFileRejected(t *testing.T) {
	s := newStorage(t)
	big := make([]byte, 11*1024*1024) // 11 MB, over the 10 MB cap
	copy(big, minimalPNG)
	_, err := s.Save(context.Background(), "PROJ/PROJ-1", "big.png", big, "image/png")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size")
}

func TestSave_WrongContentTypeRejected(t *testing.T) {
	s := newStorage(t)
	data := []byte("this is not an image")
	_, err := s.Save(context.Background(), "PROJ/PROJ-1", "hack.png", data, "image/png")
	require.Error(t, err)
}

func TestSave_DisallowedTypeRejected(t *testing.T) {
	s := newStorage(t)
	exe := []byte("MZ\x90\x00\x03\x00\x00\x00") // PE/EXE magic bytes
	_, err := s.Save(context.Background(), "PROJ/PROJ-1", "malware.exe", exe, "application/x-msdownload")
	require.Error(t, err)
}

func TestSave_OOXMLDeclaredAsZipSignatureAccepted(t *testing.T) {
	s := newStorage(t)
	_, err := s.Save(context.Background(), "PROJ/PROJ-1", "report.docx",
		minimalZip, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	require.NoError(t, err, "OOXML formats are ZIP containers — declaring docx over a ZIP signature must be accepted")
}

func TestSave_LegacyOfficeDeclaredAsOLESignatureAccepted(t *testing.T) {
	s := newStorage(t)
	_, err := s.Save(context.Background(), "PROJ/PROJ-1", "report.doc", minimalOLE, "application/msword")
	require.NoError(t, err, "legacy Office formats share the OLE Compound File signature and must be accepted")
}

func TestSave_OLESignatureDeclaredAsWrongLegacyOfficeTypeRejected(t *testing.T) {
	s := newStorage(t)
	// Actual bytes are a valid OLE compound file, but declared as a ZIP-based
	// OOXML type — the signature families don't overlap, so this must be rejected.
	_, err := s.Save(context.Background(), "PROJ/PROJ-1", "fake.docx",
		minimalOLE, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	require.Error(t, err)
}

func TestSave_TwoFilesSameFilenameGetDistinctKeys(t *testing.T) {
	s := newStorage(t)
	key1, err := s.Save(context.Background(), "PROJ/PROJ-2", "report.png", minimalPNG, "image/png")
	require.NoError(t, err)
	key2, err := s.Save(context.Background(), "PROJ/PROJ-2", "report.png", minimalPNG, "image/png")
	require.NoError(t, err)
	assert.NotEqual(t, key1, key2, "two uploads of the same filename must not collide on storage")
}

func TestPresignGet_ReturnsWorkingSignedURL(t *testing.T) {
	s := newStorage(t)
	key, err := s.Save(context.Background(), "PROJ/PROJ-3", "doc.png", minimalPNG, "image/png")
	require.NoError(t, err)

	url, err := s.PresignGet(context.Background(), key, "doc.png")
	require.NoError(t, err)
	assert.Contains(t, url, "X-Amz-Signature")

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Disposition"), `filename="doc.png"`)
}

// TestPresignGet_UsesPresignEndpointOverride guards against a real bug found
// via full docker-compose testing: the backend reaches LocalStack via the
// Docker-internal hostname "localstack", but a presigned URL is followed by
// the browser on the host machine, which can't resolve that hostname. The
// signed host must come from PresignEndpoint, not Endpoint.
func TestPresignGet_UsesPresignEndpointOverride(t *testing.T) {
	cfg := testConfig()
	cfg.Endpoint = "http://localstack:4566"       // as reached from inside a container
	cfg.PresignEndpoint = "http://localhost:4566" // as reached from the browser
	s, err := attachmentstorage.New(cfg)
	require.NoError(t, err)

	// Object writes still go to the real reachable endpoint for this test
	// environment — swap Endpoint back before Save, only PresignGet cares
	// about the override.
	writeCfg := testConfig()
	writer, err := attachmentstorage.New(writeCfg)
	require.NoError(t, err)
	resp, herr := http.Get("http://localhost:4566/_localstack/health")
	if herr != nil {
		t.Skipf("LocalStack not reachable at localhost:4566 (%v) — run `docker compose up -d localstack localstack-init`", herr)
	}
	resp.Body.Close()
	key, err := writer.Save(context.Background(), "PROJ/PROJ-6", "override.png", minimalPNG, "image/png")
	require.NoError(t, err)

	url, err := s.PresignGet(context.Background(), key, "override.png")
	require.NoError(t, err)
	assert.Contains(t, url, "http://localhost:4566/", "presigned URL must use PresignEndpoint, not the internal Endpoint")
}

func TestDelete_RemovesObject(t *testing.T) {
	s := newStorage(t)
	key, err := s.Save(context.Background(), "PROJ/PROJ-4", "temp.png", minimalPNG, "image/png")
	require.NoError(t, err)

	err = s.Delete(context.Background(), key)
	require.NoError(t, err)

	// A presigned URL for a deleted object should now 404 when fetched.
	url, err := s.PresignGet(context.Background(), key, "temp.png")
	require.NoError(t, err)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteAll_RemovesMultipleObjects(t *testing.T) {
	s := newStorage(t)
	var keys []string
	for i := 0; i < 3; i++ {
		key, err := s.Save(context.Background(), "PROJ/PROJ-5", "batch.png", minimalPNG, "image/png")
		require.NoError(t, err)
		keys = append(keys, key)
	}

	err := s.DeleteAll(context.Background(), keys)
	require.NoError(t, err)

	for _, key := range keys {
		url, err := s.PresignGet(context.Background(), key, "batch.png")
		require.NoError(t, err)
		resp, err := http.Get(url)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}
}

func TestDeleteAll_EmptyListIsNoop(t *testing.T) {
	s := newStorage(t)
	err := s.DeleteAll(context.Background(), nil)
	assert.NoError(t, err)
}
