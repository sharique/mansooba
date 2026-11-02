import type { Project, ProjectResponse, MemberResponse } from '~/types/domain.types'

export interface CreateProjectRequest {
  name: string
  key?: string
  description?: string
}

export interface AddMemberRequest {
  email: string
  role: string
}

/** Maps the wire shape to the camelCase Project the app uses. */
export function toProject(raw: ProjectResponse): Project {
  const { owner_id, my_role, ...rest } = raw
  return {
    ...rest,
    ownerId: owner_id ?? raw.ownerId ?? 0,
    myRole: my_role ?? undefined,
    permissions: raw.permissions,
  }
}

export const projectsService = {
  async list(): Promise<Project[]> {
    const { $api } = useNuxtApp()
    const raw = await $api<ProjectResponse[]>('/projects')
    return (raw ?? []).map(toProject)
  },

  async get(key: string): Promise<Project> {
    const { $api } = useNuxtApp()
    return toProject(await $api<ProjectResponse>(`/projects/${key}`))
  },

  async create(data: CreateProjectRequest): Promise<Project> {
    const { $api } = useNuxtApp()
    return toProject(await $api<ProjectResponse>('/projects', { method: 'POST', body: data }))
  },

  async update(key: string, data: Partial<CreateProjectRequest>): Promise<Project> {
    const { $api } = useNuxtApp()
    return toProject(await $api<ProjectResponse>(`/projects/${key}`, { method: 'PUT', body: data }))
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

  changeMemberRole(key: string, userId: number, role: string): Promise<MemberResponse> {
    const { $api } = useNuxtApp()
    return $api<MemberResponse>(`/projects/${key}/members/${userId}`, { method: 'PATCH', body: { role } })
  },

  /** The project must be empty; the exact project name confirms the delete. */
  deleteProject(key: string, confirmName: string): Promise<void> {
    const { $api } = useNuxtApp()
    return $api(`/projects/${key}`, { method: 'DELETE', body: { confirm_name: confirmName } })
  },
}
