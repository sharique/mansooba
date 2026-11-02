package service_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// What each role may do is decided in one place (domain/access.go, applied by
// service/access.go). This scan fails when a service or handler starts
// comparing role names or the admin flag itself again, which is how the rules
// used to drift apart.
var inlineRolePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.Role\s*(==|!=)`),
	regexp.MustCompile(`Role:\s*"(admin|member|viewer|owner)"`),
	regexp.MustCompile(`"owner"`),
	regexp.MustCompile(`\.OwnerID\s*(==|!=)`),
}

// The administrator flag is read to show it, never to decide (the services ask
// Access.RequireSystem; handlers ask UserService.RequireSystem).
var inlineAdminPattern = regexp.MustCompile(`\.IsAdmin\b`)

func scanFiles(t *testing.T, dirs ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dir := range dirs {
		require.NoError(t, filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			require.NoError(t, err)
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, err := os.ReadFile(path)
			require.NoError(t, err)
			out[path] = string(b)
			return nil
		}))
	}
	return out
}

func TestNoInlineRoleChecks(t *testing.T) {
	for path, src := range scanFiles(t, ".", "../handler") {
		if filepath.Base(path) == "access.go" {
			continue
		}
		for n, line := range strings.Split(src, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "//") {
				continue
			}
			for _, re := range inlineRolePatterns {
				if re.MatchString(line) {
					assert.Failf(t, "inline role check", "%s:%d decides access by role name; use service.Access / domain.RoleCan: %s", path, n+1, trim)
				}
			}
		}
	}
}

func TestNoInlineSystemAdminChecks(t *testing.T) {
	for path, src := range scanFiles(t, ".", "../handler") {
		if filepath.Base(path) == "access.go" {
			continue // the one place that asks
		}
		for n, line := range strings.Split(src, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "//") || !inlineAdminPattern.MatchString(line) {
				continue
			}
			// reading the flag to show it (responses, lists) is fine; deciding on it is not
			if strings.Contains(line, "IsAdmin:") || strings.Contains(line, "IsAdmin =") || strings.Contains(line, "req.IsAdmin") || strings.Contains(line, "target.IsAdmin") {
				continue
			}
			assert.Failf(t, "inline admin check", "%s:%d decides access on the admin flag; use Access.RequireSystem: %s", path, n+1, trim)
		}
	}
}
