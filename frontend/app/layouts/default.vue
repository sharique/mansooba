<template>
  <div class="min-h-screen bg-base-100">
    <div class="navbar bg-base-200 shadow-sm">
      <div class="navbar-start">
        <NuxtLink to="/projects" class="btn btn-ghost text-xl">jira-go</NuxtLink>
      </div>
      <div class="navbar-end">
        <span class="mr-4 text-sm">{{ authStore.user?.name }}</span>
        <button class="btn btn-ghost btn-sm" @click="logout">Logout</button>
      </div>
    </div>
    <main class="container mx-auto p-6">
      <slot />
    </main>
    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth.store'

const authStore = useAuthStore()
const router = useRouter()

async function logout() {
  authStore.clearAuth()
  await router.push('/login')
}
</script>
