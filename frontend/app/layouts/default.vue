<template>
  <div class="drawer lg:drawer-open min-h-screen bg-base-200">
    <input id="app-drawer" type="checkbox" class="drawer-toggle">

    <div class="drawer-content flex flex-col min-h-screen">
      <LayoutTopBar />
      <main class="flex-1 p-4 lg:p-6 max-w-7xl w-full mx-auto">
        <slot />
      </main>
    </div>

    <div class="drawer-side z-30">
      <label for="app-drawer" aria-label="close sidebar" class="drawer-overlay" />
      <LayoutSidebar />
    </div>

    <!-- Global project creation modal, triggered by TopBar via provide/inject -->
    <dialog ref="createProjectModal" class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">New Project</h3>
        <ProjectsProjectForm @saved="onProjectCreated" @cancel="createProjectModal?.close()" />
      </div>
      <form method="dialog" class="modal-backdrop"><button>close</button></form>
    </dialog>

    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { useNotificationsStore } from '~/stores/notifications.store'
import type { Project } from '~/types/domain.types'

const notifStore = useNotificationsStore()
const { showSuccess } = useToast()
const createProjectModal = ref<HTMLDialogElement | null>(null)

function triggerCreateProject() {
  createProjectModal.value?.showModal()
}

function onProjectCreated(project: Project) {
  createProjectModal.value?.close()
  showSuccess(`Project "${project.name}" created`)
}

provide('triggerCreateProject', triggerCreateProject)

onMounted(() => {
  notifStore.fetchUnread()
  const interval = setInterval(() => notifStore.fetchUnread(), 30_000)
  onUnmounted(() => clearInterval(interval))
})
</script>
