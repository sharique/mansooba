<template>
  <div class="grid grid-cols-3 gap-6">
    <!-- Main content -->
    <div class="col-span-2 space-y-4">
      <!-- Title inline-edit -->
      <div>
        <input
          v-if="editing === 'title'"
          ref="titleInput"
          v-model="editTitle"
          class="input input-bordered w-full text-xl font-bold"
          @blur="saveField('title', editTitle)"
          @keyup.enter="saveField('title', editTitle)"
          @keyup.escape="editing = null"
        />
        <h2
          v-else
          class="text-xl font-bold cursor-pointer hover:bg-base-200 rounded px-1 -mx-1"
          @click="startEdit('title')"
        >
          {{ issue.title }}
        </h2>
      </div>

      <!-- Description inline-edit -->
      <div>
        <textarea
          v-if="editing === 'description'"
          ref="descInput"
          v-model="editDescription"
          class="textarea textarea-bordered w-full"
          rows="5"
          @blur="saveField('description', editDescription)"
          @keyup.escape="editing = null"
        />
        <div
          v-else
          class="cursor-pointer hover:bg-base-200 rounded p-1 -mx-1 min-h-12 text-base-content/70 whitespace-pre-wrap"
          @click="startEdit('description')"
        >
          {{ issue.description || 'Click to add description…' }}
        </div>
      </div>
    </div>

    <!-- Sidebar -->
    <div class="space-y-4">
      <!-- Status -->
      <div class="form-control">
        <label class="label py-1"><span class="label-text font-medium">Status</span></label>
        <select
          :value="issue.status"
          class="select select-bordered select-sm"
          @change="onFieldChange('status', ($event.target as HTMLSelectElement).value)"
        >
          <option value="backlog">Backlog</option>
          <option value="todo">To Do</option>
          <option value="in_progress">In Progress</option>
          <option value="in_review">In Review</option>
          <option value="done">Done</option>
        </select>
      </div>

      <!-- Type -->
      <div>
        <span class="text-sm font-medium block mb-1">Type</span>
        <span class="badge badge-outline capitalize">{{ issue.type }}</span>
      </div>

      <!-- Priority -->
      <div>
        <span class="text-sm font-medium block mb-1">Priority</span>
        <span :class="priorityClass" class="badge capitalize">{{ issue.priority }}</span>
      </div>

      <!-- Assignee -->
      <div>
        <span class="text-sm font-medium block mb-1">Assignee</span>
        <span class="text-sm">{{ issue.assigneeId ? `User #${issue.assigneeId}` : 'Unassigned' }}</span>
      </div>

      <!-- Reporter -->
      <div>
        <span class="text-sm font-medium block mb-1">Reporter</span>
        <span class="text-sm">User #{{ issue.reporterId }}</span>
      </div>

      <!-- Delete -->
      <div v-if="canDelete" class="pt-4 border-t border-base-200">
        <button class="btn btn-error btn-sm w-full" @click="confirmModal?.showModal()">Delete Issue</button>
      </div>
    </div>
  </div>

  <!-- Delete confirm modal -->
  <dialog ref="confirmModal" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-2">Delete issue?</h3>
      <p class="text-base-content/70 mb-4">This action cannot be undone.</p>
      <div class="flex gap-2 justify-end">
        <form method="dialog"><button class="btn btn-ghost">Cancel</button></form>
        <button class="btn btn-error" :disabled="deleting" @click="deleteIssue">
          <span v-if="deleting" class="loading loading-spinner loading-sm" />
          Delete
        </button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop"><button>close</button></form>
  </dialog>
</template>

<script setup lang="ts">
import type { Issue } from '~/types/domain.types'

const props = defineProps<{ issue: Issue; projectKey: string }>()
const emit = defineEmits<{ deleted: [] }>()

const issuesStore = useIssuesStore()
const authStore = useAuthStore()

const canDelete = computed(() => authStore.user?.id === props.issue.reporterId)

const editing = ref<'title' | 'description' | null>(null)
const editTitle = ref(props.issue.title)
const editDescription = ref(props.issue.description ?? '')

const titleInput = ref<HTMLInputElement | null>(null)
const descInput = ref<HTMLTextAreaElement | null>(null)
const confirmModal = ref<HTMLDialogElement | null>(null)
const deleting = ref(false)

const priorityClass = computed(() => ({
  'badge-error': props.issue.priority === 'critical',
  'badge-warning': props.issue.priority === 'high',
  'badge-info': props.issue.priority === 'medium',
  'badge-ghost': props.issue.priority === 'low',
}))

async function startEdit(field: 'title' | 'description') {
  editTitle.value = props.issue.title
  editDescription.value = props.issue.description ?? ''
  editing.value = field
  await nextTick()
  if (field === 'title') titleInput.value?.focus()
  else descInput.value?.focus()
}

async function saveField(field: 'title' | 'description', value: string) {
  editing.value = null
  const trimmed = value.trim()
  if (field === 'title' && !trimmed) return
  if (trimmed === (field === 'title' ? props.issue.title : props.issue.description)) return
  await issuesStore.update(props.projectKey, props.issue.id, { [field]: trimmed })
}

async function onFieldChange(field: string, value: string) {
  await issuesStore.update(props.projectKey, props.issue.id, { [field]: value } as never)
}

async function deleteIssue() {
  deleting.value = true
  try {
    await issuesStore.remove(props.projectKey, props.issue.id)
    confirmModal.value?.close()
    emit('deleted')
  }
  finally {
    deleting.value = false
  }
}
</script>
