package httpserver_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Echo v4 stops receiving security fixes on 2026-12-31 (ADR-037). The module
// file must never list it again, directly or through a dependency that still
// imports it (the old Swagger UI handler was one).
func TestModule_DoesNotRequireEchoV4(t *testing.T) {
	for _, f := range []string{"../../go.mod", "../../go.sum"} {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		assert.False(t, strings.Contains(string(b), "github.com/labstack/echo/v4"), "%s must not mention echo/v4", f)
	}
}
