<template>
  <div class="min-h-screen grid lg:grid-cols-2">
    <!-- brand panel -->
    <div class="hidden lg:flex flex-col justify-center px-12 bg-neutral text-neutral-content">
      <div class="flex items-center gap-3 mb-6">
        <span class="inline-block w-9 h-9 rounded-lg bg-primary" />
        <span class="text-2xl font-bold">Mansooba</span>
      </div>
      <h1 class="text-3xl font-bold leading-tight">Plan, track, and ship your team's work.</h1>
      <p class="opacity-70 mt-3 max-w-sm">Boards, sprints, and reports — your whole project on one desk.</p>
    </div>

    <!-- form panel -->
    <div class="flex items-center justify-center bg-base-200 p-6">
      <div class="card w-full max-w-sm bg-base-100 shadow-xl border border-base-300">
        <div class="card-body">
          <h2 class="card-title">Sign in</h2>
          <AuthLoginForm @success="navigateTo('/')" />
          <p class="text-sm text-center mt-2">
            No account?
            <NuxtLink to="/register" class="link link-primary">Register</NuxtLink>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
</script>
