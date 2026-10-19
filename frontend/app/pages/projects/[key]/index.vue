<template>
  <div class="max-w-6xl mx-auto p-6">
    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li>{{ key }}</li>
      </ul>
    </div>

    <div v-if="projectLoading" class="space-y-2 mb-6">
      <div class="skeleton h-8 w-64" />
      <div class="skeleton h-4 w-96 max-w-full" />
    </div>
    <div v-else-if="project" class="mb-6">
      <h1 class="text-2xl font-bold">{{ project.name }}</h1>
      <p class="text-base-content/60 mt-1">{{ project.description }}</p>
    </div>

    <!-- Each section loads and fails on its own. -->
    <div class="space-y-4">
      <OverviewSprintSummary
        :project-key="key"
        :data="overview.sprint.data"
        :loading="overview.sprint.loading"
        :error="overview.sprint.error"
        @retry="overview.fetchSprint(key)"
      />
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <OverviewOpenTasks
          :project-key="key"
          :data="overview.tasks.data"
          :loading="overview.tasks.loading"
          :error="overview.tasks.error"
          @retry="overview.fetchTasks(key)"
        />
        <OverviewRecentActivity
          :project-key="key"
          :data="overview.activity.data"
          :loading="overview.activity.loading"
          :error="overview.activity.error"
          :scope="overview.scope"
          @update:scope="overview.fetchActivity(key, $event)"
          @retry="overview.fetchActivity(key)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const key = route.params.key as string
const projectsStore = useProjectsStore()
const overview = useOverviewStore()
const { showError } = useToast()
const projectLoading = ref(true)

// The store keeps the last project it loaded. If this project could not be loaded
// (for example the person is not a member) that leftover must not be shown.
const project = computed(() => {
  const p = projectsStore.current
  return p && p.key.toLowerCase() === key.toLowerCase() ? p : null
})

onMounted(async () => {
  // Start clean so another project's numbers are never shown while this one loads.
  overview.reset()
  // Load everything at once; every request records its own failure, so one that
  // fails never stops the others.
  await Promise.allSettled([
    loadProject(),
    overview.fetchSprint(key),
    overview.fetchTasks(key),
    overview.fetchActivity(key),
  ])
})

async function loadProject() {
  try {
    await projectsStore.fetchOne(key)
  }
  catch {
    showError('Failed to load project')
  }
  finally {
    projectLoading.value = false
  }
}
</script>
