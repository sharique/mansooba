package testparity

import (
	"net/http/httptest"
	"testing"

	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/sharique/mansooba/internal/testdb"
)

// inProcessBase starts the real server (router, middleware, handlers,
// services, repositories) on a fresh database and returns its
// base URL. The configuration matches the one used to record the golden file
// from the standalone build: a 4M body limit and an auth rate limit high
// enough that the script itself is never limited.
func inProcessBase(t *testing.T) string {
	t.Helper()
	cfg := testdb.Config(t)
	cfg.JWTSecret = "parity-secret"
	cfg.JWTAccessTTL = "15m"
	cfg.JWTRefreshTTL = "168h"
	cfg.CORSOrigins = "http://localhost:3000"
	cfg.BodySizeLimit = "4M"
	cfg.RequestTimeout = "30s"
	cfg.AuthRateLimit = 1000
	e, cleanup, err := httpserver.NewTestServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv.URL
}
