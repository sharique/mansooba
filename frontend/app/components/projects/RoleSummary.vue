<template>
  <section class="rounded-box border border-base-300 bg-base-100 p-4 mb-6" aria-labelledby="role-summary-title">
    <h2 id="role-summary-title" class="font-semibold mb-3">What each role can do</h2>
    <ul class="grid gap-2 sm:grid-cols-3">
      <li
        v-for="r in roles"
        :key="r.role"
        :data-role="r.role"
        class="rounded-lg border p-3"
        :class="r.role === myRole ? 'border-primary bg-primary/10' : 'border-base-300'"
      >
        <div class="flex items-center gap-2 mb-1">
          <span class="font-medium capitalize">{{ r.role }}</span>
          <span v-if="r.role === myRole" class="badge badge-primary badge-sm">Your role</span>
        </div>
        <p class="text-sm text-base-content/70">{{ r.description }}</p>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
// Wording only. What the signed-in person may actually do comes from the server's
// `permissions`; this just explains the three role names.
const roles = [
  { role: 'viewer', description: 'Can read everything, change nothing' },
  { role: 'member', description: 'Can work on tasks, comments, labels and files' },
  { role: 'admin', description: 'Can also manage sprints, people and the project' },
]

const { myRole } = useProjectPermissions()
</script>
