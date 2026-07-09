import type { Attachment, AttachmentUploadResult } from '~/types/domain.types'

export const attachmentsService = {
  list(issueId: number): Promise<{ attachments: Attachment[] }> {
    const { $api } = useNuxtApp()
    return $api<{ attachments: Attachment[] }>(`/issues/${issueId}/attachments`)
  },

  upload(issueId: number, files: File[]): Promise<AttachmentUploadResult> {
    const { $api } = useNuxtApp()
    const body = new FormData()
    for (const file of files) body.append('files', file)
    return $api<AttachmentUploadResult>(`/issues/${issueId}/attachments`, { method: 'POST', body })
  },

  // Fetches a short-lived presigned S3 URL (authenticated, same-origin JSON
  // call), then navigates to it directly. The URL itself needs no
  // Authorization header — a 302 redirect can't carry one, which is why this
  // endpoint returns JSON instead (research.md Decision 2's revision note).
  async download(issueId: number, attachmentId: number): Promise<void> {
    const { $api } = useNuxtApp()
    const { url } = await $api<{ url: string; filename: string }>(
      `/issues/${issueId}/attachments/${attachmentId}/download`,
    )
    window.location.href = url
  },

  delete(issueId: number, attachmentId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/issues/${issueId}/attachments/${attachmentId}`, { method: 'DELETE' })
  },
}
