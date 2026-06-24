<template>
  <div class="dropdown dropdown-end">
    <button tabindex="0" class="btn btn-ghost btn-circle relative">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
      </svg>
      <span v-if="store.unreadCount > 0"
        class="badge badge-primary badge-xs absolute top-1 right-1">
        {{ store.unreadCount > 9 ? '9+' : store.unreadCount }}
      </span>
    </button>
    <ul tabindex="0" class="dropdown-content menu bg-base-100 border border-base-200 rounded-box z-[20] shadow w-72 p-1 max-h-80 overflow-y-auto">
      <li v-if="store.unread.length === 0" class="pointer-events-none">
        <span class="text-base-content/40 text-sm">No new notifications</span>
      </li>
      <li v-for="n in store.unread" :key="n.id">
        <button class="text-left text-sm py-2" @click="open(n)">
          <span>You were mentioned in {{ n.issue_key }}</span>
          <span class="block text-xs text-base-content/40">{{ formatDate(n.created_at) }}</span>
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { useNotificationsStore } from '~/stores/notifications.store'
import type { Notification } from '~/types/domain.types'

const store = useNotificationsStore()
const router = useRouter()

async function open(n: Notification) {
  await store.markRead(n.id)
  router.push(`/projects/${n.project_key}/issues/${n.issue_id}`)
}

const { formatDate } = useTimeFormatter()
</script>
