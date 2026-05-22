import { defineStore } from 'pinia'
import { issuesService } from '~/services/issues.service'
import type { Issue, IssueFilters } from '~/types/domain.types'
import type { CreateIssueRequest, IssueListQuery } from '~/services/issues.service'

export const useIssuesStore = defineStore('issues', {
  state: () => ({
    issues: [] as Issue[],
    current: null as Issue | null,
    searchResults: [] as Issue[],
  }),
  actions: {
    async fetchForProject(key: string, filters?: IssueListQuery) {
      this.issues = await issuesService.list(key, filters)
    },
    async fetchOne(key: string, id: number) {
      this.current = await issuesService.get(key, id)
    },
    async create(key: string, data: CreateIssueRequest) {
      const issue = await issuesService.create(key, data)
      this.issues.push(issue)
      return issue
    },
    async update(key: string, id: number, data: Partial<CreateIssueRequest>) {
      const updated = await issuesService.update(key, id, data)
      const idx = this.issues.findIndex(i => i.id === id)
      if (idx !== -1) this.issues[idx] = updated
      if (this.current?.id === id) this.current = updated
      return updated
    },
    async remove(key: string, id: number) {
      await issuesService.remove(key, id)
      this.issues = this.issues.filter(i => i.id !== id)
    },
    async searchIssues(projectKey: string, filters: IssueFilters) {
      const hasFilters = !!(filters.q || filters.type || filters.status || filters.priority
        || filters.assignee_id || filters.label_id)
      if (!hasFilters) {
        this.searchResults = []
        return
      }
      this.searchResults = await issuesService.search(projectKey, filters)
    },
  },
})
