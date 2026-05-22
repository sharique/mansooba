<script setup lang="ts">
import type { VelocityDataPoint } from '~/types/domain.types'
import { reportsService } from '~/services/reports.service'

const projectsStore = useProjectsStore()

const selectedKey = ref<string>('')
const velocity = ref<VelocityDataPoint[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

// Fetch all projects on mount so the selector is populated.
onMounted(async () => {
  if (projectsStore.projects.length === 0) {
    try {
      await projectsStore.fetchAll()
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load projects'
    }
  }
})

// Reload velocity data whenever the selected project changes.
watch(selectedKey, async (key) => {
  if (!key) {
    velocity.value = []
    return
  }
  loading.value = true
  error.value = null
  try {
    velocity.value = await reportsService.getVelocity(key)
  }
  catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load velocity data'
    velocity.value = []
  }
  finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="max-w-4xl mx-auto p-6">
    <!-- Page header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold">Reports</h1>
      <p class="text-base-content/60 text-sm mt-1">
        Track team velocity across completed sprints.
      </p>
    </div>

    <!-- Project selector -->
    <div class="form-control w-full max-w-xs mb-8">
      <label class="label" for="project-select">
        <span class="label-text font-medium">Project</span>
      </label>
      <select
        id="project-select"
        v-model="selectedKey"
        class="select select-bordered"
      >
        <option value="">— Select a project —</option>
        <option
          v-for="project in projectsStore.projects"
          :key="project.key"
          :value="project.key"
        >
          {{ project.name }} ({{ project.key }})
        </option>
      </select>
    </div>

    <!-- No project selected hint -->
    <div
      v-if="!selectedKey"
      class="flex flex-col items-center justify-center h-48 text-base-content/40 gap-2"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" d="M3 3v18h18M7 16l4-4 4 4 4-6" />
      </svg>
      <p class="text-sm">Select a project to view velocity</p>
    </div>

    <!-- Loading state -->
    <div v-else-if="loading" class="flex justify-center items-center h-48">
      <span class="loading loading-spinner loading-lg" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <!-- Velocity chart -->
    <section v-else>
      <h2 class="text-lg font-semibold mb-4">Sprint Velocity</h2>
      <div class="bg-base-200 rounded-box p-6 pl-14">
        <ReportsVelocityChart :data="velocity" />
      </div>
    </section>
  </div>
</template>
