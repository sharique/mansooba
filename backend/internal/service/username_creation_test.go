package service_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// raceRepo loses the race on insert: the username looked free when it was
// checked but is taken when the account is written.
type raceRepo struct{ *usernameRepo }

func (r *raceRepo) Create(_ context.Context, _ *domain.User) error { return domain.ErrUsernameTaken }

func newRegisterSvc(repo domain.UserRepository) service.AuthService {
	return service.NewAuthService(repo, newStubRevokedRepo(), stubSystemLogService{}, zap.NewNop(), testSecret, "15m", "168h")
}

func TestAuthService_Register_Usernames(t *testing.T) {
	ctx := context.Background()
	pw := "Passw0rd!x"

	t.Run("a supplied username is validated, lower-cased and stored", func(t *testing.T) {
		repo := newUsernameRepo()
		resp, err := newRegisterSvc(repo).Register(ctx, dto.RegisterRequest{FullName: "A", Email: "a@x.com", Password: pw, Username: "Alice_1"}, 0)
		require.NoError(t, err)
		assert.Equal(t, "alice_1", resp.User.Username)
		u, _ := repo.FindByEmail(ctx, "a@x.com")
		assert.Equal(t, "alice_1", u.Username)
	})
	t.Run("no username: one is generated from the email", func(t *testing.T) {
		repo := newUsernameRepo()
		repo.add("sam@other.com", "sam")
		resp, err := newRegisterSvc(repo).Register(ctx, dto.RegisterRequest{FullName: "S", Email: "sam-smith@x.com", Password: pw}, 0)
		require.NoError(t, err)
		assert.Equal(t, "samsmith", resp.User.Username)
		resp, err = newRegisterSvc(repo).Register(ctx, dto.RegisterRequest{FullName: "S", Email: "sam@x.com", Password: pw}, 0)
		require.NoError(t, err)
		assert.Equal(t, "sam2", resp.User.Username)
	})
	t.Run("a taken, reserved or invalid username creates nothing", func(t *testing.T) {
		repo := newUsernameRepo()
		repo.add("sam@other.com", "sam")
		svc := newRegisterSvc(repo)
		for username, want := range map[string]error{"SAM": domain.ErrUsernameTaken, "admin": domain.ErrUsernameReserved, "no way": domain.ErrUsernameInvalid} {
			_, err := svc.Register(ctx, dto.RegisterRequest{FullName: "N", Email: "n@x.com", Password: pw, Username: username}, 0)
			assert.ErrorIs(t, err, want, username)
			_, ferr := repo.FindByEmail(ctx, "n@x.com")
			assert.ErrorIs(t, ferr, domain.ErrNotFound, "no account may be created for %q", username)
		}
	})
	t.Run("losing the race at insert is reported as taken", func(t *testing.T) {
		repo := &raceRepo{newUsernameRepo()}
		_, err := newRegisterSvc(repo).Register(ctx, dto.RegisterRequest{FullName: "N", Email: "n@x.com", Password: pw, Username: "freshname"}, 0)
		assert.ErrorIs(t, err, domain.ErrUsernameTaken)
		_, ferr := repo.FindByEmail(ctx, "n@x.com")
		assert.ErrorIs(t, ferr, domain.ErrNotFound)
	})
}

func TestSetupService_Usernames(t *testing.T) {
	ctx := context.Background()
	pw := "Passw0rd!x"

	t.Run("first admin: supplied username used", func(t *testing.T) {
		repo := newUsernameRepo()
		resp, err := newSetupSvc(repo, &stubProjectService{}).CreateAdmin(ctx, dto.SetupAdminRequest{FullName: "Root", Email: "root@x.com", Password: pw, Username: "RootUser"})
		require.NoError(t, err)
		assert.Equal(t, "rootuser", resp.User.Username)
	})
	t.Run("first admin: generated, and admin@ becomes admin2 because admin is reserved", func(t *testing.T) {
		repo := newUsernameRepo()
		resp, err := newSetupSvc(repo, &stubProjectService{}).CreateAdmin(ctx, dto.SetupAdminRequest{FullName: "Root", Email: "admin@x.com", Password: pw})
		require.NoError(t, err)
		assert.Equal(t, "admin2", resp.User.Username)
	})
	t.Run("first admin: reserved username refused, nothing created", func(t *testing.T) {
		repo := newUsernameRepo()
		_, err := newSetupSvc(repo, &stubProjectService{}).CreateAdmin(ctx, dto.SetupAdminRequest{FullName: "Root", Email: "root@x.com", Password: pw, Username: "admin"})
		assert.ErrorIs(t, err, domain.ErrUsernameReserved)
		has, _ := repo.HasAdmin(ctx)
		assert.False(t, has)
	})
	t.Run("team member: taken refused, generated numbered", func(t *testing.T) {
		repo := newUsernameRepo()
		repo.add("sam@a.com", "sam")
		svc := newSetupSvc(repo, &stubProjectService{})
		_, err := svc.CreateUser(ctx, dto.SetupUserRequest{FullName: "S", Email: "s2@x.com", Password: pw, Username: "sam"})
		assert.ErrorIs(t, err, domain.ErrUsernameTaken)
		resp, err := svc.CreateUser(ctx, dto.SetupUserRequest{FullName: "S", Email: "sam@x.com", Password: pw})
		require.NoError(t, err)
		assert.Equal(t, "sam2", resp.Username)
	})
}
