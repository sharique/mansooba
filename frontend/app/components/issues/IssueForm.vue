<template>
  <form @submit.prevent="submit">
    <p v-if="!canWrite" class="text-sm text-base-content/70 mb-3" data-testid="read-only-note">{{ reason('task.write') }}</p>
    <!-- Title -->
    <div class="form-control mb-4">
      <label class="label"><span class="label-text">Title <span class="text-error">*</span></span></label>
      <input
        v-model="form.title"
        data-testid="title"
        type="text"
        class="input input-bordered w-full"
        :class="{ 'input-error': titleError }"
        placeholder="Issue title"
      />
      <label v-if="titleError" class="label">
        <span class="label-text-alt text-error">Title is required</span>
      </label>
    </div>

    <!-- Description -->
    <div class="form-control mb-4">
      <label class="label"><span class="label-text">Description</span></label>
      <textarea
        v-model="form.description"
        class="textarea textarea-bordered w-full"
        rows="3"
        placeholder="Optional description"
      />
    </div>

    <!-- Type & Priority -->
    <div class="grid grid-cols-2 gap-3 mb-4">
      <div class="form-control">
        <label class="label"><span class="label-text">Type</span></label>
        <select v-model="form.type" class="select select-bordered w-full">
          <option value="task">Task</option>
          <option value="story">Story</option>
          <option value="bug">Bug</option>
          <option value="epic">Epic</option>
        </select>
      </div>
      <div class="form-control">
        <label class="label"><span class="label-text">Priority</span></label>
        <select v-model="form.priority" class="select select-bordered w-full">
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="critical">Critical</option>
        </select>
      </div>
    </div>

    <!-- Story Points -->
    <div class="form-control mb-4">
      <label class="label"><span class="label-text">Story Points</span></label>
      <input
        v-model.number="form.story_points"
        type="number"
        min="0"
        max="100"
        class="input input-bordered w-full"
        placeholder="Leave blank if unestimated"
      />
    </div>

    <!-- Assignee -->
    <div class="form-control mb-4">
      <label class="label"><span class="label-text">Assignee</span></label>
      <select v-model="form.assignee_id" class="select select-bordered w-full">
        <option :value="undefined">Unassigned</option>
        <option v-for="m in members" :key="m.user_id" :value="m.user_id">{{ m.name }}</option>
      </select>
    </div>

    <!-- Sprint (new issues only): backlog by default, or a current / upcoming sprint -->
    <div v-if="!issue && openSprints.length > 0" class="form-control mb-4">
      <label class="label"><span class="label-text">Sprint</span></label>
      <select v-model="chosenSprint" data-testid="sprint" class="select select-bordered w-full">
        <option value="">Backlog (no sprint)</option>
        <option v-for="sp in openSprints" :key="sp.id" :value="String(sp.id)">
          {{ sp.name }} ({{ sp.status === 'active' ? 'active' : 'planned' }})
        </option>
      </select>
    </div>

    <div class="flex gap-2 justify-end">
      <button type="button" class="btn btn-ghost" @click="$emit('cancelled')">Cancel</button>
      <button
        data-testid="submit"
        type="submit"
        class="btn btn-primary"
        :disabled="saving || !canWrite"
        :title="canWrite ? undefined : reason('task.write')"
      >
        <span v-if="saving" class="loading loading-spinner loading-sm" />
        {{ issue ? 'Save' : 'Create' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { projectsService } from '~/services/projects.service'
import type { Issue, MemberResponse, Sprint } from '~/types/domain.types'
import type { CreateIssueRequest } from '~/services/issues.service'

const props = defineProps<{
  projectKey: string
  issue?: Issue
  defaultStatus?: string
  /** Sprints the new issue can be put into; finished ones are ignored here. */
  sprints?: Sprint[]
}>()

const emit = defineEmits<{
  saved: [issue: Issue]
  cancelled: []
}>()

const issuesStore = useIssuesStore()
const { can, reason } = useProjectPermissions()
const canWrite = computed(() => can('task.write'))

// Only the current and upcoming sprints can take new work.
const openSprints = computed(() => (props.sprints ?? []).filter(sp => sp.status !== 'completed'))
const chosenSprint = ref('')

const form = reactive<CreateIssueRequest>({
  title: props.issue?.title ?? '',
  description: props.issue?.description ?? '',
  type: props.issue?.type ?? 'task',
  priority: props.issue?.priority ?? 'medium',
  status: props.issue?.status ?? props.defaultStatus ?? 'todo',
  assignee_id: props.issue?.assignee_id,
  story_points: props.issue?.story_points,
})

const titleError = ref(false)
const saving = ref(false)
const members = ref<MemberResponse[]>([])

onMounted(async () => {
  try {
    members.value = await projectsService.listMembers(props.projectKey)
  }
  catch { /* ignore — assignee select stays empty */ }
})

async function submit() {
  if (!canWrite.value) return
  titleError.value = !form.title.trim()
  if (titleError.value) return

  saving.value = true
  try {
    let result: Issue
    if (props.issue) {
      // The server reads 0 as "clear"; an unset select would be ignored (ADR-036).
      const payload = form.assignee_id === undefined && props.issue.assignee_id
        ? { ...form, assignee_id: 0 }
        : form
      result = await issuesStore.update(props.projectKey, props.issue.id, payload)
    }
    else {
      result = await issuesStore.create(
        props.projectKey,
        chosenSprint.value ? { ...form, sprint_id: Number(chosenSprint.value) } : form,
      )
    }
    emit('saved', result)
  }
  finally {
    saving.value = false
  }
}
</script>
