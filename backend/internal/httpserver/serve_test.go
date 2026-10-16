package httpserver_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func startServe(t *testing.T, e *echo.Echo, graceful time.Duration) (addr string, cancel context.CancelFunc, done chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	bound := make(chan net.Addr, 1)
	done = make(chan error, 1)
	go func() {
		done <- httpserver.Serve(ctx, e, "127.0.0.1:0", graceful, zap.NewNop(), func(a net.Addr) { bound <- a })
	}()
	select {
	case a := <-bound:
		return a.String(), cancel, done
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}
	return "", cancel, done
}

// SIGTERM handling (disposability, constitution VI.9): requests that are
// already running when shutdown starts finish normally, and Serve returns
// once they have.
func TestServe_ShutdownLetsInFlightRequestsFinish(t *testing.T) {
	e := newServer(t, nil)
	started := make(chan struct{})
	e.GET("/__inflight", func(c *echo.Context) error {
		close(started)
		time.Sleep(300 * time.Millisecond)
		return c.String(http.StatusOK, "done")
	})
	addr, cancel, done := startServe(t, e, 5*time.Second)

	type result struct {
		code int
		err  error
	}
	res := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/__inflight")
		if err != nil {
			res <- result{err: err}
			return
		}
		defer resp.Body.Close()
		res <- result{code: resp.StatusCode}
	}()

	<-started
	cancel() // shutdown begins while the request is running

	r := <-res
	require.NoError(t, r.err)
	assert.Equal(t, http.StatusOK, r.code)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after shutdown")
	}

	// The listener is closed: new connections are refused.
	_, err := http.Get("http://" + addr + "/api/v1/about")
	assert.Error(t, err)
}

func TestServe_ReportsShutdownTimeout(t *testing.T) {
	e := newServer(t, nil)
	started := make(chan struct{})
	e.GET("/__stuck", func(c *echo.Context) error {
		close(started)
		time.Sleep(1500 * time.Millisecond)
		return c.String(http.StatusOK, "late")
	})
	addr, cancel, done := startServe(t, e, 100*time.Millisecond)

	go func() { _, _ = http.Get("http://" + addr + "/__stuck") }()
	<-started
	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, httpserver.ErrShutdown)
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the graceful timeout")
	}
}
