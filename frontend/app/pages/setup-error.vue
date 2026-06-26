<template>
  <div class="min-h-screen flex items-center justify-center bg-base-200">
    <div class="card bg-base-100 shadow-xl max-w-md w-full mx-4">
      <div class="card-body text-center gap-4">
        <div class="text-error text-5xl">⚠</div>
        <h1 class="text-2xl font-bold">Unable to connect</h1>
        <p class="text-base-content/70">
          Could not reach the server. Please check your connection and try again.
        </p>
        <button class="btn btn-primary mt-2" @click="retry">Refresh</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })

function retry() {
  window.location.href = '/'
}
</script>
