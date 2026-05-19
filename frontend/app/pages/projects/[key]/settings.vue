<template>
  <div class="max-w-2xl mx-auto py-8 px-4">
    <div class="mb-6 flex items-center gap-4">
      <NuxtLink :to="`/projects/${key}/board`" class="btn btn-sm btn-ghost">← Board</NuxtLink>
      <h1 class="text-xl font-bold">Project Settings</h1>
    </div>
    <LabelManager :project-key="key" />
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const key = route.params.key as string
</script>
