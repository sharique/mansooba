import { defineStore } from 'pinia'
import type { RelationResponse, CreateRelationRequest } from '~/types/domain.types'
import { issueRelationsService } from '~/services/issue-relations.service'

export const useIssueRelationsStore = defineStore('issue-relations', {
  state: () => ({
    relations: [] as RelationResponse[],
    currentIssueId: null as number | null,
  }),
  actions: {
    async fetchForIssue(issueId: number) {
      if (this.currentIssueId !== issueId) {
        this.relations = []
        this.currentIssueId = issueId
      }
      this.relations = await issueRelationsService.list(issueId)
    },
    async create(issueId: number, payload: CreateRelationRequest) {
      const rel = await issueRelationsService.create(issueId, payload)
      this.relations.push(rel)
    },
    async remove(issueId: number, relationId: number) {
      await issueRelationsService.remove(issueId, relationId)
      this.relations = this.relations.filter(r => r.id !== relationId)
    },
  },
})
