<template>
  <div class="card bg-base-100 shadow hover:shadow-md transition-shadow cursor-pointer">
    <div class="card-body">
      <div class="flex items-center gap-2">
        <span class="badge badge-primary">{{ project.key }}</span>
        <h2 class="card-title text-base">{{ project.name }}</h2>
      </div>
      <p class="text-sm text-base-content/70 line-clamp-2">{{ project.description || '—' }}</p>
      <div class="card-actions justify-end mt-2">
        <NuxtLink :to="`/projects/${project.key}`" class="btn btn-sm btn-ghost">Details</NuxtLink>
        <NuxtLink :to="`/projects/${project.key}/board`" class="btn btn-sm btn-primary">Open Board</NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Project } from '~/types/domain.types'

defineProps<{ project: Project }>()
</script>
