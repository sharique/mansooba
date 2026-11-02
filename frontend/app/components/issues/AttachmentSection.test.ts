// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AttachmentSection from './AttachmentSection.vue'
import { stubPermissions, MEMBER, VIEWER, ADMIN } from '../../../test-helpers/permissions'

const perms = stubPermissions(MEMBER, 'member')
const att = (id: number, uploader_id: number) => ({
  id, filename: `f${id}.pdf`, size_bytes: 10, uploader_id, uploader_name: 'X', created_at: '2026-10-01T00:00:00Z',
})

vi.mock('~/stores/attachments.store', () => ({
  useAttachmentsStore: () => ({
    attachments: [att(1, 1), att(2, 2)], uploading: false, error: null, lastRejected: [],
    fetchAttachments: vi.fn(), uploadFiles: vi.fn(), downloadAttachment: vi.fn(), deleteAttachment: vi.fn(),
  }),
}))
vi.stubGlobal('useToast', () => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.stubGlobal('useTimeFormatter', () => ({ formatDateTime: () => 'now' }))
vi.stubGlobal('onMounted', (fn: () => void) => fn())

const mountIt = () => mount(AttachmentSection, {
  props: { issueId: 7, currentUserId: 1 },
  global: { stubs: { Icon: true } },
})
const deletes = (w: ReturnType<typeof mountIt>) => w.findAll('button[aria-label^="Delete"]').map(b => b.attributes('aria-label'))
const attach = (w: ReturnType<typeof mountIt>) => w.findAll('button').some(b => b.text().includes('Attach files'))

describe('AttachmentSection roles', () => {
  beforeEach(() => { perms.permissions = MEMBER; perms.myRole = 'member' })

  it('a viewer can download but not attach or delete', () => {
    perms.permissions = VIEWER
    perms.myRole = 'viewer'
    const w = mountIt()
    expect(attach(w)).toBe(false)
    expect(deletes(w)).toEqual([])
    expect(w.findAll('button[aria-label^="Download"]')).toHaveLength(2)
  })

  it('a member can attach and delete only their own file', () => {
    const w = mountIt()
    expect(attach(w)).toBe(true)
    expect(deletes(w)).toEqual(['Delete f1.pdf'])
  })

  it('an admin can delete other people\'s files too', () => {
    perms.permissions = ADMIN
    perms.myRole = 'admin'
    expect(deletes(mountIt())).toEqual(['Delete f1.pdf', 'Delete f2.pdf'])
  })
})
