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
      <div class="flex items-center gap-3">
        <h1 class="text-2xl font-bold">Board</h1>
        <NuxtLink :to="`/projects/${key}/backlog`" class="btn btn-sm btn-ghost">
          Backlog →
        </NuxtLink>
        <NuxtLink :to="`/projects/${key}/settings`" class="btn btn-sm btn-ghost">
          Settings
        </NuxtLink>
      </div>
      <button class="btn btn-primary btn-sm" @click="openCreateModal('')">Create Issue</button>
    </div>

    <!-- Burndown chart — shown when a sprint is Active -->
    <section v-if="sprintsStore.activeSprint" class="mb-6">
      <div class="card card-bordered bg-base-100 shadow-sm">
        <div class="card-body p-4">
          <div class="flex items-center justify-between mb-2">
            <h3 class="font-semibold">
              {{ sprintsStore.activeSprint.name }}
              <span class="badge badge-success badge-sm ml-2">Active</span>
            </h3>
            <span v-if="sprintsStore.activeSprint.end_date" class="text-sm text-base-content/50">
              Ends {{ sprintsStore.activeSprint.end_date }}
            </span>
          </div>
          <div v-if="loadingBurndown" class="skeleton h-64 w-full" />
          <ChartsBurndownChart v-else-if="sprintsStore.burndownData" :data="sprintsStore.burndownData" />
        </div>
      </div>
    </section>

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
const sprintsStore = useSprintsStore()
const { showSuccess, showError } = useToast()

const boardData = ref<BoardData | null>(null)
const loading = ref(true)
const loadingBurndown = ref(false)
const selectedStatus = ref('')
const modalOpen = ref(false)

async function fetchBoard() {
  boardData.value = await boardService.getBoard(key)
}

onMounted(async () => {
  try {
    await Promise.all([fetchBoard(), sprintsStore.fetchSprints(key)])
  }
  catch {
    showError('Failed to load board')
  }
  finally {
    loading.value = false
  }
})

watch(
  () => sprintsStore.activeSprint,
  async (sprint) => {
    if (!sprint) return
    loadingBurndown.value = true
    try {
      await sprintsStore.fetchBurndown(key, sprint.id)
    }
    catch {
      // burndownData remains null on error
    }
    finally {
      loadingBurndown.value = false
    }
  },
  { immediate: true },
)

function openCreateModal(status: string) {
  selectedStatus.value = status
  modalOpen.value = true
}

async function onIssueCreated(_issue: Issue) {
  modalOpen.value = false
  showSuccess('Issue created')
  boardData.value = await boardService.getBoard(key)
}
</script>
