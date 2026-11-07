<template>
  <div class="max-w-xl mx-auto p-6 mt-16 text-center space-y-4">
    <Icon name="lucide:lock" class="text-5xl text-base-content/40" aria-hidden="true" />
    <h1 class="text-2xl font-bold">You don't have access to {{ project }} project.</h1>
    <p class="text-base-content/70">
      Either you are not a member, or it does not exist. Ask a project admin to add you.
    </p>
    <NuxtLink to="/projects" class="btn btn-primary">Back to my projects</NuxtLink>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const project = computed(() => {
  const p = route.query.project
  return (Array.isArray(p) ? p[0] : p) || 'this'
})
</script>
