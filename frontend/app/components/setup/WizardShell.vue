<template>
  <div class="min-h-screen bg-base-200 flex items-center justify-center p-4">
    <div class="card bg-base-100 shadow-xl w-full max-w-lg">
      <div class="card-body gap-6">
        <div v-if="setupStore.currentStep > 0" class="text-sm text-base-content/60 font-medium" aria-live="polite">
          Step {{ setupStore.currentStep }} of 5
        </div>

        <SetupWelcomeStep v-if="setupStore.currentStep === 0" />
        <SetupAdminStep v-else-if="setupStore.currentStep === 1" />
        <SetupUserStep v-else-if="setupStore.currentStep === 2" />
        <SetupProjectStep v-else-if="setupStore.currentStep === 3" />
        <SetupSampleDataStep v-else-if="setupStore.currentStep === 4" />
        <SetupCompleteStep v-else-if="setupStore.currentStep === 5" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useSetupStore } from '~/stores/setup.store'

const setupStore = useSetupStore()

// On step change: scroll to top of wizard card and focus first field
watch(() => setupStore.currentStep, () => {
  window.scrollTo({ top: 0 })
  nextTick(() => {
    const firstField = document.querySelector<HTMLElement>('.card input, .card button')
    firstField?.focus()
  })
})
</script>
