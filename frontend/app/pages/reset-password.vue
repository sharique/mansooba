<template>
  <div class="min-h-screen grid lg:grid-cols-2">
    <!-- brand panel -->
    <div class="hidden lg:flex flex-col justify-center px-12 bg-neutral text-neutral-content">
      <div class="flex items-center gap-3 mb-6">
        <span class="inline-block w-9 h-9 rounded-lg bg-primary" />
        <span class="text-2xl font-bold">Mansooba</span>
      </div>
      <h1 class="text-3xl font-bold leading-tight">Set a new password.</h1>
      <p class="opacity-70 mt-3 max-w-sm">Paste your reset token and choose a new password.</p>
    </div>

    <!-- form panel -->
    <div class="flex items-center justify-center bg-base-200 p-6">
      <div class="card w-full max-w-sm bg-base-100 shadow-xl border border-base-300">
        <div class="card-body">
          <h2 class="card-title">Reset password</h2>

          <form @submit.prevent="submit">
            <div class="form-control w-full">
              <label class="label"><span class="label-text">Reset token</span></label>
              <input
                v-model="token"
                type="text"
                class="input input-bordered w-full font-mono text-sm"
                placeholder="Paste your 64-character token"
                maxlength="64"
                :disabled="loading"
              />
              <div v-if="tokenError" class="label">
                <span class="label-text-alt text-error">{{ tokenError }}</span>
              </div>
            </div>

            <div class="form-control w-full mt-3">
              <label class="label"><span class="label-text">New password</span></label>
              <input
                v-model="password"
                type="password"
                class="input input-bordered w-full"
                :class="{ 'input-error': passwordTouched && !allRulesPassed }"
                placeholder="Minimum 8 characters"
                :disabled="loading"
                @input="passwordTouched = true"
              />
              <ul class="text-sm mt-1 space-y-0.5" aria-label="Password requirements">
                <li :class="rules.length ? 'text-success' : 'text-base-content/50'">
                  {{ rules.length ? '✓' : '○' }} At least 8 characters
                </li>
                <li :class="rules.upper ? 'text-success' : 'text-base-content/50'">
                  {{ rules.upper ? '✓' : '○' }} At least one uppercase letter
                </li>
                <li :class="rules.lower ? 'text-success' : 'text-base-content/50'">
                  {{ rules.lower ? '✓' : '○' }} At least one lowercase letter
                </li>
                <li :class="rules.digit ? 'text-success' : 'text-base-content/50'">
                  {{ rules.digit ? '✓' : '○' }} At least one number
                </li>
              </ul>
            </div>

            <div v-if="errorMessage" class="alert alert-error mt-4 py-2 text-sm">
              {{ errorMessage }}
            </div>

            <button
              type="submit"
              class="btn btn-primary w-full mt-5"
              :disabled="!canSubmit || loading"
            >
              <span v-if="loading" class="loading loading-spinner loading-sm" />
              Set new password
            </button>
          </form>

          <p class="text-sm text-center mt-2">
            <NuxtLink to="/forgot-password" class="link link-primary">Request a new token</NuxtLink>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { authService } from '~/services/auth.service'

definePageMeta({ layout: false })

const route = useRoute()
const token = ref((route.query.token as string) ?? '')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')
const tokenError = ref('')
const passwordTouched = ref(false)

// Mirrors the backend's password_complexity validator exactly (main.go):
// 8+ chars, at least one uppercase, one lowercase, one digit.
const rules = reactive({ length: false, upper: false, lower: false, digit: false })
watch(password, (pw) => {
  rules.length = pw.length >= 8
  rules.upper = /[A-Z]/.test(pw)
  rules.lower = /[a-z]/.test(pw)
  rules.digit = /[0-9]/.test(pw)
})
const allRulesPassed = computed(() => rules.length && rules.upper && rules.lower && rules.digit)

const canSubmit = computed(() => token.value.length === 64 && allRulesPassed.value)

async function submit() {
  tokenError.value = ''
  errorMessage.value = ''
  loading.value = true
  try {
    await authService.resetPassword(token.value, password.value)
    await navigateTo('/login?reset=success')
  }
  catch (err: unknown) {
    const msg = (err as { data?: { message?: string } })?.data?.message ?? 'Reset failed. Please try again.'
    if (msg.includes('invalid') || msg.includes('expired')) {
      tokenError.value = msg
    }
    else {
      errorMessage.value = msg
    }
    // Keep the token field value so the user can retry without re-pasting.
  }
  finally {
    loading.value = false
  }
}
</script>
