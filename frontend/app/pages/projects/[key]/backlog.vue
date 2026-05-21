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

    <!-- Sprint management section -->
    <section class="mb-8">
      <SprintsSprintList
        :project-key="key"
        :can-manage="canManage"
        @removed-from-sprint="onRemovedFromSprint"
      />
    </section>

    <!-- Backlog issue list -->
    <section>
      <div class="flex items-center gap-2 mb-3">
        <h2 class="text-lg font-semibold">Backlog</h2>
        <span class="badge badge-neutral">{{ issues.length }}</span>
      </div>
      <IssuesIssueSearchBar @search="onSearch" />
      <BacklogList
        :issues="issues"
        :project-key="key"
        :loading="loading"
        :can-manage="canManage"
        :sprints="sprintsStore.sprints"
        @sprint-assign="onSprintAssign"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { backlogService } from '~/services/backlog.service'
import { projectsService } from '~/services/projects.service'
import { issuesService } from '~/services/issues.service'
import type { Issue, IssueFilters, MemberResponse } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const { showSuccess, showError } = useToast()

const authStore = useAuthStore()
const projectsStore = useProjectsStore()
const sprintsStore = useSprintsStore()
const issuesStore = useIssuesStore()

const issues = ref<Issue[]>([])
const members = ref<MemberResponse[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const searchActive = ref(false)

const myRole = computed(() =>
  members.value.find(m => m.user_id === authStore.user?.id)?.role
)

const canManage = computed(() =>
  projectsStore.current?.ownerId === authStore.user?.id
  || myRole.value === 'admin'
)

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
      projectsService.listMembers(key).then(v => (members.value = v)).catch(() => {}),
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
    try {
      issues.value = await backlogService.getBacklog(key)
    } catch {
      showError('Failed to refresh backlog')
    }
    return
  }
  searchActive.value = true
  await issuesStore.searchIssues(key, filters)
  issues.value = issuesStore.issues
}

async function onRemovedFromSprint({ issueId }: { issueId: number; sprintId: string }) {
  try {
    const refreshed = await backlogService.getBacklog(key)
    issues.value = refreshed
  }
  catch {
    showError('Failed to refresh backlog')
  }
}

async function onSprintAssign({ issueId, sprintId }: { issueId: number; sprintId: number }) {
  const sprint = sprintsStore.sprints.find(s => Number(s.id) === sprintId)
  try {
    await issuesService.update(key, issueId, { sprint_id: sprintId })
    issues.value = issues.value.filter(i => i.id !== issueId)
    showSuccess(`Added to ${sprint?.name ?? 'sprint'}`)
  }
  catch {
    showError('Failed to add issue to sprint')
  }
}
</script>
