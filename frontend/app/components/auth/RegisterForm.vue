<template>
  <form @submit.prevent="submit">
    <div class="space-y-4">
      <div class="form-control w-full">
        <label class="label"><span class="label-text">Full Name</span></label>
        <input
          v-model="fullName"
          type="text"
          class="input input-bordered w-full"
          placeholder="Alice Smith"
          required
        />
      </div>

      <div class="form-control w-full">
        <label class="label"><span class="label-text">Email</span></label>
        <input
          v-model="email"
          type="email"
          class="input input-bordered w-full"
          :class="{ 'input-error': emailError }"
          placeholder="you@example.com"
        />
        <label v-if="emailError" class="label">
          <span class="label-text-alt text-error">{{ emailError }}</span>
        </label>
      </div>

      <div class="form-control w-full">
        <label class="label"><span class="label-text">Password</span></label>
        <input
          v-model="password"
          type="password"
          class="input input-bordered w-full"
          :class="{ 'input-error': passwordError }"
          placeholder="••••••••"
        />
        <label v-if="passwordError" class="label">
          <span class="label-text-alt text-error">{{ passwordError }}</span>
        </label>
      </div>
    </div>

    <div v-if="errorMessage" class="alert alert-error mt-4 py-2 text-sm">
      {{ errorMessage }}
    </div>

    <button type="submit" class="btn btn-primary w-full mt-5" :disabled="loading">
      <span v-if="loading" class="loading loading-spinner loading-sm" />
      Create account
    </button>
  </form>
</template>

<script setup lang="ts">
import { authService } from '~/services/auth.service'

const emit = defineEmits<{ success: [name: string] }>()

const fullName = ref('')
const email = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')
const emailError = ref('')
const passwordError = ref('')

function validate(): boolean {
  emailError.value = ''
  passwordError.value = ''
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
    emailError.value = 'Enter a valid email address'
  }
  if (password.value.length < 8) {
    passwordError.value = 'Password must be at least 8 characters'
  }
  return !emailError.value && !passwordError.value
}

async function submit() {
  if (!validate()) return
  loading.value = true
  errorMessage.value = ''
  try {
    await authService.register(email.value, password.value, fullName.value)
    emit('success', fullName.value)
  }
  catch (err: unknown) {
    errorMessage.value = (err as { data?: { message?: string } })?.data?.message ?? 'Registration failed'
  }
  finally {
    loading.value = false
  }
}
</script>
