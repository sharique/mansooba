import { defineStore } from 'pinia'
import { aboutService } from '~/services/about.service'

// This instance's version, source location and licence, shown to every
// user. fetch() never throws — the notice must not break the page it's
// shown on if the request fails.
export const useAboutStore = defineStore('about', {
  state: () => ({
    version: '',
    sourceUrl: 'https://github.com/sharique/mansooba',
    license: 'AGPL-3.0-only',
    licenseUrl: 'https://www.gnu.org/licenses/agpl-3.0.html',
  }),
  actions: {
    async fetch() {
      try {
        const data = await aboutService.getAbout()
        this.version = data.version
        this.sourceUrl = data.source_url
        this.license = data.license
        this.licenseUrl = data.license_url
      } catch {
        // Keep the defaults above; the notice still shows a working link.
      }
    },
  },
})
