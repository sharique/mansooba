import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useLabelsStore } from '~/stores/labels.store'
import * as labelsService from '~/services/labels.service'

vi.mock('~/services/labels.service')

describe('useLabelsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchProjectLabels populates labels', async () => {
    vi.mocked(labelsService.labelsService.list).mockResolvedValue([
      { id: 1, project_id: 1, name: 'bug', color: '#e11d48', created_at: '' },
    ])
    const store = useLabelsStore()
    await store.fetchProjectLabels('PROJ')
    expect(store.projectLabels).toHaveLength(1)
    expect(store.projectLabels[0]?.name).toBe('bug')
  })

  it('createLabel appends to list', async () => {
    vi.mocked(labelsService.labelsService.list).mockResolvedValue([])
    vi.mocked(labelsService.labelsService.create).mockResolvedValue(
      { id: 2, project_id: 1, name: 'feature', color: '#3b82f6', created_at: '' }
    )
    const store = useLabelsStore()
    await store.fetchProjectLabels('PROJ')
    await store.createLabel('PROJ', 'feature', '#3b82f6')
    expect(store.projectLabels).toHaveLength(1)
  })

  it('deleteLabel removes from list', async () => {
    vi.mocked(labelsService.labelsService.list).mockResolvedValue([
      { id: 1, project_id: 1, name: 'bug', color: '#e11d48', created_at: '' },
    ])
    vi.mocked(labelsService.labelsService.delete).mockResolvedValue(undefined)
    const store = useLabelsStore()
    await store.fetchProjectLabels('PROJ')
    await store.deleteLabel('PROJ', 1)
    expect(store.projectLabels).toHaveLength(0)
  })

  it('issueLabels tracks labels attached to an issue', async () => {
    vi.mocked(labelsService.labelsService.attach).mockResolvedValue(undefined)
    const store = useLabelsStore()
    store.issueLabels[5] = []
    await store.attachLabel(5, { id: 3, project_id: 1, name: 'urgent', color: '#e11d48', created_at: '' })
    expect(store.issueLabels[5]).toHaveLength(1)
  })
})
