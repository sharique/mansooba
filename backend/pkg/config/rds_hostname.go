package config

import "strings"

// normalizeDriver maps a DB_DRIVER value onto a canonical name ("postgres"
// or "mysql"), matching pkg/database/database.go's own driver switch
// (which already treats postgres/postgresql and mysql/mariadb as
// equivalent pairs). Returns "" for sqlite or any unrecognized driver.
func normalizeDriver(driver string) string {
	switch driver {
	case "postgres", "postgresql":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	default:
		return ""
	}
}

// extractDSNHost returns the hostname embedded in dsn for the given driver,
// or "" if it can't be determined. Supports the two DSN grammars this
// project's SQL drivers use:
//   - Postgres: space-separated key=value pairs, e.g. "host=... user=..."
//   - MySQL/MariaDB: "user:pass@tcp(host:port)/dbname?params" (port optional)
func extractDSNHost(driver, dsn string) string {
	switch normalizeDriver(driver) {
	case "postgres":
		for _, field := range strings.Fields(dsn) {
			if host, ok := strings.CutPrefix(field, "host="); ok {
				return host
			}
		}
		return ""
	case "mysql":
		start := strings.Index(dsn, "tcp(")
		if start == -1 {
			return ""
		}
		rest := dsn[start+len("tcp("):]
		end := strings.IndexAny(rest, ":)")
		if end == -1 {
			return ""
		}
		return rest[:end]
	default:
		return ""
	}
}

// matchesRDSInstance reports whether host is the specific AWS RDS endpoint
// for the given instance identifier: it must end with ".rds.amazonaws.com"
// (AWS's guaranteed endpoint suffix for every RDS instance) AND start with
// "<identifier>." (case-insensitive) — confirming not just "this is some
// RDS instance" but "this is the exact instance the operator configured."
func matchesRDSInstance(host, identifier string) bool {
	if host == "" || identifier == "" {
		return false
	}
	host = strings.ToLower(host)
	identifier = strings.ToLower(identifier)
	return strings.HasSuffix(host, ".rds.amazonaws.com") &&
		strings.HasPrefix(host, identifier+".")
}

// RDSAutoStopApplies reports whether the idle auto-stop/wake-on-hit feature
// (spec 010, docs/decisions/ADR-030) should be active. All of the following
// must hold:
//  1. DBDriver is a supported SQL driver (postgres/postgresql/mysql/mariadb)
//  2. RDSAutoStopEnabled is not explicitly disabled (defaults true)
//  3. RDSInstanceIdentifier is configured
//  4. DBDSN's hostname is confirmed as that exact AWS RDS instance's
//     endpoint — not just any database using a driver AWS RDS also happens
//     to support (e.g. local Postgres via docker-compose)
func (c *Config) RDSAutoStopApplies() bool {
	if normalizeDriver(c.DBDriver) == "" {
		return false
	}
	if !c.RDSAutoStopEnabled {
		return false
	}
	if c.RDSInstanceIdentifier == "" {
		return false
	}
	host := extractDSNHost(c.DBDriver, c.DBDSN)
	return matchesRDSInstance(host, c.RDSInstanceIdentifier)
}
