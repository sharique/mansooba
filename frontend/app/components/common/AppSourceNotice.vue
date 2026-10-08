<template>
  <div data-testid="source-notice" class="text-xs opacity-60 flex items-center gap-1 whitespace-nowrap">
    <a :href="store.sourceUrl" target="_blank" rel="noopener noreferrer" class="link link-hover">Source code</a>
    <span>·</span>
    <a :href="store.licenseUrl" target="_blank" rel="noopener noreferrer" class="link link-hover">AGPL-3.0</a>
    <template v-if="store.version">
      <span>·</span>
      <span>v{{ store.version }}</span>
    </template>
  </div>
</template>

<script setup lang="ts">
// Lets any user find this instance's source and licence (AGPL-3.0 section
// 13). Mounted in TopBar and every unauthenticated page.
import { useAboutStore } from '~/stores/about.store'

const store = useAboutStore()
</script>
