// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DeleteProjectDialog from './DeleteProjectDialog.vue'
import { ApiMessageError } from '~/utils/apiError'

const deleteProject = vi.fn()
const navigate = vi.fn()
const toast = { showSuccess: vi.fn(), showError: vi.fn() }
vi.stubGlobal('useProjectsStore', () => ({ deleteProject }))
vi.stubGlobal('useToast', () => toast)
vi.stubGlobal('navigateTo', navigate)

const mountIt = () => mount(DeleteProjectDialog, { props: { projectKey: 'scr', projectName: 'Scratch Pad' } })
const submit = (w: ReturnType<typeof mountIt>) => w.find('button[data-testid="confirm-delete"]')

describe('DeleteProjectDialog', () => {
  beforeEach(() => {
    deleteProject.mockReset()
    navigate.mockReset()
    toast.showError.mockReset()
    toast.showSuccess.mockReset()
  })

  it('keeps Delete disabled until the exact project name is typed', async () => {
    const w = mountIt()
    expect(submit(w).attributes('disabled')).toBeDefined()
    await w.find('input').setValue('scratch pad')
    expect(submit(w).attributes('disabled')).toBeDefined()
    await w.find('input').setValue('Scratch Pad')
    expect(submit(w).attributes('disabled')).toBeUndefined()
  })

  it('labels the input for screen readers', () => {
    const w = mountIt()
    expect(w.find('input').attributes('aria-label')).toContain('project name')
  })

  it('deletes, confirms and goes to the projects list', async () => {
    deleteProject.mockResolvedValueOnce(undefined)
    const w = mountIt()
    await w.find('input').setValue('Scratch Pad')
    await submit(w).trigger('click')
    await flushPromises()
    expect(deleteProject).toHaveBeenCalledWith('scr', 'Scratch Pad')
    expect(toast.showSuccess).toHaveBeenCalled()
    expect(navigate).toHaveBeenCalledWith('/projects')
  })

  it('shows how many tasks remain when the server refuses, and stays on the page', async () => {
    deleteProject.mockRejectedValueOnce(new ApiMessageError('Project still has 3 tasks. Remove them first.', 409, 3))
    const w = mountIt()
    await w.find('input').setValue('Scratch Pad')
    await submit(w).trigger('click')
    await flushPromises()
    const msg = w.find('[data-testid="delete-error"]')
    expect(msg.text()).toBe('Project still has 3 tasks. Remove them first.')
    expect(w.find('[data-testid="task-count"]').text()).toContain('3')
    expect(navigate).not.toHaveBeenCalled()
  })

  it('shows another refusal (wrong name) as its message', async () => {
    deleteProject.mockRejectedValueOnce(new ApiMessageError('The name does not match', 400))
    const w = mountIt()
    await w.find('input').setValue('Scratch Pad')
    await submit(w).trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="delete-error"]').text()).toBe('The name does not match')
    expect(w.find('[data-testid="task-count"]').exists()).toBe(false)
  })
})
