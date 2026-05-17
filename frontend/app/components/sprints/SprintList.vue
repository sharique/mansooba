<script setup lang="ts">
import type { Sprint } from '~/types/domain.types'

const props = defineProps<{
  projectKey: string
  canManage: boolean
}>()

const sprintsStore = useSprintsStore()
const { showSuccess, showError } = useToast()

const showCreateForm = ref(false)
const editingSprint = ref<Sprint | null>(null)
const completingSprint = ref<Sprint | null>(null)

const otherSprints = computed(() =>
  completingSprint.value
    ? sprintsStore.openSprints.filter(s => s.id !== completingSprint.value!.id)
    : []
)

async function handleStart(sprint: Sprint) {
  try {
    await sprintsStore.startSprint(props.projectKey, sprint.id)
    showSuccess(`"${sprint.name}" is now Active`)
  }
  catch (e: any) {
    showError(e.data?.message ?? 'Failed to start sprint')
  }
}

async function handleDelete(sprint: Sprint) {
  if (!confirm(`Delete "${sprint.name}"? This cannot be undone.`)) return
  try {
    await sprintsStore.deleteSprint(props.projectKey, sprint.id)
    showSuccess('Sprint deleted')
  }
  catch (e: any) {
    showError(e.data?.message ?? 'Failed to delete sprint')
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-3">
      <h2 class="text-lg font-semibold">Sprints</h2>
      <button
        v-if="canManage"
        class="btn btn-sm btn-primary"
        @click="showCreateForm = true"
      >
        + New Sprint
      </button>
    </div>

    <div
      v-if="sprintsStore.sprints.length > 0"
      class="flex flex-col gap-2 mb-4"
    >
      <SprintsSprintCard
        v-for="sprint in sprintsStore.sprints"
        :key="sprint.id"
        :sprint="sprint"
        :project-key="projectKey"
        :can-manage="canManage"
        :has-active-sprint="!!sprintsStore.activeSprint"
        @start="handleStart"
        @complete="completingSprint = $event"
        @edit="editingSprint = $event"
        @delete="handleDelete"
      />
    </div>
    <p v-else class="text-sm text-base-content/50 mb-4">No sprints yet.</p>

    <SprintsSprintForm
      v-if="showCreateForm"
      :project-key="projectKey"
      @saved="showCreateForm = false"
      @cancel="showCreateForm = false"
    />
    <SprintsSprintForm
      v-if="editingSprint"
      :project-key="projectKey"
      :sprint="editingSprint"
      @saved="editingSprint = null"
      @cancel="editingSprint = null"
    />
    <SprintsCompleteSprintModal
      v-if="completingSprint"
      :project-key="projectKey"
      :sprint="completingSprint"
      :other-sprints="otherSprints"
      @completed="completingSprint = null"
      @cancel="completingSprint = null"
    />
  </div>
</template>
