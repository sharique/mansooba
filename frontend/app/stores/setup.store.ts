import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth.store'
import { setupService } from '~/services/setup.service'
import type { SetupAdminRequest, SetupUserRequest, SetupProjectRequest } from '~/services/setup.service'

interface SetupState {
  setupRequired: boolean | null
  currentStep: number
  createdUser: { id: number; name: string; email: string } | null
  createdProject: { id: number; key: string; name: string } | null
}

export const useSetupStore = defineStore('setup', {
  state: (): SetupState => ({
    setupRequired: null,
    currentStep: 0,
    createdUser: null,
    createdProject: null,
  }),

  getters: {
    isSetupRequired: (state) => state.setupRequired === true,
    hasCreatedUser: (state) => state.createdUser !== null,
    summaryItems: (state): { label: string; value: string }[] => {
      return [
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

    finish(): void {
      this.setupRequired = false
      this.currentStep = 0
      this.createdUser = null
      this.createdProject = null
      navigateTo('/')
    },
  },
})
