// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import SprintCard from './SprintCard.vue'
import type { Sprint, Issue } from '~/types/domain.types'

vi.stubGlobal('useTimeFormatter', () => ({
  formatDate: (iso: string | null | undefined) => iso ?? '—',
  formatDateTime: (iso: string) => iso,
}))

const base: Sprint = {
  id: 'sprint-1',
  project_id: 'p-1',
  name: 'Sprint 1',
  goal: 'Ship it',
  status: 'planning',
  start_date: '2026-05-01',
  end_date: '2026-05-14',
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

const activeSprint: Sprint = { ...base, id: 'sprint-active', status: 'active' }
const completedSprint: Sprint = { ...base, id: 'sprint-done', status: 'completed' }

describe('SprintCard', () => {
  it('renders name and status badge', () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: false, hasActiveSprint: false },
    })
    expect(w.text()).toContain('Sprint 1')
    expect(w.text()).toContain('planning')
  })

  it('shows Start button for planning sprint with no active sprint', () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: true, hasActiveSprint: false },
    })
    expect(w.find('button.btn-success').exists()).toBe(true)
  })

  it('hides Start button when hasActiveSprint is true', () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: true, hasActiveSprint: true },
    })
    expect(w.find('button.btn-success').exists()).toBe(false)
  })

  it('shows Complete button only for active sprint', () => {
    const w = mount(SprintCard, {
      props: { sprint: activeSprint, projectKey: 'P', canManage: true, hasActiveSprint: true },
    })
    expect(w.find('button.btn-warning').exists()).toBe(true)
  })

  it('hides Edit and Delete buttons for completed sprint', () => {
    const w = mount(SprintCard, {
      props: { sprint: completedSprint, projectKey: 'P', canManage: true, hasActiveSprint: false },
    })
    const buttonTexts = w.findAll('button').map(b => b.text())
    expect(buttonTexts).not.toContain('Edit')
    expect(w.find('button.btn-error').exists()).toBe(false)
  })

  it('hides action buttons when canManage is false', () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: false, hasActiveSprint: false },
    })
    expect(w.find('button.btn-success').exists()).toBe(false)
    expect(w.find('button.btn-error').exists()).toBe(false)
  })

  it('emits start when Start is clicked', async () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: true, hasActiveSprint: false },
    })
    await w.find('button.btn-success').trigger('click')
    expect(w.emitted('start')).toBeTruthy()
  })

  it('emits complete when Complete is clicked', async () => {
    const w = mount(SprintCard, {
      props: { sprint: activeSprint, projectKey: 'P', canManage: true, hasActiveSprint: true },
    })
    await w.find('button.btn-warning').trigger('click')
    expect(w.emitted('complete')).toBeTruthy()
  })

  it('emits edit when Edit is clicked', async () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: true, hasActiveSprint: false },
    })
    const editBtn = w.findAll('button').find(b => b.text() === 'Edit')
    await editBtn!.trigger('click')
    expect(w.emitted('edit')).toBeTruthy()
  })

  it('emits delete when Delete is clicked', async () => {
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: true, hasActiveSprint: false },
    })
    await w.find('button.btn-error').trigger('click')
    expect(w.emitted('delete')).toBeTruthy()
  })

  it('emits expand and renders issues inline after toggle', async () => {
    const issues: Issue[] = [
      {
        id: 1, key: 'P-1', project_id: 1, title: 'Task One', description: '',
        type: 'task', status: 'todo', priority: 'medium', reporter_id: 1, story_points: 3,
      },
    ]
    const w = mount(SprintCard, {
      props: { sprint: base, projectKey: 'P', canManage: false, hasActiveSprint: false, issues },
    })
    const expandBtn = w.findAll('button').find(b => b.text().includes('Show issues'))
    await expandBtn!.trigger('click')
    expect(w.emitted('expand')).toBeTruthy()
    expect(w.text()).toContain('Task One')
    expect(w.text()).toContain('P-1')
  })
})
