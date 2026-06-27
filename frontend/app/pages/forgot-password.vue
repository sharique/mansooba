<template>
  <div class="min-h-screen grid lg:grid-cols-2">
    <!-- brand panel -->
    <div class="hidden lg:flex flex-col justify-center px-12 bg-neutral text-neutral-content">
      <div class="flex items-center gap-3 mb-6">
        <span class="inline-block w-9 h-9 rounded-lg bg-primary" />
        <span class="text-2xl font-bold">Mansooba</span>
      </div>
      <h1 class="text-3xl font-bold leading-tight">Recover your account.</h1>
      <p class="opacity-70 mt-3 max-w-sm">Enter your email and we'll generate a reset token you can use immediately.</p>
    </div>

    <!-- form panel -->
    <div class="flex items-center justify-center bg-base-200 p-6">
      <div class="card w-full max-w-sm bg-base-100 shadow-xl border border-base-300">
        <div class="card-body">
          <h2 class="card-title">Forgot password</h2>

          <!-- Step 1: email form -->
          <form v-if="!result" @submit.prevent="submit">
            <div class="form-control w-full">
              <label class="label"><span class="label-text">Email</span></label>
              <input
                v-model="email"
                type="email"
                class="input input-bordered w-full"
                placeholder="you@example.com"
                required
                :disabled="loading"
              />
            </div>

            <div v-if="errorMessage" class="alert alert-error mt-4 py-2 text-sm">
              {{ errorMessage }}
            </div>

            <button type="submit" class="btn btn-primary w-full mt-5" :disabled="loading">
              <span v-if="loading" class="loading loading-spinner loading-sm" />
              Send reset token
            </button>
          </form>

          <!-- Step 2: success — display token -->
          <div v-else class="success-screen space-y-4">
            <div class="alert alert-success py-2 text-sm">
              {{ result.message }}
            </div>

            <div v-if="result.token">
              <p class="text-sm font-medium mb-1">Your reset token</p>
              <code class="token-block block bg-base-200 rounded p-3 text-xs break-all font-mono select-all">{{ result.token }}</code>
              <p class="text-xs text-base-content/60 mt-2">
                Expires: {{ formattedExpiry }}
              </p>
              <p class="text-sm mt-3 text-base-content/80">
                Copy this token — you will need it on the next page.
              </p>
            </div>

            <NuxtLink
              :to="result.token ? `/reset-password?token=${result.token}` : '/reset-password'"
              class="btn btn-primary w-full"
            >
              Go to reset password
            </NuxtLink>
          </div>

          <p class="text-sm text-center mt-2">
            Remember your password?
            <NuxtLink to="/login" class="link link-primary">Sign in</NuxtLink>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { authService } from '~/services/auth.service'

definePageMeta({ layout: false })

const email = ref('')
const loading = ref(false)
const errorMessage = ref('')
const result = ref<{ token: string; expires_at: string; message: string } | null>(null)

const formattedExpiry = computed(() => {
  if (!result.value?.expires_at) return ''
  return new Date(result.value.expires_at).toLocaleString()
})

async function submit() {
  loading.value = true
  errorMessage.value = ''
  try {
    result.value = await authService.forgotPassword(email.value)
  }
  catch (err: unknown) {
    errorMessage.value = (err as { data?: { message?: string } })?.data?.message ?? 'Request failed. Please try again.'
  }
  finally {
    loading.value = false
  }
}
</script>
