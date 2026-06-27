<template>
  <div class="max-w-xl mx-auto py-8 px-4">
    <h1 class="text-2xl font-bold mb-6">Global Settings</h1>
    <SettingsGlobalSettingsForm />
    <div v-if="authStore.isAdmin" class="mt-8 border-t pt-6">
      <h2 class="text-lg font-semibold mb-3">Administration</h2>
      <NuxtLink to="/system/users" class="btn btn-outline btn-sm">
        User Management
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import { useGlobalSettingsStore } from '~/stores/global-settings.store'

const authStore = useAuthStore()
const settingsStore = useGlobalSettingsStore()

onMounted(async () => {
  if (!authStore.isAdmin) {
    await navigateTo('/settings')
    return
  }
  if (!settingsStore.loaded) {
    await settingsStore.fetch()
  }
})
</script>
