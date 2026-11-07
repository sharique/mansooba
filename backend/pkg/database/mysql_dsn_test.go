package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The MySQL driver reports rows *changed* by an UPDATE unless told to report
// rows *matched*, which is what PostgreSQL does and what the repositories rely
// on to tell "nothing to change" from "no such row".
func TestMySQLDSNWithFoundRows_AddsTheFlagAndKeepsTheRest(t *testing.T) {
	in := "mansooba:secret@tcp(db:3306)/mansooba?charset=utf8mb4&parseTime=True&loc=Local"
	out, err := mysqlDSNWithFoundRows(in)
	require.NoError(t, err)
	assert.Contains(t, out, "clientFoundRows=true")
	assert.Contains(t, out, "charset=utf8mb4")
	assert.Contains(t, out, "parseTime=true")
	assert.Contains(t, out, "mansooba:secret@tcp(db:3306)/mansooba")
}

func TestMySQLDSNWithFoundRows_IsIdempotent(t *testing.T) {
	once, err := mysqlDSNWithFoundRows("u:p@tcp(h:3306)/d?clientFoundRows=true")
	require.NoError(t, err)
	twice, err := mysqlDSNWithFoundRows(once)
	require.NoError(t, err)
	assert.Equal(t, once, twice)
}

func TestMySQLDSNWithFoundRows_RejectsAnUnparseableDSN(t *testing.T) {
	_, err := mysqlDSNWithFoundRows("not a dsn")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "not a dsn", "the DSN may hold a password and must not be echoed")
}
