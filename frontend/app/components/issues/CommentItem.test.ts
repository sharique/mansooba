// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import CommentItem from './CommentItem.vue'
import { useMarkdown } from '~/composables/useMarkdown'
import type { Comment } from '~/types/domain.types'

const projects = reactive({ members: [] as { user_id: number; name: string; email: string; username: string; role: string; is_active: boolean }[] })

vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('useMarkdown', useMarkdown)
vi.stubGlobal('useTimeFormatter', () => ({ formatDateTime: () => 'now' }))

const comment = (body: string) => ({
  id: 1, issue_id: 7, author_id: 2, author_name: 'Alex', author_avatar_url: null, body, created_at: '2026-10-14T00:00:00Z',
}) as unknown as Comment

describe('CommentItem mentions', () => {
  beforeEach(() => { projects.members = [] })

  it('renders a project member mention as their display name', () => {
    projects.members = [{ user_id: 5, name: 'Priya Nair', email: 'priya.nair@example.com', username: 'priya', role: 'member', is_active: true }]
    const w = shallowMount(CommentItem, { props: { comment: comment('hi @priya'), currentUserId: 1 } })
    expect(w.find('.mention').text()).toBe('@Priya Nair')
    expect(w.find('.mention').attributes('title')).toBe('@priya')
  })

  it('shows the raw handle when no members are loaded', () => {
    const w = shallowMount(CommentItem, { props: { comment: comment('hi @priya'), currentUserId: 1 } })
    expect(w.find('.mention').exists()).toBe(false)
    expect(w.text()).toContain('@priya')
  })
})
