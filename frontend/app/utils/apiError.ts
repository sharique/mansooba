import type { ApiErrorBody } from '~/types/domain.types'

/** An error carrying the server's plain-language message and, when sent, the HTTP status. */
export class ApiMessageError extends Error {
  status?: number
  taskCount?: number
  constructor(message: string, status?: number, taskCount?: number) {
    super(message)
    this.name = 'ApiMessageError'
    this.status = status
    this.taskCount = taskCount
  }
}

/** The server's `message` from a failed request (ofetch puts the body in `data`), else the fallback. */
export function serverMessage(e: unknown, fallback: string): string {
  const err = e as { data?: ApiErrorBody, message?: string } | null
  return err?.data?.message || fallback
}

/** Turns a failed request into an ApiMessageError whose message is the server's reason. */
export function toApiMessageError(e: unknown, fallback: string): ApiMessageError {
  const err = e as { data?: ApiErrorBody, status?: number, statusCode?: number } | null
  return new ApiMessageError(
    serverMessage(e, fallback),
    err?.status ?? err?.statusCode,
    typeof err?.data?.task_count === 'number' ? err.data.task_count : undefined,
  )
}
