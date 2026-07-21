package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBDSN      string `mapstructure:"DB_DSN"`

	// Connection pool — optional, per-driver defaults apply when zero.
	// SQLite always uses MaxOpenConns=1 regardless of this setting.
	DBMaxOpenConns    int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns    int    `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetime string `mapstructure:"DB_CONN_MAX_LIFETIME"` // e.g. "5m"; "0" = never expire

	JWTSecret     string `mapstructure:"JWT_SECRET"`
	JWTAccessTTL  string `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL string `mapstructure:"JWT_REFRESH_TTL"`
	LogLevel      string `mapstructure:"LOG_LEVEL"`
	CORSOrigins   string `mapstructure:"CORS_ORIGINS"`

	ShutdownTimeout string `mapstructure:"SHUTDOWN_TIMEOUT"` // e.g. "30s"

	BodySizeLimit  string `mapstructure:"BODY_SIZE_LIMIT"` // e.g. "1M"
	RequestTimeout string `mapstructure:"REQUEST_TIMEOUT"` // e.g. "30s"
	AuthRateLimit  int    `mapstructure:"AUTH_RATE_LIMIT"` // req/s per IP

	// AppEnv controls security-sensitive defaults (e.g., Secure cookie flag).
	// Set to "development" in local dev; omit or set to "production" in deployed environments.
	AppEnv string `mapstructure:"APP_ENV"`

	// RevokedTokenCleanupInterval is how often the background goroutine purges
	// expired rows from the revoked_tokens table. Defaults to "15m".
	RevokedTokenCleanupInterval string `mapstructure:"REVOKED_TOKEN_CLEANUP_INTERVAL"`

	// SMTP — leave SMTPHost empty to disable email delivery (NoopSender is used).
	SMTPHost string `mapstructure:"SMTP_HOST"`
	SMTPPort string `mapstructure:"SMTP_PORT"`
	SMTPFrom string `mapstructure:"SMTP_FROM"`

	// AppBaseURL is the public-facing base URL of the frontend application
	// (e.g., "https://app.example.com"). Used only for magic-link construction in
	// password-reset emails. Empty string disables magic links (raw token only).
	AppBaseURL string `mapstructure:"APP_BASE_URL"`

	// Attachment storage (S3-compatible). StorageEndpoint is set for LocalStack in
	// local dev and left empty in production so the AWS SDK resolves the real
	// regional S3 endpoint. StorageAccessKeyID/StorageSecretAccessKey are
	// LocalStack-only; production credentials come from the EC2 instance's IAM role.
	StorageEndpoint        string `mapstructure:"STORAGE_ENDPOINT"`
	StorageBucket          string `mapstructure:"STORAGE_BUCKET"`
	StorageRegion          string `mapstructure:"STORAGE_REGION"`
	StorageAccessKeyID     string `mapstructure:"STORAGE_ACCESS_KEY_ID"`
	StorageSecretAccessKey string `mapstructure:"STORAGE_SECRET_ACCESS_KEY"`
	StoragePresignTTL      string `mapstructure:"STORAGE_PRESIGN_TTL"` // e.g. "1h"
	StorageUsePathStyle    bool   `mapstructure:"STORAGE_USE_PATH_STYLE"`
	// StoragePresignEndpoint overrides StorageEndpoint for presigned URLs only —
	// needed locally since the backend reaches LocalStack via a Docker-internal
	// hostname the browser can't resolve. Empty in production (not needed;
	// real S3's hostname is reachable identically everywhere).
	StoragePresignEndpoint string `mapstructure:"STORAGE_PRESIGN_ENDPOINT"`

	// Database idle auto-stop/start (spec 010, db-idle-autostop). Whether the
	// feature is actually active is decided by Config.RDSAutoStopApplies()
	// (pkg/config/rds_hostname.go), not by any single field here: it requires
	// DBDriver to be a supported SQL driver (postgres/postgresql/mysql/mariadb),
	// RDSAutoStopEnabled to not be explicitly disabled, RDSInstanceIdentifier to
	// be configured, AND DBDSN's hostname to match that exact AWS RDS instance's
	// endpoint (not just any database using a driver AWS RDS also happens to
	// support, e.g. local Postgres via docker-compose).
	//
	// RDSAutoStopEnabled is set manually in Load() below, not via a mapstructure
	// bool tag — FR-014 requires an unrecognized value to default to enabled,
	// but viper/cast.ToBool treats unparseable strings as false, the wrong
	// default here.
	RDSAutoStopEnabled    bool
	RDSInstanceIdentifier string `mapstructure:"RDS_INSTANCE_IDENTIFIER"`
	RDSIdleTimeout        string `mapstructure:"RDS_IDLE_TIMEOUT"`        // e.g. "10m"
	RDSIdleCheckInterval  string `mapstructure:"RDS_IDLE_CHECK_INTERVAL"` // e.g. "1m"
	// RDSStartFailureBound is a *count* of consecutive failed start attempts
	// (research.md Decision 6), not a duration — independent of the client's
	// separate 5-minute retry bound (FR-013).
	RDSStartFailureBound int `mapstructure:"RDS_START_FAILURE_BOUND"`
}

