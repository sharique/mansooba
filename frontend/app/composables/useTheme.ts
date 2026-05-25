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
