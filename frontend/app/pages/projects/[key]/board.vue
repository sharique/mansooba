<template>
  <div>
    <!-- Read out by screen readers after every move or failed move -->
    <div class="sr-only" role="status" aria-live="polite">{{ store.announcement }}</div>

    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li><NuxtLink :to="`/projects/${key}`">{{ key }}</NuxtLink></li>
        <li>Sprint board</li>
      </ul>
    </div>

    <div class="flex justify-between items-center mb-4">
      <div class="flex items-center gap-3">
        <h1 class="text-2xl font-bold">Sprint board</h1>
        <NuxtLink :to="`/projects/${key}/backlog`" class="btn btn-sm btn-ghost">
          Backlog →
        </NuxtLink>
        <NuxtLink :to="`/projects/${key}/sprints`" class="btn btn-sm btn-ghost">
          Sprints →
        </NuxtLink>
        <NuxtLink :to="`/projects/${key}/settings`" class="btn btn-sm btn-ghost">
          Settings
        </NuxtLink>
      </div>
      <button class="btn btn-primary btn-sm" @click="openCreateModal('')">Create Issue</button>
    </div>

    <!-- Skeleton -->
    <div v-if="store.loading" class="flex gap-4">
      <div v-for="n in 4" :key="n" class="skeleton w-72 h-96 rounded-xl" />
    </div>

    <!-- Board columns -->
    <div v-else class="flex gap-4 overflow-x-auto pb-4">
      <BoardColumn
        v-for="col in store.columns"
        :key="col.status"
        :column="col"
        :project-key="key"
        @create-issue="openCreateModal"
        @moved="onMoved"
      />
    </div>

    <IssuesCreateIssueModal
      :project-key="key"
      :default-status="selectedStatus"
      :open="modalOpen"
      @created="onIssueCreated"
      @close="modalOpen = false"
    />
  </div>
</template>

<script setup lang="ts">
import { useBoardStore } from '~/stores/board.store'
import type { Issue } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const { showSuccess, showError } = useToast()
const store = useBoardStore()

const selectedStatus = ref('')
const modalOpen = ref(false)

onMounted(async () => {
  await store.load(key)
  if (store.error) showError('Failed to load sprint board')
})

// One toast per failed move; the card has already been put back by the store.
watch(() => store.moveFailure, (failure) => {
  if (failure) showError(failure.message)
})

// Coming back to a board that has been open for a while refreshes it quietly, so an old
// copy (someone else's moves, a sprint completed meanwhile) does not stay on screen.
function refreshOnReturn() {
  if (document.visibilityState === 'visible') store.refreshIfStale(key)
}
onMounted(() => {
  document.addEventListener('visibilitychange', refreshOnReturn)
  window.addEventListener('focus', refreshOnReturn)
})
onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', refreshOnReturn)
  window.removeEventListener('focus', refreshOnReturn)
})

function openCreateModal(status: string) {
  selectedStatus.value = status
  modalOpen.value = true
}

async function onIssueCreated(_issue: Issue) {
  modalOpen.value = false
  showSuccess('Issue created')
  await store.load(key)
}

function onMoved(move: { issueId: number; toStatus: Issue['status'] }) {
  store.moveIssue(key, move.issueId, move.toStatus)
}
</script>
