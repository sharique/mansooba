<template>
  <div class="flex flex-col gap-6 text-center">
    <h1 class="text-3xl font-bold">Welcome to Mansooba</h1>
    <p class="text-base-content/70">
      This wizard will help you set up your admin account, invite a team member, and create your first project — all in a few steps.
    </p>
    <div class="flex justify-center">
      <button class="btn btn-primary btn-wide" @click="advance">
        Get Started
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useSetupStore } from '~/stores/setup.store'

const setupStore = useSetupStore()

function advance() {
  setupStore.currentStep = 1
}
</script>
