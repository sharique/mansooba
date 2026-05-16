import { defineStore } from 'pinia'
import {
  sprintsService,
  type CreateSprintPayload,
  type UpdateSprintPayload,
  type CompleteSprintPayload,
} from '~/services/sprints.service'
import type { Sprint, BurndownData } from '~/types/domain.types'

export const useSprintsStore = defineStore('sprints', () => {
  const sprints = ref<Sprint[]>([])
  const burndownData = ref<BurndownData | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const activeSprint = computed(() =>
    sprints.value.find(s => s.status === 'Active') ?? null
  )

  const openSprints = computed(() =>
    sprints.value.filter(s => s.status !== 'Completed')
  )

  function replaceInList(updated: Sprint): Sprint {
    const idx = sprints.value.findIndex(s => s.id === updated.id)
    if (idx !== -1) sprints.value[idx] = updated
    return updated
  }

  function catchError(e: any): never {
    error.value = e.data?.message ?? e.message
    throw e
  }

  async function fetchSprints(projectKey: string) {
    loading.value = true
    error.value = null
    try {
      sprints.value = await sprintsService.list(projectKey)
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    } finally {
      loading.value = false
    }
  }

  async function getSprint(projectKey: string, id: string) {
    error.value = null
    try {
      return replaceInList(await sprintsService.get(projectKey, id))
    } catch (e: any) { return catchError(e) }
  }

  async function createSprint(projectKey: string, payload: CreateSprintPayload) {
    error.value = null
    try {
      const sprint = await sprintsService.create(projectKey, payload)
      sprints.value.push(sprint)
      return sprint
    } catch (e: any) { return catchError(e) }
  }

  async function updateSprint(projectKey: string, id: string, payload: UpdateSprintPayload) {
    error.value = null
    try {
      return replaceInList(await sprintsService.update(projectKey, id, payload))
    } catch (e: any) { return catchError(e) }
  }

  async function deleteSprint(projectKey: string, id: string) {
    error.value = null
    try {
      await sprintsService.delete(projectKey, id)
      sprints.value = sprints.value.filter(s => s.id !== id)
    } catch (e: any) { return catchError(e) }
  }

  async function startSprint(projectKey: string, id: string) {
    error.value = null
    try {
      return replaceInList(await sprintsService.start(projectKey, id))
    } catch (e: any) { return catchError(e) }
  }

  async function completeSprint(projectKey: string, id: string, payload: CompleteSprintPayload) {
    error.value = null
    try {
      return replaceInList(await sprintsService.complete(projectKey, id, payload))
    } catch (e: any) { return catchError(e) }
  }

  async function fetchBurndown(projectKey: string, id: string) {
    error.value = null
    try {
      burndownData.value = await sprintsService.burndown(projectKey, id)
      return burndownData.value
    } catch (e: any) { return catchError(e) }
  }

  return {
    sprints,
    burndownData,
    activeSprint,
    openSprints,
    loading,
    error,
    fetchSprints,
    getSprint,
    createSprint,
    updateSprint,
    deleteSprint,
    startSprint,
    completeSprint,
    fetchBurndown,
  }
})
