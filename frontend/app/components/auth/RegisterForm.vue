<template>
  <form @submit.prevent="submit">
    <div class="space-y-4">
      <div class="form-control w-full">
        <label class="label"><span class="label-text">Full Name</span></label>
        <input
          v-model="fullName"
          type="text"
          class="input input-bordered w-full"
          :class="{ 'input-error': fullNameError }"
          placeholder="Alice Smith"
        />
        <label v-if="fullNameError" class="label">
          <span class="label-text-alt text-error">{{ fullNameError }}</span>
        </label>
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
          :class="{ 'input-error': passwordTouched && !allRulesPassed }"
          placeholder="••••••••"
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
const fullNameError = ref('')
const emailError = ref('')
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

function validate(): boolean {
  fullNameError.value = fullName.value.trim() ? '' : 'Full name is required'
  emailError.value = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value) ? '' : 'Enter a valid email address'
  passwordTouched.value = true
  return !fullNameError.value && !emailError.value && allRulesPassed.value
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
