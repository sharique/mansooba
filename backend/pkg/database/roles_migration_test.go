package database_test

import (
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// legacyMember is project_members as it was before one membership per person
// per project was enforced: plain indexes only.
type legacyMember struct {
	ID        uint   `gorm:"primaryKey"`
	ProjectID uint   `gorm:"not null;index"`
	UserID    uint   `gorm:"not null;index"`
	Role      string `gorm:"not null"`
	CreatedAt time.Time
}

func (legacyMember) TableName() string { return "project_members" }

type memberRow struct {
	ProjectID uint
	UserID    uint
	Role      string
}

func membersOf(t *testing.T, db *gorm.DB, projectID uint) map[uint]string {
	t.Helper()
	var rows []memberRow
	require.NoError(t, db.Table("project_members").Where("project_id = ?", projectID).Find(&rows).Error)
	out := map[uint]string{}
	for _, r := range rows {
		_, dup := out[r.UserID]
		require.False(t, dup, "user %d appears twice in project %d", r.UserID, projectID)
		out[r.UserID] = r.Role
	}
	return out
}

func legacyRolesFixture(t *testing.T) (db *gorm.DB, p map[string]uint, u map[string]uint) {
	t.Helper()
	db = openSQLite(t)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Project{}, &legacyMember{}))
	u = map[string]uint{}
	for _, n := range []string{"ada", "ben", "cy", "dee"} {
		usr := &domain.User{Name: n, Email: n + "@x.com", Username: n, Password: "h", IsActive: true}
		require.NoError(t, db.Create(usr).Error)
		u[n] = usr.ID
	}
	p = map[string]uint{}
	mkProject := func(key string, owner uint) {
		pr := &domain.Project{Key: key, Name: key, OwnerID: owner}
		require.NoError(t, db.Create(pr).Error)
		p[key] = pr.ID
	}
	mkProject("noadmin", u["ada"]) // owner has no membership at all
	mkProject("lowered", u["ben"]) // owner was lowered to member
	mkProject("viewer", u["cy"])   // owner was lowered to viewer
	mkProject("dupes", u["ada"])   // owner is admin; another user is listed twice
	mkProject("dupes2", u["dee"])  // owner listed twice, as member and viewer
	mkProject("empty", 99999)      // owner account no longer exists
	add := func(key string, user uint, role string) {
		require.NoError(t, db.Create(&legacyMember{ProjectID: p[key], UserID: user, Role: role}).Error)
	}
	add("lowered", u["ben"], "member")
	add("viewer", u["cy"], "viewer")
	add("dupes", u["ada"], "admin")
	add("dupes", u["dee"], "viewer")
	add("dupes", u["dee"], "admin")
	add("dupes2", u["dee"], "viewer")
	add("dupes2", u["dee"], "member")
	return db, p, u
}

func TestMigrate_OwnersBecomeAdmins(t *testing.T) {
	db, p, u := legacyRolesFixture(t)
	require.NoError(t, database.Migrate(db))

	assert.Equal(t, map[uint]string{u["ada"]: "admin"}, membersOf(t, db, p["noadmin"]), "missing owner membership is created")
	assert.Equal(t, map[uint]string{u["ben"]: "admin"}, membersOf(t, db, p["lowered"]), "member owner is raised")
	assert.Equal(t, map[uint]string{u["cy"]: "admin"}, membersOf(t, db, p["viewer"]), "viewer owner is raised")
	assert.Empty(t, membersOf(t, db, p["empty"]), "an owner who no longer exists gets no membership")
}

func TestMigrate_DuplicateMembershipsCollapseToTheHighestRole(t *testing.T) {
	db, p, u := legacyRolesFixture(t)
	require.NoError(t, database.Migrate(db))

	assert.Equal(t, map[uint]string{u["ada"]: "admin", u["dee"]: "admin"}, membersOf(t, db, p["dupes"]))
	// owner of dupes2 (dee) is raised to admin even though the best listed row was member
	assert.Equal(t, map[uint]string{u["dee"]: "admin"}, membersOf(t, db, p["dupes2"]))
}

func TestMigrate_ExistingAdminsKeepTheirRole(t *testing.T) {
	db, p, u := legacyRolesFixture(t)
	require.NoError(t, db.Create(&legacyMember{ProjectID: p["lowered"], UserID: u["ada"], Role: "admin"}).Error)
	require.NoError(t, db.Create(&legacyMember{ProjectID: p["lowered"], UserID: u["cy"], Role: "viewer"}).Error)
	require.NoError(t, database.Migrate(db))

	got := membersOf(t, db, p["lowered"])
	assert.Equal(t, "admin", got[u["ada"]])
	assert.Equal(t, "viewer", got[u["cy"]], "a viewer who is not the owner stays a viewer")
	assert.Equal(t, "admin", got[u["ben"]])
}

func TestMigrate_RolesStepIsIdempotent(t *testing.T) {
	db, _, _ := legacyRolesFixture(t)
	require.NoError(t, database.Migrate(db))
	var before []memberRow
	require.NoError(t, db.Table("project_members").Order("project_id, user_id").Find(&before).Error)

	require.NoError(t, database.Migrate(db))
	var after []memberRow
	require.NoError(t, db.Table("project_members").Order("project_id, user_id").Find(&after).Error)
	assert.Equal(t, before, after)
}

func TestMigrate_OneMembershipPerPersonPerProjectIsEnforced(t *testing.T) {
	db, p, u := legacyRolesFixture(t)
	require.NoError(t, database.Migrate(db))
	err := db.Exec(`INSERT INTO project_members (project_id, user_id, role) VALUES (?, ?, 'viewer')`, p["lowered"], u["ben"]).Error
	assert.Error(t, err, "a second membership for the same person is refused by the database")
}

// A fresh install has no project_members table when the dedupe step runs.
func TestMigrate_FreshDatabaseSucceedsAndHasTheUniqueIndex(t *testing.T) {
	db := openSQLite(t)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, db.Exec(`INSERT INTO project_members (project_id, user_id, role) VALUES (1, 1, 'admin')`).Error)
	assert.Error(t, db.Exec(`INSERT INTO project_members (project_id, user_id, role) VALUES (1, 1, 'member')`).Error)
}

// The owner step is a one-time upgrade. After it has run, an admin may demote or
// remove the creator, and a restart must not undo that (FR-002: no permanent
// right for the creator).
func TestMigrate_OwnerStepDoesNotUndoLaterDemotions(t *testing.T) {
	db, p, u := legacyRolesFixture(t)
	require.NoError(t, database.Migrate(db))
	require.Equal(t, "admin", membersOf(t, db, p["lowered"])[u["ben"]])

	// someone else becomes admin, then the creator is demoted / removed
	require.NoError(t, db.Create(&domain.ProjectMember{ProjectID: p["lowered"], UserID: u["cy"], Role: "admin"}).Error)
	require.NoError(t, db.Model(&domain.ProjectMember{}).Where("project_id = ? AND user_id = ?", p["lowered"], u["ben"]).Update("role", "viewer").Error)
	require.NoError(t, db.Where("project_id = ? AND user_id = ?", p["viewer"], u["cy"]).Delete(&domain.ProjectMember{}).Error)

	require.NoError(t, database.Migrate(db)) // a restart or a deploy

	assert.Equal(t, "viewer", membersOf(t, db, p["lowered"])[u["ben"]], "the demoted creator stays demoted")
	_, back := membersOf(t, db, p["viewer"])[u["cy"]]
	assert.False(t, back, "a removed creator stays removed")
}
