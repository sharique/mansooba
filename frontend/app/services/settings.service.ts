import type { SettingsResponse, PatchSettingsRequest } from '~/types/domain.types'

export const settingsService = {
  getSettings(): Promise<SettingsResponse> {
    const { $api } = useNuxtApp()
    return $api<SettingsResponse>('/settings')
  },

  patchSettings(payload: PatchSettingsRequest): Promise<SettingsResponse> {
    const { $api } = useNuxtApp()
    return $api<SettingsResponse>('/settings', { method: 'PATCH', body: payload })
  },
}
