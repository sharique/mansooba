<template>
  <div class="max-w-4xl mx-auto p-6">
    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li><NuxtLink :to="`/projects/${key}`">{{ key }}</NuxtLink></li>
        <li>Sprints</li>
      </ul>
    </div>

    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">Sprints</h1>
      <NuxtLink :to="`/projects/${key}/backlog`" class="btn btn-sm btn-ghost">
        ← Backlog
      </NuxtLink>
    </div>

    <div v-if="error" class="alert alert-error mb-4">
      <span>Failed to load sprints: {{ error }}</span>
    </div>

    <div v-else-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-md" />
    </div>

    <SprintsSprintList v-else :project-key="key" :can-manage="canManage" />
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const key = route.params.key as string

const projectsStore = useProjectsStore()
const sprintsStore = useSprintsStore()
const { canManage } = useCanManageProject()

const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    // fetchOne doubles as the access check: it fails for non-members.
    await Promise.all([
      projectsStore.fetchOne(key),
      projectsStore.fetchMembers(key),
      sprintsStore.fetchSprints(key),
    ])
  }
  catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Unknown error'
  }
  finally {
    loading.value = false
  }
})
</script>
