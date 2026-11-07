package testdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TEST_DB_DSN may be a key/value string or a postgres:// URL; either way the
// per-test database replaces whatever database the string named.
func TestWithDatabase(t *testing.T) {
	assert.Equal(t, "host=h user=u dbname=x dbname=test_1", withDatabase("host=h user=u dbname=x", "test_1"),
		"in a key/value string the last dbname wins")
	assert.Equal(t, "postgres://u:p@h:5432/test_1?sslmode=disable",
		withDatabase("postgres://u:p@h:5432/mansooba_test?sslmode=disable", "test_1"))
	assert.Equal(t, "postgres://u:p@h/test_1", withDatabase("postgres://u:p@h", "test_1"))
	assert.Equal(t, "postgresql://u@h/test_1", withDatabase("postgresql://u@h/other", "test_1"))
}

// The stale sweep only touches databases this helper made: test_<unix>_<suffix>.
func TestIsTestDatabaseName(t *testing.T) {
	for name, want := range map[string]bool{
		"test_1700000000_ab12cd": true,
		"test_1699999999_0":      true,
		"test_data":              false,
		"test_1700000000":        false,
		"test_abc_xyz":           false,
		"test_1700000000_A-b":    false,
		"mytest_1700000000_ab":   false,
		"mansooba_tmpl_0123abc":  false,
		"":                       false,
	} {
		_, ok := parseTestDatabaseName(name)
		assert.Equal(t, want, ok, name)
	}
	unix, ok := parseTestDatabaseName("test_1700000000_ab12cd")
	assert.True(t, ok)
	assert.Equal(t, int64(1700000000), unix)
}
