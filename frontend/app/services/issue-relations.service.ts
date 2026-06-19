import type { RelationResponse, CreateRelationRequest } from '~/types/domain.types'

export const issueRelationsService = {
  list(issueId: number): Promise<RelationResponse[]> {
    const { $api } = useNuxtApp()
    return $api<RelationResponse[]>(`/issues/${issueId}/relations`)
  },

  create(issueId: number, payload: CreateRelationRequest): Promise<RelationResponse> {
    const { $api } = useNuxtApp()
    return $api<RelationResponse>(`/issues/${issueId}/relations`, { method: 'POST', body: payload })
  },

  remove(issueId: number, relationId: number): Promise<void> {
    const { $api } = useNuxtApp()
    return $api<void>(`/issues/${issueId}/relations/${relationId}`, { method: 'DELETE' })
  },
}
