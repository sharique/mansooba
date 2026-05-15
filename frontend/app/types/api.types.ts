export interface ApiError {
  code: string
  message: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
}
