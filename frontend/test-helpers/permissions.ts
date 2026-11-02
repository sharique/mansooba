import { computed, reactive } from 'vue'
import { vi } from 'vitest'

// Test fixtures only: the app never holds a role table, it reads `permissions`
// from the server. These lists just give tests something to hand the components.
export const VIEWER = ['project.view']
export const MEMBER = ['project.view', 'task.write', 'comment.write', 'label.write', 'attachment.write']
export const ADMIN = [
  ...MEMBER, 'task.delete_any', 'comment.delete_any', 'label.delete', 'attachment.delete_any',
  'sprint.manage', 'project.edit', 'member.manage', 'project.delete',
]

/** Replaces the global `useProjectPermissions` with a fake driven by the returned state. */
export function stubPermissions(permissions: string[] = ADMIN, myRole = 'admin') {
  const state = reactive({ permissions, myRole })
  vi.stubGlobal('useProjectPermissions', () => ({
    myRole: computed(() => state.myRole),
    permissions: computed(() => state.permissions),
    can: (c: string) => state.permissions.includes(c),
    isReadOnly: computed(() => state.myRole === 'viewer' || state.permissions.length === 0),
    reason: (c: string) => (state.permissions.includes(c) ? undefined : `denied:${c}`),
  }))
  return state
}
