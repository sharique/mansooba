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
          <label class="label"><span class="label-text">Full Name <span class="text-error">*</span></span></label>
          <input
            v-model="form.fullName"
            type="text"
            class="input input-bordered w-full"
            :class="{ 'input-error': fullNameError }"
            placeholder="Your full name"
          />
          <label v-if="fullNameError" class="label">
            <span class="label-text-alt text-error">Full name is required</span>
          </label>
        </div>

        <div class="form-control">
          <label class="label"><span class="label-text">Email</span></label>
          <input
            :value="authStore.profile?.email ?? authStore.user?.email"
            type="email"
            class="input input-bordered input-disabled w-full"
            disabled
          />
        </div>

        <div class="form-control">
          <label class="label"><span class="label-text">Timezone</span></label>
          <select v-model="form.timezone" class="select select-bordered w-full">
            <option value="">UTC (default)</option>
            <option v-for="tz in commonTimezones" :key="tz" :value="tz">{{ tz }}</option>
          </select>
        </div>

        <!-- Avatar section -->
        <div class="form-control">
          <label class="label"><span class="label-text">Profile Photo</span></label>
          <div class="flex items-center gap-4">
            <UserAvatar
              :avatarUrl="authStore.profile?.avatar_url || undefined"
              :name="authStore.profile?.name || authStore.user?.name || ''"
              :userId="authStore.profile?.id || 0"
              size="lg"
            />
            <div class="flex flex-col gap-2">
              <input
                ref="fileInput"
                type="file"
                accept="image/jpeg,image/png,image/webp"
                class="hidden"
                @change="onFileSelect"
              />
              <button class="btn btn-sm btn-outline" @click="fileInput?.click()">
                Choose photo
              </button>
              <button
                v-if="authStore.profile?.avatar_url"
                class="btn btn-sm btn-ghost text-error"
                :disabled="uploading"
                @click="removeAvatar"
              >
                Remove photo
              </button>
            </div>
          </div>
          <p v-if="avatarError" class="text-error text-sm mt-1">{{ avatarError }}</p>
          <p v-if="selectedFile" class="text-sm text-base-content/70 mt-1">
            Selected: {{ selectedFile.name }} ({{ (selectedFile.size / 1024).toFixed(1) }} KB)
          </p>
        </div>

        <div class="card-actions justify-end">
          <button
            class="btn btn-primary"
            :disabled="saving || uploading"
            @click="saveProfile"
          >
            <span v-if="saving || uploading" class="loading loading-spinner loading-xs" />
            {{ saving ? 'Saving…' : uploading ? 'Uploading…' : 'Save Changes' }}
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
import { authService } from '~/services/auth.service'
import UserAvatar from '~/components/common/UserAvatar.vue'

const authStore = useAuthStore()
const { showSuccess, showError } = useToast()

const activeTab = ref<'profile' | 'activity'>('profile')
const saving = ref(false)
const uploading = ref(false)
const loadingActivity = ref(false)
const activityLoaded = ref(false)
const selectedFile = ref<File | null>(null)
const avatarError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const fullNameError = ref(false)

const form = reactive({
  fullName: authStore.profile?.name ?? authStore.user?.name ?? '',
  timezone: authStore.profile?.timezone ?? '',
})

const commonTimezones = [
  'America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles',
  'America/Sao_Paulo', 'Europe/London', 'Europe/Paris', 'Europe/Berlin',
  'Asia/Dubai', 'Asia/Kolkata', 'Asia/Singapore', 'Asia/Tokyo',
  'Australia/Sydney', 'Pacific/Auckland',
]

const MAX_BYTES = 2 * 1024 * 1024
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp']

onMounted(async () => {
  await authStore.fetchMe()
  form.fullName = authStore.profile?.name ?? ''
  form.timezone = authStore.profile?.timezone ?? ''
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

function onFileSelect(e: Event) {
  avatarError.value = ''
  selectedFile.value = null
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  if (file.size > MAX_BYTES) {
    avatarError.value = 'File must be under 2 MB'
    return
  }
  if (!ALLOWED_TYPES.includes(file.type)) {
    avatarError.value = 'Only JPEG, PNG, or WebP images are accepted'
    return
  }
  selectedFile.value = file
}

async function saveProfile() {
  fullNameError.value = !form.fullName.trim()
  if (fullNameError.value) return

  saving.value = true
  try {
    if (selectedFile.value) {
      await uploadAvatar(selectedFile.value)
      selectedFile.value = null
    }
    await authStore.updateProfile({
      full_name: form.fullName.trim(),
      timezone: form.timezone || undefined,
    })
    showSuccess('Profile updated')
  } catch {
    showError('Failed to update profile')
  } finally {
    saving.value = false
  }
}

async function uploadAvatar(file: File) {
  uploading.value = true
  try {
    await authService.uploadAvatar(file)
    await authStore.fetchMe()
  } finally {
    uploading.value = false
  }
}

async function removeAvatar() {
  uploading.value = true
  try {
    await authService.deleteAvatar()
    await authStore.fetchMe()
    showSuccess('Avatar removed')
  } catch {
    showError('Failed to remove avatar')
  } finally {
    uploading.value = false
  }
}
</script>
