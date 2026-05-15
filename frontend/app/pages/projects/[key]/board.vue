<template>
  <div>
    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li><NuxtLink :to="`/projects/${key}`">{{ key }}</NuxtLink></li>
        <li>Board</li>
      </ul>
    </div>

    <div class="flex justify-between items-center mb-4">
      <h1 class="text-2xl font-bold">Board</h1>
      <button class="btn btn-primary btn-sm" @click="openCreateModal('')">Create Issue</button>
    </div>

    <!-- Skeleton -->
    <div v-if="loading" class="flex gap-4">
      <div v-for="n in 4" :key="n" class="skeleton w-72 h-96 rounded-xl" />
    </div>

    <!-- Board columns -->
    <div v-else class="flex gap-4 overflow-x-auto pb-4">
      <BoardColumn
        v-for="col in boardData?.columns"
        :key="col.status"
        :column="col"
        :project-key="key"
        @issue-status-changed="onStatusChange"
        @create-issue="openCreateModal"
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
import { boardService } from '~/services/board.service'
import type { BoardData } from '~/services/board.service'
import type { Issue } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const issuesStore = useIssuesStore()
const { showSuccess, showError } = useToast()

const boardData = ref<BoardData | null>(null)
const loading = ref(true)
const selectedStatus = ref('')
const modalOpen = ref(false)

async function fetchBoard() {
  boardData.value = await boardService.getBoard(key)
}

onMounted(async () => {
  try {
    await fetchBoard()
  }
  catch {
    showError('Failed to load board')
  }
  finally {
    loading.value = false
  }
})

function openCreateModal(status: string) {
  selectedStatus.value = status
  modalOpen.value = true
}

async function onStatusChange(issueId: number, newStatus: string) {
  try {
    await issuesStore.update(key, issueId, { status: newStatus } as never)
    boardData.value = await boardService.getBoard(key)
  }
  catch {
    showError('Failed to update status')
  }
}

async function onIssueCreated(_issue: Issue) {
  modalOpen.value = false
  showSuccess('Issue created')
  boardData.value = await boardService.getBoard(key)
}
</script>
