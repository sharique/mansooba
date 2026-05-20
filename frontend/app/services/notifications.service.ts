import type { Notification } from '~/types/domain.types'

export const notificationsService = {
  listUnread(): Promise<Notification[]> {
    const { $api } = useNuxtApp()
    return $api<Notification[]>('/notifications')
  },

  markRead(id: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/notifications/${id}/read`, { method: 'PUT' })
  },
}
