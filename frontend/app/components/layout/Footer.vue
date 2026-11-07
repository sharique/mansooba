<template>
  <footer class="px-4 lg:px-6 py-3 border-t border-base-300 bg-base-100 flex justify-center lg:justify-end shrink-0">
    <AppSourceNotice />
  </footer>
</template>

<script setup lang="ts">
// Lets any signed-in user find this instance's source and licence (AGPL-3.0
// section 13). The unauthenticated pages show the same notice in their corner.
import AppSourceNotice from '~/components/common/AppSourceNotice.vue'
</script>
