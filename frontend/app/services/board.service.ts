import type { Issue } from '~/types/domain.types'

export interface BoardColumn { status: string; issues: Issue[] }
export interface BoardData { columns: BoardColumn[] }

export const boardService = {
  getBoard(projectKey: string): Promise<BoardData> {
    const { $api } = useNuxtApp()
    return $api<BoardData>(`/projects/${projectKey}/board`)
  },
}
