<template>
  <div class="space-y-8">
    <!-- Greeting -->
    <DashboardGreeting />

    <!-- My Tasks -->
    <DashboardMyTasksWidget :tasks="myTasks" :loading="loading" />

    <!-- Recent Activity -->
    <DashboardActivityWidget :loading="loading" />
  </div>
</template>

<script setup lang="ts">
import { useProjectsStore } from '~/stores/projects.store'
import { useAuthStore } from '~/stores/auth.store'
import { issuesService } from '~/services/issues.service'
import type { Issue } from '~/types/domain.types'

const projectsStore = useProjectsStore()
const authStore = useAuthStore()
const { showError } = useToast()

const loading = ref(true)
const myTasks = ref<Issue[]>([])

onMounted(async () => {
  try {
    await Promise.all([
      projectsStore.fetchAll(),
      authStore.fetchMyActivity(10, 0),
    ])

    // Fetch issues assigned to the current user across all projects in parallel
    const userId = authStore.user?.id
    if (userId && projectsStore.projects.length > 0) {
      const perProject = await Promise.all(
        projectsStore.projects.map(p =>
          issuesService.list(p.key, { assignee_id: userId }).catch(() => [] as Issue[]),
        ),
      )
      myTasks.value = perProject.flat()
    }
  }
  catch {
    showError('Failed to load dashboard')
  }
  finally {
    loading.value = false
  }
})
</script>
