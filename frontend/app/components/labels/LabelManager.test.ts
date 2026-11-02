// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LabelManager from './LabelManager.vue'
import { stubPermissions, MEMBER, VIEWER, ADMIN } from '../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')

vi.mock('~/stores/labels.store', () => ({
  useLabelsStore: () => ({
    projectLabels: [{ id: 1, name: 'bug', color: '#f00' }],
    fetchProjectLabels: vi.fn(), createLabel: vi.fn(), deleteLabel: vi.fn(),
  }),
}))
vi.stubGlobal('onMounted', (fn: () => void) => fn())

const LabelBadge = { props: ['label'], template: '<span>{{ label.name }}</span>' }
const mountIt = () => mount(LabelManager, {
  props: { projectKey: 'P' },
  global: { components: { IssuesLabelBadge: LabelBadge } },
})
const hasAdd = (w: ReturnType<typeof mountIt>) => w.findAll('button').some(b => b.text() === 'Add label')
const hasDelete = (w: ReturnType<typeof mountIt>) => w.findAll('button').some(b => b.text() === 'Delete')

describe('LabelManager roles', () => {
  beforeEach(() => { perms.permissions = MEMBER; perms.myRole = 'member' })

  it('a viewer sees the labels, a reason, and no controls', () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const w = mountIt()
    expect(w.text()).toContain('bug')
    expect(w.find('[data-testid="label-write-denied"]').text()).toBe('denied:label.write')
    expect(hasAdd(w)).toBe(false)
    expect(hasDelete(w)).toBe(false)
  })

  it('a member can create labels but not delete them', () => {
    const w = mountIt()
    expect(hasAdd(w)).toBe(true)
    expect(hasDelete(w)).toBe(false)
  })

  it('an admin can create and delete labels', () => {
    perms.permissions = ADMIN
    perms.myRole = 'admin'
    const w = mountIt()
    expect(hasAdd(w)).toBe(true)
    expect(hasDelete(w)).toBe(true)
  })
})
