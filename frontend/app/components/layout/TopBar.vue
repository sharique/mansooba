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

    <!-- role-aware create control -->
    <div v-if="showCreateProject && showCreateTask" class="dropdown dropdown-end" data-testid="create-dropdown">
      <button tabindex="0" class="btn btn-primary btn-sm gap-1">
        <Icon name="mdi:plus" class="w-4 h-4" /> Create
      </button>
      <ul tabindex="0" class="dropdown-content menu bg-base-100 border border-base-300 rounded-box z-20 w-44 p-1 shadow">
        <li><button @click="onCreateProject">Create project</button></li>
        <li><button @click="openCreateIssue">Create task</button></li>
      </ul>
    </div>
    <button
      v-else-if="showCreateTask"
      class="btn btn-primary btn-sm gap-1"
      data-testid="create-task-btn"
      @click="openCreateIssue"
    >
      <Icon name="mdi:plus" class="w-4 h-4" /> Create task
    </button>

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

    <!-- create issue modal (global, no project context) -->
    <IssuesCreateIssueModal
      v-if="createIssueOpen"
      :open="createIssueOpen"
      @close="createIssueOpen = false"
    />
  </header>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import { useProjectsStore } from '~/stores/projects.store'
import UserAvatar from '~/components/common/UserAvatar.vue'

const authStore = useAuthStore()
const projectsStore = useProjectsStore()
const router = useRouter()
const triggerCreateProject = inject<() => void>('triggerCreateProject')

const q = ref('')
const createIssueOpen = ref(false)

const displayName = computed(() => authStore.profile?.name || authStore.user?.name || 'Account')
const showCreateProject = computed(() => authStore.isAdmin)
const showCreateTask = computed(() => projectsStore.projects.length > 0)

function search() {
  const term = q.value.trim()
  if (term) router.push({ path: '/projects', query: { q: term } })
}

function onCreateProject() {
  triggerCreateProject?.()
}

function openCreateIssue() {
  createIssueOpen.value = true
}

async function logout() {
  authStore.clearAuth()
  await router.push('/login')
}
</script>
