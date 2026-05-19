import type { Comment } from '~/types/domain.types'

export const commentsService = {
  list(issueId: number): Promise<Comment[]> {
    const { $api } = useNuxtApp()
    return $api<Comment[]>(`/issues/${issueId}/comments`)
  },

  create(issueId: number, body: string): Promise<Comment> {
    const { $api } = useNuxtApp()
    return $api<Comment>(`/issues/${issueId}/comments`, { method: 'POST', body: { body } })
  },

  update(issueId: number, commentId: number, body: string): Promise<Comment> {
    const { $api } = useNuxtApp()
    return $api<Comment>(`/issues/${issueId}/comments/${commentId}`, { method: 'PUT', body: { body } })
  },

  delete(issueId: number, commentId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/issues/${issueId}/comments/${commentId}`, { method: 'DELETE' })
  },
}
