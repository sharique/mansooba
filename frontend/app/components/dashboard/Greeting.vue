<template>
  <div class="flex flex-wrap items-end justify-between gap-2">
    <div>
      <h1 class="text-2xl font-bold tracking-tight">{{ greeting }}, {{ displayName }} 👋</h1>
      <p class="text-base-content/60 mt-1 text-sm">Here's what's on your desk today.</p>
    </div>
    <span class="text-sm text-base-content/40">{{ today }}</span>
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

const today = computed(() =>
  new Date().toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' }),
)
</script>
