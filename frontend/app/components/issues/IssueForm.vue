<template>
  <form @submit.prevent="submit">
    <!-- Title -->
    <div class="form-control mb-3">
      <label class="label"><span class="label-text">Title <span class="text-error">*</span></span></label>
      <input
        v-model="form.title"
        data-testid="title"
        type="text"
        class="input input-bordered"
        :class="{ 'input-error': titleError }"
        placeholder="Issue title"
      />
      <span v-if="titleError" class="label-text-alt text-error mt-1">Title is required</span>
    </div>

    <!-- Description -->
    <div class="form-control mb-3">
      <label class="label"><span class="label-text">Description</span></label>
      <textarea
        v-model="form.description"
        class="textarea textarea-bordered"
        rows="3"
        placeholder="Optional description"
      />
    </div>

    <!-- Type & Priority -->
    <div class="grid grid-cols-2 gap-3 mb-3">
      <div class="form-control">
        <label class="label"><span class="label-text">Type</span></label>
        <select v-model="form.type" class="select select-bordered">
          <option value="task">Task</option>
          <option value="story">Story</option>
          <option value="bug">Bug</option>
          <option value="epic">Epic</option>
        </select>
      </div>
      <div class="form-control">
        <label class="label"><span class="label-text">Priority</span></label>
        <select v-model="form.priority" class="select select-bordered">
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="critical">Critical</option>
        </select>
      </div>
    </div>

    <!-- Story Points -->
    <div class="form-control mb-3">
      <label class="label"><span class="label-text">Story Points</span></label>
      <input
        v-model.number="form.story_points"
        type="number"
        min="0"
        max="100"
        class="input input-bordered"
        placeholder="Leave blank if unestimated"
      />
    </div>

    <!-- Assignee -->
    <div class="form-control mb-4">
      <label class="label"><span class="label-text">Assignee</span></label>
      <select v-model="form.assignee_id" class="select select-bordered">
        <option :value="undefined">Unassigned</option>
        <option v-for="m in members" :key="m.user_id" :value="m.user_id">{{ m.name }}</option>
      </select>
    </div>

    <div class="flex gap-2 justify-end">
      <button type="button" class="btn btn-ghost" @click="$emit('cancelled')">Cancel</button>
      <button data-testid="submit" type="submit" class="btn btn-primary" :disabled="saving">
        <span v-if="saving" class="loading loading-spinner loading-sm" />
        {{ issue ? 'Save' : 'Create' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { projectsService } from '~/services/projects.service'
import type { Issue } from '~/types/domain.types'
import type { MemberResponse } from '~/types/domain.types'
import type { CreateIssueRequest } from '~/services/issues.service'

const props = defineProps<{
  projectKey: string
  issue?: Issue
  defaultStatus?: string
}>()

const emit = defineEmits<{
  saved: [issue: Issue]
  cancelled: []
}>()

const issuesStore = useIssuesStore()

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
  titleError.value = !form.title.trim()
  if (titleError.value) return

  saving.value = true
  try {
    let result: Issue
    if (props.issue) {
      result = await issuesStore.update(props.projectKey, props.issue.id, form)
    }
    else {
      result = await issuesStore.create(props.projectKey, form)
    }
    emit('saved', result)
  }
  finally {
    saving.value = false
  }
}
</script>
