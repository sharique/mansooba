<template>
  <div>
    <!-- Breadcrumb -->
    <div class="breadcrumbs text-sm mb-4">
      <ul>
        <li><NuxtLink to="/projects">Projects</NuxtLink></li>
        <li>{{ key }}</li>
      </ul>
    </div>

    <div v-if="loading" class="space-y-4">
      <div class="skeleton h-8 w-64" />
      <div class="skeleton h-4 w-96" />
    </div>

    <template v-else-if="projectsStore.current">
      <div class="flex items-start justify-between mb-6">
        <div>
          <h1 class="text-2xl font-bold">{{ projectsStore.current.name }}</h1>
          <p class="text-base-content/60 mt-1">{{ projectsStore.current.description }}</p>
        </div>
        <button class="btn btn-ghost btn-sm" @click="editModal?.showModal()">Edit</button>
      </div>

      <ProjectsMemberList :project-key="key" :owner-id="projectsStore.current.ownerId" />
    </template>

    <!-- Edit modal -->
    <dialog ref="editModal" class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Edit Project</h3>
        <ProjectsProjectForm :project="projectsStore.current ?? undefined" @saved="onSaved" @cancel="editModal?.close()" />
      </div>
      <form method="dialog" class="modal-backdrop"><button>close</button></form>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { useProjectsStore } from '~/stores/projects.store'
import type { Project } from '~/types/domain.types'

const route = useRoute()
const key = route.params.key as string
const projectsStore = useProjectsStore()
const { showSuccess, showError } = useToast()
const editModal = ref<HTMLDialogElement | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    await projectsStore.fetchOne(key)
  }
  catch {
    showError('Failed to load project')
  }
  finally {
    loading.value = false
  }
})

function onSaved(project: Project) {
  editModal.value?.close()
  showSuccess(`Project "${project.name}" updated`)
}
</script>
