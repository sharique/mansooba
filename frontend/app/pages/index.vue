<template>
  <div class="space-y-8">
    <!-- Greeting -->
    <DashboardGreeting />

    <!-- My Tasks -->
    <DashboardMyTasksWidget :loading="loading" />

    <!-- Recent Activity -->
    <DashboardActivityWidget :loading="loading" />
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
    showError('Failed to load dashboard')
  }
  finally {
    loading.value = false
  }
})
</script>
