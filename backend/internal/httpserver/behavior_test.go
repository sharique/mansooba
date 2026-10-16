package httpserver_test

// Characterisation tests for behaviour that is not visible in the JSON parity
// replay: limits, timeouts, panics, CORS and rate limiting. They were written
// against the Echo v4 build and must keep passing, unchanged in meaning, on v5
// (feature 018, FR-002 to FR-004). Where Echo v5 intentionally differs (forwarded
// header trust), the difference is called out in the test and in ADR-037.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServer(t *testing.T, mutate func(*config.Config)) *echo.Echo {
	t.Helper()
	cfg := testConfig()
	if mutate != nil {
		mutate(cfg)
	}
	e, cleanup, err := httpserver.NewInMemory(cfg)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	return e
}

func do(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestBehavior_RequestTimeoutReturns503(t *testing.T) {
	e := newServer(t, func(c *config.Config) { c.RequestTimeout = "50ms" })
	// A handler that honours its request context, as every handler that calls
	// the database does (GORM is given the request context).
	e.GET("/__slow", func(c *echo.Context) error {
		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-time.After(300 * time.Millisecond):
			return c.String(http.StatusOK, "late")
		}
	})

	rec := do(e, httptest.NewRequest(http.MethodGet, "/__slow", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	// Recorded from the v4 build: the timeout middleware writes its own small
	// HTML body, outside the central error handler, and sets no Content-Type.
	assert.Equal(t, `<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>`, rec.Body.String())
	assert.Empty(t, rec.Header().Get("Content-Type"))
}

// Known, accepted difference from Echo v4 (ADR-037): v4 abandoned a handler
// that was still running when the timeout fired; v5 only sets a deadline on
// the request context, so a handler that never looks at it runs to completion.
func TestBehavior_RequestTimeout_HandlerIgnoringContextIsNotInterruptedUnderV5(t *testing.T) {
	e := newServer(t, func(c *config.Config) { c.RequestTimeout = "50ms" })
	e.GET("/__ignores", func(c *echo.Context) error {
		time.Sleep(150 * time.Millisecond)
		return c.String(http.StatusOK, "late")
	})

	rec := do(e, httptest.NewRequest(http.MethodGet, "/__ignores", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "late", rec.Body.String())
}

func TestBehavior_BodyOverLimitReturns413(t *testing.T) {
	e := newServer(t, func(c *config.Config) { c.BodySizeLimit = "1K" })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"`+strings.Repeat("a", 4096)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := do(e, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.JSONEq(t, `{"code":"Request Entity Too Large","message":"Request Entity Too Large"}`, rec.Body.String())
}

func TestBehavior_PanicReturnsStandard500AndServerKeepsServing(t *testing.T) {
	e := newServer(t, nil)
	e.GET("/__panic", func(c *echo.Context) error { panic("boom") })

	rec := do(e, httptest.NewRequest(http.MethodGet, "/__panic", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"code":"Internal Server Error","message":"internal server error"}`, rec.Body.String())

	rec = do(e, httptest.NewRequest(http.MethodGet, "/api/v1/about", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBehavior_CORS(t *testing.T) {
	e := newServer(t, nil)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := do(e, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))

	req = httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec = do(e, req)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "other origins must not be allowed")
}

func loginFrom(e *echo.Echo, remote, forwardedFor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"x@example.com","password":"Wrong1234"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remote
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	return do(e, req)
}

func TestBehavior_AuthRateLimit_ReturnsOwn429Body(t *testing.T) {
	e := newServer(t, func(c *config.Config) { c.AuthRateLimit = 2 })

	var last *httptest.ResponseRecorder
	for i := 0; i < 8; i++ {
		last = loginFrom(e, "10.0.0.9:5555", "")
	}
	assert.Equal(t, http.StatusTooManyRequests, last.Code)
	// The limiter writes its own body, outside the central error handler.
	assert.JSONEq(t, `{"error":"rate limit exceeded"}`, last.Body.String())
}

func TestBehavior_AuthRateLimit_IsPerClientBehindTrustedProxy(t *testing.T) {
	e := newServer(t, func(c *config.Config) { c.AuthRateLimit = 2 })
	proxy := "172.18.0.5:4444" // the frontend nginx container on the Docker network

	// Client A exhausts its own allowance...
	var a *httptest.ResponseRecorder
	for i := 0; i < 8; i++ {
		a = loginFrom(e, proxy, "198.51.100.10")
	}
	assert.Equal(t, http.StatusTooManyRequests, a.Code)

	// ...while client B, arriving through the same proxy, is still served.
	b := loginFrom(e, proxy, "198.51.100.20")
	assert.NotEqual(t, http.StatusTooManyRequests, b.Code)
}

// Echo v4 honoured X-Forwarded-For from any peer, so a client connecting
// directly could rotate the header to dodge the limit. With the forwarded-header
// extractor (ADR-037) the header is only believed when it arrives through a
// private or loopback proxy; from a public peer it is ignored and the peer
// itself is the bucket, so rotating the header no longer helps.
func TestBehavior_AuthRateLimit_IgnoresForwardedHeaderFromPublicPeer(t *testing.T) {
	e := newServer(t, func(c *config.Config) { c.AuthRateLimit = 2 })
	public := "203.0.113.50:6666"

	limited := false
	for i := 0; i < 8; i++ {
		rec := loginFrom(e, public, "198.51.100."+string(rune('1'+i)))
		if rec.Code == http.StatusTooManyRequests {
			limited = true
		}
	}
	assert.True(t, limited, "rotating X-Forwarded-For from a public peer must not escape the limit")
}

func TestBehavior_SwaggerUIServes(t *testing.T) {
	e := newServer(t, nil)
	rec := do(e, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}
