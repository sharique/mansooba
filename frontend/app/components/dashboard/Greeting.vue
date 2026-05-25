<template>
  <div>
    <h1 class="text-3xl font-bold">{{ greeting }}, {{ displayName }} 👋</h1>
    <p class="text-base-content/60 mt-1">Here's what's happening across your projects.</p>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'

const authStore = useAuthStore()

const displayName = computed(() =>
  authStore.profile?.name || authStore.user?.name || 'there',
)

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
})
</script>
