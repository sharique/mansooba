<template>
  <div>
    <SetupWizardShell />
    <AppSourceNotice class="fixed bottom-2 right-2 bg-base-100/80 px-2 py-1 rounded" />
  </div>
</template>

<script setup lang="ts">
import { useAboutStore } from '~/stores/about.store'
import AppSourceNotice from '~/components/common/AppSourceNotice.vue'
definePageMeta({ layout: false })

useAboutStore().fetch()
</script>
