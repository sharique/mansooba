<template>
  <div class="flex flex-col items-center justify-center text-center py-12 px-4">
    <div class="rounded-full bg-base-200 p-4 mb-4">
      <Icon :name="icon" class="w-7 h-7 text-base-content/40" />
    </div>
    <h3 class="font-semibold text-base-content/80">{{ title }}</h3>
    <p v-if="description" class="text-sm text-base-content/50 mt-1 max-w-sm">{{ description }}</p>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  description?: string
  icon?: string
}>(), {
  icon: 'mdi:inbox-outline',
})
</script>
