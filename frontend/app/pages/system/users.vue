<template>
  <div class="max-w-5xl mx-auto py-8 px-4">
    <h1 class="text-2xl font-bold mb-6">User Management</h1>

    <!-- Error banner -->
    <div v-if="error" class="alert alert-error mb-4 flex items-center gap-2">
      <span>{{ error }}</span>
      <button class="btn btn-sm" @click="fetchUsers(page)">Retry</button>
    </div>

    <!-- Skeleton loader -->
    <div v-if="loading" class="space-y-2">
      <div v-for="n in 5" :key="n" class="skeleton h-10 w-full rounded" />
    </div>

    <!-- User table -->
    <div v-else-if="users.length" class="overflow-x-auto">
      <table class="table table-zebra w-full">
        <thead>
          <tr>
            <th>Name</th>
            <th>Email</th>
            <th>Role</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.name }}</td>
            <td>{{ user.email }}</td>
            <td>
              <span
                :class="user.is_admin ? 'badge badge-primary' : 'badge badge-ghost'"
                :aria-label="user.is_admin ? 'Role: Admin' : 'Role: Member'"
              >
                {{ user.is_admin ? 'Admin' : 'Member' }}
              </span>
            </td>
            <td>
              <span
                :class="user.is_active ? 'badge badge-success' : 'badge badge-error'"
                :aria-label="user.is_active ? 'Status: Active' : 'Status: Disabled'"
              >
                {{ user.is_active ? 'Active' : 'Disabled' }}
              </span>
            </td>
            <td class="flex gap-2">
              <button
                class="btn btn-xs btn-outline"
                :aria-label="user.is_admin ? `Remove admin from ${user.name}` : `Make ${user.name} admin`"
                :disabled="actionLoading === user.id"
                @click="handleRoleToggle(user)"
              >
                {{ user.is_admin ? 'Demote' : 'Promote' }}
              </button>
              <button
                class="btn btn-xs"
                :class="user.is_active ? 'btn-warning' : 'btn-success'"
                :aria-label="user.is_active ? `Disable ${user.name}'s account` : `Re-enable ${user.name}'s account`"
                :disabled="actionLoading === user.id"
                @click="handleActiveToggle(user)"
              >
                {{ user.is_active ? 'Disable' : 'Enable' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="total > pageSize" class="flex justify-center gap-2 mt-4">
        <button class="btn btn-sm" :disabled="page <= 1" @click="goToPage(page - 1)">«</button>
        <span class="flex items-center px-2">Page {{ page }} / {{ totalPages }}</span>
        <button class="btn btn-sm" :disabled="page >= totalPages" @click="goToPage(page + 1)">»</button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="text-center py-16 text-base-content/50">
      No other users yet.
    </div>

    <!-- Confirmation dialog (role demote / account disable) -->
    <dialog ref="confirmDialog" class="modal">
      <div class="modal-box" role="dialog" :aria-label="confirmTitle">
        <h3 class="font-bold text-lg">{{ confirmTitle }}</h3>
        <p v-if="confirmTarget" class="py-2">
          {{ confirmTarget.name }} ({{ confirmTarget.email }})
        </p>
        <p v-if="confirmError" class="text-error text-sm mt-1">{{ confirmError }}</p>
        <div class="modal-action">
          <button class="btn" @click="closeDialog">Cancel</button>
          <button class="btn btn-error" :disabled="confirmLoading" @click="confirmAction">
            {{ confirmLabel }}
          </button>
        </div>
      </div>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'
import { useAdminUsers, type AdminUser } from '~/composables/useAdminUsers'

const authStore = useAuthStore()
const { users, total, page, loading, error, fetchUsers, patchUser } = useAdminUsers()

const pageSize = 20
const totalPages = computed(() => Math.ceil(total.value / pageSize))

const actionLoading = ref<number | null>(null)

// Confirmation dialog state
const confirmDialog = ref<HTMLDialogElement | null>(null)
const confirmTarget = ref<AdminUser | null>(null)
const confirmTitle = ref('')
const confirmLabel = ref('')
const confirmLoading = ref(false)
const confirmError = ref('')
let pendingAction: (() => Promise<void>) | null = null
let triggerElement: HTMLElement | null = null

onMounted(async () => {
  if (!authStore.isAdmin) {
    await navigateTo('/')
    return
  }
  await fetchUsers(1, pageSize)
})

async function goToPage(p: number) {
  await fetchUsers(p, pageSize)
}

async function handleRoleToggle(user: AdminUser) {
  if (user.is_admin) {
    // Destructive — show confirmation
    openDialog(
      user,
      'Remove admin access?',
      'Remove',
      async () => {
        await applyPatch(user, { is_admin: false })
      }
    )
  } else {
    // Promote — no confirmation needed
    await applyPatch(user, { is_admin: true })
  }
}

async function handleActiveToggle(user: AdminUser) {
  if (user.is_active) {
    openDialog(
      user,
      'Disable account?',
      'Disable',
      async () => {
        await applyPatch(user, { is_active: false })
      }
    )
  } else {
    await applyPatch(user, { is_active: true })
  }
}

async function applyPatch(user: AdminUser, patch: { is_admin?: boolean; is_active?: boolean }) {
  const prevAdmin = user.is_admin
  const prevActive = user.is_active

  // Optimistic update
  if (patch.is_admin !== undefined) user.is_admin = patch.is_admin
  if (patch.is_active !== undefined) user.is_active = patch.is_active
  actionLoading.value = user.id

  try {
    const updated = await patchUser(user.id, patch)
    user.is_admin = updated.is_admin
    user.is_active = updated.is_active
  } catch (e: unknown) {
    // Revert on error
    user.is_admin = prevAdmin
    user.is_active = prevActive

    const msg = e instanceof Error ? e.message : 'An error occurred'
    if (msg.includes('LAST_ADMIN') || msg.includes('last active admin')) {
      confirmError.value = 'Cannot remove the last active admin.'
    } else {
      error.value = msg
    }
  } finally {
    actionLoading.value = null
  }
}

function openDialog(user: AdminUser, title: string, label: string, action: () => Promise<void>) {
  triggerElement = document.activeElement as HTMLElement | null
  confirmTarget.value = user
  confirmTitle.value = title
  confirmLabel.value = label
  confirmError.value = ''
  pendingAction = action
  confirmDialog.value?.showModal()
}

function closeDialog() {
  confirmDialog.value?.close()
  confirmError.value = ''
  pendingAction = null
  triggerElement?.focus()
  triggerElement = null
}

async function confirmAction() {
  if (!pendingAction) return
  confirmLoading.value = true
  confirmError.value = ''
  try {
    await pendingAction()
    closeDialog()
  } catch {
    // error handled in applyPatch
  } finally {
    confirmLoading.value = false
  }
}
</script>
