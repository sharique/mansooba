import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAttachmentsStore } from '~/stores/attachments.store'
import * as attachmentsService from '~/services/attachments.service'

vi.mock('~/services/attachments.service')

const attachment = {
  id: 1,
  issue_id: 5,
  filename: 'a.png',
  content_type: 'image/png',
  size_bytes: 1024,
  uploader_id: 1,
  uploader_name: 'Alice',
  created_at: '2026-07-10T00:00:00Z',
}

describe('useAttachmentsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAttachments populates the list', async () => {
    vi.mocked(attachmentsService.attachmentsService.list).mockResolvedValue({ attachments: [attachment] })
    const store = useAttachmentsStore()
    await store.fetchAttachments(5)
    expect(store.attachments).toHaveLength(1)
    expect(store.attachments[0]?.filename).toBe('a.png')
  })

  it('uploadFiles prepends uploaded files and records rejections', async () => {
    vi.mocked(attachmentsService.attachmentsService.upload).mockResolvedValue({
      uploaded: [attachment],
      rejected: [{ filename: 'bad.exe', reason: 'content type not accepted' }],
    })
    const store = useAttachmentsStore()
    const file = new File(['x'], 'a.png', { type: 'image/png' })
    await store.uploadFiles(5, [file])

    expect(store.attachments).toHaveLength(1)
    expect(store.lastRejected).toHaveLength(1)
    expect(store.lastRejected[0]?.filename).toBe('bad.exe')
  })

  it('uploadFiles surfaces a service error without throwing', async () => {
    vi.mocked(attachmentsService.attachmentsService.upload).mockRejectedValue(
      { data: { message: 'attachment limit reached for this issue' } },
    )
    const store = useAttachmentsStore()
    await store.uploadFiles(5, [new File(['x'], 'a.png')])

    expect(store.error).toBe('attachment limit reached for this issue')
    expect(store.attachments).toHaveLength(0)
  })

  it('deleteAttachment removes the attachment from the list', async () => {
    vi.mocked(attachmentsService.attachmentsService.list).mockResolvedValue({ attachments: [attachment] })
    vi.mocked(attachmentsService.attachmentsService.delete).mockResolvedValue(undefined)
    const store = useAttachmentsStore()
    await store.fetchAttachments(5)
    await store.deleteAttachment(5, 1)
    expect(store.attachments).toHaveLength(0)
  })

  it('downloadAttachment delegates to the service and surfaces errors', async () => {
    vi.mocked(attachmentsService.attachmentsService.download).mockRejectedValue(
      { data: { message: 'forbidden' } },
    )
    const store = useAttachmentsStore()
    await store.downloadAttachment(5, 1)
    expect(store.error).toBe('forbidden')
  })
})
