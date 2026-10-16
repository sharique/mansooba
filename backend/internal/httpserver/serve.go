package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

// ErrShutdown is wrapped by the error Serve returns when in-flight requests
// did not finish within the graceful timeout.
var ErrShutdown = errors.New("graceful shutdown did not complete")

// Serve runs e on addr until ctx is cancelled, then stops accepting
// connections and lets in-flight requests finish for up to gracefulTimeout
// (SHUTDOWN_TIMEOUT). It returns once shutdown has finished: nil for a clean
// stop, an error wrapping ErrShutdown if requests were still running when the
// timeout ended, or the listen/serve error. listening, when non-nil, receives
// the bound address (tests use ":0").
func Serve(ctx context.Context, e *echo.Echo, addr string, gracefulTimeout time.Duration, log *zap.Logger, listening func(net.Addr)) error {
	var shutdownErr error
	sc := echo.StartConfig{
		Address:          addr,
		HideBanner:       true,
		HidePort:         true,
		GracefulTimeout:  gracefulTimeout,
		ListenerAddrFunc: listening,
		OnShutdownError: func(err error) {
			shutdownErr = err
			log.Error("graceful shutdown failed", zap.Error(err))
		},
		// Echo v5 sets a 30 s ReadTimeout by default; v4 had none. Uploads of
		// up to 25 MB over a slow link must keep working, so keep v4's
		// behaviour (ADR-037).
		BeforeServeFunc: func(s *http.Server) error {
			s.ReadTimeout = 0
			return nil
		},
	}
	if err := sc.Start(ctx, e); err != nil {
		return err
	}
	if shutdownErr != nil {
		return fmt.Errorf("%w: %v", ErrShutdown, shutdownErr)
	}
	return nil
}
