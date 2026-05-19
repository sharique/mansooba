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

  return { unread, unreadCount, error, fetchUnread, markRead }
})
