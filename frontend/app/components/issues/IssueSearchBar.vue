<template>
  <div class="flex flex-col gap-3 mb-4">
    <!-- Text search -->
    <div class="relative">
      <input
        v-model="localQ"
        type="text"
        class="input input-bordered w-full pr-10"
        placeholder="Search issues…"
        aria-label="Search issues"
      />
      <button
        v-if="localQ"
        class="absolute right-3 top-1/2 -translate-y-1/2 text-base-content/40 hover:text-base-content"
        aria-label="Clear search query"
        @click="localQ = ''"
      >✕</button>
    </div>

    <!-- Filter row -->
    <div class="flex flex-wrap gap-2">
      <select v-model="localFilters.type" class="select select-bordered select-sm" aria-label="Filter by type">
        <option value="">All types</option>
        <option value="task">Task</option>
        <option value="story">Story</option>
        <option value="bug">Bug</option>
        <option value="epic">Epic</option>
      </select>

      <select v-model="localFilters.status" class="select select-bordered select-sm" aria-label="Filter by status">
        <option value="">All statuses</option>
        <option value="backlog">Backlog</option>
        <option value="todo">Todo</option>
        <option value="in_progress">In Progress</option>
        <option value="in_review">In Review</option>
        <option value="done">Done</option>
      </select>

      <select v-model="localFilters.priority" class="select select-bordered select-sm" aria-label="Filter by priority">
        <option value="">All priorities</option>
        <option value="critical">Critical</option>
        <option value="high">High</option>
        <option value="medium">Medium</option>
        <option value="low">Low</option>
      </select>

      <button
        v-if="hasActiveFilters"
        class="btn btn-ghost btn-sm"
        @click="clearAll"
      >Clear all</button>
    </div>

    <!-- Active filter chips -->
    <div v-if="hasActiveFilters" class="flex flex-wrap gap-1">
      <span v-if="localQ" class="badge badge-neutral gap-1">
        "{{ localQ }}" <button aria-label="Clear search query" @click="localQ = ''">✕</button>
      </span>
      <span v-if="localFilters.type" class="badge badge-neutral gap-1">
        {{ localFilters.type }} <button aria-label="Clear type filter" @click="localFilters.type = ''">✕</button>
      </span>
      <span v-if="localFilters.status" class="badge badge-neutral gap-1">
        {{ localFilters.status }} <button aria-label="Clear status filter" @click="localFilters.status = ''">✕</button>
      </span>
      <span v-if="localFilters.priority" class="badge badge-neutral gap-1">
        {{ localFilters.priority }} <button aria-label="Clear priority filter" @click="localFilters.priority = ''">✕</button>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { IssueFilters } from '~/types/domain.types'

const emit = defineEmits<{ search: [filters: IssueFilters] }>()

const localQ = ref('')
const localFilters = reactive<Omit<IssueFilters, 'q' | 'assignee_id' | 'label_id'>>({
  type: '',
  status: '',
  priority: '',
})

const hasActiveFilters = computed(() =>
  !!localQ.value || !!localFilters.type || !!localFilters.status || !!localFilters.priority
)

// Debounce: emit 300ms after any change.
let debounceTimer: ReturnType<typeof setTimeout>
function emitSearch() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    emit('search', {
      q: localQ.value || undefined,
      type: localFilters.type || undefined,
      status: localFilters.status || undefined,
      priority: localFilters.priority || undefined,
    })
  }, 300)
}

// Clear any in-flight debounce when the component is destroyed.
onUnmounted(() => clearTimeout(debounceTimer))

watch([localQ, () => localFilters.type, () => localFilters.status, () => localFilters.priority], emitSearch)

function clearAll() {
  localQ.value = ''
  localFilters.type = ''
  localFilters.status = ''
  localFilters.priority = ''
}
</script>
