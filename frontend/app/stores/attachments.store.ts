import { defineStore } from 'pinia'
import { attachmentsService } from '~/services/attachments.service'
import type { Attachment, AttachmentRejection } from '~/types/domain.types'

export const useAttachmentsStore = defineStore('attachments', () => {
  const attachments = ref<Attachment[]>([])
  const loading = ref(false)
  const uploading = ref(false)
  const error = ref<string | null>(null)
  const lastRejected = ref<AttachmentRejection[]>([])

  async function fetchAttachments(issueId: number) {
    loading.value = true
    error.value = null
    try {
      const resp = await attachmentsService.list(issueId)
      attachments.value = resp.attachments
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    } finally {
      loading.value = false
    }
  }

  async function uploadFiles(issueId: number, files: File[]) {
    uploading.value = true
    error.value = null
    lastRejected.value = []
    try {
      const result = await attachmentsService.upload(issueId, files)
      attachments.value = [...result.uploaded, ...attachments.value]
      lastRejected.value = result.rejected
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    } finally {
      uploading.value = false
    }
  }

  async function downloadAttachment(issueId: number, attachmentId: number) {
    try {
      await attachmentsService.download(issueId, attachmentId)
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    }
  }

  async function deleteAttachment(issueId: number, attachmentId: number) {
    await attachmentsService.delete(issueId, attachmentId)
    attachments.value = attachments.value.filter(a => a.id !== attachmentId)
  }

  return {
    attachments,
    loading,
    uploading,
    error,
    lastRejected,
    fetchAttachments,
    uploadFiles,
    downloadAttachment,
    deleteAttachment,
  }
})
