// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import IssueLabelPicker from './IssueLabelPicker.vue'
import { stubPermissions, MEMBER, VIEWER } from '../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')
const bug = { id: 1, name: 'bug', color: '#f00' }
const ui = { id: 2, name: 'ui', color: '#0f0' }
const detachLabel = vi.fn()

vi.mock('~/stores/labels.store', () => ({
  useLabelsStore: () => ({
    projectLabels: [bug, ui], issueLabels: { 7: [bug] },
    fetchProjectLabels: vi.fn(), fetchIssueLabels: vi.fn(), attachLabel: vi.fn(), detachLabel: (...a: unknown[]) => detachLabel(...a),
  }),
}))
vi.stubGlobal('onMounted', (fn: () => void) => fn())

const LabelBadge = { props: ['label'], template: '<span class="badge">{{ label.name }}</span>' }
const mountIt = () => mount(IssueLabelPicker, {
  props: { issueId: 7, projectKey: 'P' },
  global: { components: { IssuesLabelBadge: LabelBadge } },
})

describe('IssueLabelPicker roles', () => {
  beforeEach(() => { perms.permissions = MEMBER; perms.myRole = 'member'; detachLabel.mockReset() })

  it('a viewer sees the labels but cannot add or remove them', async () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const w = mountIt()
    expect(w.text()).toContain('bug')
    expect(w.text()).not.toContain('+ Label')
    await w.find('.badge').trigger('click')
    expect(detachLabel).not.toHaveBeenCalled()
  })

  it('a member can add and remove labels', async () => {
    const w = mountIt()
    expect(w.text()).toContain('+ Label')
    await w.find('.badge').trigger('click')
    expect(detachLabel).toHaveBeenCalledWith(7, 1)
  })
})
