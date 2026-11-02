<template>
  <div class="max-w-2xl mx-auto py-8 px-4">
    <div class="mb-6 flex items-center gap-4">
      <NuxtLink :to="`/projects/${key}/board`" class="btn btn-sm btn-ghost">← Sprint board</NuxtLink>
      <h1 class="text-xl font-bold">Project Settings</h1>
    </div>
    <LabelsLabelManager :project-key="key" />

    <section v-if="project && can('project.delete')" class="mt-10 border-t border-base-300 pt-6" aria-labelledby="danger-title">
      <h2 id="danger-title" class="text-lg font-semibold text-error mb-2">Delete this project</h2>
      <p class="text-sm text-base-content/70 mb-3">Remove all tasks first. Then the project and its people are removed for good.</p>
      <ProjectsDeleteProjectDialog :project-key="key" :project-name="project.name" />
    </section>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const key = route.params.key as string
const projectsStore = useProjectsStore()
const { can } = useProjectPermissions()
const project = computed(() => {
  const p = projectsStore.current
  return p && p.key.toLowerCase() === key.toLowerCase() ? p : null
})

onMounted(() => {
  // The project carries what the signed-in user may do here.
  projectsStore.fetchOne(key).catch(() => {})
})
</script>
