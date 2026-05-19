<template>
  <span
    class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"
    :style="{ backgroundColor: label.color, color: textColor }"
  >
    {{ label.name }}
  </span>
</template>

<script setup lang="ts">
import type { Label } from '~/types/domain.types'

const props = defineProps<{ label: Label }>()

const textColor = computed(() => {
  const hex = props.label.color.replace('#', '')
  const r = parseInt(hex.slice(0, 2), 16)
  const g = parseInt(hex.slice(2, 4), 16)
  const b = parseInt(hex.slice(4, 6), 16)
  const lum = (0.299 * r + 0.587 * g + 0.114 * b) / 255
  return lum > 0.5 ? '#000000' : '#ffffff'
})
</script>
