<template>
  <div class="min-h-screen flex items-center justify-center bg-base-200">
    <div class="card w-96 bg-base-100 shadow-xl">
      <div class="card-body">
        <h2 class="card-title">Create account</h2>
        <AuthRegisterForm @success="navigateTo('/projects')" />
        <p class="text-sm text-center mt-2">
          Already have an account?
          <NuxtLink to="/login" class="link">Sign in</NuxtLink>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
</script>
