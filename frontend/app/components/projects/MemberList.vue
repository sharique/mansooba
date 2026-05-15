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
          <th>Email</th>
          <th>Role</th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="m in members" :key="m.user_id">
          <td>{{ m.name }}</td>
          <td>{{ m.email }}</td>
          <td><span class="badge badge-ghost capitalize">{{ m.role }}</span></td>
          <td>
            <button
              v-if="isOwner"
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
  </div>
</template>

<script setup lang="ts">
import { projectsService } from '~/services/projects.service'
import { useAuthStore } from '~/stores/auth.store'
import type { MemberResponse } from '~/types/domain.types'

type MemberRow = MemberResponse

const props = defineProps<{ projectKey: string; ownerId: number }>()
const { showSuccess, showError } = useToast()

const authStore = useAuthStore()
const isOwner = computed(() => authStore.user?.id === props.ownerId)

const members = ref<MemberRow[]>([])
const loading = ref(true)
const removing = ref<number | null>(null)
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

async function remove(userId: number) {
  removing.value = userId
  try {
    await projectsService.removeMember(props.projectKey, userId)
    members.value = members.value.filter(m => m.user_id !== userId)
    showSuccess('Member removed')
  }
  catch {
    showError('Failed to remove member')
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
