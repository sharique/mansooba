package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	ServerPort    string `mapstructure:"SERVER_PORT"`
	DBDriver      string `mapstructure:"DB_DRIVER"`
	DBDSN         string `mapstructure:"DB_DSN"`
	JWTSecret     string `mapstructure:"JWT_SECRET"`
	JWTAccessTTL  string `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL string `mapstructure:"JWT_REFRESH_TTL"`
	LogLevel      string `mapstructure:"LOG_LEVEL"`
	CORSOrigins   string `mapstructure:"CORS_ORIGINS"`
}

// Load reads configuration from a .env file and environment variables.
// It panics if required fields (e.g. JWTSecret) are missing.
func Load() *Config {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")          // look in working directory
	viper.AddConfigPath("../../")     // also try repo root when running from cmd/server
	viper.AddConfigPath("../../../")  // extra fallback

	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DB_DRIVER", "sqlite")
	viper.SetDefault("DB_DSN", "./dev.db")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("CORS_ORIGINS", "http://localhost:3000")

	if err := viper.ReadInConfig(); err != nil {
		// If .env is missing, rely on env vars + defaults — that's fine.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("warning: could not read config file: %v", err)
		}
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("config: failed to unmarshal: %v", err)
	}

	// Normalize
	cfg.ServerPort = strings.TrimPrefix(cfg.ServerPort, ":")

	if cfg.JWTSecret == "" {
		log.Fatal("config: JWT_SECRET must not be empty")
	}

	return cfg
}
