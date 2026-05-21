<template>
  <div class="min-h-screen bg-base-100">
    <div class="navbar bg-base-200 shadow-sm">
      <div class="navbar-start">
        <NuxtLink to="/projects" class="btn btn-ghost text-xl">jira-go</NuxtLink>
      </div>
      <div class="navbar-end">
        <NuxtLink to="/settings" class="mr-4 text-sm hover:underline">{{ authStore.user?.name }}</NuxtLink>
        <LayoutNotificationBell class="mr-2" />
        <button class="btn btn-ghost btn-sm" @click="logout">Logout</button>
      </div>
    </div>
    <!-- Project sub-nav shown when on a /projects/:key route -->
    <div v-if="currentProjectKey" class="bg-base-200 border-t border-base-300 px-6 py-1 flex gap-2">
      <NuxtLink :to="`/projects/${currentProjectKey}`" class="btn btn-ghost btn-sm">Overview</NuxtLink>
      <NuxtLink :to="`/projects/${currentProjectKey}/board`" class="btn btn-ghost btn-sm">Board</NuxtLink>
      <NuxtLink :to="`/projects/${currentProjectKey}/backlog`" class="btn btn-ghost btn-sm">Backlog</NuxtLink>
      <NuxtLink :to="`/projects/${currentProjectKey}/reports`" class="btn btn-ghost btn-sm">Reports</NuxtLink>
    </div>
    <main class="container mx-auto p-6">
      <slot />
    </main>
    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import { useNotificationsStore } from '~/stores/notifications.store'

const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()
const notifStore = useNotificationsStore()

const currentProjectKey = computed(() => {
  const key = route.params.key
  return typeof key === 'string' ? key : null
})

async function logout() {
  authStore.clearAuth()
  await router.push('/login')
}

onMounted(() => {
  notifStore.fetchUnread()
  const interval = setInterval(() => notifStore.fetchUnread(), 30_000)
  onUnmounted(() => clearInterval(interval))
})
</script>
