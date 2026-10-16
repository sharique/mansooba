package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// usernameRepo is a small in-memory UserRepository for the username rules.
// failNextSetWithTaken simulates losing a race: the name looked free but another
// writer took it before this write.
type usernameRepo struct {
	*stubUserRepo
	failNextSetWithTaken int
}

func newUsernameRepo() *usernameRepo { return &usernameRepo{stubUserRepo: newStubUserRepo()} }

func (r *usernameRepo) add(email, username string) *domain.User {
	u := &domain.User{Email: email, Name: email, Username: username}
	_ = r.stubUserRepo.Create(context.Background(), u)
	got, _ := r.stubUserRepo.FindByEmail(context.Background(), email)
	return got
}

func (r *usernameRepo) SetUsername(ctx context.Context, id uint, name string) error {
	if r.failNextSetWithTaken > 0 {
		r.failNextSetWithTaken--
		return domain.ErrUsernameTaken
	}
	return r.stubUserRepo.SetUsername(ctx, id, name)
}

func TestUsernameService_Resolve(t *testing.T) {
	ctx := context.Background()
	repo := newUsernameRepo()
	repo.add("sam@a.com", "sam")
	repo.add("sam2@a.com", "sam2")
	svc := service.NewUsernameService(repo)

	t.Run("a supplied valid free username is normalised and used", func(t *testing.T) {
		got, err := svc.Resolve(ctx, "  NewName ", "x@a.com")
		require.NoError(t, err)
		assert.Equal(t, "newname", got)
	})
	t.Run("a supplied taken username is refused, ignoring case", func(t *testing.T) {
		_, err := svc.Resolve(ctx, "SAM", "x@a.com")
		assert.ErrorIs(t, err, domain.ErrUsernameTaken)
	})
	t.Run("a supplied reserved or invalid username is refused", func(t *testing.T) {
		_, err := svc.Resolve(ctx, "admin", "x@a.com")
		assert.ErrorIs(t, err, domain.ErrUsernameReserved)
		_, err = svc.Resolve(ctx, "no way", "x@a.com")
		assert.ErrorIs(t, err, domain.ErrUsernameInvalid)
	})
	t.Run("none supplied: generated from the email and numbered when taken", func(t *testing.T) {
		got, err := svc.Resolve(ctx, "", "Sam@b.com")
		require.NoError(t, err)
		assert.Equal(t, "sam3", got, "sam and sam2 are taken")
	})
	t.Run("none supplied: reserved bases are numbered", func(t *testing.T) {
		got, err := svc.Resolve(ctx, "", "admin@b.com")
		require.NoError(t, err)
		assert.Equal(t, "admin2", got)
	})
	t.Run("generated names stay within 30 characters when numbered", func(t *testing.T) {
		long := strings.Repeat("a", 30)
		repo.add("long@a.com", long)
		got, err := svc.Resolve(ctx, "", long+"@b.com")
		require.NoError(t, err)
		assert.LessOrEqual(t, len(got), 30)
		assert.NotEqual(t, long, got)
		assert.NoError(t, domain.ValidateUsername(got))
	})
}

func TestUsernameService_Check(t *testing.T) {
	ctx := context.Background()
	repo := newUsernameRepo()
	repo.add("sam@a.com", "sam")
	svc := service.NewUsernameService(repo)

	cases := []struct {
		input, reason string
		available     bool
	}{
		{"fresh_name", "", true},
		{" Fresh_Name ", "", true},
		{"SAM", "taken", false},
		{"admin", "reserved", false},
		{"ab", "invalid", false},
		{"sam-smith", "invalid", false},
	}
	for _, c := range cases {
		got := svc.Check(ctx, c.input)
		assert.Equal(t, c.available, got.Available, c.input)
		assert.Equal(t, c.reason, got.Reason, c.input)
		if !c.available {
			assert.NotEmpty(t, got.Message, c.input)
		}
	}
}

func TestUsernameService_BackfillMissing_AssignsEveryAccountOnceAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newUsernameRepo()
	repo.add("sam-smith@x.com", "")
	repo.add("sam@a.com", "")
	repo.add("sam@b.com", "")
	repo.add("admin@x.com", "")
	repo.add("ab@x.com", "")
	keep := repo.add("keep@x.com", "kept_name")
	svc := service.NewUsernameService(repo)

	n, err := svc.CountMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	changed, err := svc.BackfillMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, changed)

	want := map[string]string{"sam-smith@x.com": "samsmith", "sam@a.com": "sam", "sam@b.com": "sam2", "admin@x.com": "admin2", "ab@x.com": "userab"}
	for email, name := range want {
		u, _ := repo.FindByEmail(ctx, email)
		assert.Equal(t, name, u.Username, email)
	}
	k, _ := repo.FindByID(ctx, keep.ID)
	assert.Equal(t, "kept_name", k.Username, "existing usernames are never touched")

	changed, err = svc.BackfillMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, changed, "a second run changes nothing")
}

func TestUsernameService_BackfillMissing_RetriesWhenAnotherWriterTakesTheName(t *testing.T) {
	ctx := context.Background()
	repo := newUsernameRepo()
	repo.add("sam@a.com", "")
	repo.failNextSetWithTaken = 1 // "sam" looked free but was taken just before the write
	svc := service.NewUsernameService(repo)

	changed, err := svc.BackfillMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, changed)
	u, _ := repo.FindByEmail(ctx, "sam@a.com")
	assert.Equal(t, "sam2", u.Username)
}
