import { defineStore } from 'pinia'
import { commentsService } from '~/services/comments.service'
import { activityService } from '~/services/activity.service'
import type { Comment, ActivityEvent } from '~/types/domain.types'

export const useCommentsStore = defineStore('comments', () => {
  const comments = ref<Comment[]>([])
  const activity = ref<ActivityEvent[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchComments(issueId: number) {
    loading.value = true
    error.value = null
    try {
      comments.value = await commentsService.list(issueId)
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchActivity(issueId: number) {
    try {
      activity.value = await activityService.listByIssue(issueId)
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    }
  }

  async function addComment(issueId: number, body: string) {
    const comment = await commentsService.create(issueId, body)
    comments.value.push(comment)
  }

  async function updateComment(issueId: number, commentId: number, body: string) {
    const updated = await commentsService.update(issueId, commentId, body)
    const idx = comments.value.findIndex(c => c.id === commentId)
    if (idx !== -1) comments.value[idx] = updated
  }

  async function deleteComment(issueId: number, commentId: number) {
    await commentsService.delete(issueId, commentId)
    comments.value = comments.value.filter(c => c.id !== commentId)
  }

  return { comments, activity, loading, error, fetchComments, fetchActivity, addComment, updateComment, deleteComment }
})
