package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.Config {
	return &config.Config{
		DBDriver:       "sqlite",
		DBDSN:          ":memory:",
		JWTSecret:      "test-secret",
		JWTAccessTTL:   "15m",
		JWTRefreshTTL:  "168h",
		CORSOrigins:    "http://localhost:3000",
		BodySizeLimit:  "4M",
		RequestTimeout: "30s",
		AuthRateLimit:  1000,
	}
}

func TestNewInMemory_ServesPublicHealthAndRefusesProtectedRoutes(t *testing.T) {
	e, cleanup, err := httpserver.NewInMemory(testConfig())
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
