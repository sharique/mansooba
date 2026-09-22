<template>
  <div v-if="store.demoBannerEnabled" data-testid="demo-banner" class="alert alert-warning rounded-none justify-center text-center py-2">
    {{ store.demoBannerMessage }}
  </div>
</template>

<script setup lang="ts">
const store = useSetupStore()
</script>
