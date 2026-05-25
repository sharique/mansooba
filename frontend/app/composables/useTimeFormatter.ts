import { useAuthStore } from '~/stores/auth.store'

export function useTimeFormatter() {
  const authStore = useAuthStore()

  function formatDateTime(iso: string): string {
    const tz = authStore.profile?.timezone || undefined
    return new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      timeZone: tz,
    }).format(new Date(iso))
  }

  return { formatDateTime }
}
