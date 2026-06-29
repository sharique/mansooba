<template>
  <header class="h-14 bg-base-100 border-b border-base-300 flex items-center gap-3 px-4 shrink-0">
    <!-- mobile drawer toggle -->
    <label for="app-drawer" class="btn btn-ghost btn-sm btn-square lg:hidden">
      <Icon name="mdi:menu" class="w-5 h-5" />
    </label>

    <!-- global search -->
    <form class="flex-1 max-w-md" @submit.prevent="search">
      <label class="input input-sm input-bordered flex items-center gap-2 w-full">
        <Icon name="mdi:magnify" class="w-4 h-4 opacity-50" />
        <input v-model="q" type="search" class="grow" placeholder="Search issues…">
      </label>
    </form>

    <div class="flex-1" />

    <LayoutNotificationBell />
    <LayoutThemeToggle />

    <!-- user menu -->
    <div class="dropdown dropdown-end">
      <button tabindex="0" class="btn btn-ghost btn-sm btn-circle">
        <UserAvatar
          :avatarUrl="authStore.profile?.avatar_url || undefined"
          :name="displayName"
          :userId="authStore.profile?.id || 0"
          size="sm"
        />
      </button>
      <ul tabindex="0" class="dropdown-content menu bg-base-100 border border-base-300 rounded-box z-20 w-44 p-1 shadow">
        <li class="menu-title text-xs">{{ displayName }}</li>
        <li><NuxtLink to="/settings">Settings</NuxtLink></li>
        <li><button @click="logout">Logout</button></li>
      </ul>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import { authService } from '~/services/auth.service'
import UserAvatar from '~/components/common/UserAvatar.vue'

const authStore = useAuthStore()
const router = useRouter()

const q = ref('')

const displayName = computed(() => authStore.profile?.name || authStore.user?.name || 'Account')

function search() {
  const term = q.value.trim()
  if (term) router.push({ path: '/projects', query: { q: term } })
}

async function logout() {
  await authService.logout()
}
</script>
