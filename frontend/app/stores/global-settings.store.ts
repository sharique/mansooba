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
    system_log_retention_days: '90',
    demo_banner_enabled: 'false',
    demo_banner_message: 'This is a demo instance. Data may be reset at any time.',
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
      this.system_log_retention_days = data.system_log_retention_days
      this.demo_banner_enabled = data.demo_banner_enabled
      this.demo_banner_message = data.demo_banner_message
      this.loaded = true
    },
    async patch(payload: PatchSettingsRequest) {
      const data = await settingsService.patchSettings(payload)
      this.organization_name = data.organization_name
      this.date_format = data.date_format
      this.time_format = data.time_format
      this.locale = data.locale
      this.week_start_day = data.week_start_day
      this.system_log_retention_days = data.system_log_retention_days
      this.demo_banner_enabled = data.demo_banner_enabled
      this.demo_banner_message = data.demo_banner_message
    },
  },
})
