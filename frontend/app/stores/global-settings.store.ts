import { defineStore } from 'pinia'
import type { PatchSettingsRequest } from '~/types/domain.types'
import { settingsService } from '~/services/settings.service'

export const useGlobalSettingsStore = defineStore('global-settings', {
  state: () => ({
    organization_name: 'Mansooba',
    date_format: 'YYYY-MM-DD',
    time_format: '24h',
    locale: 'en-US',
    week_start_day: 'monday',
    loaded: false,
  }),
  actions: {
    async fetch() {
      const data = await settingsService.getSettings()
      this.organization_name = data.organization_name
      this.date_format = data.date_format
      this.time_format = data.time_format
      this.locale = data.locale
      this.week_start_day = data.week_start_day
      this.loaded = true
    },
    async patch(payload: PatchSettingsRequest) {
      const data = await settingsService.patchSettings(payload)
      this.organization_name = data.organization_name
      this.date_format = data.date_format
      this.time_format = data.time_format
      this.locale = data.locale
      this.week_start_day = data.week_start_day
    },
  },
})
