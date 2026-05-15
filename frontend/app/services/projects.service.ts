import type { Project, MemberResponse } from '~/types/domain.types'

export interface CreateProjectRequest {
  name: string
  key?: string
  description?: string
}

export interface AddMemberRequest {
  email: string
  role: string
}

export const projectsService = {
  list(): Promise<Project[]> {
    const { $api } = useNuxtApp()
    return $api<Project[]>('/projects')
  },

  get(key: string): Promise<Project> {
    const { $api } = useNuxtApp()
    return $api<Project>(`/projects/${key}`)
  },

  create(data: CreateProjectRequest): Promise<Project> {
    const { $api } = useNuxtApp()
    return $api<Project>('/projects', { method: 'POST', body: data })
  },

  update(key: string, data: Partial<CreateProjectRequest>): Promise<Project> {
    const { $api } = useNuxtApp()
    return $api<Project>(`/projects/${key}`, { method: 'PUT', body: data })
  },

  listMembers(key: string): Promise<MemberResponse[]> {
    const { $api } = useNuxtApp()
    return $api<MemberResponse[]>(`/projects/${key}/members`)
  },

  addMember(key: string, email: string, role: string): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/projects/${key}/members`, { method: 'POST', body: { email, role } })
  },

  removeMember(key: string, userId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/projects/${key}/members/${userId}`, { method: 'DELETE' })
  },
}
