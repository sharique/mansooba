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
      <SprintsSprintList :project-key="key" :can-manage="canManage" />
    </section>

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
import { projectsService } from '~/services/projects.service'
import type { Issue, MemberResponse } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const { showError } = useToast()

const authStore = useAuthStore()
const projectsStore = useProjectsStore()
const sprintsStore = useSprintsStore()

const issues = ref<Issue[]>([])
const members = ref<MemberResponse[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

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
</script>
