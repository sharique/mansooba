<template>
  <div class="space-y-8">
    <!-- Greeting -->
    <DashboardGreeting />

    <!-- My Projects -->
    <section>
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-semibold">My Projects</h2>
        <NuxtLink to="/projects" class="text-sm text-primary hover:underline">View all →</NuxtLink>
      </div>

      <!-- Skeleton -->
      <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <div v-for="i in 3" :key="i" class="skeleton h-40 w-full rounded-xl" />
      </div>

      <!-- Empty state -->
      <div v-else-if="projectsStore.projects.length === 0" class="text-base-content/50 text-sm py-4">
        No projects yet.
        <NuxtLink to="/projects" class="text-primary hover:underline ml-1">Create one →</NuxtLink>
      </div>

      <!-- Grid (first 6 projects) -->
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <ProjectsProjectCard
          v-for="project in projectsStore.projects.slice(0, 6)"
          :key="project.id"
          :project="project"
        />
      </div>
    </section>

    <!-- Recent Activity -->
    <DashboardActivityWidget :loading="loading" />
  </div>
</template>

<script setup lang="ts">
import { useProjectsStore } from '~/stores/projects.store'
import { useAuthStore } from '~/stores/auth.store'

const projectsStore = useProjectsStore()
const authStore = useAuthStore()
const { showError } = useToast()

const loading = ref(true)

onMounted(async () => {
  try {
    await Promise.all([
      projectsStore.fetchAll(),
      authStore.fetchMyActivity(10, 0),
    ])
  }
  catch {
    showError('Failed to load dashboard')
  }
  finally {
    loading.value = false
  }
})
</script>
