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

    <!-- Search bar (always visible) -->
    <IssuesIssueSearchBar @search="onSearch" />

    <!-- Search results (shown when a search is active, hides normal view) -->
    <div v-if="searchActive" class="mb-8">
      <div class="flex items-center gap-2 mb-3">
        <h2 class="text-lg font-semibold">Search Results</h2>
        <span class="badge badge-neutral">{{ issuesStore.searchResults.length }}</span>
      </div>
      <p v-if="!loading && issuesStore.searchResults.length === 0" class="text-base-content/50 text-sm py-6 text-center">
        No issues match your search.
      </p>
      <div v-else class="flex flex-col gap-2">
        <div
          v-for="result in issuesStore.searchResults"
          :key="result.id"
          class="flex items-center gap-3 p-3 bg-base-200 rounded-lg"
        >
          <span class="badge badge-ghost font-mono text-xs shrink-0">{{ result.key }}</span>
          <NuxtLink
            :to="`/projects/${key}/issues/${result.id}`"
            class="font-medium hover:underline flex-1 truncate"
          >{{ result.title }}</NuxtLink>
          <span class="badge badge-sm shrink-0">{{ result.status }}</span>
          <span class="badge badge-sm badge-outline shrink-0">{{ result.priority }}</span>
        </div>
      </div>
    </div>

    <!-- Backlog view (hidden while search is active) -->
    <template v-else>
      <!-- Backlog issue list -->
      <section>
        <div class="flex items-center gap-2 mb-3">
          <h2 class="text-lg font-semibold">Backlog</h2>
          <span class="badge badge-neutral">{{ issues.length }}</span>
        </div>
        <BacklogList
          :issues="issues"
          :project-key="key"
          :loading="loading"
          :sprints="sprintsStore.sprints"
          @sprint-assign="onSprintAssign"
        />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { backlogService } from '~/services/backlog.service'
import { issuesService } from '~/services/issues.service'
import type { Issue, IssueFilters } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const { showSuccess, showError } = useToast()

const projectsStore = useProjectsStore()
const sprintsStore = useSprintsStore()
const issuesStore = useIssuesStore()

const issues = ref<Issue[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const searchActive = ref(false)

onMounted(async () => {
  try {
    await Promise.all([
      backlogService.getBacklog(key).then(v => (issues.value = v)).catch((e: unknown) => {
        const msg = e instanceof Error ? e.message : 'Unknown error'
        error.value = msg
        showError('Failed to load backlog')
      }),
      projectsStore.fetchOne(key).catch((e: unknown) => {
        const msg = e instanceof Error ? e.message : 'Unknown error'
        error.value = error.value ?? msg
        showError('Failed to load project')
      }),
      sprintsStore.fetchSprints(key),
    ])
  }
  finally {
    loading.value = false
  }
})

async function onSearch(filters: IssueFilters) {
  const isEmpty = !filters.q && !filters.type && !filters.status && !filters.priority
  if (isEmpty) {
    searchActive.value = false
    await issuesStore.searchIssues(key, {}) // clears searchResults
    return
  }
  searchActive.value = true
  loading.value = true
  try {
    await issuesStore.searchIssues(key, filters)
  }
  finally {
    loading.value = false
  }
}

async function onSprintAssign({ issueId, sprintId }: { issueId: number; sprintId: number }) {
  const sprint = sprintsStore.sprints.find(s => Number(s.id) === sprintId)
  try {
    await issuesService.update(key, issueId, { sprint_id: sprintId })
    issues.value = issues.value.filter(i => i.id !== issueId)
    // The Sprints page must not show a stale list for this sprint.
    sprintsStore.invalidateSprintIssues(String(sprintId))
    showSuccess(`Added to ${sprint?.name ?? 'sprint'}`)
  }
  catch {
    showError('Failed to add issue to sprint')
  }
}
</script>
