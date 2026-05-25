import { useThemeStore, type ThemeName } from '~/stores/theme.store'

export function useTheme() {
  const store = useThemeStore()

  function systemTheme(): ThemeName {
    if (import.meta.client && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'mansooba-dark'
    }
    return 'mansooba'
  }

  // The theme actually in effect right now.
  const effective = computed<ThemeName>(() => store.selected ?? systemTheme())

  // Keep <html data-theme="..."> in sync whenever effective changes
  // (e.g., when store.setTheme() is called directly, not just via toggle())
  if (import.meta.client) {
    watchEffect(() => {
      document.documentElement.setAttribute('data-theme', effective.value)
    })
  }

  function apply() {
    if (import.meta.client) {
      document.documentElement.setAttribute('data-theme', effective.value)
    }
  }

  function toggle() {
    store.toggle(effective.value)
    apply()
  }

  return { effective, apply, toggle }
}
