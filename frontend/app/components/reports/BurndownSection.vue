<script setup lang="ts">
import { SprintStatus } from '~/types/domain.types'
import type { Sprint } from '~/types/domain.types'

const props = defineProps<{
  projectKey: string
}>()

const sprintsStore = useSprintsStore()
const { formatDate } = useTimeFormatter()
const loading = ref(false)
const failed = ref(false)
const selectedId = ref('')

// Planned sprints have no burndown yet; the rest are offered, newest first.
const choices = computed<Sprint[]>(() =>
  sprintsStore.sprints
    .filter(s => s.status !== SprintStatus.Planning)
    .sort((a, b) => (b.start_date ?? '').localeCompare(a.start_date ?? '')),
)

// The API returns numeric ids (typed as string) while a <select> value is always a string.
const selected = computed(() => choices.value.find(s => String(s.id) === selectedId.value) ?? null)

// Only the latest request may touch the page, so a slow answer for a sprint
// that is no longer selected cannot overwrite the chart.
let requestSeq = 0

async function loadBurndown(id: string) {
  const seq = ++requestSeq
  failed.value = false
  if (!id) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    await sprintsStore.fetchBurndown(props.projectKey, id)
  }
  catch {
    if (seq === requestSeq) failed.value = true
  }
  finally {
    if (seq === requestSeq) loading.value = false
  }
}

function onSelect(e: Event) {
  selectedId.value = (e.target as HTMLSelectElement).value
  void loadBurndown(selectedId.value)
}

// Defaults to the sprint in progress, or none when there is not one.
onMounted(async () => {
  loading.value = true
  try {
    await sprintsStore.fetchSprints(props.projectKey)
  }
  catch {
    failed.value = true
    loading.value = false
    return
  }
  selectedId.value = sprintsStore.activeSprint ? String(sprintsStore.activeSprint.id) : ''
  await loadBurndown(selectedId.value)
})
</script>

<template>
  <section class="mb-8">
    <div class="card card-bordered bg-base-100 shadow-sm">
      <div class="card-body p-4">
        <div class="flex flex-wrap items-center justify-between gap-2 mb-2">
          <h2 class="text-lg font-semibold">Burndown</h2>
          <div class="flex items-center gap-2">
            <span v-if="selected && (selected.start_date || selected.end_date)" class="text-sm text-base-content/50 whitespace-nowrap">
              {{ formatDate(selected.start_date) }} → {{ formatDate(selected.end_date) }}
            </span>
            <select
              class="select select-bordered select-sm"
              aria-label="Sprint"
              :value="selectedId"
              @change="onSelect"
            >
              <option value="">None</option>
              <option v-for="s in choices" :key="s.id" :value="String(s.id)">
                {{ s.name }}{{ s.status === 'active' ? ' (current)' : '' }}
              </option>
            </select>
          </div>
        </div>
        <div v-if="loading" class="skeleton h-64 w-full" />
        <div v-else-if="failed" class="alert alert-error"><span>Failed to load burndown</span></div>
        <ChartsBurndownChart
          v-else-if="selected && sprintsStore.burndownData"
          :key="String(selected.id)"
          :data="sprintsStore.burndownData"
        />
        <p v-else class="text-sm text-base-content/50 py-10 text-center">
          {{ sprintsStore.activeSprint ? 'Select a sprint to see its burndown.' : 'No sprint is in progress. Pick a completed sprint to see its burndown.' }}
        </p>
      </div>
    </div>
  </section>
</template>
