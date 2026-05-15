<template>
  <form @submit.prevent="submit">
    <div class="form-control w-full">
      <label class="label"><span class="label-text">Name <span class="text-error">*</span></span></label>
      <input
        v-model="form.name"
        type="text"
        class="input input-bordered w-full"
        placeholder="My Project"
        required
        @input="autoKey"
      />
    </div>

    <div class="form-control w-full mt-3">
      <label class="label"><span class="label-text">Key</span></label>
      <input
        v-model="form.key"
        type="text"
        class="input input-bordered w-full font-mono uppercase"
        placeholder="PROJ"
        maxlength="10"
      />
      <label class="label"><span class="label-text-alt text-base-content/60">Auto-generated from name, max 10 chars</span></label>
    </div>

    <div class="form-control w-full mt-3">
      <label class="label"><span class="label-text">Description</span></label>
      <textarea
        v-model="form.description"
        class="textarea textarea-bordered w-full"
        placeholder="What is this project about?"
        rows="3"
      />
    </div>

    <div v-if="errorMessage" class="alert alert-error mt-4 py-2 text-sm">
      {{ errorMessage }}
    </div>

    <div class="modal-action">
      <button type="submit" class="btn btn-primary" :disabled="loading">
        <span v-if="loading" class="loading loading-spinner loading-sm" />
        {{ project ? 'Save changes' : 'Create project' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import type { Project } from '~/types/domain.types'
import { useProjectsStore } from '~/stores/projects.store'
import { projectsService } from '~/services/projects.service'

const props = defineProps<{ project?: Project }>()
const emit = defineEmits<{ saved: [project: Project] }>()

const projectsStore = useProjectsStore()
const loading = ref(false)
const errorMessage = ref('')

const form = reactive({
  name: props.project?.name ?? '',
  key: props.project?.key ?? '',
  description: props.project?.description ?? '',
})

function autoKey() {
  if (props.project) return
  form.key = form.name.replace(/[^A-Za-z]/g, '').toUpperCase().slice(0, 10)
}

async function submit() {
  loading.value = true
  errorMessage.value = ''
  try {
    const saved = props.project
      ? await projectsStore.update(props.project.key, form)
      : await projectsStore.create(form)
    emit('saved', saved)
  }
  catch (err: unknown) {
    errorMessage.value = (err as { data?: { message?: string } })?.data?.message ?? 'Failed to save project'
  }
  finally {
    loading.value = false
  }
}
</script>
