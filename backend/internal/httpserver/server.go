package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	apimw "github.com/sharique/mansooba/internal/middleware"
	"github.com/sharique/mansooba/internal/pkg/rdsclient"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/apierror"
	"github.com/sharique/mansooba/pkg/config"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i any) error { return cv.v.Struct(i) }

// Deps is what New needs besides the handlers.
type Deps struct {
	Cfg *config.Config
	Log *zap.Logger
	H   Handlers
	// Tracker and RDS are nil unless RDS idle auto-stop is enabled; when set
	// the wake-up and activity middleware are registered.
	Tracker      *service.DBLifecycleTracker
	RDS          *rdsclient.Client
	SystemLogSvc service.SystemLogService
}

// limiterConfig is the shared per-client-IP rate limiter configuration used by
// the account-creation and authentication route groups. denyBody is the JSON
// body written when the limit is exceeded (429).
func limiterConfig(cfg *config.Config, denyBody map[string]string) echomw.RateLimiterConfig {
	return echomw.RateLimiterConfig{
		Skipper: echomw.DefaultSkipper,
		Store: echomw.NewRateLimiterMemoryStoreWithConfig(
			echomw.RateLimiterMemoryStoreConfig{
				Rate:      float64(cfg.AuthRateLimit),
				Burst:     cfg.AuthRateLimit,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context *echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		},
		DenyHandler: func(context *echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, denyBody)
		},
	}
}

