<template>
  <div class="flex flex-col gap-6">
    <div>
      <h2 class="text-xl font-bold">Import sample data</h2>
      <p class="text-base-content/70 mt-1 text-sm">
        Start with a realistic demo project — a sprint, issues, and labels are created for you.
        You can delete them any time.
      </p>
    </div>

    <!-- Idle state: two choice cards -->
    <div v-if="state === 'idle'" class="flex flex-col gap-3">
      <button
        class="btn btn-primary btn-block"
        @click="importData"
      >
        Import example data
      </button>
      <button
        class="btn btn-ghost btn-block"
        @click="skip"
      >
        Start with a clean workspace
      </button>
    </div>

    <!-- Loading state -->
    <div v-else-if="state === 'loading'" class="flex flex-col items-center gap-4 py-4">
      <span class="loading loading-spinner loading-lg text-primary" aria-busy="true" />
      <p class="text-sm text-base-content/70">Importing sample data…</p>
    </div>

    <!-- Error state (first failure: show Try again) -->
    <div v-else-if="state === 'error-retry'" class="flex flex-col gap-4">
      <div class="alert alert-error" role="alert" aria-live="assertive">
        <span>Failed to import sample data. Please try again.</span>
      </div>
      <div class="flex flex-col gap-2">
        <button class="btn btn-primary btn-block" @click="importData">
          Try again
        </button>
        <button class="btn btn-ghost btn-block" @click="skip">
          Continue without sample data
        </button>
      </div>
    </div>

    <!-- Permanent error state (second failure: show CLI recovery) -->
    <div v-else-if="state === 'error-permanent'" class="flex flex-col gap-4">
      <div class="alert alert-warning" role="alert" aria-live="assertive">
        <div class="flex flex-col gap-2 text-sm">
          <p class="font-medium">Sample data import failed after two attempts.</p>
          <p>You can import it later by running this command in your backend directory:</p>
          <code class="bg-base-200 rounded px-2 py-1 font-mono text-xs select-all">
            go run ./cmd/seed
          </code>
        </div>
      </div>
      <button class="btn btn-ghost btn-block" @click="skip">
        Continue without sample data
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useSetupStore } from '~/stores/setup.store'

const setupStore = useSetupStore()

type State = 'idle' | 'loading' | 'error-retry' | 'error-permanent'
const state = ref<State>('idle')
const attempts = ref(0)

async function importData() {
  state.value = 'loading'
  try {
    await setupStore.completeSampleData()
    // completeSampleData advances currentStep to 5 on success
  } catch {
    attempts.value++
    state.value = attempts.value >= 2 ? 'error-permanent' : 'error-retry'
  }
}

function skip() {
  setupStore.skipSampleData()
}
</script>
