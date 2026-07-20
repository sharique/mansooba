package config

import "testing"

func TestNormalizeDriver(t *testing.T) {
	tests := []struct {
		driver string
		want   string
	}{
		{"postgres", "postgres"},
		{"postgresql", "postgres"},
		{"mysql", "mysql"},
		{"mariadb", "mysql"},
		{"sqlite", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			if got := normalizeDriver(tt.driver); got != tt.want {
				t.Errorf("normalizeDriver(%q) = %q, want %q", tt.driver, got, tt.want)
			}
		})
	}
}

func TestExtractDSNHost(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		dsn    string
		want   string
	}{
		{
			name:   "postgres local",
			driver: "postgres",
			dsn:    "host=localhost user=mansooba password=secret dbname=mansooba port=5432 sslmode=disable",
			want:   "localhost",
		},
		{
			name:   "postgres RDS",
			driver: "postgres",
			dsn:    "host=mansooba-db.abc123xyz.eu-central-1.rds.amazonaws.com user=mansooba password=secret dbname=mansooba port=5432 sslmode=require",
			want:   "mansooba-db.abc123xyz.eu-central-1.rds.amazonaws.com",
		},
		{
			name:   "postgresql alias",
			driver: "postgresql",
			dsn:    "host=localhost dbname=mansooba",
			want:   "localhost",
		},
		{
			name:   "mysql local",
			driver: "mysql",
			dsn:    "mansooba:secret@tcp(localhost:3306)/mansooba?charset=utf8mb4&parseTime=True&loc=Local",
			want:   "localhost",
		},
		{
			name:   "mariadb RDS",
			driver: "mariadb",
			dsn:    "mansooba:secret@tcp(mansooba-db.abc123xyz.eu-central-1.rds.amazonaws.com:3306)/mansooba?charset=utf8mb4",
			want:   "mansooba-db.abc123xyz.eu-central-1.rds.amazonaws.com",
		},
		{
			name:   "mysql DSN with no port",
			driver: "mysql",
			dsn:    "mansooba:secret@tcp(localhost)/mansooba",
			want:   "localhost",
		},
		{
			name:   "sqlite has no host",
			driver: "sqlite",
			dsn:    "./dev.db",
			want:   "",
		},
		{
			name:   "malformed postgres DSN",
			driver: "postgres",
			dsn:    "user=mansooba dbname=mansooba",
			want:   "",
		},
		{
			name:   "malformed mysql DSN",
			driver: "mysql",
			dsn:    "not-a-valid-dsn",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractDSNHost(tt.driver, tt.dsn); got != tt.want {
				t.Errorf("extractDSNHost(%q, %q) = %q, want %q", tt.driver, tt.dsn, got, tt.want)
			}
		})
	}
}

func TestMatchesRDSInstance(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		identifier string
		want       bool
	}{
		{"matching RDS host", "mansooba-db.abc123xyz.eu-central-1.rds.amazonaws.com", "mansooba-db", true},
		{"case-insensitive match", "Mansooba-DB.abc123xyz.eu-central-1.rds.amazonaws.com", "mansooba-db", true},
		{"local host does not match", "localhost", "mansooba-db", false},
		{"wrong identifier prefix", "other-db.abc123xyz.eu-central-1.rds.amazonaws.com", "mansooba-db", false},
		{"empty host", "", "mansooba-db", false},
		{"empty identifier", "mansooba-db.abc123xyz.eu-central-1.rds.amazonaws.com", "", false},
		{"prefix substring but not a real label boundary", "mansooba-db-other.abc123xyz.eu-central-1.rds.amazonaws.com", "mansooba-db", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesRDSInstance(tt.host, tt.identifier); got != tt.want {
				t.Errorf("matchesRDSInstance(%q, %q) = %v, want %v", tt.host, tt.identifier, got, tt.want)
			}
		})
	}
}
