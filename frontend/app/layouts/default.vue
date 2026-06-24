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

    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { useNotificationsStore } from '~/stores/notifications.store'
import { useGlobalSettingsStore } from '~/stores/global-settings.store'

const notifStore = useNotificationsStore()
const globalSettingsStore = useGlobalSettingsStore()

onMounted(async () => {
  if (!globalSettingsStore.loaded) {
    await globalSettingsStore.fetch()
  }
  notifStore.fetchUnread()
  const interval = setInterval(() => notifStore.fetchUnread(), 30_000)
  onUnmounted(() => clearInterval(interval))
})
</script>
