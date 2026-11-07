// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import RoleSummary from './RoleSummary.vue'
import { stubPermissions, VIEWER, MEMBER, ADMIN } from '../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')
const item = (w: ReturnType<typeof mount>, role: string) => w.find(`[data-role="${role}"]`)

describe('RoleSummary', () => {
  beforeEach(() => { perms.permissions = MEMBER; perms.myRole = 'member' })

  it('describes the three roles in one plain line each', () => {
    const w = mount(RoleSummary)
    expect(item(w, 'viewer').text()).toContain('Can read everything, change nothing')
    expect(item(w, 'member').text()).toContain('Can work on tasks, comments, labels and files')
    expect(item(w, 'admin').text()).toContain('Can also manage sprints, people and the project')
  })

  it.each([['viewer', VIEWER], ['member', MEMBER], ['admin', ADMIN]])('marks only the person\'s own role (%s)', (role, perm) => {
    perms.permissions = perm
    perms.myRole = role
    const w = mount(RoleSummary)
    const marked = w.findAll('[data-role]').filter(e => e.text().includes('Your role'))
    expect(marked).toHaveLength(1)
    expect(marked[0]!.attributes('data-role')).toBe(role)
  })

  it('marks no role when the role is unknown', () => {
    perms.permissions = []
    perms.myRole = ''
    expect(mount(RoleSummary).text()).not.toContain('Your role')
  })

  it('does not list capabilities, only the three one-line descriptions', () => {
    const text = mount(RoleSummary).text()
    expect(text).not.toMatch(/task\.write|sprint\.manage|member\.manage/)
  })
})
