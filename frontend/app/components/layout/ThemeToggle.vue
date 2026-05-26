<template>
  <button class="btn btn-ghost btn-circle btn-sm" :aria-label="label" :title="label" @click="toggle">
    <Icon :name="effective === 'mansooba-dark' ? 'mdi:weather-sunny' : 'mdi:weather-night'" class="w-5 h-5" />
  </button>
</template>

<script setup lang="ts">
const { effective, toggle } = useTheme()
const label = computed(() =>
  effective.value === 'mansooba-dark' ? 'Switch to light mode' : 'Switch to dark mode',
)
</script>
