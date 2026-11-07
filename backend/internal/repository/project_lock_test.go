package repository

import (
	"context"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Adding to a project takes a shared lock on it: any number of inserts proceed
// together, but a project delete (which takes an exclusive lock) waits for them
// and they wait for it.
func TestProjectLock_InsertsShareItAndADeleteExcludesThem(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	p := &domain.Project{Key: "LOCK", Name: "p", OwnerID: 1}
	require.NoError(t, NewProjectRepository(db).Create(ctx, p))

	// One insert is in flight (its transaction holds the shared lock, uncommitted).
	holder := db.Begin()
	require.NoError(t, holder.Error)
	require.NoError(t, lockProjectShared(holder, p.ID))

	// A second insert into the same project is not held up by the first.
	quick, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	require.NoError(t, NewSprintRepository(db).Create(quick, &domain.Sprint{ProjectID: p.ID, Name: "second"}))
	assert.Less(t, time.Since(start), 2*time.Second, "inserts into one project must not queue behind each other")

	// A delete does wait for the insert that is still in flight.
	deleted := make(chan error, 1)
	go func() { deleted <- NewProjectRepository(db).DeleteIfEmpty(ctx, p.ID) }()
	select {
	case err := <-deleted:
		t.Fatalf("the delete ran while an insert held the project (result: %v)", err)
	case <-time.After(400 * time.Millisecond):
	}

	require.NoError(t, holder.Rollback().Error)
	select {
	case err := <-deleted:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the delete never finished after the insert ended")
	}
}
