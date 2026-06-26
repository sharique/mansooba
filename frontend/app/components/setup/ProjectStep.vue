<template>
  <div class="flex flex-col gap-4">
    <h2 class="text-2xl font-bold">Create a project</h2>
    <p class="text-base-content/70 text-sm">Optional — you can create projects later from the dashboard.</p>

    <div class="form-control gap-1">
      <label for="project-name" class="label label-text font-medium">Project name</label>
      <input
        id="project-name"
        v-model="form.name"
        type="text"
        class="input input-bordered w-full"
        :class="{ 'input-error': errors.name }"
        :disabled="loading"
      />
      <span v-if="errors.name" role="alert" class="text-error text-sm">{{ errors.name }}</span>
    </div>

    <div class="form-control gap-1">
      <label for="project-description" class="label label-text font-medium">
        Description <span class="text-base-content/50 font-normal">(optional)</span>
      </label>
      <textarea
        id="project-description"
        v-model="form.description"
        class="textarea textarea-bordered w-full"
        :disabled="loading"
      />
    </div>

    <div v-if="setupStore.hasCreatedUser" class="bg-base-200 rounded-lg p-4 space-y-3">
      <p class="font-medium text-sm">
        Should <strong>{{ setupStore.createdUser?.name }}</strong> be added to this project?
      </p>
      <div class="flex gap-3">
        <button
          class="btn btn-sm"
          :class="addUser === true ? 'btn-primary' : 'btn-outline'"
          :aria-label="`Add ${setupStore.createdUser?.name} to this project`"
          @click="addUser = true"
        >
          Yes
        </button>
        <button
          class="btn btn-sm"
          :class="addUser === false ? 'btn-primary' : 'btn-outline'"
          :aria-label="`Do not add ${setupStore.createdUser?.name} to this project`"
          @click="addUser = false"
        >
          No
        </button>
      </div>
    </div>

    <span v-if="errors.server" role="alert" class="text-error text-sm">{{ errors.server }}</span>

    <div class="flex items-center justify-between mt-2">
      <button
        class="btn btn-ghost text-sm font-normal"
        :disabled="loading"
        @click="skip"
      >
        Skip for now
      </button>
      <button
        class="btn btn-primary"
        :disabled="loading || !form.name.trim()"
        @click="submit"
      >
        <span v-if="loading" class="loading loading-spinner loading-sm" />
        Create project
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useSetupStore } from '~/stores/setup.store'

const setupStore = useSetupStore()
const form = reactive({ name: '', description: '' })
const errors = reactive({ name: '', server: '' })
const loading = ref(false)
const addUser = ref<boolean | null>(null)

function skip() {
  setupStore.skipProject()
}

async function submit() {
  errors.server = ''
  if (!form.name.trim()) { errors.name = 'Project name is required.'; return }
  errors.name = ''

  loading.value = true
  try {
    await setupStore.completeProject(
      { name: form.name, description: form.description || undefined },
      addUser.value === true,
    )
  } catch (err: any) {
    if (err?.statusCode === 404) {
      errors.server = 'The team member could not be found. Please try without adding a member.'
    } else {
      errors.server = 'Something went wrong. Please try again.'
    }
  } finally {
    loading.value = false
  }
}
</script>
