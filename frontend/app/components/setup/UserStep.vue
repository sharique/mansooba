<template>
  <div class="flex flex-col gap-4">
    <h2 class="text-2xl font-bold">Invite a team member</h2>
    <p class="text-base-content/70 text-sm">Optional — you can do this later from the admin panel.</p>

    <div class="form-control gap-1">
      <label for="user-name" class="label label-text font-medium">Display name</label>
      <input
        id="user-name"
        v-model="form.full_name"
        type="text"
        autocomplete="name"
        class="input input-bordered w-full"
        :class="{ 'input-error': errors.full_name }"
        :disabled="loading"
      />
      <span v-if="errors.full_name" role="alert" class="text-error text-sm">{{ errors.full_name }}</span>
    </div>

    <div class="form-control gap-1">
      <label for="user-email" class="label label-text font-medium">Email</label>
      <input
        id="user-email"
        v-model="form.email"
        type="email"
        autocomplete="email"
        class="input input-bordered w-full"
        :class="{ 'input-error': errors.email }"
        :disabled="loading"
        @blur="validateEmail"
      />
      <span v-if="errors.email" role="alert" class="text-error text-sm">{{ errors.email }}</span>
    </div>

    <div class="form-control gap-1">
      <label for="user-password" class="label label-text font-medium">Password</label>
      <input
        id="user-password"
        v-model="form.password"
        type="password"
        autocomplete="new-password"
        class="input input-bordered w-full"
        :disabled="loading"
        @input="validatePassword"
      />
      <ul class="text-sm mt-1 space-y-0.5" aria-label="Password requirements">
        <li :class="rules.length ? 'text-success' : 'text-base-content/50'">{{ rules.length ? '✓' : '○' }} At least 8 characters</li>
        <li :class="rules.upper ? 'text-success' : 'text-base-content/50'">{{ rules.upper ? '✓' : '○' }} At least one uppercase letter</li>
        <li :class="rules.lower ? 'text-success' : 'text-base-content/50'">{{ rules.lower ? '✓' : '○' }} At least one lowercase letter</li>
        <li :class="rules.digit ? 'text-success' : 'text-base-content/50'">{{ rules.digit ? '✓' : '○' }} At least one number</li>
      </ul>
    </div>

    <span v-if="errors.server" role="alert" class="text-error text-sm">{{ errors.server }}</span>

    <div class="flex items-center justify-between mt-2">
      <button
        class="btn btn-ghost text-sm font-normal"
        :disabled="loading"
        @click="skip"
      >
        Skip for now
      </button>
      <button
        class="btn btn-primary"
        :disabled="loading || !canSubmit"
        @click="submit"
      >
        <span v-if="loading" class="loading loading-spinner loading-sm" />
        Create user
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, computed, ref } from 'vue'
import { useSetupStore } from '~/stores/setup.store'

const setupStore = useSetupStore()
const form = reactive({ full_name: '', email: '', password: '' })
const errors = reactive({ full_name: '', email: '', server: '' })
const loading = ref(false)
const rules = reactive({ length: false, upper: false, lower: false, digit: false })

function validatePassword() {
  const pw = form.password
  rules.length = pw.length >= 8
  rules.upper = /[A-Z]/.test(pw)
  rules.lower = /[a-z]/.test(pw)
  rules.digit = /[0-9]/.test(pw)
}

function validateEmail() {
  if (!form.email) {
    errors.email = 'Email is required.'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    errors.email = 'Please enter a valid email address.'
  } else {
    errors.email = ''
  }
}

const allRulesPassed = computed(() => rules.length && rules.upper && rules.lower && rules.digit)
const canSubmit = computed(() => form.full_name.trim() !== '' && form.email !== '' && allRulesPassed.value)

function skip() {
  setupStore.skipUser()
}

async function submit() {
  errors.server = ''
  if (!form.full_name.trim()) { errors.full_name = 'Display name is required.'; return }
  errors.full_name = ''
  validateEmail()
  if (errors.email) return

  loading.value = true
  try {
    await setupStore.completeUser({ full_name: form.full_name, email: form.email, password: form.password })
  } catch (err: any) {
    if (err?.statusCode === 429) {
      errors.server = 'Too many attempts. Please wait a minute and try again.'
    } else if (err?.statusCode === 409) {
      errors.server = 'This email is already registered.'
    } else {
      errors.server = 'Something went wrong. Please try again.'
    }
  } finally {
    loading.value = false
  }
}
</script>
