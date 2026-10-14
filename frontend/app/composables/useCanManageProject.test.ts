import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { useCanManageProject } from './useCanManageProject'

const projects = reactive({ current: null as null | { ownerId: number }, members: [] as { user_id: number; role: string }[] })
const auth = reactive({ user: { id: 5 } as null | { id: number } })

vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useAuthStore', () => auth)

describe('useCanManageProject', () => {
  beforeEach(() => {
    projects.current = null
    projects.members = []
    auth.user = { id: 5 }
  })

  it('is false before the project and members are loaded', () => {
    expect(useCanManageProject().canManage.value).toBe(false)
  })

  it('is true for the project owner', () => {
    projects.current = { ownerId: 5 }
    expect(useCanManageProject().canManage.value).toBe(true)
  })

  it('is true for a member with role admin', () => {
    projects.current = { ownerId: 1 }
    projects.members = [{ user_id: 5, role: 'admin' }]
    expect(useCanManageProject().canManage.value).toBe(true)
  })

  it('is false for a regular member', () => {
    projects.current = { ownerId: 1 }
    projects.members = [{ user_id: 5, role: 'member' }]
    expect(useCanManageProject().canManage.value).toBe(false)
  })

  it('reacts when members load after the project', () => {
    projects.current = { ownerId: 1 }
    const { canManage } = useCanManageProject()
    expect(canManage.value).toBe(false)
    projects.members = [{ user_id: 5, role: 'admin' }]
    expect(canManage.value).toBe(true)
  })

  it('is false when nobody is signed in', () => {
    auth.user = null
    projects.current = { ownerId: 5 }
    expect(useCanManageProject().canManage.value).toBe(false)
  })
})
