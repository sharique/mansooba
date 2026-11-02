import { defineStore } from 'pinia'
import { adminService } from '~/services/admin.service'
import { serverMessage, toApiMessageError } from '~/utils/apiError'
import type { ProjectWithoutAdmin } from '~/types/domain.types'

/** Projects with no active admin, for system administrators. */
export const useAdminProjectsStore = defineStore('admin-projects', () => {
  const projects = ref<ProjectWithoutAdmin[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetch() {
    loading.value = true
    error.value = null
    try {
      projects.value = await adminService.listProjectsWithoutAdmin()
    }
    catch (e) {
      error.value = serverMessage(e, 'Failed to load projects')
    }
    finally {
      loading.value = false
    }
  }

  /** Takes the admin role in a project; throws an Error with the server's reason when refused. */
  async function claim(key: string) {
    try {
      await adminService.claimProjectAdmin(key)
    }
    catch (e) {
      throw toApiMessageError(e, 'Failed to take admin')
    }
    projects.value = projects.value.filter(p => p.key !== key)
  }

  return { projects, loading, error, fetch, claim }
})
