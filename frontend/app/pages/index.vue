<template>
  <div class="space-y-6">
    <DashboardGreeting />

    <div class="grid gap-6 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <DashboardMyTasksWidget :loading="loading" />
      </div>
      <div class="lg:col-span-1">
        <DashboardActivityWidget :loading="loading" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'

const authStore = useAuthStore()
const { showError } = useToast()

const loading = ref(true)

onMounted(async () => {
  try {
    await Promise.all([
      authStore.fetchMyIssues(),
      authStore.fetchMyActivity(10, 0),
    ])
  }
  catch {
    showError('Failed to load your desk')
  }
  finally {
    loading.value = false
  }
})
</script>
