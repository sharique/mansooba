package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/sharique/mansooba/internal/testdb"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig points the server at its own fresh database (internal/testdb).
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := testdb.Config(t)
	cfg.JWTSecret = "test-secret"
	cfg.JWTAccessTTL = "15m"
	cfg.JWTRefreshTTL = "168h"
	cfg.CORSOrigins = "http://localhost:3000"
	cfg.BodySizeLimit = "4M"
	cfg.RequestTimeout = "30s"
	cfg.AuthRateLimit = 1000
	return cfg
}

func TestNewTestServer_ServesPublicHealthAndRefusesProtectedRoutes(t *testing.T) {
	e, cleanup, err := httpserver.NewTestServer(testConfig(t))
	require.NoError(t, err)
	t.Cleanup(cleanup)

	// Public route: answers (status follows the health handler's own rules,
	// which also check Loki when it is wired, so only assert it is not a
	// routing or auth failure).
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.NotEqual(t, http.StatusNotFound, rec.Code)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)

	// Protected route without a token is refused with the standard body.
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.JSONEq(t, `{"code":"Unauthorized","message":"Unauthorized"}`, rec.Body.String())
}
