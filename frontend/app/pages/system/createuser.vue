<template>
  <div class="max-w-md mx-auto py-8 px-4">
    <h1 class="text-2xl font-bold mb-2">Create User</h1>
    <p class="text-base-content/60 text-sm mb-6">
      New account credentials will be shared with the user directly.
    </p>

    <div v-if="createdName" class="alert alert-success mb-6">
      <span>Account created for <strong>{{ createdName }}</strong>.</span>
      <button class="btn btn-sm btn-ghost ml-auto" @click="createdName = ''">Create another</button>
    </div>

    <AuthRegisterForm v-if="!createdName" @success="onSuccess" />
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'

const authStore = useAuthStore()
const createdName = ref('')

onMounted(async () => {
  if (!authStore.isAdmin) {
    await navigateTo('/system/users')
  }
})

function onSuccess(name: string) {
  createdName.value = name || 'New user'
}
</script>
