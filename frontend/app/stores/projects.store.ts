import { defineStore } from 'pinia'
import type { Project } from '~/types/domain.types'
import { projectsService, type CreateProjectRequest } from '~/services/projects.service'

export const useProjectsStore = defineStore('projects', {
  state: () => ({
    projects: [] as Project[],
    current: null as Project | null,
  }),
  actions: {
    async fetchAll() {
      this.projects = (await projectsService.list()) ?? []
    },
    async fetchOne(key: string) {
      this.current = await projectsService.get(key)
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
  },
})
