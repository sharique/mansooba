import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { useProjectPermissions } from './useProjectPermissions'

const projects = reactive({
  current: null as null | { myRole?: string; permissions?: string[] },
})
vi.stubGlobal('useProjectsStore', () => projects)

describe('useProjectPermissions', () => {
  beforeEach(() => {
    projects.current = null
  })

  it('allows nothing before the project is loaded', () => {
    const p = useProjectPermissions()
    expect(p.can('task.write')).toBe(false)
    expect(p.isReadOnly.value).toBe(true)
    expect(p.myRole.value).toBeNull()
  })

  it('allows nothing when the server sent no permissions', () => {
    projects.current = { myRole: 'admin' }
    const p = useProjectPermissions()
    expect(p.can('project.view')).toBe(false)
    expect(p.isReadOnly.value).toBe(true)
  })

  it('viewer can only view and is read-only', () => {
    projects.current = { myRole: 'viewer', permissions: ['project.view'] }
    const p = useProjectPermissions()
    expect(p.can('project.view')).toBe(true)
    expect(p.can('task.write')).toBe(false)
    expect(p.isReadOnly.value).toBe(true)
    expect(p.myRole.value).toBe('viewer')
  })

  it('member can write tasks but not manage sprints', () => {
    projects.current = { myRole: 'member', permissions: ['project.view', 'task.write'] }
    const p = useProjectPermissions()
    expect(p.can('task.write')).toBe(true)
    expect(p.can('sprint.manage')).toBe(false)
    expect(p.isReadOnly.value).toBe(false)
  })

  it('admin has what the server lists', () => {
    projects.current = { myRole: 'admin', permissions: ['project.view', 'sprint.manage', 'member.manage'] }
    const p = useProjectPermissions()
    expect(p.can('sprint.manage')).toBe(true)
    expect(p.can('member.manage')).toBe(true)
  })

  it('reacts when the permissions are refreshed', () => {
    projects.current = { myRole: 'member', permissions: ['task.write'] }
    const p = useProjectPermissions()
    expect(p.can('task.write')).toBe(true)
    projects.current = { myRole: 'viewer', permissions: ['project.view'] }
    expect(p.can('task.write')).toBe(false)
    expect(p.isReadOnly.value).toBe(true)
  })

  it('gives a one-line reason for a denied capability and none for an allowed one', () => {
    projects.current = { myRole: 'viewer', permissions: ['project.view'] }
    const p = useProjectPermissions()
    expect(p.reason('task.write')).toBe("Viewers can't edit tasks")
    expect(p.reason('project.view')).toBeUndefined()
  })
})
