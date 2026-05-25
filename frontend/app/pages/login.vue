<template>
  <div class="min-h-screen flex items-center justify-center bg-base-200">
    <div class="card w-96 bg-base-100 shadow-xl">
      <div class="card-body">
        <h2 class="card-title">Sign in</h2>
        <AuthLoginForm @success="navigateTo('/')" />
        <p class="text-sm text-center mt-2">
          No account?
          <NuxtLink to="/register" class="link">Register</NuxtLink>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
</script>
