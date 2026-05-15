import type { User } from '~/types/domain.types'

export interface AuthResponse {
  access_token: string
  user: User
}
