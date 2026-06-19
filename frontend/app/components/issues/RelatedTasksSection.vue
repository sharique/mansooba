<template>
  <div class="mt-6">
    <div class="flex items-center justify-between mb-3">
      <h3 class="font-semibold text-sm text-base-content/70 uppercase tracking-wide">Related Tasks</h3>
      <button class="btn btn-ghost btn-xs gap-1" @click="togglePopover">
        <Icon name="mdi:plus" class="w-4 h-4" /> Add relation
      </button>
    </div>

    <!-- Relation list -->
    <div v-if="store.relations.length" class="space-y-2">
      <div
        v-for="rel in store.relations"
        :key="rel.id"
        class="flex items-center justify-between gap-2 rounded-lg border border-base-300 px-3 py-2 bg-base-100"
      >
        <div class="flex items-center gap-2 min-w-0">
          <span :class="['badge badge-sm', relationBadgeClass(rel.relation_type)]">
            {{ relationLabel(rel.relation_type) }}
          </span>
          <span class="font-mono text-xs text-base-content/50 shrink-0">{{ rel.related_issue.key }}</span>
          <span class="text-sm truncate">{{ rel.related_issue.title }}</span>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <span class="badge badge-outline badge-xs">{{ rel.related_issue.status }}</span>
          <button
            class="btn btn-ghost btn-xs text-error"
            :aria-label="`Remove relation ${rel.id}`"
            @click="removeRelation(rel.id)"
          >
            <Icon name="mdi:close" class="w-3 h-3" />
          </button>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <p v-else class="text-sm text-base-content/40 py-2">No related tasks.</p>

    <!-- Add relation popover -->
    <div v-if="popoverOpen" class="mt-3 rounded-lg border border-base-300 bg-base-100 p-4 space-y-3">
      <div class="form-control">
        <label class="label"><span class="label-text text-xs">Search tasks in this project</span></label>
        <input
          v-model="searchQuery"
          type="text"
          class="input input-sm input-bordered"
          placeholder="Type to search…"
          @input="onSearch"
        >
        <ul v-if="searchResults.length" class="menu bg-base-200 rounded-box mt-1 max-h-40 overflow-y-auto">
          <li v-for="issue in searchResults" :key="issue.id">
            <button class="text-sm text-left" @click="selectIssue(issue)">
              <span class="font-mono text-xs text-base-content/50">{{ issue.key }}</span>
              {{ issue.title }}
            </button>
          </li>
        </ul>
      </div>
      <div v-if="selectedIssue" class="text-sm text-base-content/70">
        Selected: <strong>{{ selectedIssue.key }}</strong> — {{ selectedIssue.title }}
      </div>
      <div class="form-control">
        <label class="label"><span class="label-text text-xs">Relation type</span></label>
        <select v-model="relationType" class="select select-sm select-bordered">
          <option value="blocks">Blocks</option>
          <option value="relates_to">Relates to</option>
          <option value="duplicates">Duplicates</option>
        </select>
      </div>
      <div class="flex gap-2 justify-end">
        <button class="btn btn-ghost btn-sm" @click="popoverOpen = false">Cancel</button>
        <button
          class="btn btn-primary btn-sm"
          :disabled="!selectedIssue"
          @click="addRelation"
        >
          Add
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useIssueRelationsStore } from '~/stores/issue-relations.store'
import type { Issue } from '~/types/domain.types'

const props = defineProps<{ issueId: number; projectKey: string }>()
const store = useIssueRelationsStore()
const { showSuccess, showError } = useToast()

const popoverOpen = ref(false)
const searchQuery = ref('')
const searchResults = ref<Issue[]>([])
const selectedIssue = ref<Issue | null>(null)
const relationType = ref('relates_to')

const relationLabels: Record<string, string> = {
  blocks: 'Blocks',
  is_blocked_by: 'Is blocked by',
  relates_to: 'Relates to',
  duplicates: 'Duplicates',
}

const relationBadgeClasses: Record<string, string> = {
  blocks: 'badge-error',
  is_blocked_by: 'badge-warning',
  relates_to: 'badge-info',
  duplicates: 'badge-ghost',
}

function relationLabel(type: string) {
  return relationLabels[type] ?? type
}

function relationBadgeClass(type: string) {
  return relationBadgeClasses[type] ?? 'badge-neutral'
}

function togglePopover() {
  popoverOpen.value = !popoverOpen.value
  if (!popoverOpen.value) resetPopover()
}

function resetPopover() {
  searchQuery.value = ''
  searchResults.value = []
  selectedIssue.value = null
  relationType.value = 'relates_to'
}

async function onSearch() {
  const q = searchQuery.value.trim()
  if (!q) { searchResults.value = []; return }
  try {
    const { $api } = useNuxtApp()
    const results = await $api<Issue[]>(`/projects/${props.projectKey}/issues`, { query: { q } })
    searchResults.value = results.filter(i => i.id !== props.issueId)
  }
  catch { searchResults.value = [] }
}

function selectIssue(issue: Issue) {
  selectedIssue.value = issue
  searchQuery.value = issue.title
  searchResults.value = []
}

async function addRelation() {
  if (!selectedIssue.value) return
  try {
    await store.create(props.issueId, {
      target_issue_id: selectedIssue.value.id,
      relation_type: relationType.value,
    })
    showSuccess('Relation added')
    popoverOpen.value = false
    resetPopover()
  }
  catch {
    showError('Failed to add relation')
  }
}

async function removeRelation(relationId: number) {
  try {
    await store.remove(props.issueId, relationId)
    showSuccess('Relation removed')
  }
  catch {
    showError('Failed to remove relation')
  }
}

onMounted(() => {
  store.fetchForIssue(props.issueId)
})
</script>
