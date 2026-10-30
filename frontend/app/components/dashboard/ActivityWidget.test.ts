// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ActivityWidget from './ActivityWidget.vue'
import { useAuthStore } from '~/stores/auth.store'
import type { ActivityEvent } from '~/types/domain.types'

vi.stubGlobal('useTimeFormatter', () => ({ formatDateTime: (iso: string) => `at ${iso}` }))

const event = (overrides: Partial<ActivityEvent>): ActivityEvent => ({
  id: 1, issue_id: 5, actor_id: 3, actor_name: 'Carol', actor_avatar_url: null,
  issue_key: 'P-5', issue_title: 'Some task', kind: 'status_changed', old_value: 'todo', new_value: 'done',
  created_at: '2026-10-29T10:00:00Z', ...overrides,
})

const mountWith = (events: ActivityEvent[]) => {
  setActivePinia(createPinia())
  useAuthStore().myActivity = events
  return mount(ActivityWidget, { props: { loading: false } })
}

describe('ActivityWidget (my activity)', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('says a task was deleted, naming it, even though the task no longer exists', () => {
    const w = mountWith([event({ id: 2, kind: 'issue_deleted', issue_key: 'P-77', issue_title: 'Gone task', old_value: undefined, new_value: undefined })])
    const text = w.text()
    expect(text).toContain('[P-77]')
    expect(text).toContain('Carol deleted')
    expect(text).toContain('Gone task')
    expect(text).not.toContain('updated issue')
  })

  it('still describes the other kinds as before', () => {
    const w = mountWith([event({})])
    expect(w.text()).toContain('changed status from "todo" to "done"')
  })
})
