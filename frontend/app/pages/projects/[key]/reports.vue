<script setup lang="ts">
import type { VelocityDataPoint } from '~/types/domain.types'
import { reportsService } from '~/services/reports.service'

const route = useRoute()
const key = route.params.key as string

const velocity = ref<VelocityDataPoint[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
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
  <div>
    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li><NuxtLink :to="`/projects/${key}`">{{ key }}</NuxtLink></li>
        <li>Reports</li>
      </ul>
    </div>

    <div class="mb-6">
      <h1 class="text-2xl font-bold">Reports</h1>
      <p class="text-base-content/60 text-sm mt-1">
        Sprint velocity for <span class="font-mono font-semibold">{{ key }}</span>.
      </p>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="flex justify-center items-center h-48">
      <span class="loading loading-spinner loading-lg" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <!-- No data -->
    <div
      v-else-if="velocity.length === 0"
      class="flex flex-col items-center justify-center h-48 text-base-content/40 gap-2"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" d="M3 3v18h18M7 16l4-4 4 4 4-6" />
      </svg>
      <p class="text-sm">No completed sprints yet</p>
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