// New builds the Echo instance: validator, error handler, middleware and routes.
func New(d Deps) *echo.Echo {
	cfg, log, h := d.Cfg, d.Log, d.H

	e := echo.New()

	// Behind the frontend nginx the peer is the proxy, so the rate limiters
	// must key on the client address nginx forwards. This trusts loopback and
	// private proxies and returns the nearest untrusted address in
	// X-Forwarded-For (reading from the right), so a client cannot choose its
	// own bucket. Echo v5's default would use the proxy's address for everyone;
	// Echo v4 trusted the left-most header from any peer (ADR-037).
	e.IPExtractor = echo.ExtractIPFromXFFHeader()

	v := validator.New()
	v.RegisterValidation("password_complexity", func(fl validator.FieldLevel) bool { //nolint:errcheck
		pw := fl.Field().String()
		var hasUpper, hasLower, hasDigit bool
		for _, r := range pw {
			switch {
			case unicode.IsUpper(r):
				hasUpper = true
			case unicode.IsLower(r):
				hasLower = true
			case unicode.IsDigit(r):
				hasDigit = true
			}
		}
		return len(pw) >= 8 && hasUpper && hasLower && hasDigit
	})
	e.Validator = &customValidator{v}
	e.HTTPErrorHandler = apierror.HTTPErrorHandler

	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     strings.Split(cfg.CORSOrigins, ","),
		AllowCredentials: true,
	}))

	// Body limit — reject payloads > BodySizeLimit with 413
	bodyLimit, err := parseSize(cfg.BodySizeLimit)
	if err != nil {
		// Same outcome as Echo v4, whose BodyLimit panicked on a bad value:
		// refuse to start with a misconfigured limit.
		panic("invalid BODY_SIZE_LIMIT: " + err.Error())
	}
	e.Use(echomw.BodyLimit(bodyLimit))

	// Request timeout — abort requests exceeding RequestTimeout with 503
	requestTimeout, err := time.ParseDuration(cfg.RequestTimeout)
	if err != nil {
		log.Warn("invalid REQUEST_TIMEOUT value, defaulting to 30s", zap.String("value", cfg.RequestTimeout), zap.Error(err))
		requestTimeout = 30 * time.Second
	}
	e.Use(echomw.ContextTimeoutWithConfig(echomw.ContextTimeoutConfig{
		Timeout:      requestTimeout,
		ErrorHandler: timeoutResponse,
	}))

	// Database idle auto-stop / wake-on-hit (spec 010) — only registered when
	// the feature is actually enabled, so both are a complete no-op otherwise,
	// not just an inert call. DBWake runs first so a request hitting a stopped
	// database gets the waking_up signal immediately, without DBActivity or any
	// handler attempting real (doomed) DB work.
	if d.Tracker != nil {
		e.Use(apimw.DBWake(d.Tracker, d.RDS, log, d.SystemLogSvc))
		e.Use(apimw.DBActivity(d.Tracker))
	}

	tooMany := map[string]string{"error": "Too many attempts. Please wait a minute and try again."}

	// Public routes
	e.GET("/health", h.Health.Check)
	// Public by design (016-agpl-dual-licensing): lets any user of this
	// instance, including on the sign-in page, find its source (AGPL-3.0
	// section 13). Returns no user or instance-configuration data.
	e.GET("/api/v1/about", h.About.Get)

	auth := e.Group("/api/v1/auth")
	auth.Use(echomw.RateLimiterWithConfig(limiterConfig(cfg, map[string]string{"error": "rate limit exceeded"})))
	auth.POST("/login", h.Auth.Login)
	auth.POST("/refresh", h.Auth.Refresh)
	auth.POST("/logout", h.Auth.Logout)
	auth.POST("/forgot-password", h.PasswordReset.ForgotPassword)
	auth.POST("/reset-password", h.PasswordReset.ResetPassword)

	// Setup wizard routes (public group — no JWT required)
	setup := e.Group("/api/v1/setup")
	setup.GET("/status", h.Setup.Status)

	// Account-creation steps are rate-limited (same config as auth)
	setupRateLimited := setup.Group("", echomw.RateLimiterWithConfig(limiterConfig(cfg, tooMany)))
	setupRateLimited.POST("/admin", h.Setup.CreateAdmin)

	// Authenticated setup steps (require JWT from step 1)
	setupAuth := setup.Group("", apimw.JWTAuth(cfg.JWTSecret))
	setupAuth.Use(echomw.RateLimiterWithConfig(limiterConfig(cfg, tooMany)))
	setupAuth.POST("/user", h.Setup.CreateUser)
	setupAuth.POST("/project", h.Setup.CreateProject)
	setupAuth.POST("/seed", h.Setup.SeedData)

	// Protected routes
	api := e.Group("/api/v1", apimw.JWTAuth(cfg.JWTSecret))

	authMe := api.Group("/auth")
	authMe.POST("/register", h.Auth.Register)
	authMe.GET("/me", h.User.GetProfile)
	authMe.PUT("/me", h.User.UpdateProfile)
	authMe.GET("/me/activity", h.User.GetMyActivity)
	authMe.GET("/me/issues", h.User.GetMyIssues)
	authMe.POST("/me/avatar", h.User.UploadAvatar)
	authMe.DELETE("/me/avatar", h.User.DeleteAvatar)

	// Change-password is rate-limited per source IP, unlike authMe's other
	// routes — mirrors the setupAuth pattern (the only existing precedent for
	// "JWT-protected + rate-limited" in this codebase).
	authMePasswordLimited := authMe.Group("")
	authMePasswordLimited.Use(echomw.RateLimiterWithConfig(limiterConfig(cfg, tooMany)))
	authMePasswordLimited.PUT("/me/password", h.PasswordChange.ChangePassword)

	// Public avatar serving, streamed from S3 (avatars/ prefix of the shared
	// bucket). Intentionally unauthenticated — browser <img> tags fetch images
	// as unauthenticated sub-resources and avatars carry no per-viewer access
	// restriction. Rationale documented in ADR-033.
	e.GET("/avatars/:filename", h.User.ServeAvatar)

	projects := api.Group("/projects")
	projects.GET("", h.Project.List)
	projects.POST("", h.Project.Create)
	projects.GET("/:key", h.Project.Get)
	projects.PUT("/:key", h.Project.Update)
	projects.DELETE("/:key", h.Project.Delete)
	projects.GET("/:key/members", h.Project.ListMembers)
	projects.POST("/:key/members", h.Project.AddMember)
	projects.DELETE("/:key/members/:userId", h.Project.RemoveMember)
	projects.GET("/:key/labels", h.Label.List)
	projects.POST("/:key/labels", h.Label.Create)
	projects.DELETE("/:key/labels/:lid", h.Label.Delete)

	issues := api.Group("/projects/:key/issues")
	issues.GET("", h.Issue.List)
	issues.POST("", h.Issue.Create)
	issues.GET("/:id", h.Issue.Get)
	issues.PUT("/:id", h.Issue.Update)
	issues.DELETE("/:id", h.Issue.Delete)

	api.GET("/projects/:key/board", h.Board.GetBoard)

	sprints := api.Group("/projects/:key/sprints")
	sprints.GET("", h.Sprint.List)
	sprints.POST("", h.Sprint.Create)
	sprints.GET("/:id", h.Sprint.Get)
	sprints.PUT("/:id", h.Sprint.Update)
	sprints.DELETE("/:id", h.Sprint.Delete)
	sprints.POST("/:id/start", h.Sprint.Start)
	sprints.POST("/:id/complete", h.Sprint.Complete)
	sprints.GET("/:id/burndown", h.Sprint.Burndown)
	sprints.GET("/:id/issues", h.Sprint.GetIssues)

	api.GET("/projects/:key/backlog", h.Sprint.Backlog)
	api.GET("/projects/:key/velocity", h.Sprint.Velocity)

	issueItems := api.Group("/issues/:id")
	issueItems.GET("/comments", h.Comment.List)
	issueItems.POST("/comments", h.Comment.Create)
	issueItems.PUT("/comments/:cid", h.Comment.Update)
	issueItems.DELETE("/comments/:cid", h.Comment.Delete)
	issueItems.GET("/activity", h.Activity.ListByIssue)
	issueItems.GET("/labels", h.Label.ListByIssue)
	issueItems.POST("/labels/:lid", h.Label.Attach)
	issueItems.DELETE("/labels/:lid", h.Label.Detach)
	issueItems.GET("/relations", h.IssueRelation.List)
	issueItems.POST("/relations", h.IssueRelation.Create)
	issueItems.DELETE("/relations/:rid", h.IssueRelation.Delete)

	// Attachment upload gets a larger body-size limit than the global default
	// (research.md Decision 3) — batch uploads of multiple ~10MB files would
	// otherwise be rejected by the global 4M BodyLimit applied to every route.
	issueItems.POST("/attachments", h.Attachment.Upload, echomw.BodyLimit(25<<20))
	issueItems.GET("/attachments", h.Attachment.List)
	issueItems.GET("/attachments/:aid/download", h.Attachment.Download)
	issueItems.DELETE("/attachments/:aid", h.Attachment.Delete)

	admin := api.Group("/admin")
	admin.GET("/users", h.AdminUser.ListUsers)
	admin.PATCH("/users/:id", h.AdminUser.PatchUser)
	admin.GET("/usernames/check", h.Username.Check)
	admin.GET("/system-logs", h.SystemLog.List)

	settings := api.Group("/settings")
	settings.GET("", h.Setting.GetAll)
	settings.PATCH("", h.Setting.Patch)

	notifs := api.Group("/notifications")
	notifs.GET("", h.Notification.List)
	notifs.PUT("/:id/read", h.Notification.MarkRead)

	e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))

	return e
}

// requestTimeoutBody is the body Echo v4's timeout middleware answered with.
const requestTimeoutBody = "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"

// timeoutResponse turns a context deadline error returned by a handler into the
// response Echo v4 produced when a request exceeded REQUEST_TIMEOUT: status 503
// with a small HTML body and no explicit Content-Type (net/http sniffs it).
// Echo v5's own ContextTimeout would answer with JSON instead; keeping the
// bytes identical keeps the behaviour unchanged for clients (ADR-037).
func timeoutResponse(c *echo.Context, err error) error {
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if resp, uerr := echo.UnwrapResponse(c.Response()); uerr == nil && resp.Committed {
		return err
	}
	c.Response().WriteHeader(http.StatusServiceUnavailable)
	_, _ = c.Response().Write([]byte(requestTimeoutBody))
	return nil
}
