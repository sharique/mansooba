import { defineStore } from 'pinia'

export type ThemeName = 'mansooba' | 'mansooba-dark'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    // null = follow system preference; otherwise an explicit choice
    selected: null as ThemeName | null,
  }),
  actions: {
    setTheme(theme: ThemeName) {
      this.selected = theme
    },
    // `current` is the effective theme right now (used when no explicit choice yet)
    toggle(current: ThemeName) {
      const base = this.selected ?? current
      this.selected = base === 'mansooba-dark' ? 'mansooba' : 'mansooba-dark'
    },
  },
  persist: true,
})
