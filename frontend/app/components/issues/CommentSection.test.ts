// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import CommentSection from './CommentSection.vue'
import MentionTextarea from './MentionTextarea.vue'
import { useCommentsStore } from '~/stores/comments.store'

const mem = (id: number, name: string, username: string) =>
  ({ user_id: id, name, email: `${username}@example.com`, username, role: 'member', is_active: true })
const projects = reactive({ members: [mem(1, 'Me', 'me'), mem(2, 'Sam Smith', 'sam')] })

vi.stubGlobal('useProjectsStore', () => projects)
vi.stubGlobal('ref', ref)
vi.stubGlobal('onMounted', (fn: () => void) => fn())

async function typeInto(w: ReturnType<typeof mountIt>, text: string) {
  const el = w.find('textarea').element as HTMLTextAreaElement
  el.value = text
  el.setSelectionRange(text.length, text.length)
  await w.find('textarea').trigger('input')
}

function mountIt() {
  return mount(CommentSection, {
    props: { issueId: 7, currentUserId: 1 },
    global: { components: { IssuesMentionTextarea: MentionTextarea, IssuesCommentItem: { template: '<div />' } } },
  })
}

describe('CommentSection mentions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const store = useCommentsStore()
    vi.spyOn(store, 'fetchComments').mockResolvedValue(undefined as never)
    vi.spyOn(store, 'addComment').mockResolvedValue(undefined as never)
    projects.members = [mem(1, 'Me', 'me'), mem(2, 'Sam Smith', 'sam')]
  })

  it('posts the exact text built from the list', async () => {
    const w = mountIt()
    await typeInto(w, '@sa')
    await w.findAll('[role="option"]')[0]!.trigger('mousedown')
    await w.find('textarea').setValue('@sam please look')
    await w.find('button').trigger('click')
    expect(useCommentsStore().addComment).toHaveBeenCalledWith(7, '@sam please look')
  })

  it('suggests only the current project members after the project changes', async () => {
    const w = mountIt()
    projects.members = [mem(1, 'Me', 'me'), mem(9, 'Newcomer', 'newcomer')]
    await typeInto(w, '@')
    expect(w.findAll('[role="option"]').map(o => o.text())).toEqual([expect.stringContaining('Newcomer')])
  })

  it('a non-member warning does not disable the Comment button', async () => {
    vi.useFakeTimers()
    const w = mountIt()
    await typeInto(w, 'hi @lee')
    vi.advanceTimersByTime(400)
    await w.vm.$nextTick()
    expect(w.find('[role="status"]').text()).toContain('@lee is not a member')
    expect(w.find('button').attributes('disabled')).toBeUndefined()
    vi.useRealTimers()
  })
})
