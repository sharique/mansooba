import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth.store'
import { setupService } from '~/services/setup.service'
import type { SetupAdminRequest, SetupUserRequest, SetupProjectRequest } from '~/services/setup.service'

interface SetupState {
  setupRequired: boolean | null
  currentStep: number
  createdUser: { id: number; name: string; email: string } | null
  createdProject: { id: number; key: string; name: string } | null
  seedImported: boolean | null
  seedProjectKey: string | null
}

export const useSetupStore = defineStore('setup', {
  state: (): SetupState => ({
    setupRequired: null,
    currentStep: 0,
    createdUser: null,
    createdProject: null,
    seedImported: null,
    seedProjectKey: null,
  }),

  getters: {
    isSetupRequired: (state) => state.setupRequired === true,
    hasCreatedUser: (state) => state.createdUser !== null,
    summaryItems: (state): { label: string; value: string }[] => {
      const items: { label: string; value: string }[] = [
        {
          label: 'Team member',
          value: state.createdUser
            ? `${state.createdUser.name} (${state.createdUser.email})`
            : 'No team member added',
        },
        {
          label: 'Project',
          value: state.createdProject
            ? `${state.createdProject.name} [${state.createdProject.key}]`
            : 'No project added',
        },
      ]
      if (state.seedImported !== null) {
        items.push({
          label: 'Sample data',
          value: state.seedImported
            ? `Imported (project: ${state.seedProjectKey})`
            : 'Skipped',
        })
      }
      return items
    },
  },

  actions: {
    async checkSetupStatus(): Promise<boolean> {
      if (this.setupRequired !== null) {
        return this.setupRequired
      }
      const data = await setupService.getStatus()
      this.setupRequired = data.setup_required
      return this.setupRequired
    },

    async completeAdmin(req: SetupAdminRequest): Promise<void> {
      const data = await setupService.createAdmin(req)
      useAuthStore().setAuth(data.user, data.access_token)
      this.currentStep = 2
    },

    async completeUser(req: SetupUserRequest): Promise<void> {
      const data = await setupService.createUser(req)
      this.createdUser = { id: data.user_id, name: data.name, email: data.email }
      this.currentStep = 3
    },

    skipUser(): void {
      this.createdUser = null
      this.currentStep = 3
    },

    async completeProject(req: SetupProjectRequest, addUser: boolean): Promise<void> {
      const payload: SetupProjectRequest = {
        name: req.name,
        description: req.description,
        add_user_id: addUser && this.createdUser ? this.createdUser.id : 0,
      }
      const data = await setupService.createProject(payload)
      this.createdProject = { id: data.project_id, key: data.project_key, name: data.name }
      this.currentStep = 4
    },

    skipProject(): void {
      this.createdProject = null
      this.currentStep = 4
    },

    async completeSampleData(): Promise<void> {
      const data = await setupService.seedData()
      this.seedImported = !data.skipped
      this.seedProjectKey = data.project_key
      this.currentStep = 5
    },

    skipSampleData(): void {
      this.seedImported = false
      this.seedProjectKey = null
      this.currentStep = 5
    },

    finish(): void {
      this.setupRequired = false
      this.currentStep = 0
      this.createdUser = null
      this.createdProject = null
      this.seedImported = null
      this.seedProjectKey = null
      navigateTo('/')
    },
  },
})
