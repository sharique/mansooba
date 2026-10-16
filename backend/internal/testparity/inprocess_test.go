package testparity

import (
	"net/http/httptest"
	"testing"

	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/sharique/mansooba/pkg/config"
)

// inProcessBase starts the real server (router, middleware, handlers,
// services, repositories) on a fresh in-memory SQLite database and returns its
// base URL. The configuration matches the one used to record the golden file
// from the standalone build: a 4M body limit and an auth rate limit high
// enough that the script itself is never limited.
func inProcessBase(t *testing.T) string {
	t.Helper()
	cfg := &config.Config{
		DBDriver:       "sqlite",
		DBDSN:          ":memory:",
		JWTSecret:      "parity-secret",
		JWTAccessTTL:   "15m",
		JWTRefreshTTL:  "168h",
		CORSOrigins:    "http://localhost:3000",
		BodySizeLimit:  "4M",
		RequestTimeout: "30s",
		AuthRateLimit:  1000,
	}
	e, cleanup, err := httpserver.NewInMemory(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv.URL
}
