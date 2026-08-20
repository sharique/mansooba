<template>
  <form class="space-y-4" @submit.prevent="submit">
    <div class="form-control">
      <label class="label"><span class="label-text">Current Password</span></label>
      <input
        v-model="form.currentPassword"
        data-testid="current-password"
        type="password"
        class="input input-bordered w-full"
      />
    </div>

    <div class="form-control">
      <label class="label"><span class="label-text">New Password</span></label>
      <input
        v-model="form.newPassword"
        data-testid="new-password"
        type="password"
        class="input input-bordered w-full"
        :class="{ 'input-error': newPasswordTouched && !allRulesPassed }"
        @input="newPasswordTouched = true"
      />
      <ul class="text-sm mt-1 space-y-0.5" aria-label="Password requirements">
        <li :class="rules.length ? 'text-success' : 'text-base-content/50'">{{ rules.length ? '✓' : '○' }} At least 8 characters</li>
        <li :class="rules.upper ? 'text-success' : 'text-base-content/50'">{{ rules.upper ? '✓' : '○' }} At least one uppercase letter</li>
        <li :class="rules.lower ? 'text-success' : 'text-base-content/50'">{{ rules.lower ? '✓' : '○' }} At least one lowercase letter</li>
        <li :class="rules.digit ? 'text-success' : 'text-base-content/50'">{{ rules.digit ? '✓' : '○' }} At least one number</li>
      </ul>
    </div>

    <div class="form-control">
      <label class="label"><span class="label-text">Confirm New Password</span></label>
      <input
        v-model="form.confirmPassword"
        data-testid="confirm-password"
        type="password"
        class="input input-bordered w-full"
      />
      <label v-if="mismatchError" class="label">
        <span data-testid="mismatch-error" class="label-text-alt text-error">{{ mismatchError }}</span>
      </label>
    </div>

    <div v-if="fieldError" class="alert alert-error py-2 text-sm">{{ fieldError }}</div>

    <div class="card-actions justify-end">
      <button data-testid="submit" type="submit" class="btn btn-primary" :disabled="saving">
        <span v-if="saving" class="loading loading-spinner loading-xs" />
        {{ saving ? 'Changing…' : 'Change Password' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
const authStore = useAuthStore()
const { showSuccess, showError } = useToast()

const form = reactive({ currentPassword: '', newPassword: '', confirmPassword: '' })
const saving = ref(false)
const newPasswordTouched = ref(false)
const mismatchError = ref('')
const fieldError = ref('')

// Mirrors reset-password.vue's rule block and the backend's password_complexity
// validator exactly: 8+ chars, one uppercase, one lowercase, one digit.
const rules = reactive({ length: false, upper: false, lower: false, digit: false })
watch(() => form.newPassword, (pw) => {
  rules.length = pw.length >= 8
  rules.upper = /[A-Z]/.test(pw)
  rules.lower = /[a-z]/.test(pw)
  rules.digit = /[0-9]/.test(pw)
})
const allRulesPassed = computed(() => rules.length && rules.upper && rules.lower && rules.digit)

async function submit() {
  mismatchError.value = ''
  fieldError.value = ''

  // Confirm-password mismatch guard — blocks before any network call.
  if (!form.confirmPassword || form.newPassword !== form.confirmPassword) {
    mismatchError.value = 'New password and confirmation do not match.'
    return
  }

  saving.value = true
  try {
    await authStore.changePassword(form.currentPassword, form.newPassword)
    showSuccess('Password changed')
    form.currentPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
    newPasswordTouched.value = false
  }
  catch (err: unknown) {
    // Maps the backend's {"message": "..."} body to a specific inline
    // error, following reset-password.vue's extraction pattern.
    const msg = (err as { data?: { message?: string } })?.data?.message ?? 'Failed to change password. Please try again.'
    fieldError.value = msg
    showError(msg)
  }
  finally {
    saving.value = false
  }
}
</script>
