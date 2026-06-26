<template>
  <div class="flex flex-col gap-6 text-center">
    <div class="text-success text-5xl" aria-hidden="true">✓</div>
    <h2 class="text-2xl font-bold">Setup complete!</h2>

    <div class="text-left bg-base-200 rounded-lg p-4 space-y-2">
      <div>
        <span class="font-medium">Admin account:</span>
        {{ authStore.user?.name }} ({{ authStore.user?.email }})
      </div>
      <div v-for="item in setupStore.summaryItems" :key="item.label">
        <span class="font-medium">{{ item.label }}:</span> {{ item.value }}
      </div>
    </div>

    <div class="flex justify-center">
      <button
        class="btn btn-primary btn-wide"
        :disabled="navigating"
        @click="goToDashboard"
      >
        <span v-if="navigating" class="loading loading-spinner loading-sm" />
        Go to Dashboard
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useSetupStore } from '~/stores/setup.store'
import { useAuthStore } from '~/stores/auth.store'

const setupStore = useSetupStore()
const authStore = useAuthStore()
const navigating = ref(false)

function goToDashboard() {
  navigating.value = true
  setupStore.finish()
}
</script>
