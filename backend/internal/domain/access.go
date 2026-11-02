package domain

import (
	"sort"
	"sync"
)

// This file is the single definition of what each project role and the system
// administrator may do. Every allow-or-refuse decision in the product reads it
// (through service.Access); nothing compares a role string or the admin flag
// anywhere else. Standard library only, so it can sit in the domain layer.

// Project roles. A project member has exactly one.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// Roles lists the three project roles, most privileged first.
func Roles() []string { return []string{RoleAdmin, RoleMember, RoleViewer} }

// IsValidRole reports whether role is one of the three project roles.
func IsValidRole(role string) bool {
	return role == RoleAdmin || role == RoleMember || role == RoleViewer
}

// Capability names one thing a project role may be allowed to do.
type Capability string

// Project capabilities. "Own" rules (deleting your own task or comment) are an
// authorship test on top of the matching *Write capability, not separate
// capabilities.
const (
	CapProjectView         Capability = "project.view"
	CapTaskWrite           Capability = "task.write"
	CapTaskDeleteAny       Capability = "task.delete_any"
	CapCommentWrite        Capability = "comment.write"
	CapCommentDeleteAny    Capability = "comment.delete_any"
	CapLabelWrite          Capability = "label.write"
	CapLabelDelete         Capability = "label.delete"
	CapAttachmentWrite     Capability = "attachment.write"
	CapAttachmentDeleteAny Capability = "attachment.delete_any"
	CapSprintManage        Capability = "sprint.manage"
	CapProjectEdit         Capability = "project.edit"
	CapMemberManage        Capability = "member.manage"
	CapProjectDelete       Capability = "project.delete"
)

// AccessTable maps each project role to the capabilities it holds.
type AccessTable struct {
	roles map[string]map[Capability]bool
}

// DefaultAccessTable returns a fresh copy of the shipped rules. Tests change a
// copy and install it with UseAccessTable.
func DefaultAccessTable() *AccessTable {
	grant := func(caps ...Capability) map[Capability]bool {
		m := make(map[Capability]bool, len(caps))
		for _, c := range caps {
			m[c] = true
		}
		return m
	}
	member := []Capability{
		CapProjectView, CapTaskWrite, CapCommentWrite, CapLabelWrite, CapAttachmentWrite,
	}
	admin := append(append([]Capability{}, member...),
		CapTaskDeleteAny, CapCommentDeleteAny, CapLabelDelete, CapAttachmentDeleteAny,
		CapSprintManage, CapProjectEdit, CapMemberManage, CapProjectDelete,
	)
	return &AccessTable{roles: map[string]map[Capability]bool{
		RoleViewer: grant(CapProjectView),
		RoleMember: grant(member...),
		RoleAdmin:  grant(admin...),
	}}
}

// Can reports whether role holds c. An unknown role holds nothing.
func (t *AccessTable) Can(role string, c Capability) bool {
	return t.roles[role][c]
}

// Set grants or removes one capability for one role (used by tests).
func (t *AccessTable) Set(role string, c Capability, allowed bool) {
	if t.roles[role] == nil {
		t.roles[role] = map[Capability]bool{}
	}
	t.roles[role][c] = allowed
}

// capabilities returns the role's capabilities, sorted by name.
func (t *AccessTable) capabilities(role string) []Capability {
	var out []Capability
	for c, ok := range t.roles[role] {
		if ok {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

var (
	activeMu    sync.RWMutex
	activeTable = DefaultAccessTable()
)

// UseAccessTable installs t as the table in force and returns a function that
// puts the previous one back. It exists for tests that change one rule and
// check that every path follows; production code never calls it.
func UseAccessTable(t *AccessTable) (restore func()) {
	activeMu.Lock()
	prev := activeTable
	activeTable = t
	activeMu.Unlock()
	return func() {
		activeMu.Lock()
		activeTable = prev
		activeMu.Unlock()
	}
}

// RoleCan reports whether a project role holds a capability.
func RoleCan(role string, c Capability) bool {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return activeTable.Can(role, c)
}

// CapabilitiesOf lists a role's capabilities, sorted by name. An unknown role
// has none.
func CapabilitiesOf(role string) []Capability {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return activeTable.capabilities(role)
}

// SystemCapability names a thing only a system administrator may do. The
// system administrator is a per-account setting (users.is_admin), independent
// of every project role, and grants nothing inside a project.
type SystemCapability string

const (
	SysUsers         SystemCapability = "system.users"
	SysSettings      SystemCapability = "system.settings"
	SysLogs          SystemCapability = "system.logs"
	SysRescueProject SystemCapability = "system.rescue_project"
)

// SystemAdminCan reports whether an account holds a system capability: it has
// to be a system administrator and an active account.
func SystemAdminCan(isAdmin, isActive bool, c SystemCapability) bool {
	if !isAdmin || !isActive {
		return false
	}
	switch c {
	case SysUsers, SysSettings, SysLogs, SysRescueProject:
		return true
	}
	return false
}
