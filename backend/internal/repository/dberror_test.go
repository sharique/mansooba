package repository

import (
	"errors"
	"fmt"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestUniqueViolation(t *testing.T) {
	pg := &pgconn.PgError{Code: "23505", ConstraintName: "idx_users_username"}

	name, ok := uniqueViolation(pg)
	assert.True(t, ok)
	assert.Equal(t, "idx_users_username", name)

	name, ok = uniqueViolation(fmt.Errorf("create user: %w", pg))
	assert.True(t, ok, "a wrapped error is still recognised")
	assert.Equal(t, "idx_users_username", name)

	_, ok = uniqueViolation(&pgconn.PgError{Code: "23503", ConstraintName: "fk_x"})
	assert.False(t, ok, "a foreign-key violation is not a unique violation")

	// MySQL 8 names the table as well; MariaDB gives the index name alone.
	name, ok = uniqueViolation(&mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'sam' for key 'users.idx_users_username'"})
	assert.True(t, ok)
	assert.Equal(t, "idx_users_username", name)
	name, ok = uniqueViolation(&mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'sam' for key 'idx_users_username'"})
	assert.True(t, ok)
	assert.Equal(t, "idx_users_username", name)

	_, ok = uniqueViolation(&mysqldriver.MySQLError{Number: 1452, Message: "Cannot add or update a child row"})
	assert.False(t, ok)
	_, ok = uniqueViolation(errors.New("duplicate key value violates unique constraint"))
	assert.False(t, ok, "wording alone is not a database error code")
}

func TestTranslateUsernameConflict_OnlyForTheUsernameIndex(t *testing.T) {
	username := &pgconn.PgError{Code: "23505", ConstraintName: "idx_users_username"}
	assert.ErrorIs(t, translateUsernameConflict(username), domain.ErrUsernameTaken)

	email := &pgconn.PgError{Code: "23505", ConstraintName: "idx_users_email"}
	assert.Equal(t, error(email), translateUsernameConflict(email), "a duplicate e-mail is not a taken username")

	// Wording that merely mentions the words must not be mistaken for the violation.
	lookalike := errors.New("duplicate unique username in the log message")
	assert.Equal(t, lookalike, translateUsernameConflict(lookalike))

	mysql := &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'sam' for key 'users.idx_users_username'"}
	assert.ErrorIs(t, translateUsernameConflict(mysql), domain.ErrUsernameTaken)
}

func TestTranslateMembershipConflict_OnlyForTheMembershipIndex(t *testing.T) {
	member := &pgconn.PgError{Code: "23505", ConstraintName: "idx_project_user"}
	assert.ErrorIs(t, translateMembershipConflict(member), domain.ErrConflict)

	other := &pgconn.PgError{Code: "23505", ConstraintName: "idx_projects_key"}
	assert.Equal(t, error(other), translateMembershipConflict(other))

	mysql := &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry '1-2' for key 'project_members.idx_project_user'"}
	assert.ErrorIs(t, translateMembershipConflict(mysql), domain.ErrConflict)
}
