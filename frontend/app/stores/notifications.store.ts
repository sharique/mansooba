import { defineStore } from 'pinia'
import { notificationsService } from '~/services/notifications.service'
import type { Notification } from '~/types/domain.types'

export const useNotificationsStore = defineStore('notifications', () => {
  const unread = ref<Notification[]>([])
  const error = ref<string | null>(null)

  const unreadCount = computed(() => unread.value.length)

  async function fetchUnread() {
    try {
      unread.value = await notificationsService.listUnread()
    } catch (e: any) {
      error.value = e.data?.message ?? e.message
    }
  }

  async function markRead(id: number) {
    await notificationsService.markRead(id)
    unread.value = unread.value.filter(n => n.id !== id)
  }

  // Pinia's setup-syntax stores don't get an auto-generated $reset() (that's
  // an options-store-only feature) — calling .$reset() on a setup store
  // throws. resetDomainStores() (auth.service.ts) calls $reset() uniformly
  // across stores on logout, so this store defines its own.
  function $reset() {
    unread.value = []
    error.value = null
  }

  return { unread, unreadCount, error, fetchUnread, markRead, $reset }
})