// Load reads configuration from a .env file and environment variables.
// It panics if required fields (e.g. JWTSecret) are missing.
func Load() *Config {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../../")
	viper.AddConfigPath("../../../")

	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DB_DRIVER", "sqlite")
	viper.SetDefault("DB_DSN", "./dev.db")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 0)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 2)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "0")
	viper.SetDefault("JWT_SECRET", "")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001")
	viper.SetDefault("SHUTDOWN_TIMEOUT", "30s")
	viper.SetDefault("BODY_SIZE_LIMIT", "4M")
	viper.SetDefault("REQUEST_TIMEOUT", "30s")
	viper.SetDefault("AUTH_RATE_LIMIT", 20)
	viper.SetDefault("APP_ENV", "production")
	viper.SetDefault("REVOKED_TOKEN_CLEANUP_INTERVAL", "15m")
	viper.SetDefault("SMTP_HOST", "")
	viper.SetDefault("SMTP_PORT", "1025")
	viper.SetDefault("SMTP_FROM", "noreply@mansooba.local")
	viper.SetDefault("APP_BASE_URL", "")
	viper.SetDefault("STORAGE_ENDPOINT", "")
	viper.SetDefault("STORAGE_BUCKET", "mansooba-attachments")
	viper.SetDefault("STORAGE_REGION", "us-east-1")
	viper.SetDefault("STORAGE_ACCESS_KEY_ID", "")
	viper.SetDefault("STORAGE_SECRET_ACCESS_KEY", "")
	viper.SetDefault("STORAGE_PRESIGN_TTL", "1h")
	viper.SetDefault("STORAGE_USE_PATH_STYLE", false)
	viper.SetDefault("STORAGE_PRESIGN_ENDPOINT", "")
	viper.SetDefault("RDS_AUTOSTOP_ENABLED", "true")
	viper.SetDefault("RDS_INSTANCE_IDENTIFIER", "")
	viper.SetDefault("RDS_IDLE_TIMEOUT", "10m")
	viper.SetDefault("RDS_IDLE_CHECK_INTERVAL", "1m")
	viper.SetDefault("RDS_START_FAILURE_BOUND", 3)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("warning: could not read config file: %v", err)
		}
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("config: failed to unmarshal: %v", err)
	}

	cfg.ServerPort = strings.TrimPrefix(cfg.ServerPort, ":")
	cfg.RDSAutoStopEnabled = !isFalsey(viper.GetString("RDS_AUTOSTOP_ENABLED"))

	if cfg.JWTSecret == "" {
		log.Fatal("config: JWT_SECRET must not be empty")
	}

	return cfg
}

// isFalsey reports whether v is an explicit, recognized "disable" value
// (case-insensitive "false", "0", or "no"). Anything else — including unset,
// unrecognized, or an explicit "true" — is not falsey, per FR-014's
// "unrecognized value defaults to enabled" requirement.
func isFalsey(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "no":
		return true
	default:
		return false
	}
}
