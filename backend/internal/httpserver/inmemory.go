package httpserver

import (
	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/pkg/attachmentstorage"
	"github.com/sharique/mansooba/internal/pkg/avatarstorage"
	"github.com/sharique/mansooba/internal/pkg/lokiclient"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
)

// NewInMemory builds a complete server on a fresh database (cfg.DBDSN,
// normally ":memory:") for tests and the parity replay. Object storage and
// Loki point at addresses nothing listens on, so only code paths that really
// need them fail, exactly as on a stack without LocalStack or Loki. It is
// test support kept next to the code it exercises; production code uses New.
func NewInMemory(cfg *config.Config) (*echo.Echo, func(), error) {
	// Some handlers and services log through the package-level logger, which
	// main initialises at startup; do the same (quietly) when nothing has.
	if logger.Logger == nil {
		logger.Init("error")
	}
	log := zap.NewNop()
	db, err := database.Open(cfg)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = sqlDB.Close() }
	if err := database.Migrate(db); err != nil {
		cleanup()
		return nil, nil, err
	}

	storage, err := attachmentstorage.New(attachmentstorage.Config{
		Endpoint:        "http://127.0.0.1:1",
		Bucket:          "test",
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		UsePathStyle:    true,
	})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	lokiClient := lokiclient.New("http://127.0.0.1:1")
	settingRepo := repository.NewSettingRepository(db)
	systemLogSvc := service.NewSystemLogService(repository.NewLokiSystemLogRepository(lokiClient), settingRepo, "", log)

	h := NewHandlers(Inputs{
		Cfg: cfg, Log: log, DB: db, SQLDB: sqlDB, Loki: lokiClient, SystemLogSvc: systemLogSvc,
		AttachmentStorage: storage,
		AvatarStorage:     avatarstorage.New(storage.Client(), "test"),
		Version:           "dev", SourceCodeURL: "https://github.com/sharique/mansooba",
	})
	return New(Deps{Cfg: cfg, Log: log, H: h, SystemLogSvc: systemLogSvc}), cleanup, nil
}
