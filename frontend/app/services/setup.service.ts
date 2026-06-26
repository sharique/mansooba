import type { AuthResponse } from '~/types/auth.types'

export interface SetupStatusResponse {
  setup_required: boolean
}

export interface SetupAdminRequest {
  full_name: string
  email: string
  password: string
}

export interface SetupUserRequest {
  full_name: string
  email: string
  password: string
}

export interface SetupUserResponse {
  user_id: number
  name: string
  email: string
}

export interface SetupProjectRequest {
  name: string
  description?: string
  add_user_id?: number
}

export interface SetupProjectResponse {
  project_id: number
  project_key: string
  name: string
}

export const setupService = {
  async getStatus(): Promise<SetupStatusResponse> {
    const { $api } = useNuxtApp()
    return $api<SetupStatusResponse>('/setup/status')
  },

  async createAdmin(req: SetupAdminRequest): Promise<AuthResponse> {
    const { $api } = useNuxtApp()
    return $api<AuthResponse>('/setup/admin', {
      method: 'POST',
      body: req,
    })
  },

  async createUser(req: SetupUserRequest): Promise<SetupUserResponse> {
    const { $api } = useNuxtApp()
    return $api<SetupUserResponse>('/setup/user', {
      method: 'POST',
      body: req,
    })
  },

  async createProject(req: SetupProjectRequest): Promise<SetupProjectResponse> {
    const { $api } = useNuxtApp()
    return $api<SetupProjectResponse>('/setup/project', {
      method: 'POST',
      body: req,
    })
  },
}
