import { defineStore } from 'pinia'
import type { MemberResponse, Project } from '~/types/domain.types'
import { toApiMessageError } from '~/utils/apiError'
import { projectsService, type CreateProjectRequest } from '~/services/projects.service'

export const useProjectsStore = defineStore('projects', {
  state: () => ({
    projects: [] as Project[],
    current: null as Project | null,
    members: [] as MemberResponse[],
  }),
  actions: {
    async fetchAll() {
      this.projects = (await projectsService.list()) ?? []
    },
    async fetchOne(key: string) {
      // Never show another project's permissions while this one loads.
      if (this.current && this.current.key.toLowerCase() !== key.toLowerCase()) this.current = null
      this.current = await projectsService.get(key)
    },
    async fetchMembers(key: string) {
      this.members = []
      try {
        this.members = (await projectsService.listMembers(key)) ?? []
      }
      catch {
        // members stay empty; callers treat that as "no manage rights"
      }
    },
    async create(data: CreateProjectRequest) {
      const project = await projectsService.create(data)
      this.projects.push(project)
      return project
    },
    async update(key: string, data: Partial<CreateProjectRequest>) {
      const project = await projectsService.update(key, data)
      const idx = this.projects.findIndex(p => p.key === key)
      if (idx !== -1) this.projects[idx] = project
      if (this.current?.key === key) this.current = project
      return project
    },
    async changeMemberRole(key: string, userId: number, role: string) {
      try {
        const updated = await projectsService.changeMemberRole(key, userId, role)
        const idx = this.members.findIndex(m => m.user_id === userId)
        if (idx !== -1) this.members[idx] = { ...this.members[idx]!, ...updated }
        return updated
      }
      catch (e) {
        throw toApiMessageError(e, 'Failed to change role')
      }
    },
    /** Throws an ApiMessageError (server message; `taskCount` on a 409) when the delete is refused. */
    async deleteProject(key: string, confirmName: string) {
      try {
        await projectsService.deleteProject(key, confirmName)
      }
      catch (e) {
        throw toApiMessageError(e, 'Failed to delete project')
      }
      this.projects = this.projects.filter(p => p.key !== key)
      if (this.current?.key === key) this.current = null
    },
    /**
     * Refetches the open project so `myRole` and `permissions` follow a role change
     * made by someone else. A 404 means access is gone: nothing is allowed any more.
     * Other failures keep what is shown.
     */
    async refreshPermissions(key: string) {
      try {
        const fresh = await projectsService.get(key)
        if (this.current?.key === key) this.current = fresh
      }
      catch (e) {
        const status = (e as { status?: number, statusCode?: number } | null)?.status
          ?? (e as { statusCode?: number } | null)?.statusCode
        if (status === 404 && this.current?.key === key) {
          this.current = { ...this.current, myRole: undefined, permissions: [] }
        }
      }
    },
  },
})
