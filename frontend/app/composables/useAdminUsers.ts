export interface AdminUser {
  id: number
  name: string
  email: string
  is_admin: boolean
  is_active: boolean
  created_at: string
}

interface AdminUserListResponse {
  users: AdminUser[]
  total: number
  page: number
  size: number
}

export function useAdminUsers() {
  const { $api } = useNuxtApp()

  const users = ref<AdminUser[]>([])
  const total = ref(0)
  const page = ref(1)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchUsers(p = 1, size = 20) {
    loading.value = true
    error.value = null
    try {
      const data = await $api<AdminUserListResponse>(`/admin/users?page=${p}&size=${size}`)
      users.value = data.users
      total.value = data.total
      page.value = data.page
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load users'
    } finally {
      loading.value = false
    }
  }

  async function patchUser(id: number, patch: { is_admin?: boolean; is_active?: boolean }): Promise<AdminUser> {
    return $api<AdminUser>(`/admin/users/${id}`, {
      method: 'PATCH',
      body: patch,
    })
  }

  return { users, total, page, loading, error, fetchUsers, patchUser }
}
