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
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001")
	viper.SetDefault("SHUTDOWN_TIMEOUT", "30s")

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

	if cfg.JWTSecret == "" {
		log.Fatal("config: JWT_SECRET must not be empty")
	}

	return cfg
}
