<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold">Projects</h1>
      <button class="btn btn-primary" @click="createModal?.showModal()">New Project</button>
    </div>

    <!-- Skeleton -->
    <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="i in 6" :key="i" class="skeleton h-40 w-full rounded-xl" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!projectsStore.projects.length" class="text-center py-20 text-base-content/50">
      <p class="text-lg">No projects yet.</p>
      <p class="text-sm mt-1">Click "New Project" to create your first one.</p>
    </div>

    <!-- Grid -->
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <ProjectsProjectCard
        v-for="project in projectsStore.projects"
        :key="project.id"
        :project="project"
      />
    </div>

    <!-- Create modal -->
    <dialog ref="createModal" class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">New Project</h3>
        <ProjectsProjectForm @saved="onCreated" />
      </div>
      <form method="dialog" class="modal-backdrop"><button>close</button></form>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { useProjectsStore } from '~/stores/projects.store'
import type { Project } from '~/types/domain.types'

const projectsStore = useProjectsStore()
const { showSuccess, showError } = useToast()
const createModal = ref<HTMLDialogElement | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    await projectsStore.fetchAll()
  }
  catch {
    showError('Failed to load projects')
  }
  finally {
    loading.value = false
  }
})

function onCreated(project: Project) {
  createModal.value?.close()
  showSuccess(`Project "${project.name}" created`)
}
</script>
