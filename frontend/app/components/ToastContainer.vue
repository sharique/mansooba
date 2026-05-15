<template>
  <div class="toast toast-top toast-end z-50">
    <div
      v-for="toast in toasts"
      :key="toast.id"
      :class="['alert', toast.type === 'success' ? 'alert-success' : 'alert-error']"
    >
      <span>{{ toast.message }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
const { toasts } = useToast()
</script>
