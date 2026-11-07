package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildServer compiles the real server binary once per test run.
func buildServer(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "server")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	require.NoError(t, err, string(out))
	return bin
}

// runServer starts the binary with only the given settings and returns what it
// printed, its exit status and how long it ran.
func runServer(t *testing.T, bin string, env ...string) (stderr string, exit int, took time.Duration) {
	t.Helper()
	// If a refusal ever stopped working the server would really start: give it a
	// random port (never the developer's 8080) and a hard stop.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "JWT_SECRET=test-secret", "SERVER_PORT=0"}, env...)
	cmd.Dir = t.TempDir() // no .env file to pick up
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	start := time.Now()
	err := cmd.Run()
	took = time.Since(start)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exit = exitErr.ExitCode()
	} else {
		require.NoError(t, err)
	}
	return errBuf.String(), exit, took
}

func TestServer_RefusesSQLiteAtOnceAndTouchesNothing(t *testing.T) {
	bin := buildServer(t)
	dbFile := filepath.Join(t.TempDir(), "x.db")

	stderr, exit, took := runServer(t, bin, "DB_DRIVER=sqlite", "DB_DSN="+dbFile)

	assert.Equal(t, 1, exit)
	assert.Contains(t, stderr, "SQLite is no longer supported")
	assert.Contains(t, stderr, "moving-off-sqlite.md")
	assert.Less(t, took, time.Second, "the refusal must come within the first second")
	_, statErr := os.Stat(dbFile)
	assert.True(t, os.IsNotExist(statErr), "the SQLite file must not be created")
}

func TestServer_RefusesAMissingDriverOrConnectionString(t *testing.T) {
	bin := buildServer(t)

	stderr, exit, _ := runServer(t, bin, "DB_DSN=host=x")
	assert.Equal(t, 1, exit)
	assert.Contains(t, stderr, "DB_DRIVER is not set")

	stderr, exit, _ = runServer(t, bin, "DB_DRIVER=postgres")
	assert.Equal(t, 1, exit)
	assert.Contains(t, stderr, "DB_DSN is not set")
	assert.False(t, strings.Contains(stderr, "panic"), stderr)
}
