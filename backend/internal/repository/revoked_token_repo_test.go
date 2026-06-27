package repository_test

// T010: Integration tests for gormRevokedTokenRepository against real SQLite.
// Constitution Principle III: security paths must use real infrastructure, no mocks.

import (
	"context"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRevokedTokenDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&domain.RevokedToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRevokedTokenRepo_Create_IsIdempotent(t *testing.T) {
	repo := repository.NewRevokedTokenRepository(newRevokedTokenDB(t))
	ctx := context.Background()
	tok := &domain.RevokedToken{
		JTI:       "test-jti-idempotent",
		UserID:    1,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: time.Now(),
	}
	if err := repo.Create(ctx, tok); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	// Second insert with same JTI must return nil (ON CONFLICT DO NOTHING)
	if err := repo.Create(ctx, tok); err != nil {
		t.Errorf("second Create (idempotent) returned error: %v", err)
	}
}

func TestRevokedTokenRepo_Exists_ReturnsTrue_WhenPresent(t *testing.T) {
	repo := repository.NewRevokedTokenRepository(newRevokedTokenDB(t))
	ctx := context.Background()
	_ = repo.Create(ctx, &domain.RevokedToken{
		JTI: "present-jti", UserID: 1,
		ExpiresAt: time.Now().Add(time.Hour), RevokedAt: time.Now(),
	})
	ok, err := repo.Exists(ctx, "present-jti")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Error("expected Exists to return true for inserted JTI")
	}
}

func TestRevokedTokenRepo_Exists_ReturnsFalse_WhenAbsent(t *testing.T) {
	repo := repository.NewRevokedTokenRepository(newRevokedTokenDB(t))
	ok, err := repo.Exists(context.Background(), "nonexistent-jti")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if ok {
		t.Error("expected Exists to return false for unknown JTI")
	}
}

func TestRevokedTokenRepo_DeleteExpired_RemovesOnlyExpiredRows(t *testing.T) {
	repo := repository.NewRevokedTokenRepository(newRevokedTokenDB(t))
	ctx := context.Background()

	_ = repo.Create(ctx, &domain.RevokedToken{
		JTI: "expired-jti", UserID: 1,
		ExpiresAt: time.Now().Add(-time.Hour), // already expired
		RevokedAt: time.Now(),
	})
	_ = repo.Create(ctx, &domain.RevokedToken{
		JTI: "valid-jti", UserID: 1,
		ExpiresAt: time.Now().Add(time.Hour), // still valid
		RevokedAt: time.Now(),
	})

	n, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	// Valid record must still exist
	ok, _ := repo.Exists(ctx, "valid-jti")
	if !ok {
		t.Error("valid-jti was incorrectly deleted")
	}
}
