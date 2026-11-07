package database

import (
	"fmt"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// roleRank orders roles for the duplicate clean-up: the highest is kept.
var roleRank = map[string]int{domain.RoleAdmin: 3, domain.RoleMember: 2, domain.RoleViewer: 1}

// dedupeProjectMembers removes repeated (project, user) rows from
// project_members, keeping the row with the highest role (then the oldest), so
// the unique index that enforces one membership per person per project can be
// created. It does nothing on a fresh database, where the table does not exist
// yet, and is safe to run repeatedly. It must run before AutoMigrate.
func dedupeProjectMembers(db *gorm.DB) (removed int, err error) {
	if !db.Migrator().HasTable("project_members") {
		return 0, nil
	}
	type row struct {
		ID        uint
		ProjectID uint
		UserID    uint
		Role      string
	}
	var rows []row
	if err := db.Raw(`SELECT id, project_id, user_id, role FROM project_members ORDER BY project_id, user_id, id`).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("read project_members: %w", err)
	}
	type key struct{ p, u uint }
	best := map[key]row{}
	var losers []uint
	for _, r := range rows {
		k := key{r.ProjectID, r.UserID}
		cur, seen := best[k]
		switch {
		case !seen:
			best[k] = r
		case roleRank[r.Role] > roleRank[cur.Role]:
			losers = append(losers, cur.ID)
			best[k] = r
		default:
			losers = append(losers, r.ID)
		}
	}
	if len(losers) == 0 {
		return 0, nil
	}
	if err := db.Exec(`DELETE FROM project_members WHERE id IN ?`, losers).Error; err != nil {
		return 0, fmt.Errorf("remove duplicate memberships: %w", err)
	}
	return len(losers), nil
}

// ownerStepMarker records that the one-time owner-to-admin upgrade has run. The
// step must not run again: after it, an admin may demote or remove a project's
// creator, and a restart must not undo that.
const ownerStepMarker = "project_roles_owner_step_done"

// runOwnerStepOnce runs ensureOwnerAdmins the first time only. A fresh database
// has no projects, so the marker is simply recorded.
func runOwnerStepOnce(db *gorm.DB) (created, raised int, err error) {
	var done int64
	if err := db.Model(&domain.GlobalSetting{}).Where("setting_key = ?", ownerStepMarker).Count(&done).Error; err != nil {
		return 0, 0, fmt.Errorf("read upgrade marker: %w", err)
	}
	if done > 0 {
		return 0, 0, nil
	}
	created, raised, err = ensureOwnerAdmins(db)
	if err != nil {
		return created, raised, err
	}
	if err := db.Create(&domain.GlobalSetting{SettingKey: ownerStepMarker, SettingValue: "1"}).Error; err != nil {
		return created, raised, fmt.Errorf("record upgrade marker: %w", err)
	}
	return created, raised, nil
}

// ensureOwnerAdmins makes every project's recorded owner an admin member: a
// missing membership is created and a lower role is raised. An owner whose
// account no longer exists is skipped (a system administrator can rescue such a
// project). Existing admins and other members are untouched, so it is safe to
// run again. It must run after AutoMigrate, and only through runOwnerStepOnce.
func ensureOwnerAdmins(db *gorm.DB) (created, raised int, err error) {
	var projects []domain.Project
	if err := db.Select("id", "owner_id").Find(&projects).Error; err != nil {
		return 0, 0, fmt.Errorf("read projects: %w", err)
	}
	for _, p := range projects {
		var users int64
		if err := db.Model(&domain.User{}).Where("id = ?", p.OwnerID).Count(&users).Error; err != nil {
			return created, raised, err
		}
		if users == 0 {
			continue
		}
		var m domain.ProjectMember
		res := db.Where("project_id = ? AND user_id = ?", p.ID, p.OwnerID).Limit(1).Find(&m)
		if res.Error != nil {
			return created, raised, res.Error
		}
		switch {
		case res.RowsAffected == 0:
			if err := db.Create(&domain.ProjectMember{ProjectID: p.ID, UserID: p.OwnerID, Role: domain.RoleAdmin}).Error; err != nil {
				return created, raised, fmt.Errorf("create owner membership: %w", err)
			}
			created++
		case m.Role != domain.RoleAdmin:
			if err := db.Model(&domain.ProjectMember{}).Where("id = ?", m.ID).Update("role", domain.RoleAdmin).Error; err != nil {
				return created, raised, fmt.Errorf("raise owner to admin: %w", err)
			}
			raised++
		}
	}
	return created, raised, nil
}

func logRolesStep(removed, created, raised int) {
	if logger.Logger == nil || (removed == 0 && created == 0 && raised == 0) {
		return
	}
	logger.Logger.Info("project roles migration",
		zap.Int("duplicate_memberships_removed", removed),
		zap.Int("owner_memberships_created", created),
		zap.Int("owners_raised_to_admin", raised))
}
