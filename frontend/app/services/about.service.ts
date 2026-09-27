export interface AboutResponse {
  version: string
  source_url: string
  license: string
  license_url: string
}

export const aboutService = {
  getAbout(): Promise<AboutResponse> {
    const { $api } = useNuxtApp()
    return $api<AboutResponse>('/about')
  },
}
