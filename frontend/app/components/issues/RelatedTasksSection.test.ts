// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RelatedTasksSection from './RelatedTasksSection.vue'
import { stubPermissions, MEMBER, VIEWER } from '../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')
const rel = { id: 5, relation_type: 'blocks', related_issue: { key: 'P-2', title: 'Other', status: 'todo' } }

vi.mock('~/stores/issue-relations.store', () => ({
  useIssueRelationsStore: () => ({ relations: [rel], fetchForIssue: vi.fn(), create: vi.fn(), remove: vi.fn() }),
}))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.stubGlobal('onMounted', (fn: () => void) => fn())

const mountIt = () => mount(RelatedTasksSection, {
  props: { issueId: 7, projectKey: 'P' },
  global: { stubs: { Icon: true } },
})

describe('RelatedTasksSection roles', () => {
  beforeEach(() => { perms.permissions = MEMBER; perms.myRole = 'member' })

  it('a viewer sees the relations but cannot add or remove any', () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const w = mountIt()
    expect(w.text()).toContain('Other')
    expect(w.text()).not.toContain('Add relation')
    expect(w.find('button[aria-label^="Remove relation"]').exists()).toBe(false)
  })

  it('a member can add and remove relations', () => {
    const w = mountIt()
    expect(w.text()).toContain('Add relation')
    expect(w.find('button[aria-label^="Remove relation"]').exists()).toBe(true)
  })
})
