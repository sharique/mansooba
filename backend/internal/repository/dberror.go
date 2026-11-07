package repository

import (
	"errors"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

// Constraint names the schema gives its unique indexes. The database error says
// which one was violated; these let a repository tell a taken username from a
// taken e-mail without reading the error's wording.
const (
	usernameIndex   = "idx_users_username"
	membershipIndex = "idx_project_user"
)

// uniqueViolation reports whether err is a unique-constraint violation and, when
// the database says so, which constraint or index. It reads the driver's error
// code (PostgreSQL 23505, MySQL and MariaDB 1062), so it does not depend on the
// wording of the message or on the server's language.
func uniqueViolation(err error) (constraint string, ok bool) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return pg.ConstraintName, true
	}
	var my *mysqldriver.MySQLError
	if errors.As(err, &my) && my.Number == 1062 {
		return mysqlKeyName(my.Message), true
	}
	return "", false
}

// mysqlKeyName extracts the key from "Duplicate entry 'x' for key 'users.idx'"
// (MySQL 8 prefixes the table) or "... for key 'idx'" (MariaDB).
func mysqlKeyName(message string) string {
	_, after, found := strings.Cut(message, "for key '")
	if !found {
		return ""
	}
	key, _, _ := strings.Cut(after, "'")
	if i := strings.LastIndex(key, "."); i >= 0 {
		key = key[i+1:]
	}
	return key
}
