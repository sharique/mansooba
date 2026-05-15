<template>
  <form @submit.prevent="submit">
    <div class="form-control w-full">
      <label class="label"><span class="label-text">Email</span></label>
      <input
        v-model="email"
        type="email"
        class="input input-bordered w-full"
        placeholder="you@example.com"
        required
      />
    </div>

    <div class="form-control w-full mt-3">
      <label class="label"><span class="label-text">Password</span></label>
      <input
        v-model="password"
        type="password"
        class="input input-bordered w-full"
        placeholder="••••••••"
        required
      />
    </div>

    <div v-if="errorMessage" class="alert alert-error mt-4 py-2 text-sm">
      {{ errorMessage }}
    </div>

    <button type="submit" class="btn btn-primary w-full mt-5" :disabled="loading">
      <span v-if="loading" class="loading loading-spinner loading-sm" />
      Sign in
    </button>
  </form>
</template>

<script setup lang="ts">
import { authService } from '~/services/auth.service'

const emit = defineEmits<{ success: [] }>()

const email = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')

async function submit() {
  loading.value = true
  errorMessage.value = ''
  try {
    await authService.login(email.value, password.value)
    emit('success')
  }
  catch (err: unknown) {
    errorMessage.value = (err as { data?: { message?: string } })?.data?.message ?? 'Login failed'
  }
  finally {
    loading.value = false
  }
}
</script>
