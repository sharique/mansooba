<template>
  <div class="max-w-2xl mx-auto p-6">
    <h1 class="text-2xl font-bold mb-6">Settings</h1>

    <div role="tablist" class="tabs tabs-bordered mb-6">
      <button
        role="tab"
        :class="['tab', activeTab === 'profile' && 'tab-active']"
        @click="activeTab = 'profile'"
      >Profile</button>
      <button
        role="tab"
        :class="['tab', activeTab === 'activity' && 'tab-active']"
        @click="activeTab = 'activity'"
      >My Activity</button>
    </div>

    <!-- Profile Tab -->
    <div v-if="activeTab === 'profile'" class="card bg-base-100 shadow border border-base-200">
      <div class="card-body gap-4">
        <div class="form-control">
          <label class="label"><span class="label-text">Full Name</span></label>
          <input
            v-model="form.fullName"
            type="text"
            class="input input-bordered"
            placeholder="Your full name"
          />
        </div>

        <div class="form-control">
          <label class="label"><span class="label-text">Email</span></label>
          <input
            :value="authStore.profile?.email ?? authStore.user?.email"
            type="email"
            class="input input-bordered input-disabled"
            disabled
          />
        </div>

        <div class="form-control">
          <label class="label"><span class="label-text">Timezone</span></label>
          <select v-model="form.timezone" class="select select-bordered">
            <option value="">UTC (default)</option>
            <option v-for="tz in commonTimezones" :key="tz" :value="tz">{{ tz }}</option>
          </select>
        </div>

        <div class="form-control">
          <label class="label"><span class="label-text">Avatar URL</span></label>
          <input
            v-model="form.avatarURL"
            type="url"
            class="input input-bordered"
            placeholder="https://example.com/avatar.jpg"
          />
          <div v-if="form.avatarURL" class="mt-2">
            <img :src="form.avatarURL" alt="Avatar preview" class="w-16 h-16 rounded-full object-cover" />
          </div>
        </div>

        <div class="card-actions justify-end">
          <button
            class="btn btn-primary"
            :disabled="saving"
            @click="saveProfile"
          >
            {{ saving ? 'Saving…' : 'Save Changes' }}
          </button>
        </div>
      </div>
    </div>

    <!-- My Activity Tab -->
    <div v-if="activeTab === 'activity'">
      <div v-if="loadingActivity" class="skeleton h-32 w-full" />
      <div v-else-if="authStore.myActivity.length === 0" class="text-base-content/50 text-sm">
        No activity yet.
      </div>
      <IssuesActivityFeed
        v-else
        :comments="[]"
        :activity="authStore.myActivity"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'

const authStore = useAuthStore()
const { showSuccess, showError } = useToast()

const activeTab = ref<'profile' | 'activity'>('profile')
const saving = ref(false)
const loadingActivity = ref(false)
const activityLoaded = ref(false)

const form = reactive({
  fullName: authStore.profile?.name ?? authStore.user?.name ?? '',
  timezone: authStore.profile?.timezone ?? '',
  avatarURL: authStore.profile?.avatar_url ?? '',
})

const commonTimezones = [
  'America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles',
  'America/Sao_Paulo', 'Europe/London', 'Europe/Paris', 'Europe/Berlin',
  'Asia/Dubai', 'Asia/Kolkata', 'Asia/Singapore', 'Asia/Tokyo',
  'Australia/Sydney', 'Pacific/Auckland',
]

onMounted(async () => {
  await authStore.fetchMe()
  form.fullName = authStore.profile?.name ?? ''
  form.timezone = authStore.profile?.timezone ?? ''
  form.avatarURL = authStore.profile?.avatar_url ?? ''
})

watch(activeTab, async (tab) => {
  if (tab === 'activity' && !activityLoaded.value) {
    loadingActivity.value = true
    try {
      await authStore.fetchMyActivity()
    } finally {
      loadingActivity.value = false
      activityLoaded.value = true
    }
  }
})

async function saveProfile() {
  saving.value = true
  try {
    await authStore.updateProfile({
      full_name: form.fullName || undefined,
      timezone: form.timezone || undefined,
      avatar_url: form.avatarURL || undefined,
    })
    showSuccess('Profile updated')
  } catch {
    showError('Failed to update profile')
  } finally {
    saving.value = false
  }
}
</script>
