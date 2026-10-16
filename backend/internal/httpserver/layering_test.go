package httpserver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Clean architecture (constitution principle II, CLAUDE.md): the web framework
// belongs to the transport layer. Services, the domain, repositories and
// infrastructure packages must not import it and must return domain errors,
// which handlers map to HTTP.
func TestLayering_OnlyTransportPackagesImportTheWebFramework(t *testing.T) {
	for _, dir := range []string{"../service", "../domain", "../repository", "../dto", "../seed", "../email", "../pkg"} {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			require.NoError(t, err)
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			b, rerr := os.ReadFile(path)
			require.NoError(t, rerr)
			assert.NotContains(t, string(b), "github.com/labstack/echo", "%s imports the web framework", path)
			return nil
		})
		require.NoError(t, err)
	}
}
