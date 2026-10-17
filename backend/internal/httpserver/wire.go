// Package httpserver builds the HTTP server: it wires repositories, services
// and handlers (NewHandlers) and registers middleware and routes (New).
// It was extracted from cmd/server/main.go (feature 018) so the real
// middleware stack can be exercised by tests. Infrastructure clients (database,
// object storage, Loki, RDS) are still constructed once in main and injected.
package httpserver

import (
	"database/sql"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/email"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/pkg/attachmentstorage"
	"github.com/sharique/mansooba/internal/pkg/avatarstorage"
	"github.com/sharique/mansooba/internal/pkg/lokiclient"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

// Inputs is the injected infrastructure and settings NewHandlers needs.
type Inputs struct {
	Cfg               *config.Config
	Log               *zap.Logger
	DB                *gorm.DB
	SQLDB             *sql.DB
	Loki              *lokiclient.Client
	SystemLogSvc      service.SystemLogService
	AttachmentStorage *attachmentstorage.Storage
	AvatarStorage     *avatarstorage.Storage
	Version           string
	SourceCodeURL     string
}

// Handlers holds every HTTP handler the router registers.
type Handlers struct {
	Health         *handler.HealthHandler
	About          *handler.AboutHandler
	SystemLog      *handler.SystemLogHandler
	Auth           *handler.AuthHandler
	Setup          *handler.SetupHandler
	User           *handler.UserHandler
	Project        *handler.ProjectHandler
	Issue          *handler.IssueHandler
	Board          *handler.BoardHandler
	Sprint         *handler.SprintHandler
	Overview       *handler.OverviewHandler
	Comment        *handler.CommentHandler
	Attachment     *handler.AttachmentHandler
	Activity       *handler.ActivityHandler
	Label          *handler.LabelHandler
	Notification   *handler.NotificationHandler
	Setting        *handler.SettingHandler
	IssueRelation  *handler.IssueRelationHandler
	AdminUser      *handler.AdminUserHandler
	PasswordReset  *handler.PasswordResetHandler
	PasswordChange *handler.PasswordChangeHandler
	Username       *handler.UsernameHandler
}

// NewHandlers constructs repositories, services and handlers exactly as
// cmd/server/main.go did before the extraction.
func NewHandlers(in Inputs) Handlers {
	cfg, log, db := in.Cfg, in.Log, in.DB

	// Repositories
	userRepo := repository.NewUserRepository(db)
	revokedTokenRepo := repository.NewRevokedTokenRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	projectMemberRepo := repository.NewProjectMemberRepository(db)
	issueRepo := repository.NewIssueRepository(db)
	sprintRepo := repository.NewSprintRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	issueRelationRepo := repository.NewIssueRelationRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)

	// Services
	authSvc := service.NewAuthService(userRepo, revokedTokenRepo, in.SystemLogSvc, log, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userSvc := service.NewUserService(userRepo, in.AvatarStorage)
	notifSvc := service.NewNotificationService(notifRepo, projectMemberRepo, log)
	projectSvc := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, issueRepo, notifSvc)
	activitySvc := service.NewActivityService(activityRepo, userRepo, issueRepo, projectMemberRepo)
	issueSvc := service.NewIssueService(issueRepo, projectRepo, projectMemberRepo, activitySvc, userRepo, sprintRepo).
		WithAttachments(attachmentRepo, in.AttachmentStorage).
		WithNotifier(notifSvc)
	boardSvc := service.NewBoardService(issueRepo, projectRepo, projectMemberRepo)
	sprintSvc := service.NewSprintService(sprintRepo, issueRepo, projectRepo, projectMemberRepo)
	overviewSvc := service.NewOverviewService(projectRepo, projectMemberRepo, sprintRepo, issueRepo, activityRepo, userRepo)
	commentSvc := service.NewCommentService(commentRepo, issueRepo, projectMemberRepo, activitySvc, notifRepo, userRepo)
	attachmentSvc := service.NewAttachmentService(attachmentRepo, issueRepo, projectRepo, projectMemberRepo, activitySvc, userRepo, in.AttachmentStorage)
	labelSvc := service.NewLabelService(repository.NewLabelRepository(db), issueRepo, projectRepo, projectMemberRepo, activitySvc)
	settingSvc := service.NewSettingService(settingRepo, userRepo, in.SystemLogSvc)
	issueRelationSvc := service.NewIssueRelationService(issueRelationRepo, issueRepo, projectMemberRepo)
	var emailSender domain.EmailSender
	if cfg.SMTPHost != "" {
		emailSender = email.SMTPSender{Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.SMTPFrom, AppBaseURL: cfg.AppBaseURL}
		log.Info("email delivery enabled", zap.String("smtp_host", cfg.SMTPHost), zap.String("smtp_port", cfg.SMTPPort))
	} else {
		emailSender = email.NoopSender{}
		log.Info("email delivery disabled — token returned in API response only")
	}
	passwordResetSvc := service.NewPasswordResetService(userRepo, passwordResetRepo, emailSender)
	passwordChangeSvc := service.NewPasswordChangeService(userRepo, in.SystemLogSvc, emailSender)

	adminUserSvc := service.NewAdminUserService(userRepo, in.SystemLogSvc)

	// Setup service (wizard)
	accessTTL, _ := time.ParseDuration(cfg.JWTAccessTTL)
	setupSvc := service.NewSetupService(userRepo, projectSvc, cfg.JWTSecret, accessTTL, log, db)

	// Instance info: version and source location, both resolved by the
	// caller; License/LicenseURL are fixed.
	instanceSvc := service.NewInstanceService(domain.InstanceInfo{
		Version:   in.Version,
		SourceURL: in.SourceCodeURL,
	})

	h := Handlers{
		Health:         handler.NewHealthHandler(in.SQLDB).WithLoki(in.Loki),
		About:          handler.NewAboutHandler(instanceSvc),
		SystemLog:      handler.NewSystemLogHandler(in.SystemLogSvc, userSvc),
		Auth:           handler.NewAuthHandler(authSvc, userSvc),
		Setup:          handler.NewSetupHandler(setupSvc, settingSvc),
		User:           handler.NewUserHandler(userSvc, activitySvc, issueSvc),
		Project:        handler.NewProjectHandler(projectSvc),
		Issue:          handler.NewIssueHandler(issueSvc).WithRelationRepo(issueRelationRepo),
		Board:          handler.NewBoardHandler(boardSvc),
		Sprint:         handler.NewSprintHandler(sprintSvc),
		Overview:       handler.NewOverviewHandler(overviewSvc),
		Comment:        handler.NewCommentHandler(commentSvc),
		Attachment:     handler.NewAttachmentHandler(attachmentSvc),
		Activity:       handler.NewActivityHandler(activitySvc),
		Label:          handler.NewLabelHandler(labelSvc),
		Notification:   handler.NewNotificationHandler(notifRepo),
		Setting:        handler.NewSettingHandler(settingSvc, userSvc),
		IssueRelation:  handler.NewIssueRelationHandler(issueRelationSvc),
		AdminUser:      handler.NewAdminUserHandler(adminUserSvc, userSvc),
		PasswordReset:  handler.NewPasswordResetHandler(passwordResetSvc),
		PasswordChange: handler.NewPasswordChangeHandler(passwordChangeSvc, authSvc),
		Username:       handler.NewUsernameHandler(service.NewUsernameService(userRepo), userSvc),
	}
	return h
}
