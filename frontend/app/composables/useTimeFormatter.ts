import { useAuthStore } from '~/stores/auth.store'
import { useGlobalSettingsStore } from '~/stores/global-settings.store'

export function useTimeFormatter() {
  const authStore = useAuthStore()
  const settingsStore = useGlobalSettingsStore()

  function dateParts(d: Date, tz: string | undefined) {
    const numericParts = new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      timeZone: tz,
    }).formatToParts(d)

    const monthShortParts = new Intl.DateTimeFormat('en-US', {
      month: 'short',
      timeZone: tz,
    }).formatToParts(d)

    return {
      year: numericParts.find(p => p.type === 'year')?.value ?? '',
      month: numericParts.find(p => p.type === 'month')?.value ?? '',
      day: numericParts.find(p => p.type === 'day')?.value ?? '',
      monthShort: monthShortParts.find(p => p.type === 'month')?.value ?? '',
    }
  }

  function applyFormat(fmt: string, year: string, month: string, day: string, monthShort: string): string {
    const dayNopad = String(Number.parseInt(day, 10))
    return fmt
      .replace('YYYY', year)
      .replace('MMM', monthShort)   // before MM so "MMM" isn't partially consumed
      .replace('MM', month)
      .replace('DD', day)           // before D so "DD" isn't partially consumed
      .replace(/\bD\b/, dayNopad)   // standalone D only (after DD is already replaced)
  }

  function formatDate(iso: string | null | undefined): string {
    if (!iso) return '—'
    const tz = authStore.profile?.timezone || undefined
    const { year, month, day, monthShort } = dateParts(new Date(iso), tz)
    return applyFormat(settingsStore.date_format, year, month, day, monthShort)
  }

  function formatDateTime(iso: string): string {
    const tz = authStore.profile?.timezone || undefined
    const d = new Date(iso)
    const { year, month, day, monthShort } = dateParts(d, tz)
    const datePart = applyFormat(settingsStore.date_format, year, month, day, monthShort)
    const timePart = new Intl.DateTimeFormat('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: settingsStore.time_format === '12h',
      timeZone: tz,
    }).format(d)
    return `${datePart} ${timePart}`
  }

  return { formatDate, formatDateTime }
}
