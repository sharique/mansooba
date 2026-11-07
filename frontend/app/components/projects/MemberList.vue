<template>
  <div>
    <h3 class="font-semibold text-lg mb-3">Members</h3>

    <div v-if="loading" class="space-y-2">
      <div v-for="i in 3" :key="i" class="skeleton h-10 w-full" />
    </div>

    <table v-else class="table w-full">
      <thead>
        <tr>
          <th>Name</th>
          <th>Username</th>
          <th>Email</th>
          <th>Role</th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="m in members" :key="m.user_id">
          <td>{{ m.name }}</td>
          <td><span v-if="m.username" class="text-base-content/70">@{{ m.username }}</span></td>
          <td>{{ m.email }}</td>
          <td>
            <select
              v-if="canManage"
              class="select select-bordered select-sm capitalize"
              :aria-label="`Role of ${m.name}`"
              :value="m.role"
              :disabled="isOnlyActiveAdmin(m) || changing === m.user_id"
              :title="isOnlyActiveAdmin(m) ? ONLY_ADMIN_REASON : undefined"
              @change="changeRole(m, $event)"
            >
              <option value="admin">Admin</option>
              <option value="member">Member</option>
              <option value="viewer">Viewer</option>
            </select>
            <span v-else class="badge badge-ghost capitalize">{{ m.role }}</span>
          </td>
          <td>
            <button
              v-if="canManage && m.user_id !== authStore.user?.id"
              class="btn btn-ghost btn-xs text-error"
              :disabled="removing === m.user_id"
              @click="remove(m.user_id)"
            >
              Remove
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <template v-if="canManage">
    <div class="divider" />

    <h4 class="font-medium mb-2">Invite member</h4>
    <div class="flex gap-2">
      <input
        v-model="inviteEmail"
        type="email"
        class="input input-bordered flex-1"
        placeholder="colleague@example.com"
      />
      <select v-model="inviteRole" class="select select-bordered">
        <option value="member">Member</option>
        <option value="admin">Admin</option>
        <option value="viewer">Viewer</option>
      </select>
      <button class="btn btn-primary" :disabled="inviting" @click="invite">
        <span v-if="inviting" class="loading loading-spinner loading-sm" />
        Add
      </button>
    </div>
    <div v-if="inviteError" class="alert alert-error mt-2 py-2 text-sm">{{ inviteError }}</div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { projectsService } from '~/services/projects.service'
import { useAuthStore } from '~/stores/auth.store'
import type { MemberResponse } from '~/types/domain.types'

type MemberRow = MemberResponse

const props = defineProps<{ projectKey: string }>()
const { showSuccess, showError } = useToast()

const authStore = useAuthStore()
const projectsStore = useProjectsStore()
const { can } = useProjectPermissions()
const canManage = computed(() => can('member.manage'))

const ONLY_ADMIN_REASON = 'A project needs at least one active admin'
const activeAdmins = computed(() => members.value.filter(m => m.role === 'admin' && m.is_active))
const isOnlyActiveAdmin = (m: MemberRow) =>
  m.role === 'admin' && m.is_active && activeAdmins.value.length === 1

const members = ref<MemberRow[]>([])
const loading = ref(true)
const removing = ref<number | null>(null)
const changing = ref<number | null>(null)
const inviteEmail = ref('')
const inviteRole = ref('member')
const inviting = ref(false)
const inviteError = ref('')

async function fetchMembers() {
  loading.value = true
  try {
    members.value = await projectsService.listMembers(props.projectKey)
  }
  finally {
    loading.value = false
  }
}

async function changeRole(m: MemberRow, event: Event) {
  const select = event.target as HTMLSelectElement
  const role = select.value
  changing.value = m.user_id
  try {
    const updated = await projectsStore.changeMemberRole(props.projectKey, m.user_id, role)
    members.value = members.value.map(x => (x.user_id === m.user_id ? { ...x, ...updated } : x))
    showSuccess(`${m.name} is now ${role}`)
    // Changing your own role changes what this page lets you do.
    if (m.user_id === authStore.user?.id) await projectsStore.refreshPermissions(props.projectKey)
  }
  catch (e) {
    select.value = m.role
    showError((e as Error).message)
  }
  finally {
    changing.value = null
  }
}

async function remove(userId: number) {
  removing.value = userId
  try {
    await projectsService.removeMember(props.projectKey, userId)
    members.value = members.value.filter(m => m.user_id !== userId)
    showSuccess('Member removed')
  }
  catch (e: unknown) {
    showError((e as { data?: { message?: string } })?.data?.message ?? 'Failed to remove member')
  }
  finally {
    removing.value = null
  }
}

async function invite() {
  inviteError.value = ''
  inviting.value = true
  try {
    await projectsService.addMember(props.projectKey, inviteEmail.value, inviteRole.value)
    inviteEmail.value = ''
    showSuccess('Member added')
    await fetchMembers()
  }
  catch (err: unknown) {
    inviteError.value = (err as { data?: { message?: string } })?.data?.message ?? 'Failed to add member'
  }
  finally {
    inviting.value = false
  }
}

onMounted(fetchMembers)
</script>
