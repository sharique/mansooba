package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const guideURL = "https://github.com/sharique/mansooba/blob/main/docs/deployments/moving-off-sqlite.md"

func TestValidateDatabase_Refusals(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"no driver", Config{DBDSN: "host=x"},
			"database: DB_DRIVER is not set. Set it to one of: postgres, mysql, mariadb."},
		{"blank driver", Config{DBDriver: "   ", DBDSN: "host=x"},
			"database: DB_DRIVER is not set. Set it to one of: postgres, mysql, mariadb."},
		{"sqlite", Config{DBDriver: "sqlite", DBDSN: "./dev.db"},
			"database: SQLite is no longer supported. Set DB_DRIVER to one of: postgres, mysql, mariadb. See " + guideURL},
		{"sqlite3, any case", Config{DBDriver: "SQLite3", DBDSN: "/data/dev.db"},
			"database: SQLite is no longer supported. Set DB_DRIVER to one of: postgres, mysql, mariadb. See " + guideURL},
		{"unknown driver", Config{DBDriver: "mongo", DBDSN: "mongodb://localhost"},
			`database: unsupported DB_DRIVER "mongo". Supported: postgres, mysql, mariadb.`},
		{"postgres without a connection string", Config{DBDriver: "postgres"},
			"database: DB_DSN is not set. Give the connection string for postgres; see backend/.env.example."},
		{"mariadb with a blank connection string", Config{DBDriver: "mariadb", DBDSN: "  "},
			"database: DB_DSN is not set. Give the connection string for mariadb; see backend/.env.example."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabase(&tt.cfg)
			require.Error(t, err)
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

// The driver is checked before the connection string, so a SQLite setup whose
// connection string is a file path still gets the SQLite message.
func TestValidateDatabase_DriverIsCheckedBeforeTheConnectionString(t *testing.T) {
	err := ValidateDatabase(&Config{DBDriver: "sqlite"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SQLite is no longer supported")
}

func TestValidateDatabase_AcceptsTheSupportedDrivers(t *testing.T) {
	for _, driver := range []string{"postgres", "postgresql", "mysql", "mariadb", "PostgreSQL", " mariadb "} {
		t.Run(driver, func(t *testing.T) {
			assert.NoError(t, ValidateDatabase(&Config{DBDriver: driver, DBDSN: "anything"}))
		})
	}
}

// The settings have no defaults: an empty environment must reach the refusal,
// not silently become some database.
func TestLoad_DatabaseSettingsHaveNoDefaults(t *testing.T) {
	t.Chdir(t.TempDir()) // Load also reads a .env from the current and parent directories
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DB_DSN", "")
	cfg := Load()
	assert.Empty(t, cfg.DBDriver)
	assert.Empty(t, cfg.DBDSN)
	assert.Error(t, ValidateDatabase(cfg))
}

// DatabaseKind is the one canonical name the rest of the code switches on.
func TestConfig_DatabaseKind(t *testing.T) {
	tests := map[string]string{
		" PostgreSQL ": "postgres", "postgres": "postgres",
		"MySQL": "mysql", "mariadb": "mysql", " MariaDB": "mysql",
		"sqlite": "", "mongo": "", "": "",
	}
	for in, want := range tests {
		assert.Equal(t, want, (&Config{DBDriver: in}).DatabaseKind(), "driver %q", in)
	}
}

// The pool has bounded defaults (it used to be unlimited).
func TestLoad_PoolDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	cfg := Load()
	assert.Equal(t, 25, cfg.DBMaxOpenConns)
	assert.Equal(t, 5, cfg.DBMaxIdleConns)
}

// The SQLite refusal names a guide by URL. An operator following it must land on
// a page that exists, so map the URL back to a file in this repository.
func TestSQLiteRefusal_PointsAtAGuideThatExists(t *testing.T) {
	err := ValidateDatabase(&Config{DBDriver: "sqlite", DBDSN: "./dev.db"})
	require.Error(t, err)
	_, path, found := strings.Cut(err.Error(), "/blob/main/")
	require.True(t, found, "the message must link to a file in the repository: %v", err)

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..") // backend/pkg/config -> repository root
	info, statErr := os.Stat(filepath.Join(repoRoot, strings.TrimSpace(path)))
	require.NoError(t, statErr, "the guide named in the refusal message does not exist")
	assert.Greater(t, info.Size(), int64(500), "the guide should not be a stub")
}
