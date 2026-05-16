<template>
  <div class="max-w-4xl mx-auto p-6">
    <!-- Page header -->
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">Backlog</h1>
      <NuxtLink :to="`/projects/${key}/board`" class="btn btn-sm btn-ghost">
        ← Board
      </NuxtLink>
    </div>

    <!-- Error state -->
    <div v-if="error" class="alert alert-error mb-4">
      <span>Failed to load backlog: {{ error }}</span>
    </div>

    <!-- Backlog issue list -->
    <section>
      <div class="flex items-center gap-2 mb-3">
        <h2 class="text-lg font-semibold">Backlog</h2>
        <span class="badge badge-neutral">{{ issues.length }}</span>
      </div>
      <BacklogList :issues="issues" :project-key="key" :loading="loading" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { backlogService } from '~/services/backlog.service'
import type { Issue } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const { showError } = useToast()

const issues = ref<Issue[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    issues.value = await backlogService.getBacklog(key)
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'Unknown error'
    error.value = msg
    showError('Failed to load backlog')
  }
  finally {
    loading.value = false
  }
})
</script>
